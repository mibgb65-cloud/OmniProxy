package workbuddyai

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"omniproxy/internal/core/netproxy"
	"omniproxy/internal/core/proxy"
)

const (
	// chatPath 是唯一的推理端点；上游只接受流式 Chat Completions。
	chatPath = "/v2/chat/completions"
	// configPath 是按账号发放的模型目录。
	configPath = "/v3/config"
	// creditsPath 是 Credits 资源总览（额度主接口）。
	creditsPath = "/billing/meter/get-user-resource-summary"
	// signInStatePath 发起浏览器登录并取回 state 与授权页地址。
	signInStatePath = "/v2/plugin/auth/state"
	// signInTokenPath 轮询登录是否完成并取回令牌。
	signInTokenPath = "/v2/plugin/auth/token"
	// refreshPath 用 refresh token 换新令牌。
	refreshPath = "/v2/plugin/auth/token/refresh"
	// accountPath 按 state 读取登录后的账号身份。
	accountPath = "/v2/plugin/login/account"
	// messagesPath 是国内版为 Claude 模型提供的原生 Anthropic Messages 端点。
	messagesPath = "/v1/messages"
	// 以下指纹来自桌面 CLI 内置的 openai 6.25.0 与 Electron 的 Node 版本。
	openAISDKVersion   = "6.25.0"
	nodeRuntimeVersion = "v22.21.1"
	// stainlessTimeout 是 OpenAI SDK 自带的请求超时（秒）。
	stainlessTimeout = "1200"

	// responseHeaderTimeout 允许思考型模型在首个分片前等待较久。
	responseHeaderTimeout = 3 * time.Minute
	// tlsHandshakeTimeout 与其它适配器保持一致。
	tlsHandshakeTimeout = 10 * time.Second
	// maxURLBytes 限制可注入基址的长度。
	maxURLBytes = 2048
	// maxClassifiedBytes 是错误正文的识别上限：只用于读固定业务码，不保存原文。
	maxClassifiedBytes = 8 << 10
	// maxSignInBytes 限制登录接口正文；它们都只是少量 JSON。
	maxSignInBytes = 64 << 10
	// maxStateBytes 限制登录 state 与生成的会话 ID 长度。
	maxStateBytes = 64
	// randomIDBytes 是会话/请求 ID 的随机字节数（hex 后 32 字符，≤64）。
	randomIDBytes = 16
)

// Client 只访问固定官方端点；禁止重定向与客户端头部透传。
type Client struct {
	http *http.Client
	base string
	// build 决定站点、登录平台标识、UA 与 Messages 能力；国际化与国内版共用本实现。
	build Build
	// 登录窗口内的 state 只用于 /v2/plugin/login/account：该接口按 state 选中
	// 用户刚在浏览器里选的账号，而 ReadIdentity 的契约签名只带 Token。
	stateMu sync.Mutex
	state   string
	stateAt time.Time
}

// New 构造国际版的隔离传输；可注入 transport 进行无真实凭据的契约测试。
func New(transport http.RoundTripper) *Client {
	return NewForBuild(International, transport)
}

// NewForBuild 按构建构造隔离传输；base 取 build.BaseURL。
func NewForBuild(build Build, transport http.RoundTripper) *Client {
	if transport == nil {
		standard := http.DefaultTransport.(*http.Transport).Clone()
		standard.ResponseHeaderTimeout = responseHeaderTimeout
		standard.TLSHandshakeTimeout = tlsHandshakeTimeout
		standard.MaxIdleConnsPerHost = 16
		transport = netproxy.Wrap(standard)
	}
	return &Client{
		http:  &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		base:  build.BaseURL,
		build: build,
	}
}

// NewClientAt 用指定基址构造国际版客户端；非 HTTPS 基址只允许回环地址，仅测试使用，
// 生产路径始终使用固定基址。
func NewClientAt(base string, transport http.RoundTripper) (*Client, error) {
	return NewClientAtFor(International, base, transport)
}

// NewClientAtFor 按构建与注入基址构造客户端（同样只允许回环 HTTP），用于契约测试。
func NewClientAtFor(build Build, base string, transport http.RoundTripper) (*Client, error) {
	validated, err := validateBase(base)
	if err != nil {
		return nil, err
	}
	client := NewForBuild(build, transport)
	client.base = validated
	return client, nil
}

