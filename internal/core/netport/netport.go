// Package netport 判断本机回环端口是否可用，并为被占用的固定端口挑选替代端口。
//
// 网关只监听 127.0.0.1（见 internal/core/proxy），因此可用性的判定标准就是
// 「能否在 127.0.0.1 上成功监听一次」，不依赖操作系统之外的端口表。
package netport

import (
	"errors"
	"fmt"
	"net"
	"strconv"
)

// ErrNoAvailablePort 表示给定区间内没有可监听的端口。
var ErrNoAvailablePort = errors.New("netport: no available port in range")

// MaxPort 是 TCP 端口上限。
const MaxPort = 65535

// Available 报告端口当前能否在 127.0.0.1 上监听。
//
// 已被其它进程监听、被 Windows 保留（Hyper-V 动态区间等）或越界的端口都返回
// false：这些情况下网关同样无法监听，没有必要区分原因。
func Available(port int) bool {
	if port < 1 || port > MaxPort {
		return false
	}
	listener, err := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return false
	}
	// 探测端口立即释放；调用方随后自行监听，期间的竞争由监听失败路径处理。
	return listener.Close() == nil
}

// NextAvailable 从 from 开始（含）在到 to（含）的区间内寻找第一个可用端口。
func NextAvailable(from, to int) (int, error) {
	if from < 1 {
		from = 1
	}
	if to > MaxPort {
		to = MaxPort
	}
	for port := from; port <= to; port++ {
		if Available(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("%w: %d-%d", ErrNoAvailablePort, from, to)
}
