// Package subscription 协调官方 Codex 登录与账号元数据，授权令牌由系统凭证库持有。
package subscription

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/infrastructure/codex"
)

const loginTimeout = 10 * time.Minute
const operationTimeout = 30 * time.Second

// Input 只接受账号标识、显示名称与固定授权方式。
type Input struct {
	ProviderID string `json:"providerId"`
	Name       string `json:"name"`
	Mode       string `json:"mode"`
	AccountID  string `json:"accountId,omitempty"`
}

// State 不含令牌；URL 与用户码只用于进行中的官方授权。
type State struct {
	ID             string        `json:"id"`
	State          string        `json:"state"`
	URL            string        `json:"url"`
	UserCode       string        `json:"userCode"`
	ErrorCode      string        `json:"errorCode"`
	ExpiresAt      string        `json:"expiresAt"`
	Account        *account.Item `json:"account,omitempty"`
	CleanupWarning bool          `json:"cleanupWarning"`
}

type session struct {
	view    State
	input   Input
	profile string
	client  codex.Client
	loginID string
	cancel  context.CancelFunc
	done    chan struct{}
}
type Factory func(context.Context, string) (codex.Client, error)

// Service 一次只允许一个授权会话，避免官方回调端口及重复提交冲突。
type Service struct {
	mu                sync.Mutex
	operations        operationGate
	accounts          *account.Service
	root              string
	factory           Factory
	readToken         func(string, string, func([]byte, string) error) error
	current           *session
	history           map[string]State
	onChange          func()
	closed            bool
	quotaObservations map[string]quotaObservation
}

// New 使用按运行环境隔离的固定授权配置根目录。
func New(accounts *account.Service, root string, factory Factory) *Service {
	if factory == nil {
		factory = codex.Open
	}
	return &Service{accounts: accounts, root: root, factory: factory, readToken: codex.UseToken, history: map[string]State{}}
}
func failure(code string) error { return errors.New("subscription: " + code) }
func randomID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", failure("initialization_failed")
	}
	return hex.EncodeToString(value[:]), nil
}
func validID(value string) bool {
	if len(value) != 32 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func (s *Service) profilePath(profile string) (string, error) {
	if !filepath.IsAbs(s.root) || !validID(profile) {
		return "", failure("invalid_input")
	}
	path := filepath.Join(filepath.Clean(s.root), profile)
	relative, err := filepath.Rel(s.root, path)
	if err != nil || relative != profile {
		return "", failure("invalid_input")
	}
	return path, nil
}
func (s *Service) open(ctx context.Context, profile string) (codex.Client, error) {
	path, err := s.profilePath(profile)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		return nil, failure("initialization_failed")
	}
	return s.factory(ctx, path)
}

// OnChange 只发送状态变更信号，事件中不携带授权位置或个人信息。
func (s *Service) OnChange(listener func()) { s.mu.Lock(); s.onChange = listener; s.mu.Unlock() }
func (s *Service) changed() {
	s.mu.Lock()
	listener := s.onChange
	s.mu.Unlock()
	if listener != nil {
		listener()
	}
}

// Read 使用随机会话 ID 获取有限历史状态。
func (s *Service) Read(id string) (State, error) {
	if !validID(id) {
		return State{}, failure("invalid_input")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current != nil && s.current.view.ID == id {
		return s.current.view, nil
	}
	if value, ok := s.history[id]; ok {
		return value, nil
	}
	return State{}, failure("session_not_found")
}

func safeCode(err error) string { return ErrorCode(err) }