// Forward 只接受 chat 协议，把调用方已构造好的正文按官方客户端形态（gzip 请求体、
// 固定签名头、OpenAI SDK 指纹）发往 /v2/chat/completions。
//
// 流式约定：上游只接受 stream=true。本适配器**不改写**正文里的 stream 字段，也不
// 补 stream_options——把正文改成流式是调用方（网关层）的责任；request.Stream 为
// false 时这里直接拒绝，避免把必定被上游拒绝的非流式请求发出去。
func (c *Client) Forward(ctx context.Context, cred Credentials, request proxy.Request, consume func(proxy.Response) error) (result error) {
	if request.Protocol != ChatProtocol {
		return proxy.Failure("unsupported_protocol", 400)
	}
	if !request.Stream {
		return proxy.Failure("unsupported_protocol", 400)
	}
	if !cred.Valid() || consume == nil {
		return proxy.Failure("invalid_credential", 503)
	}
	if len(request.Body) == 0 || len(request.Body) > proxy.MaxRequestBytes {
		return proxy.Failure("invalid_request", 400)
	}
	// 上游要求首条消息必须是 system：缺了会被 11128 拒绝，这里先补齐形态。
	prepared := prepareChatBody(request.Body)
	if len(prepared) == 0 {
		prepared = request.Body
	}
	if len(prepared) > proxy.MaxRequestBytes {
		return proxy.Failure("invalid_request", 400)
	}
	reply, err := c.sendChat(ctx, cred, prepared)
	if err != nil {
		return err
	}
	// 11128 是渠道校验拒绝了系统提示词。按官方客户端形态把系统提示词读成用户消息重试一次；
	// 仍被拒才作为 channel_rejected 返回。重试发生在推理之前，不产生计费。
	if reply.hasCode && reply.code == chatChannelRejected {
		if retry := systemAsUserBody(prepared); len(retry) > 0 && len(retry) <= proxy.MaxRequestBytes {
			_ = reply.response.Body.Close()
			next, retryErr := c.sendChat(ctx, cred, retry)
			if retryErr != nil {
				return retryErr
			}
			reply = next
		}
	}
	response := reply.response
	defer func() {
		if err := response.Body.Close(); err != nil && result == nil {
			result = proxy.Failure(CodeUpstreamUnavailable, http.StatusBadGateway)
		}
	}()
	switch response.StatusCode {
	case http.StatusUnauthorized:
		return proxy.Failure(CodeUnauthorized, http.StatusUnauthorized)
	case http.StatusTooManyRequests:
		return proxy.Failure(CodeRateLimited, http.StatusTooManyRequests)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		// 非 2xx：先按已识别的业务码（10085/11128/11159/11158）返回；识别不了的
		// 状态与正文原样交给网关决定，本适配器不替网关吞掉响应；已读前缀会拼回。
		if reply.hasCode {
			if mapped, known := businessFailure(reply.code); known {
				return mapped
			}
		}
		response.Body = replayedBody{Reader: io.MultiReader(bytes.NewReader(reply.prefix), response.Body), Closer: response.Body}
	}
	return consume(proxy.Response{
		StatusCode:  response.StatusCode,
		ContentType: response.Header.Get("Content-Type"),
		RetryAfter:  response.Header.Get("Retry-After"),
		Body:        response.Body,
	})
}

// chatReply 是一次对话请求的响应，以及非 2xx 时已读取的错误前缀与业务码。
type chatReply struct {
	response *http.Response
	prefix   []byte
	code     int
	hasCode  bool
}

