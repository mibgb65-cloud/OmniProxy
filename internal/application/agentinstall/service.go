// Package agentinstall 管理客户端安装状态和单个后台安装任务。
package agentinstall

import (
	"context"
	"sync"
	"time"

	runtime "omniproxy/internal/infrastructure/agentinstall"
)

// Backend 允许测试替换检测和安装，测试不会修改本机软件。
type Backend interface {
	Available() bool
	Inspect(context.Context, string) runtime.Detection
	Install(context.Context, string, func(runtime.Progress)) error
}

// Item 仅返回安装元数据和稳定状态，不暴露安装器输出。
type Item struct {
	ID            string            `json:"id"`
	Detection     runtime.Detection `json:"detection"`
	InstallState  string            `json:"installState"`
	ErrorCode     string            `json:"errorCode"`
	OfficialURL   string            `json:"officialURL"`
	CanInstall    bool              `json:"canInstall"`
	Progress      *runtime.Progress `json:"progress,omitempty"`
	StartedAt     string            `json:"startedAt,omitempty"`
	FinishedAt    string            `json:"finishedAt,omitempty"`
	Operation     string            `json:"operation,omitempty"`
	UpdateVersion string            `json:"updateVersion,omitempty"`
}

// Snapshot 将安装器可用性与各客户端状态一次性提供给界面。
type Snapshot struct {
	Items              []Item `json:"items"`
	Busy               bool   `json:"busy"`
	InstallerAvailable bool   `json:"installerAvailable"`
}

// Service 一次只运行一个安装任务，结果由真实检测确定。
type Service struct {
	backend        Backend
	mu             sync.Mutex
	refreshMu      sync.Mutex
	items          map[string]Item
	initialized    bool
	active         string
	closed         bool
	cancel         context.CancelFunc
	done           chan struct{}
	installTimeout time.Duration
	afterInstalled func(string, runtime.Detection)
	lastUpdate     *checkedUpdate
}

// New 创建安装协调服务，不触发检测或安装。
func New(backend Backend) *Service {
	s := &Service{backend: backend, items: map[string]Item{}, installTimeout: 20 * time.Minute}
	for _, product := range runtime.Products() {
		s.items[product.ID] = Item{ID: product.ID, Detection: runtime.Detection{State: "unknown"}, InstallState: "idle", OfficialURL: product.OfficialURL, CanInstall: product.PackageID != ""}
	}
	return s
}

// OnInstalled 在安装成功后同步应用中的客户端版本能力。
func (s *Service) OnInstalled(fn func(string, runtime.Detection)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.afterInstalled = fn
}

// List 首次读取时检测，轮询后台任务时只返回缓存，不反复启动检测进程。
func (s *Service) List(ctx context.Context) Snapshot {
	s.mu.Lock()
	initialized := s.initialized
	s.mu.Unlock()
	if !initialized {
		return s.Refresh(ctx)
	}
	return s.snapshot()
}

// Refresh 只做本机只读检测；正在安装时保留任务快照。
func (s *Service) Refresh(ctx context.Context) Snapshot {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()
	s.mu.Lock()
	busy := s.active != ""
	s.mu.Unlock()
	if busy {
		return s.snapshot()
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	type result struct {
		id        string
		detection runtime.Detection
	}
	results := make(chan result, len(runtime.Products()))
	var group sync.WaitGroup
	for _, product := range runtime.Products() {
		group.Add(1)
		go func(id string) { defer group.Done(); results <- result{id, s.backend.Inspect(ctx, id)} }(product.ID)
	}
	group.Wait()
	close(results)
	s.mu.Lock()
	for result := range results {
		if result.id == s.active {
			continue
		}
		item := s.items[result.id]
		item.Detection = result.detection
		if item.Detection.State == "installed" && item.InstallState == "failed" && (item.Operation != "update" || runtime.DesktopVersionAtLeast(item.Detection.Version, item.UpdateVersion)) {
			item.InstallState, item.ErrorCode = "completed", ""
		}
		if item.Detection.State != "installed" && item.InstallState == "completed" {
			item.InstallState, item.ErrorCode = "idle", ""
		}
		s.items[result.id] = item
	}
	s.initialized = true
	s.mu.Unlock()
	return s.snapshot()
}

func (s *Service) snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := Snapshot{Items: []Item{}, Busy: s.active != "", InstallerAvailable: s.backend.Available()}
	for _, product := range runtime.Products() {
		result.Items = append(result.Items, s.items[product.ID])
	}
	return result
}
