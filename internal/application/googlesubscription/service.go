// Package googlesubscription 管理与普通 API Key 隔离的 Google 订阅授权。
package googlesubscription

import (
	"context"
	"sync"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/subscription"
	"omniproxy/internal/provider/antigravity"
)

// Service 持有有限后台授权会话；长期凭证仅保存在受保护账号存储。
type Service struct {
	mu       sync.Mutex
	ops      sync.Mutex
	accounts *account.Service
	client   *antigravity.Client
	config   func() (antigravity.OAuthConfig, error)
	current  *session
	onChange func()
	closed   bool
}

type session struct {
	view   subscription.State
	input  subscription.Input
	cancel context.CancelFunc
	done   chan struct{}
}

// New 创建独立订阅服务，测试可注入仅访问契约服务的客户端。
func New(accounts *account.Service, client *antigravity.Client) *Service {
	if client == nil {
		client = antigravity.New(nil)
	}
	return &Service{accounts: accounts, client: client, config: antigravity.InstalledOAuthConfig}
}

// OnChange 注册只发送变更信号的通知。
func (s *Service) OnChange(listener func()) { s.mu.Lock(); s.onChange = listener; s.mu.Unlock() }

func (s *Service) notify() {
	s.mu.Lock()
	listener := s.onChange
	s.mu.Unlock()
	if listener != nil {
		listener()
	}
}

// Read 读取不含 token 的会话状态。
func (s *Service) Read(id string) (subscription.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil || id != s.current.view.ID {
		return subscription.State{}, antigravity.Failure("invalid_input")
	}
	view := s.current.view
	if view.Account != nil {
		item := *view.Account
		view.Account = &item
	}
	return view, nil
}

// Current 恢复相同提供商和账号的后台授权。
func (s *Service) Current(providerID, id string) (subscription.State, error) {
	if s.accounts.CheckGoogleProvider(context.Background(), providerID) != nil || len(id) > 128 {
		return subscription.State{}, antigravity.Failure("invalid_input")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil || s.current.input.ProviderID != providerID || s.current.input.AccountID != id || !subscription.InProgress(s.current.view.State) {
		return subscription.State{}, nil
	}
	return s.current.view, nil
}

// Cancel 结束仍在进行中的授权，完成后的账号不会被取消删除。
func (s *Service) Cancel(id string) (subscription.State, error) {
	s.mu.Lock()
	if s.current == nil || s.current.view.ID != id {
		s.mu.Unlock()
		return subscription.State{}, antigravity.Failure("invalid_input")
	}
	job := s.current
	if subscription.InProgress(job.view.State) {
		job.cancel()
	}
	s.mu.Unlock()
	select {
	case <-job.done:
	case <-time.After(5 * time.Second):
		return subscription.State{}, antigravity.Failure("busy")
	}
	return s.Read(id)
}

// Close 在退出时取消并等待当前回调释放端口。
func (s *Service) Close() {
	s.mu.Lock()
	s.closed = true
	job := s.current
	if job != nil {
		job.cancel()
	}
	s.mu.Unlock()
	if job != nil {
		<-job.done
	}
}

func (s *Service) busy(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed || s.current != nil && subscription.InProgress(s.current.view.State) && s.current.input.AccountID == id
}