// sendChat 发一次对话请求：gzip 正文、签名头、会话 ID，并对非 2xx 做有界业务码识别。
// 已读前缀留给调用方回放，避免把上游正文丢掉。
func (c *Client) sendChat(ctx context.Context, cred Credentials, body []byte) (*chatReply, error) {
	compressed, err := gzipBody(body)
	if err != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	defer clear(compressed)
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+chatPath, bytes.NewReader(compressed))
	if err != nil {
		return nil, proxy.UpstreamFailure(err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Content-Encoding", "gzip")
	c.applySign(httpRequest, cred)
	defer httpRequest.Header.Del("Authorization")
	applyChatHeaders(httpRequest)
	if err := applyConversationIDs(httpRequest, body); err != nil {
		return nil, err
	}
	response, err := c.http.Do(httpRequest)
	if err != nil {
		return nil, proxy.UpstreamFailure(err)
	}
	reply := &chatReply{response: response}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		reply.prefix, _ = io.ReadAll(io.LimitReader(response.Body, maxClassifiedBytes))
		if code, found := upstreamBusinessCode(reply.prefix); found {
			reply.code, reply.hasCode = code, true
		}
	}
	return reply, nil
}

// replayedBody 把已读取的前缀与剩余正文拼回一个可关闭的正文。
type replayedBody struct {
	io.Reader
	io.Closer
}

// applySign 写入所有业务请求共用的签名头；Authorization 由调用方 defer 删除。
func (c *Client) applySign(request *http.Request, cred Credentials) {
	request.Header.Set("Authorization", "Bearer "+cred.AccessToken)
	request.Header.Set("X-User-Id", cred.UID)
	request.Header.Set("X-Domain", c.domainOf(cred))
	request.Header.Set("X-Product", "SaaS")
	request.Header.Set("X-IDE-Type", "WorkBuddy")
	request.Header.Set("X-IDE-Name", "WorkBuddy")
	request.Header.Set("X-IDE-Version", c.build.AppVersion)
	request.Header.Set("User-Agent", c.build.UserAgent())
	request.Header.Set("X-Requested-With", "XMLHttpRequest")
	request.Header.Set("X-Agent-Intent", "craft")
	request.Header.Set("X-Agent-Purpose", "conversation")
	request.Header.Set("Accept", "application/json")
	// 企业（团队）账号必须带企业头，否则上游按个人版回答目录与额度；个人账号没有。
	if cred.EnterpriseID != "" {
		request.Header.Set("X-Enterprise-Id", cred.EnterpriseID)
		request.Header.Set("X-Tenant-Id", cred.EnterpriseID)
	}
}

// applyBilling 是计费接口的头集（照社区插件 meterAsked）：短 UA、POST 空对象正文，
// 不带对话用的 Agent／Stainless 头；企业账号才带企业头。
func (c *Client) applyBilling(request *http.Request, cred Credentials) {
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", c.build.ShortUserAgent())
	request.Header.Set("Authorization", "Bearer "+cred.AccessToken)
	request.Header.Set("X-User-Id", cred.UID)
	request.Header.Set("X-Domain", c.domainOf(cred))
	request.Header.Set("X-Product", "SaaS")
	request.Header.Set("X-IDE-Type", "WorkBuddy")
	if cred.EnterpriseID != "" {
		request.Header.Set("X-Enterprise-Id", cred.EnterpriseID)
		request.Header.Set("X-Tenant-Id", cred.EnterpriseID)
	}
}

// domainOf 取账号登录时拿到的域名；缺失时回退到基址主机名。
func (c *Client) domainOf(cred Credentials) string {
	if cred.Domain != "" {
		return cred.Domain
	}
	parsed, err := url.Parse(c.base)
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}

// applyChatHeaders 只给对话请求补上桌面 CLI 的 Agent 头与 OpenAI SDK 指纹。
func applyChatHeaders(request *http.Request) {
	request.Header.Set("X-Agent-Type", "main")
	request.Header.Set("X-Agent-Runtime", "local-agent")
	request.Header.Set("X-Stainless-Retry-Count", "0")
	request.Header.Set("X-Stainless-Timeout", stainlessTimeout)
	request.Header.Set("X-Stainless-Lang", "js")
	request.Header.Set("X-Stainless-Package-Version", openAISDKVersion)
	request.Header.Set("X-Stainless-Runtime", "node")
	request.Header.Set("X-Stainless-Runtime-Version", nodeRuntimeVersion)
	request.Header.Set("X-Stainless-OS", stainlessOS(runtime.GOOS))
	request.Header.Set("X-Stainless-Arch", stainlessArch(runtime.GOARCH))
}

