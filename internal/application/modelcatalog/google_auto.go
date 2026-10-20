package modelcatalog

import (
	"context"
	"errors"
	"time"

	"omniproxy/internal/application/account"
)

const googleSyncTimeout = 30 * time.Second

// GoogleAutoSync 串行发现 Google 订阅目录，并合并在途通知，避免授权回调阻塞或重复并发请求。
type GoogleAutoSync struct {
	source  *Service
	refresh func(context.Context) error
	report  func(error)
	pending chan struct{}
	cancel  context.CancelFunc
	done    chan struct{}
}

// StartGoogleAutoSync 在后台执行首次同步；不依赖账号页面是否打开，也不启动网关。
func StartGoogleAutoSync(ctx context.Context, source *Service, refresh func(context.Context) error, report func(error)) *GoogleAutoSync {
	ctx, cancel := context.WithCancel(ctx)
	s := &GoogleAutoSync{source: source, refresh: refresh, report: report, pending: make(chan struct{}, 1), cancel: cancel, done: make(chan struct{})}
	s.Request()
	go s.run(ctx)
	return s
}

// Request 合并待处理通知；失败不自行重试，由下一次授权／刷新或用户手动操作重新触发。
func (s *GoogleAutoSync) Request() {
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
func (s *GoogleAutoSync) Close() {
	s.cancel()
	<-s.done
}

func (s *GoogleAutoSync) run(ctx context.Context) {
	defer close(s.done)
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.pending:
			if ctx.Err() != nil {
				return
			}
			err := s.sync(ctx)
			if ctx.Err() == nil && s.report != nil {
				s.report(err)
			}
		}
	}
}

func (s *GoogleAutoSync) sync(ctx context.Context) error {
	providers, err := s.source.providers.List(ctx)
	if err != nil {
		return ErrUnavailable
	}
	enabled := make(map[string]bool)
	for _, provider := range providers {
		enabled[provider.ID] = provider.Enabled && provider.Kind == "google"
	}
	accounts, err := s.source.accounts.List(ctx)
	if err != nil {
		return ErrUnavailable
	}
	var result error
	for _, item := range accounts {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !enabled[item.ProviderID] || !item.Enabled || item.AuthMethod != account.AuthSubscription || item.AuthState != "authenticated" {
			continue
		}
		operation, cancel := context.WithTimeout(ctx, googleSyncTimeout)
		_, syncError := s.source.Sync(operation, item.ID)
		cancel()
		result = errors.Join(result, syncError)
	}
	// 账号退出或删除后也要复核受管目录；空目录、限选与外部改动的保护仍由配置层决定。
	if ctx.Err() == nil && s.refresh != nil {
		result = errors.Join(result, s.refresh(ctx))
	}
	return result
}
