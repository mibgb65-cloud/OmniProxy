// Package quotaactivation 自动激活已授权账号的未开始额度窗口，不改变路由或客户端会话。
package quotaactivation

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/application/subscription"
)

// Accounts 仅提供公开账号元数据，不向自动检测服务暴露凭据。
type Accounts interface {
	List(context.Context) ([]account.Item, error)
	Metadata(context.Context, string) (account.Item, error)
}

const (
	detectionInterval = 5 * time.Minute
	wakeDebounce      = 30 * time.Second
	activationTimeout = 60 * time.Second
)

// Providers 用于核对账号所属服务商的类型和启用状态。
type Providers interface {
	List(context.Context) ([]provider.Configuration, error)
}

// Quotas 读取由订阅服务验证身份后的官方窗口快照。
type Quotas interface {
	ReadQuota(context.Context, string) (subscription.Quota, error)
}

// Sender 固定使用指定账号发送一次最小请求。
type Sender interface {
	Send(context.Context, account.Item, SendPlan) error
}

// Status 只返回检测计数和安全错误，不暴露身份、凭据、提示词或原始响应。
type Status struct {
	Running   bool   `json:"running"`
	Checked   int    `json:"checked"`
	Sent      int    `json:"sent"`
	Activated int    `json:"activated"`
	Failed    int    `json:"failed"`
	Pending   int    `json:"pending"`
	UpdatedAt string `json:"updatedAt"`
	ErrorCode string `json:"errorCode"`
}

// Service 将串行检测、持久去重和生命周期统一在后台，前端只负责保存授权。
type Service struct {
	log          *slog.Logger
	db           *sql.DB
	accounts     Accounts
	providers    Providers
	quotas       Quotas
	sender       Sender
	checkMu      sync.Mutex
	mu           sync.Mutex
	status       Status
	changed      bool
	observations map[string]string
	cancel       context.CancelFunc
	activeCancel context.CancelFunc
	done         chan struct{}
	wake         chan struct{}
	force        atomic.Bool
	onChange     func()
	now          func() time.Time
}

// New 创建默认尚未运行的自动激活服务；不读取凭据或发送模型请求。
func New(db *sql.DB, accounts Accounts, providers Providers, quotas Quotas, sender Sender) *Service {
	return &Service{db: db, accounts: accounts, providers: providers, quotas: quotas, sender: sender, wake: make(chan struct{}, 1), now: time.Now, observations: map[string]string{}}
}

// codeError 携带稳定错误码，让状态与日志能说明具体原因，而不是笼统的失败。
type codeError struct{ code string }

func (e *codeError) Error() string { return "activation: " + e.code }

func failure(code string) error { return &codeError{code: code} }

// codeOf 取错误链里第一个稳定码；组合错误按顺序查找，识别不到时返回空串。
func codeOf(err error) string {
	for err != nil {
		if coded, ok := err.(*codeError); ok {
			return coded.code
		}
		switch wrapped := err.(type) {
		case interface{ Unwrap() []error }:
			for _, inner := range wrapped.Unwrap() {
				if code := codeOf(inner); code != "" {
					return code
				}
			}
			return ""
		case interface{ Unwrap() error }:
			err = wrapped.Unwrap()
		default:
			return ""
		}
	}
	return ""
}

// ErrAlreadyActive 表示发送前的复查已确认无需消息，调用方只更新额度快照。
var ErrAlreadyActive = errors.New("activation: already_active")

// ErrNotSent 表示失败发生在 HTTP 发送之前，可安全释放本次登记供后续检查。
var ErrNotSent = errors.New("activation: not_sent")

// ErrAwaitingQuota 表示发送前时序证据不足，只复查而非报告网络／授权失败。
var ErrAwaitingQuota = errors.New("activation: awaiting_quota")

// Status 返回线程安全的最近检测结果。
func (s *Service) Status() Status { s.mu.Lock(); defer s.mu.Unlock(); return s.status }

// OnChange 设置只包含变更信号的通知，避免将账号身份写入事件。
func (s *Service) OnChange(callback func()) { s.mu.Lock(); defer s.mu.Unlock(); s.onChange = callback }

// SetLogger 安装统一脱敏日志器；通常在后台启动前调用。
func (s *Service) SetLogger(log *slog.Logger) { s.mu.Lock(); defer s.mu.Unlock(); s.log = log }