// stainlessOS 按 OpenAI SDK 的取值映射运行时；未知平台用 Unknown。
func stainlessOS(goos string) string {
	switch goos {
	case "darwin":
		return "MacOS"
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	case "freebsd":
		return "FreeBSD"
	case "openbsd":
		return "OpenBSD"
	case "android":
		return "Android"
	}
	return "Unknown"
}

// stainlessArch 按 OpenAI SDK 的取值映射架构；未知架构用 Unknown。
func stainlessArch(goarch string) string {
	switch goarch {
	case "amd64":
		return "x64"
	case "arm64":
		return "arm64"
	case "arm":
		return "arm"
	case "386":
		return "x32"
	}
	return "Unknown"
}

// applyConversationIDs 写入上游用来归属会话与计费的四个 ID；全部不超过 64 字符。
func applyConversationIDs(request *http.Request, body []byte) error {
	requestID, err := randomID()
	if err != nil {
		return proxy.UpstreamFailure(err)
	}
	if conversation := conversationID(body); conversation != "" {
		request.Header.Set("X-Conversation-ID", conversation)
	}
	request.Header.Set("X-Conversation-Request-ID", requestID)
	request.Header.Set("X-Conversation-Message-ID", requestID)
	request.Header.Set("X-Request-ID", requestID)
	return nil
}

