package netproxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	proxyDialTimeout    = 1500 * time.Millisecond
	proxyDialOKTTL      = 30 * time.Second
	proxyDialFailTTL    = 5 * time.Second
	protocolSwitchGuard = 60 * time.Second
	detectRecordTimeout = 2 * time.Second
)

// Resolver 由应用层实现（数据库 + 缓存）；Core 只消费快照。
type Resolver interface {
	Snapshot(context.Context) (Snapshot, error)
}

// DetectionRecorder 可选：Transport 识别出协议后回写缓存。
type DetectionRecorder interface {
	RememberDetectedScheme(context.Context, string) error
}

var (
	registryMu sync.RWMutex
	registry   Resolver
)

// Install 注入进程级 Resolver；传 nil 取消注入（Wrap 恢复原样返回）。
func Install(resolver Resolver) {
	registryMu.Lock()
	registry = resolver
	registryMu.Unlock()
}

// Installed 返回当前注入的 Resolver。
func Installed() Resolver {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry
}

// Wrap 把底层 RoundTripper 包上按提供商分流的能力。
// 未安装 Resolver、base 已经是包装层、或 base 不是 *http.Transport（测试注入的
// 契约传输）时原样返回：这类传输没有可克隆的超时与拨号配置，包起来只会绕过它。
func Wrap(base http.RoundTripper) http.RoundTripper {
	resolver := Installed()
	if resolver == nil {
		return base
	}
	if _, wrapped := base.(*Transport); wrapped {
		return base
	}
	if base == nil {
		base = http.DefaultTransport.(*http.Transport).Clone()
	} else if _, standard := base.(*http.Transport); !standard {
		return base
	}
	return &Transport{base: base, resolver: resolver, now: time.Now, pools: map[string]*http.Transport{}, endpoints: map[string]*endpointState{}, dials: map[string]dialState{}}
}

// NewClient 构造走包装层的 HTTP 客户端，供 WebSocket 建连等直接使用 RoundTripper 的调用方。
func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{Transport: Wrap(nil), Timeout: timeout}
}

type endpointState struct {
	scheme     string
	switchedAt time.Time
}

type dialState struct {
	ok bool
	at time.Time
}

// Transport 按请求上下文里的提供商身份选择直连或代理，并按端点复用连接池。
type Transport struct {
	base     http.RoundTripper
	resolver Resolver
	now      func() time.Time

	mu        sync.Mutex
	version   string
	pools     map[string]*http.Transport
	endpoints map[string]*endpointState
	dials     map[string]dialState
}

// RoundTrip 实现 http.RoundTripper。
func (t *Transport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request == nil || request.URL == nil {
		return nil, &Error{Code: "invalid_request"}
	}
	snapshot, err := t.resolver.Snapshot(request.Context())
	if err != nil {
		return nil, &Error{Code: "proxy_configuration_unavailable", Err: err}
	}
	t.adopt(snapshot)
	decision := Decide(snapshot, mustScope(request.Context()), request.URL.Host)
	if decision.Reason == ReasonNotConfigured {
		return nil, &Error{Code: "proxy_not_configured", Err: errors.New("proxy endpoint is not configured")}
	}
	if !decision.UseProxy {
		return t.base.RoundTrip(request)
	}
	address := snapshot.Endpoint.Address()
	if err := t.ensureReachable(request.Context(), address); err != nil {
		return nil, err
	}
	scheme := t.schemeFor(request.Context(), snapshot, address)
	response, err := t.roundTripVia(request, snapshot.Endpoint, scheme)
	if err == nil {
		return response, nil
	}
	if !isProtocolFailure(err) || !t.canSwitch(address) || !replayable(request) {
		return nil, classifyProxyError(err, address)
	}
	retry, cloneErr := cloneForRetry(request)
	if cloneErr != nil {
		return nil, classifyProxyError(err, address)
	}
	other := otherScheme(scheme)
	response, retryErr := t.roundTripVia(retry, snapshot.Endpoint, other)
	if retryErr != nil {
		return nil, classifyProxyError(err, address)
	}
	t.remember(address, other)
	t.recordDetection(other)
	return response, nil
}

func mustScope(ctx context.Context) Scope {
	scope, _ := ScopeFrom(ctx)
	return scope
}

// adopt 在设置版本变化时丢弃协议缓存与连接池，使保存后的改动立即生效。
func (t *Transport) adopt(snapshot Snapshot) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.version == snapshot.Version {
		return
	}
	for _, pool := range t.pools {
		pool.CloseIdleConnections()
	}
	t.version = snapshot.Version
	t.pools = map[string]*http.Transport{}
	t.endpoints = map[string]*endpointState{}
	t.dials = map[string]dialState{}
}

