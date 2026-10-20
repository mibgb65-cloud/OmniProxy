// Package proxy 实现独立于桌面、数据库和提供商的本机 HTTP 代理。
package proxy

import (
	"context"
	"io"
	"log/slog"
	"time"
)

const (
	MaxRequestBytes = 2 << 20
	// Responses 图片以 Base64 内嵌；使用独立有界容量，不放大普通 Chat Completions。
	MaxResponsesRequestBytes = 16 << 20
	// 首事件会回显工具定义；订阅推断和 Responses 两种传输使用一致的有界容量。
	MaxResponsesEventBytes = 2 << 20
	// 百万上下文的文本及 JSON 编码可能超过 2 MiB，Messages 使用独立的有界容量。
	MaxMessagesRequestBytes = 16 << 20
	// 对齐 Messages 官方的单请求消息上限；工具结果转换后也不能无限膨胀。
	MaxConversationMessages = 100000
	maxResponseBytes        = 8 << 20
	maxStreamBytes          = 32 << 20
)

// Request 保留协议正文；只解析代理需要的模型和流式标志。
type Request struct {
	CodexMetadata CodexMetadata
	// Compaction 仅由固定 HTTP 端点设置，不接受正文声明或跨协议伪造。
	Compaction   bool
	Candidates   []AccountCandidate
	FixedAccount bool
	// 仅有界保留真实客户端标识供要求工具身份的上游使用，不包含认证头或进入日志。
	ClientUserAgent  string
	AccountUpdatedAt string
	Trace            *Trace
	Protocol         string
	// 只由网关完成 Messages → Chat 转换后设置，不能从客户端 JSON 赋值。
	ConvertedFromMessages bool
	Body                  []byte
	Model                 string
	Stream                bool
	// 路由目标由受信解析器填入，HTTP 正文不能指定这些字段。
	ProviderID string
	AccountID  string
}

// Response 只允许提供商返回有限的非敏感响应元数据。
type Response struct {
	// 仅固定提供商确认首个完整事件明确拒绝执行时设置，不用于一般流中错误。
	PreExecutionQuotaRejected bool
	QuotaResetAt              time.Time
	// ContentTypeInferred 只标记提供商已验证并补全的缺失类型，不放宽 Core 校验。
	ContentTypeInferred bool
	// 类型推断仅携带固定原因和有界预读大小，禁止携带响应正文或动态错误消息。
	ContentTypeInferenceReason string
	ContentTypeInferenceBytes  int
	StatusCode                 int
	ContentType                string
	RetryAfter                 string
	Body                       io.Reader
}

// Upstream 在 consume 完成前持有响应资源，不能向 Core 暴露凭证。
type Upstream interface {
	Forward(context.Context, Request, func(Response) error) error
}

// RequestRouter 在白名单检查前解析别名；未实现此能力的上游保持原行为。
type RequestRouter interface {
	RouteRequest(context.Context, Request) (Request, error)
}

// ModelCatalog 动态提供已配置别名，不把配置读取失败解释为无模型。
type ModelCatalog interface {
	AvailableModels(context.Context) ([]string, error)
}

// Status 是无敏感数据的运行状态快照。
type Status struct {
	State     string `json:"state"`
	Address   string `json:"address"`
	Port      int    `json:"port"`
	StartedAt string `json:"startedAt"`
	ErrorCode string `json:"errorCode"`
}

// Options 配置代理的资源边界；Port 为 0 时由系统分配，供隔离测试使用。
type Options struct {
	FailoverEnabled  func(context.Context) (bool, error)
	WebSocketAllowed func(context.Context) (bool, error)
	Observer         RequestObserver
	Authorize        func(context.Context, string) bool
	Port             int
	RequestTimeout   time.Duration
	MaxConcurrent    int
	Upstream         Upstream
	Models           []string
	Logger           *slog.Logger
}

// Error 的 Code 可用于 UI 和协议错误；禁止放入原始上游正文、凭证或请求内容。
type Error struct {
	RetryAfter   string
	Code         string
	StatusCode   int
	imageContent bool
}

func (e *Error) Error() string { return "gateway: " + e.Code }

// Failure 创建安全的边界错误。
func Failure(code string, status int) error { return &Error{Code: code, StatusCode: status} }
