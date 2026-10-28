package proxy

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"omniproxy/internal/core/netproxy"
)

// Server 管理单个本机监听器，启停操作串行化，停止会取消正在进行的上游请求。
type Server struct {
	mu       sync.Mutex
	opts     Options
	status   Status
	http     *http.Server
	cancel   context.CancelFunc
	listener net.Listener
	onStatus func(Status)
	// passthrough 是透明转发用的 HTTP 客户端：不跟随重定向，也不设整体期限。
	passthrough *http.Client
	slots       chan struct{}
	socketSlots chan struct{}
}

// New 创建尚未监听的代理；默认请求期限 120 秒，并发上限 16。
func New(opts Options) *Server {
	if opts.RequestTimeout <= 0 {
		opts.RequestTimeout = 120 * time.Second
	}
	if opts.MaxConcurrent <= 0 {
		opts.MaxConcurrent = 16
	}
	opts.Models = append([]string(nil), opts.Models...)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// 只限制响应头到达时间：订阅端点的流式响应可能持续数分钟。
	transport.ResponseHeaderTimeout = passthroughHeaderTimeout
	return &Server{
		opts: opts, status: Status{State: "stopped", Port: opts.Port},
		slots:       make(chan struct{}, opts.MaxConcurrent),
		socketSlots: make(chan struct{}, opts.MaxConcurrent),
		passthrough: &http.Client{Transport: netproxy.Wrap(transport), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}
}

// Status 返回不会暴露内部 HTTP 对象的快照。
func (s *Server) Status() Status { s.mu.Lock(); defer s.mu.Unlock(); return s.status }

// OnStatus 设置状态通知；回调在锁外执行，UI 应重新读取状态而非依赖事件顺序。
func (s *Server) OnStatus(listener func(Status)) { s.mu.Lock(); s.onStatus = listener; s.mu.Unlock() }

func (s *Server) notify() {
	s.mu.Lock()
	status, listener := s.status, s.onStatus
	s.mu.Unlock()
	if listener != nil {
		listener(status)
	}
}

// Start 仅监听 IPv4 loopback；同一实例重复启动保持幂等。
func (s *Server) Start() (Status, error) {
	s.mu.Lock()
	if s.status.State == "running" {
		status := s.status
		s.mu.Unlock()
		return status, nil
	}
	if s.status.State == "stopping" {
		s.mu.Unlock()
		return s.Status(), Failure("gateway_busy", 409)
	}
	if s.opts.Port < 0 || s.opts.Port > 65535 || s.opts.Upstream == nil {
		s.mu.Unlock()
		return s.Status(), Failure("invalid_configuration", 400)
	}
	listener, err := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(s.opts.Port)))
	if err != nil {
		// Windows 对占用／保留端口可能返回不同错误，统一说明端口不可用而不暴露系统错误。
		code := "port_unavailable"
		s.status.State, s.status.ErrorCode = "error", code
		s.mu.Unlock()
		s.notify()
		return s.Status(), Failure(code, 503)
	}
	ctx, cancel := context.WithCancel(context.Background())
	log := s.opts.Logger
	if log == nil {
		log = slog.Default()
	}
	server := &http.Server{
		Handler: s, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10,
		ErrorLog:    slog.NewLogLogger(log.Handler(), slog.LevelError),
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	s.http, s.listener, s.cancel = server, listener, cancel
	port := listener.Addr().(*net.TCPAddr).Port
	s.status = Status{State: "running", Address: "http://" + listener.Addr().String(), Port: port, StartedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	s.mu.Unlock()
	s.notify()
	go s.serve(server, listener)
	return s.Status(), nil
}

// SetPort 切换监听端口：先保存新端口，运行中则先停止（取消在途请求）再在新端口上
// 恢复监听。未运行或停止失败时不会自行启动监听器。
func (s *Server) SetPort(ctx context.Context, port int) (Status, error) {
	if port < 0 || port > 65535 {
		return s.Status(), Failure("invalid_configuration", 400)
	}
	s.mu.Lock()
	if s.status.State == "stopping" {
		s.mu.Unlock()
		return s.Status(), Failure("gateway_busy", 409)
	}
	running := s.http != nil
	s.opts.Port = port
	s.status.Port = port
	s.mu.Unlock()
	s.notify()
	if !running {
		return s.Status(), nil
	}
	if _, err := s.Stop(ctx); err != nil {
		return s.Status(), err
	}
	return s.Start()
}

func (s *Server) serve(server *http.Server, listener net.Listener) {
	err := server.Serve(listener)
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return
	}
	s.mu.Lock()
	if s.http != server {
		s.mu.Unlock()
		return
	}
	s.cancel()
	s.http, s.listener, s.cancel = nil, nil, nil
	s.status.State, s.status.ErrorCode, s.status.Address, s.status.StartedAt = "error", "serve_failed", "", ""
	s.mu.Unlock()
	if s.opts.Logger != nil {
		s.opts.Logger.Error("gateway listener failed", "code", "serve_failed")
	}
	s.notify()
}

// Stop 在关闭监听器前取消请求，不会等待长时间的流式生成。
func (s *Server) Stop(ctx context.Context) (Status, error) {
	s.mu.Lock()
	if s.http == nil {
		status := s.status
		s.mu.Unlock()
		return status, nil
	}
	if s.status.State == "stopping" {
		s.mu.Unlock()
		return s.Status(), Failure("gateway_busy", 409)
	}
	server := s.http
	s.status.State = "stopping"
	s.cancel()
	s.mu.Unlock()
	s.notify()
	err := server.Shutdown(ctx)
	if err != nil {
		err = errors.Join(err, server.Close())
	}
	s.mu.Lock()
	s.http, s.listener, s.cancel = nil, nil, nil
	s.status.State, s.status.Address, s.status.StartedAt, s.status.ErrorCode = "stopped", "", "", ""
	if err != nil {
		s.status.ErrorCode = "stop_timeout"
	}
	s.mu.Unlock()
	s.notify()
	if err != nil {
		return s.Status(), Failure("stop_timeout", 503)
	}
	return s.Status(), nil
}
