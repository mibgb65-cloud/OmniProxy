package proxy

import (
	"context"
	"net/http"
	"time"
)

// PassthroughRequest 是交给宿主账号策略的上游转发输入。
// Header 是客户端原始头部的副本；Body 保持客户端编码，不做解析或重写。
type PassthroughRequest struct {
	Header http.Header
	Method string
	// Path 是折叠入口前缀之后的原始路径，仍带 /v1 等版本段。
	Path   string
	Query  string
	Client string
	Body   []byte
}

// PassthroughAttempt 是一次尝试的上游基址、鉴权头与记账身份。
// Header 只放网关替换的鉴权头；Core 在请求结束后清零。
type PassthroughAttempt struct {
	Header    http.Header
	Snapshot  TargetSnapshot
	BaseURL   string
	AccountID string
	Model     string
}

// PassthroughOutcome 描述一次透明转发尝试的结束原因，由宿主换算成冷却或额度阻断。
// UpstreamQuotaRejected 只在首个完整事件明确携带执行前额度拒绝时设置。
type PassthroughOutcome struct {
	// RetryAfter 是上游原样给出的重试提示，仅在状态码拒绝时参与冷却。
	RetryAfter string
	// QuotaResetAt 是执行前额度拒绝的重置时间；未知时为零值。
	QuotaResetAt time.Time
	// StatusCode 是本次尝试实际收到的上游状态码，不因分类而改写。
	StatusCode            int
	UpstreamQuotaRejected bool
}

// PassthroughSession 由宿主按账号策略实现：按顺序给出可用账号，并判定是否换号。
// Next 返回错误表示没有可用账号；Retryable 只应依据明确拒绝分类，不能凭猜测。
type PassthroughSession interface {
	Next(context.Context) (PassthroughAttempt, error)
	Retryable(PassthroughAttempt, int) bool
	// Switchable 报告宿主是否允许换号：固定账号或关闭故障切换时为 false。
	// 执行前额度拒绝仍会登记，只是不再换到下一个候选。
	Switchable() bool
	Done(context.Context, PassthroughAttempt, PassthroughOutcome)
}

// PassthroughBinding 把一个透明转发请求交给宿主的账号与路由策略。
type PassthroughBinding interface {
	OpenPassthrough(context.Context, PassthroughRequest) (PassthroughSession, error)
}
