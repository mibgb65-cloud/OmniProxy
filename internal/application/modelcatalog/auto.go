package modelcatalog

import (
	"context"
	"errors"
	"sync"
	"time"

	"omniproxy/internal/application/account"
)

// autoSyncTimeout 是单个账号目录同步的上限，避免后台任务无限占用官方组件。
const autoSyncTimeout = 30 * time.Second

// SyncStatus 是后台目录同步的只读内存状态，不包含账号身份或上游正文。
type SyncStatus struct {
	Running   bool   `json:"running"`
	Stale     int    `json:"stale"`
	LastError string `json:"lastError,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// AutoSync 串行发现目录，合并并发通知；staleOnly 时只处理当前版本没有目录的账号。
type AutoSync struct {
	source    *Service
	kind      string
	staleOnly bool
	// apiKey 表示该服务商按 API Key 账号发现目录（OpenAI、HowToFix 等）；
	// 为 false 时只处理已授权的订阅账号（如 Google 订阅）。
	apiKey  bool
	refresh func(context.Context) error
	report  func(error)
	pending chan struct{}
	cancel  context.CancelFunc
	done    chan struct{}

	mu        sync.Mutex
	attempted map[string]bool
	running   bool
	stale     int
	lastError string
	updatedAt string
}

// StartAutoSync 在后台执行首次扫描；不依赖账号页面是否打开，也不启动网关。
func StartAutoSync(ctx context.Context, source *Service, kind string, staleOnly bool, refresh func(context.Context) error, report func(error)) *AutoSync {
	return startAutoSync(ctx, source, kind, staleOnly, false, refresh, report)
}

// StartAPIKeyAutoSync 在后台补齐当前版本没有目录的 API Key 账号；失败保留旧快照，
// 由下一次配置变更或手动同步重试。
func StartAPIKeyAutoSync(ctx context.Context, source *Service, kind string, refresh func(context.Context) error, report func(error)) *AutoSync {
	return startAutoSync(ctx, source, kind, true, true, refresh, report)
}

func startAutoSync(ctx context.Context, source *Service, kind string, staleOnly, apiKey bool, refresh func(context.Context) error, report func(error)) *AutoSync {
	ctx, cancel := context.WithCancel(ctx)
	s := &AutoSync{
		source: source, kind: kind, staleOnly: staleOnly, apiKey: apiKey, refresh: refresh, report: report,
		pending: make(chan struct{}, 1), cancel: cancel, done: make(chan struct{}),
		attempted: map[string]bool{},
	}
	s.Request()
	go s.run(ctx)
	return s
}

// Request 合并待处理通知；失败不自行重试，由下一次授权／刷新或手动操作重新触发。
func (s *AutoSync) Request() {
	select {
	case <-s.done:
		return
	default:
	}
	select {
	case s.pending <- struct{}{}:
	default:
	}
}

// Close 取消在途请求并等待后台任务退出；可重复调用。
func (s *AutoSync) Close() {
	s.cancel()
	<-s.done
}

// Status 返回最近一次扫描的只读状态。
func (s *AutoSync) Status() SyncStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return SyncStatus{Running: s.running, Stale: s.stale, LastError: s.lastError, UpdatedAt: s.updatedAt}
}

func (s *AutoSync) run(ctx context.Context) {
	defer close(s.done)
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.pending:
			if ctx.Err() != nil {
				return
			}
			modelErr, refreshErr := s.sync(ctx)
			s.finish(modelErr)
			if ctx.Err() == nil && s.report != nil {
				s.report(errors.Join(modelErr, refreshErr))
			}
		}
	}
}

func (s *AutoSync) sync(ctx context.Context) (error, error) {
	s.begin()
	providers, err := s.source.providers.List(ctx)
	if err != nil {
		return ErrUnavailable, nil
	}
	enabled := make(map[string]bool)
	for _, provider := range providers {
		enabled[provider.ID] = provider.Enabled && provider.Kind == s.kind
	}
	accounts, err := s.source.accounts.List(ctx)
	if err != nil {
		return ErrUnavailable, nil
	}
	stale := 0
	var result error
	for _, item := range accounts {
		if ctx.Err() != nil {
			return ctx.Err(), nil
		}
		subscriptionReady := item.AuthMethod == account.AuthSubscription && item.AuthState == "authenticated"
		// API Key 没有订阅登录状态，但同样需要官方模型快照才能参与路由。
		apiKeyReady := s.apiKey && item.AuthMethod == account.AuthAPIKey
		if !enabled[item.ProviderID] || !item.Enabled || !subscriptionReady && !apiKeyReady {
			continue
		}
		if s.staleOnly {
			version, current, stateErr := s.source.catalogCurrent(ctx, item.ID)
			if stateErr != nil {
				result = errors.Join(result, ErrUnavailable)
				continue
			}
			if current {
				continue
			}
			stale++
			if s.attemptedVersion(item.ID, version) {
				continue
			}
		}
		operation, cancel := context.WithTimeout(ctx, autoSyncTimeout)
		_, syncError := s.source.Sync(operation, item.ID)
		cancel()
		result = errors.Join(result, syncError)
	}
	if s.staleOnly {
		s.setStale(stale)
	}
	// 账号退出或删除后也要复核受管目录；空目录、限选与外部改动的保护仍由配置层决定。
	// 客户端刷新失败与账号目录失败分开返回：前者不应让模型目录页误报同步失败。
	if ctx.Err() == nil && s.refresh != nil {
		return result, s.refresh(ctx)
	}
	return result, nil
}

func (s *AutoSync) begin() {
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()
}

func (s *AutoSync) finish(err error) {
	s.mu.Lock()
	s.running = false
	s.updatedAt = time.Now().UTC().Format(time.RFC3339)
	if err == nil || errors.Is(err, context.Canceled) {
		s.lastError = ""
	} else {
		s.lastError = s.kind + "_auto_sync_failed"
	}
	s.mu.Unlock()
}

func (s *AutoSync) setStale(value int) {
	s.mu.Lock()
	s.stale = value
	s.mu.Unlock()
}

// attemptedVersion 在单次应用运行内按账号＋凭据版本去重，避免空目录反复启动官方组件。
func (s *AutoSync) attemptedVersion(id, version string) bool {
	key := id + "\x00" + version
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.attempted[key] {
		return true
	}
	if len(s.attempted) >= 4096 {
		s.attempted = map[string]bool{}
	}
	s.attempted[key] = true
	return false
}

// catalogCurrent 返回账号当前目录版本以及该版本是否已有任何模型行。
func (s *Service) catalogCurrent(ctx context.Context, id string) (string, bool, error) {
	var version string
	var current int
	err := s.db.QueryRowContext(ctx, `SELECT v.version,
		EXISTS(SELECT 1 FROM account_models m WHERE m.account_id=v.id AND m.account_version=v.version)
		FROM account_catalog_versions v WHERE v.id=?`, id).Scan(&version, &current)
	if err != nil {
		return "", false, err
	}
	return version, current != 0, nil
}
