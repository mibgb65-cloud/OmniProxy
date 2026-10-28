package wails

import (
	"context"
	"errors"
	"fmt"

	"omniproxy/internal/core/netport"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/platform/profile"
)

// proxyPortSearchSpan 是端口被占用时自动避让的搜索宽度：只在用户选择的端口
// 之后的小区间内寻找替代端口，避免悄悄跳到毫不相关的端口。
const proxyPortSearchSpan = 20

// ProxyPortState 描述本地网关端口的当前设置与占用状态。
type ProxyPortState struct {
	// Port 是当前生效的监听端口。
	Port int `json:"port"`
	// DefaultPort 是该 profile 未做改动时使用的端口。
	DefaultPort int `json:"defaultPort"`
	// Available 表示端口现在可以被网关监听（已由本应用监听时也为 true）。
	Available bool `json:"available"`
	// Running 表示网关正在监听。
	Running bool `json:"running"`
	// EnvLocked 表示 OMNIPROXY_PROXY_PORT 固定了端口，界面只能读取。
	EnvLocked bool `json:"envLocked"`
}

// ProxyPortUpdate 是保存端口后的结果，包含自动避让与受管客户端的处理情况。
type ProxyPortUpdate struct {
	State ProxyPortState `json:"state"`
	// Requested 是用户请求的端口；0 表示恢复默认端口。
	Requested int `json:"requested"`
	// Adjusted 表示请求端口被占用，已自动改用 State.Port。
	Adjusted bool `json:"adjusted"`
	// Updated 是地址已随端口改写的受管客户端。
	Updated []string `json:"updated"`
	// Manual 是需要用户重新一键配置的客户端。
	Manual []string `json:"manual"`
}

// ProxyPort 返回本地网关端口的当前设置。
func (s *AppService) ProxyPort() (ProxyPortState, error) {
	if s.prof == nil {
		return ProxyPortState{}, errors.New("gateway: unavailable")
	}
	return s.proxyPortState(), nil
}

// SetProxyPort 保存并立即应用新的网关端口。
//
// 端口被占用时自动避让到同区间内第一个可用端口（见 ProxyPortUpdate.Adjusted）；
// 端口由 OMNIPROXY_PROXY_PORT 固定时拒绝修改，避免界面与环境变量互相覆盖。
func (s *AppService) SetProxyPort(port int) (ProxyPortUpdate, error) {
	if s.prof == nil {
		return ProxyPortUpdate{}, errors.New("gateway: unavailable")
	}
	if profile.ProxyPortEnvOverride() {
		return ProxyPortUpdate{}, errors.New("gateway: proxy port is fixed by OMNIPROXY_PROXY_PORT")
	}
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	return s.applyProxyPort(context.Background(), port)
}

func (s *AppService) proxyPortState() ProxyPortState {
	state := ProxyPortState{
		Port:        s.prof.ProxyPort,
		DefaultPort: profile.DefaultProxyPort(s.prof.Kind),
		EnvLocked:   profile.ProxyPortEnvOverride(),
	}
	if s.gateway != nil && s.gateway.Info().Runtime.State == "running" {
		// 自己持有的端口当然无法再被探测到，但它是可用的。
		state.Running, state.Available = true, true
		return state
	}
	state.Available = netport.Available(state.Port)
	return state
}

// applyProxyPort 解析目标端口、避让冲突、持久化偏好并同步运行时。
//
// 顺序固定为「挑端口 → 停止监听 → 落盘 → 换端口 → 改写受管客户端 → 恢复监听」：
// 落盘失败时保持原端口继续服务，客户端改写只在落盘成功后执行，避免出现客户端
// 指向新端口而网关仍在旧端口的状态。
func (s *AppService) applyProxyPort(ctx context.Context, requested int) (ProxyPortUpdate, error) {
	update := ProxyPortUpdate{Requested: requested}
	target := requested
	if target == 0 {
		target = profile.DefaultProxyPort(s.prof.Kind)
	}
	if target < 1 || target > 65535 {
		return update, fmt.Errorf("gateway: invalid port %d", requested)
	}
	if !netport.Available(target) {
		next, err := netport.NextAvailable(target, target+proxyPortSearchSpan)
		if err != nil {
			return update, fmt.Errorf("gateway: port %d is unavailable: %w", target, err)
		}
		target, update.Adjusted = next, true
	}
	old := s.prof.ProxyPort
	running := s.gateway != nil && s.gateway.Info().Runtime.State == "running"
	// 端口未变化时不打断在途请求：运行中的同一个端口本来就是我们自己持有的。
	if old == target && running {
		update.State = s.proxyPortState()
		return update, nil
	}
	if running {
		if _, err := s.GatewayStop(); err != nil {
			return update, err
		}
	}
	if err := profile.SetProxyPort(s.prof.Kind, target); err != nil {
		if running {
			if _, startErr := s.GatewayStart(); startErr != nil && s.log != nil {
				s.log.Error("gateway restart failed after port persistence error",
					"module", "gateway", "code", "restart_failed")
			}
		}
		return update, err
	}
	s.prof.ProxyPort = target
	if s.gateway != nil {
		if _, err := s.gateway.SetPort(ctx, target); err != nil {
			// 运行中的网关已被停止：尽力恢复监听，避免为了换端口而丢掉服务。
			if running {
				if _, startErr := s.GatewayStart(); startErr != nil && s.log != nil {
					s.log.Error("gateway restart failed after port switch error",
						"module", "gateway", "code", "restart_failed")
				}
			}
			update.State = s.proxyPortState()
			return update, err
		}
	}
	if s.agents != nil {
		s.agents.SetPort(target)
		if old != target {
			rebase, err := s.agents.RebasePort(ctx, old, target)
			update.Updated, update.Manual = rebase.Updated, rebase.Manual
			if err != nil && s.log != nil {
				s.log.Warn("managed client addresses were not fully rebased",
					"module", "agentconfig", "code", "port_rebase_failed")
			}
		}
	}
	if running {
		if _, err := s.GatewayStart(); err != nil {
			update.State = s.proxyPortState()
			return update, err
		}
	}
	update.State = s.proxyPortState()
	if s.log != nil {
		s.log.Info("gateway port updated", "module", "gateway",
			"old_port", old, "new_port", target, "adjusted", update.Adjusted)
	}
	return update, nil
}

// isPortUnavailable 区分「端口被占用」与其它启动失败：只有前者才值得自动避让。
func isPortUnavailable(err error) bool {
	var failure *proxy.Error
	return errors.As(err, &failure) && failure.Code == "port_unavailable"
}