// ensureReachable 预连代理端口，让"代理没开"和"上游挂了"在错误码上可以区分。
func (t *Transport) ensureReachable(ctx context.Context, address string) error {
	now := t.now()
	t.mu.Lock()
	state, seen := t.dials[address]
	t.mu.Unlock()
	if seen {
		if state.ok && now.Sub(state.at) < proxyDialOKTTL {
			return nil
		}
		if !state.ok && now.Sub(state.at) < proxyDialFailTTL {
			return &Error{Code: "proxy_unavailable", Err: fmt.Errorf("%w: %s", ErrUnavailable, address)}
		}
	}
	dialer := &net.Dialer{Timeout: proxyDialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	ok := err == nil
	if conn != nil {
		_ = conn.Close()
	}
	t.mu.Lock()
	t.dials[address] = dialState{ok: ok, at: now}
	t.mu.Unlock()
	if !ok {
		return &Error{Code: "proxy_unavailable", Err: fmt.Errorf("%w: %s", ErrUnavailable, address)}
	}
	return nil
}

// schemeFor 依次取运行期缓存、持久化识别结果、显式配置，最后现场探测。
func (t *Transport) schemeFor(ctx context.Context, snapshot Snapshot, address string) string {
	t.mu.Lock()
	state := t.endpoints[address]
	t.mu.Unlock()
	if state != nil && state.scheme != "" {
		return state.scheme
	}
	if snapshot.Detected == SchemeHTTP || snapshot.Detected == SchemeSOCKS5 {
		t.remember(address, snapshot.Detected)
		return snapshot.Detected
	}
	if snapshot.Endpoint.Scheme == SchemeHTTP || snapshot.Endpoint.Scheme == SchemeSOCKS5 {
		t.remember(address, snapshot.Endpoint.Scheme)
		return snapshot.Endpoint.Scheme
	}
	detectCtx, cancel := context.WithTimeout(ctx, 2*detectAttemptTimeout)
	defer cancel()
	if detection, err := Detect(detectCtx, snapshot.Endpoint); err == nil && detection.Scheme != "" {
		t.remember(address, detection.Scheme)
		if _, ok := t.resolver.(DetectionRecorder); ok {
			t.recordDetection(detection.Scheme)
		}
		return detection.Scheme
	}
	// 探测失败时先猜 HTTP：纯 HTTP 端口上它必然成功，纯 SOCKS 端口由纠偏兜住。
	t.remember(address, SchemeHTTP)
	return SchemeHTTP
}

func (t *Transport) remember(address, scheme string) {
	t.mu.Lock()
	state := t.endpoints[address]
	if state == nil {
		state = &endpointState{}
		t.endpoints[address] = state
	}
	state.scheme = scheme
	t.mu.Unlock()
}

// canSwitch 限制同一端点的协议切换频率，避免在坏端口上反复试探。
func (t *Transport) canSwitch(address string) bool {
	now := t.now()
	t.mu.Lock()
	defer t.mu.Unlock()
	state := t.endpoints[address]
	if state == nil {
		t.endpoints[address] = &endpointState{switchedAt: now}
		return true
	}
	if now.Sub(state.switchedAt) < protocolSwitchGuard {
		return false
	}
	state.switchedAt = now
	return true
}

func (t *Transport) recordDetection(scheme string) {
	recorder, ok := t.resolver.(DetectionRecorder)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), detectRecordTimeout)
	defer cancel()
	_ = recorder.RememberDetectedScheme(ctx, scheme)
}

func (t *Transport) roundTripVia(request *http.Request, endpoint Endpoint, scheme string) (*http.Response, error) {
	return t.transportFor(endpoint, scheme).RoundTrip(request)
}

func (t *Transport) transportFor(endpoint Endpoint, scheme string) *http.Transport {
	key := endpoint.Address() + "|" + scheme + "|" + endpoint.Username
	t.mu.Lock()
	defer t.mu.Unlock()
	if pool, ok := t.pools[key]; ok {
		return pool
	}
	var pool *http.Transport
	if base, ok := t.base.(*http.Transport); ok {
		pool = base.Clone()
	} else {
		pool = http.DefaultTransport.(*http.Transport).Clone()
	}
	pool.Proxy = http.ProxyURL(endpoint.URL(scheme))
	t.pools[key] = pool
	return pool
}

// replayable 判断这次请求能否换协议重放：无正文，或正文有可重取副本。
func replayable(request *http.Request) bool {
	if request.Body == nil || request.Body == http.NoBody {
		return true
	}
	return request.GetBody != nil
}

func cloneForRetry(request *http.Request) (*http.Request, error) {
	clone := request.Clone(request.Context())
	if request.Body == nil || request.Body == http.NoBody {
		clone.Body = request.Body
		return clone, nil
	}
	if request.GetBody == nil {
		return nil, errors.New("netproxy: body cannot be replayed")
	}
	body, err := request.GetBody()
	if err != nil {
		return nil, err
	}
	clone.Body = body
	return clone, nil
}

func otherScheme(scheme string) string {
	if scheme == SchemeSOCKS5 {
		return SchemeHTTP
	}
	return SchemeSOCKS5
}

// isProtocolFailure 只认"连接建立但对话不成立"的信号，网络中断不在此列。
func isProtocolFailure(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrProtocolMismatch) || errors.Is(err, io.EOF) {
		return true
	}
	text := strings.ToLower(err.Error())
	// 只列"对端说的是另一套协议"的信号：SOCKS 握手成功后的 CONNECT 失败不属于此类。
	for _, marker := range []string{
		"malformed http response",
		"malformed http status",
		"invalid response version",
		"unexpected response version",
		"unexpected eof",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

// classifyProxyError 把代理层失败换算成稳定错误码，绝不冒充上游故障。
func classifyProxyError(err error, address string) error {
	if err == nil {
		return nil
	}
	var typed *Error
	if errors.As(err, &typed) {
		return err
	}
	if isProtocolFailure(err) {
		return &Error{Code: "proxy_protocol_mismatch", Err: fmt.Errorf("%w: %s", ErrProtocolMismatch, address)}
	}
	text := strings.ToLower(err.Error())
	if strings.Contains(text, "proxyconnect") || strings.Contains(text, address) {
		return &Error{Code: "proxy_unavailable", Err: fmt.Errorf("%w: %s", ErrUnavailable, address)}
	}
	return err
}
