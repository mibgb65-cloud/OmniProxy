// Package workbuddy 管理 WorkBuddy AI（腾讯 CodeBuddy 计划国际版）的浏览器登录、
// 令牌续期、模型目录与 Credits 额度读取。
package workbuddy

import (
	"context"
	"sync"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/subscription"
	"omniproxy/internal/provider/workbuddyai"
)

const (
	// loginTimeout 是一次浏览器登录的总预算；上游插件同样按 5 分钟轮询。
	loginTimeout = 5 * time.Minute
	// operationTimeout 约束一次续期或额度读取。
	operationTimeout = 30 * time.Second
	// refreshMargin 是提前续期的余量；上游访问令牌寿命约 1 小时。
	refreshMargin = 5 * time.Minute
	// quotaFreshness 是内存额度快照可用于展示的时长。
	quotaFreshness = time.Hour
	// maxQuotas 限制内存快照条数。
	maxQuotas = 64
)

// Service 持有有限后台登录会话；长期令牌只保存在受保护账号存储。
type Service struct {
	mu       sync.Mutex
	ops      sync.Mutex
	accounts *account.Service
	client   *workbuddyai.Client
	current  *session
	onChange func()
	warn     func(message string, args ...any)
	closed   bool

	quotaMu sync.Mutex
	quotas  map[string]quotaSnapshot
}

type session struct {
	view   subscription.State
	input  subscription.Input
	cancel context.CancelFunc
	done   chan struct{}
}

type quotaSnapshot struct {
	quota     Quota
	fetchedAt time.Time
}

// New 创建独立订阅服务；测试可注入仅访问契约服务的客户端。
func New(accounts *account.Service, client *workbuddyai.Client) *Service {
	if client == nil {
		client = workbuddyai.New(nil)
	}
	return &Service{accounts: accounts, client: client, quotas: map[string]quotaSnapshot{}}
}

// OnChange 注册只发送变更信号的通知。
func (s *Service) OnChange(listener func()) {
	s.mu.Lock()
	s.onChange = listener
	s.mu.Unlock()
}

// SetWarn 注册失败诊断输出；登录在后台线程结束，调用方只能靠它拿到失败原因。
// 回调只接收稳定错误码与阶段，不接收令牌或上游原文。
func (s *Service) SetWarn(listener func(message string, args ...any)) {
	s.mu.Lock()
	s.warn = listener
	s.mu.Unlock()
}

// report 在登录失败时交给调用方的诊断钩子；没有钩子时静默（测试与单元场景）。
func (s *Service) report(stage, code string) {
	s.mu.Lock()
	listener := s.warn
	s.mu.Unlock()
	if listener != nil {
		listener("WorkBuddy AI sign-in failed", "module", "workbuddy", "stage", stage, "code", code)
	}
}

func (s *Service) notify() {
	s.mu.Lock()
	listener := s.onChange
	s.mu.Unlock()
	if listener != nil {
		listener()
	}
}

// Read 读取不含令牌的会话状态。
func (s *Service) Read(id string) (subscription.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil || id != s.current.view.ID {
		return subscription.State{}, workbuddyai.Failure("invalid_input")
	}
	view := s.current.view
	if view.Account != nil {
		item := *view.Account
		view.Account = &item
	}
	return view, nil
}

// Current 恢复相同提供商和账号的后台登录。
func (s *Service) Current(providerID, id string) (subscription.State, error) {
	if s.accounts.CheckWorkBuddyProvider(context.Background(), providerID) != nil || len(id) > 128 {
		return subscription.State{}, workbuddyai.Failure("invalid_input")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil || s.current.input.ProviderID != providerID || s.current.input.AccountID != id || !subscription.InProgress(s.current.view.State) {
		return subscription.State{}, nil
	}
	return s.current.view, nil
}

// Cancel 结束仍在进行中的登录，已完成的账号不会被取消删除。
func (s *Service) Cancel(id string) (subscription.State, error) {
	s.mu.Lock()
	if s.current == nil || s.current.view.ID != id {
		s.mu.Unlock()
		return subscription.State{}, workbuddyai.Failure("invalid_input")
	}
	job := s.current
	if subscription.InProgress(job.view.State) {
		job.cancel()
	}
	s.mu.Unlock()
	select {
	case <-job.done:
	case <-time.After(5 * time.Second):
		return subscription.State{}, workbuddyai.Failure("busy")
	}
	return s.Read(id)
}

// Close 在退出时取消并等待当前登录会话结束。
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

func (s *Service) recordQuota(id string, quota Quota) {
	s.quotaMu.Lock()
	defer s.quotaMu.Unlock()
	if len(s.quotas) >= maxQuotas {
		var oldest string
		var oldestAt time.Time
		for key, entry := range s.quotas {
			if oldest == "" || entry.fetchedAt.Before(oldestAt) {
				oldest, oldestAt = key, entry.fetchedAt
			}
		}
		delete(s.quotas, oldest)
	}
	s.quotas[id] = quotaSnapshot{quota: quota, fetchedAt: time.Now()}
}

func (s *Service) forgetQuota(id string) {
	s.quotaMu.Lock()
	delete(s.quotas, id)
	s.quotaMu.Unlock()
}
