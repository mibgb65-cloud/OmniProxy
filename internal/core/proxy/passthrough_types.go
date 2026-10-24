package proxy

import (
	"context"
	"net/http"
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

// PassthroughSession 由宿主按账号策略实现：按顺序给出可用账号，并判定是否换号。
// Next 返回错误表示没有可用账号；Retryable 只应依据明确拒绝分类，不能凭猜测。
type PassthroughSession interface {
	Next(context.Context) (PassthroughAttempt, error)
	Retryable(PassthroughAttempt, int) bool
	Done(context.Context, PassthroughAttempt, int)
}

// PassthroughBinding 把一个透明转发请求交给宿主的账号与路由策略。
type PassthroughBinding interface {
	OpenPassthrough(context.Context, PassthroughRequest) (PassthroughSession, error)
}
