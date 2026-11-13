package dumate

import "errors"

// 稳定的错误码；界面和日志只使用这些码，不回显上游原文。
const (
	// CodeSignInExpired 表示 Cookie 登录态失效，需要用户重新导入或登录。
	CodeSignInExpired = "sign_in_expired"
	// CodeConsoleUnavailable 表示控制台网络或 HTTP 失败。
	CodeConsoleUnavailable = "console_unavailable"
	// CodeInvalidResponse 表示控制台返回了无法识别的结构。
	CodeInvalidResponse = "invalid_response"
	// CodeReauthRequired 表示 gateway token 被上游拒绝，需要重新换取。
	CodeReauthRequired = "reauth_required"
	// CodeCredentialUnavailable 表示本地受保护凭据不可用。
	CodeCredentialUnavailable = "credential_unavailable"
	// CodeRateLimited 表示控制台或网关限流。
	CodeRateLimited = "rate_limited"
	// CodeRejected 表示控制台以业务码拒绝了请求（例如当日签到不可领）。
	CodeRejected = "request_rejected"
)

// Error 是不含上游原文的安全错误。
type Error struct {
	Code string
	// SignIn 为 true 时表示登录态本身失效，而不是接口暂时不可用。
	SignIn bool
}

func (e *Error) Error() string { return "dumate: " + e.Code }

// Failure 构造固定错误码。
func Failure(code string) *Error { return &Error{Code: code} }

// SignInFailure 构造登录态失效错误。
func SignInFailure() *Error { return &Error{Code: CodeSignInExpired, SignIn: true} }

// ErrorCode 返回错误中的稳定码；其他错误返回空串。
func ErrorCode(err error) string {
	var failure *Error
	if errors.As(err, &failure) {
		return failure.Code
	}
	return ""
}

// SignInExpired 报告错误是否意味着需要重新登录。
func SignInExpired(err error) bool {
	var failure *Error
	return errors.As(err, &failure) && failure.SignIn
}