// randomID 返回 32 位 hex 随机 ID（16 字节），满足上游 64 字符以内的约束。
func randomID() (string, error) {
	buffer := make([]byte, randomIDBytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

// conversationID 由正文第一条 user 消息的内容决定：同一会话的每一步得到同一 ID，
// 用户的下一条消息得到新 ID。没有 user 消息时返回空串（不发送该头）。
func conversationID(body []byte) string {
	var payload struct {
		Messages []struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return ""
	}
	for _, message := range payload.Messages {
		if message.Role != "user" {
			continue
		}
		content := contentBytes(message.Content)
		digest := sha256.Sum256(content)
		return hex.EncodeToString(digest[:16])
	}
	return ""
}

// contentBytes 取消息内容的稳定字节：字符串取原文，结构化内容取紧凑 JSON，
// null 与空内容取空字节。
func contentBytes(content json.RawMessage) []byte {
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}
	var text string
	if json.Unmarshal(trimmed, &text) == nil {
		return []byte(text)
	}
	var compact bytes.Buffer
	if json.Compact(&compact, trimmed) != nil {
		return trimmed
	}
	return compact.Bytes()
}

// gzipBody 按官方客户端的 GzipRequestProcessor 压缩对话正文。
func gzipBody(body []byte) ([]byte, error) {
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write(body); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// validateBase 校验注入基址：HTTPS，或仅供测试的回环 HTTP。
func validateBase(base string) (string, error) {
	if base == "" || len(base) > maxURLBytes {
		return "", proxy.Failure("invalid_configuration", 503)
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Hostname() == "" || parsed.Opaque != "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.ForceQuery {
		return "", proxy.Failure("invalid_configuration", 503)
	}
	if parsed.Scheme != "https" {
		ip := net.ParseIP(parsed.Hostname())
		if parsed.Scheme != "http" || !(strings.EqualFold(parsed.Hostname(), "localhost") || ip != nil && ip.IsLoopback()) {
			return "", proxy.Failure("invalid_configuration", 503)
		}
	}
	if port := parsed.Port(); port != "" {
		if value, err := strconv.Atoi(port); err != nil || value < 1 || value > 65535 {
			return "", proxy.Failure("invalid_configuration", 503)
		}
	}
	return strings.TrimRight(base, "/"), nil
}

// statusFailure 把非 2xx 状态映射成固定错误；2xx 返回 nil。
//
// 4xx 保留上游状态码，便于续期等调用路径区分「拒绝」与「暂时不可用」；
// 5xx 与其它异常状态统一折算成 502。
func statusFailure(status int) error {
	switch status {
	case http.StatusUnauthorized:
		return proxy.Failure(CodeUnauthorized, http.StatusUnauthorized)
	case http.StatusTooManyRequests:
		return proxy.Failure(CodeRateLimited, http.StatusTooManyRequests)
	}
	if status >= 400 && status < 500 {
		return proxy.Failure(CodeUpstreamUnavailable, status)
	}
	if status < 200 || status >= 300 {
		return proxy.Failure(CodeUpstreamUnavailable, http.StatusBadGateway)
	}
	return nil
}

// businessStatus 在非 2xx 时先识别固定业务码，再退回状态映射。
func businessStatus(status int, body []byte) error {
	if status >= 200 && status < 300 {
		return nil
	}
	if code, found := upstreamBusinessCode(body); found {
		if mapped, known := businessFailure(code); known {
			return mapped
		}
	}
	return statusFailure(status)
}

// businessFailure 把插件记录的业务码映射成固定错误码；11217（登录未完成）
// 只在 WaitForToken 里处理，不能当成终态错误。
func businessFailure(code int) (error, bool) {
	switch code {
	case 10085:
		return Failure(CodeUserAgentRefused), true
	case 11128:
		return Failure(CodeChannelRejected), true
	case 11159:
		return Failure(CodeSignatureInvalid), true
	case 11158:
		return Failure(CodeModelNotEntitled), true
	}
	return nil, false
}

// decodeBusiness 解析插件 API 的公共外壳 {code,msg,data}：code 非 0 时返回映射
// 后的错误；否则返回 data，没有 data 时返回原正文。
func decodeBusiness(body []byte) (json.RawMessage, error) {
	var envelope struct {
		Code json.RawMessage `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if len(body) == 0 || json.Unmarshal(body, &envelope) != nil {
		return nil, Failure(CodeInvalidResponse)
	}
	if code, found := rawCode(envelope.Code); found && code != 0 {
		if mapped, known := businessFailure(code); known {
			return nil, mapped
		}
		return nil, Failure(CodeUpstreamUnavailable)
	}
	if len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		return envelope.Data, nil
	}
	return body, nil
}

// rawCode 解析 RawMessage 形式的业务码。
func rawCode(raw json.RawMessage) (int, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return 0, false
	}
	return numericCode(value)
}

// numericCode 把数字或数字字符串解析成业务码；"11158:no entitlement" 形式取前缀。
func numericCode(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		if typed != math.Trunc(typed) || typed < 0 || typed > 1e9 {
			return 0, false
		}
		return int(typed), true
	case json.Number:
		parsed, err := strconv.ParseInt(typed.String(), 10, 64)
		if err != nil || parsed < 0 || parsed > 1e9 {
			return 0, false
		}
		return int(parsed), true
	case string:
		text := strings.TrimSpace(typed)
		if index := strings.IndexByte(text, ':'); index > 0 {
			text = text[:index]
		}
		parsed, err := strconv.Atoi(text)
		if err != nil || parsed < 0 || parsed > 1e9 {
			return 0, false
		}
		return parsed, true
	}
	return 0, false
}

// upstreamBusinessCode 在错误正文里找出固定业务码。按插件记录的三种形态读取：
// {"code":11128}、{"error":{"internal_code":"11159"}}、{"code":11102,"error":"11158:…"}，
// 字符串错误里的码优先于外层容器码。
func upstreamBusinessCode(body []byte) (int, bool) {
	var root any
	if len(body) == 0 || json.Unmarshal(body, &root) != nil {
		return 0, false
	}
	budget := 512
	return findBusinessCode(root, 0, &budget)
}

func findBusinessCode(value any, depth int, budget *int) (int, bool) {
	if depth > 4 || *budget <= 0 {
		return 0, false
	}
	*budget = *budget - 1
	object, ok := value.(map[string]any)
	if !ok {
		return 0, false
	}
	for _, key := range []string{"internal_code", "internalCode"} {
		if code, found := numericCode(object[key]); found {
			return code, true
		}
	}
	if nested, exists := object["error"]; exists {
		if code, found := numericCode(nested); found {
			return code, true
		}
		if code, found := findBusinessCode(nested, depth+1, budget); found {
			return code, true
		}
	}
	if inner, exists := object["data"]; exists {
		if code, found := findBusinessCode(inner, depth+1, budget); found {
			return code, true
		}
	}
	return numericCode(object["code"])
}

// readBounded 有界读取响应正文；超限按无法识别处理。
func readBounded(body io.Reader, limit int) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, int64(limit)+1))
	if err != nil || len(data) > limit {
		return nil, Failure(CodeInvalidResponse)
	}
	return data, nil
}
