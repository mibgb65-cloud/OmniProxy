package workbuddyai

import (
	"errors"

	"omniproxy/internal/core/proxy"
)

// 稳定的错误码；界面、日志和跨层错误只使用这些码，不回显上游原文。
const (
	// CodeSignInExpired 表示登录态失效（刷新被拒或令牌不可用），需要重新登录。
	CodeSignInExpired = "sign_in_expired"
	// CodeCredentialUnavailable 表示本地受保护凭据缺失或无法解密。
	CodeCredentialUnavailable = "credential_unavailable"
	// CodeUnauthorized 是上游 401：访问令牌被拒，调用方应先续期再重试。
	CodeUnauthorized = "unauthorized"
	// CodeChannelRejected 是业务码 11128：系统提示词或请求渠道被上游拒绝。
	CodeChannelRejected = "channel_rejected"
	// CodeSignatureInvalid 是业务码 11159：thinking 块签名与当前对话不匹配。
	CodeSignatureInvalid = "signature_invalid"
	// CodeModelNotEntitled 是业务码 11158：账号没有该模型权益。
	CodeModelNotEntitled = "model_not_entitled"
	// CodeUserAgentRefused 是业务码 10085：客户端指纹被上游拒绝。
	CodeUserAgentRefused = "user_agent_refused"
	// CodeRateLimited 是上游 429。
	CodeRateLimited = "rate_limited"
	// CodeUpstreamUnavailable 表示上游网络、5xx 或无法识别的业务码。
	CodeUpstreamUnavailable = "upstream_unavailable"
	// CodeInvalidResponse 表示上游返回了无法识别的结构；不得把未知当 0。
	CodeInvalidResponse = "invalid_response"
	// CodeRefreshRejected 表示续期被上游明确拒绝（不带 401 的拒绝）。
	CodeRefreshRejected = "refresh_rejected"
)

// Error 是不含上游原文的安全错误。
type Error struct {
	Code string
	// SignIn 为 true 时表示登录态本身失效，而不是接口暂时不可用。
	SignIn bool
}

func (e *Error) Error() string { return "workbuddy_ai: " + e.Code }

// Failure 构造固定错误码。
func Failure(code string) error { return &Error{Code: code} }

// signInFailure 构造需要用户重新登录的错误。
func signInFailure(code string) error { return &Error{Code: code, SignIn: true} }

// ErrorCode 返回错误中的稳定码；其他错误返回空串。
//
// 边界错误（401/429/网络失败与非法请求）用 proxy.Error 表达，本包的业务错误用
// Error 表达；两层都返回同一组码，调用方不必分辨来源。
func ErrorCode(err error) string {
	var failure *Error
	if errors.As(err, &failure) {
		return failure.Code
	}
	var boundary *proxy.Error
	if errors.As(err, &boundary) {
		return boundary.Code
	}
	return ""
}

// SignInExpired 报告错误是否意味着需要重新登录。
func SignInExpired(err error) bool {
	var failure *Error
	return errors.As(err, &failure) && failure.SignIn
}
