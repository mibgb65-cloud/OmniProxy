package netproxy

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	detectAttemptTimeout = 2 * time.Second
	discoverDialTimeout  = 300 * time.Millisecond
	discoverBudget       = 2 * time.Second
	detectReadLimit      = 512
)

// CandidatePorts 是本机代理客户端常见的监听端口（v2rayN / Clash / mihomo）。
var CandidatePorts = []int{10808, 10809, 7890, 7891, 7897, 1080, 2080, 1087}

// Detection 是一次端点识别结果。mixed 端口会同时满足 SOCKS5 与 HTTP。
type Detection struct {
	SOCKS5    bool
	HTTP      bool
	Auth      bool
	Scheme    string
	LatencyMs int64
}

// Detect 识别端点实际支持的协议。两者都可用（mixed）时返回 socks5：
// 纯 TCP 隧道对 ws:// 与 wss:// 都更稳，不依赖代理侧的 HTTP 解析。
func Detect(ctx context.Context, endpoint Endpoint) (Detection, error) {
	start := time.Now()
	socks, socksErr := probeSOCKS5(ctx, endpoint)
	http, httpErr := probeHTTP(ctx, endpoint)
	result := Detection{SOCKS5: socks.OK, HTTP: http.OK, Auth: socks.Auth || http.Auth, LatencyMs: time.Since(start).Milliseconds()}
	switch {
	case result.SOCKS5:
		result.Scheme = SchemeSOCKS5
		return result, nil
	case result.HTTP:
		result.Scheme = SchemeHTTP
		return result, nil
	}
	if isDialFailure(socksErr) && isDialFailure(httpErr) {
		return result, fmt.Errorf("%w: %s", ErrUnavailable, endpoint.Address())
	}
	return result, fmt.Errorf("%w: %s", ErrProtocolMismatch, endpoint.Address())
}

type probeResult struct {
	OK   bool
	Auth bool
}

// probeSOCKS5 只做 greeting（必要时补一轮用户名密码子协商），不发起 CONNECT：
// 目的只是确认端口在说 SOCKS5，不会产生真实出网流量。
func probeSOCKS5(ctx context.Context, endpoint Endpoint) (probeResult, error) {
	conn, err := dialEndpoint(ctx, endpoint)
	if err != nil {
		return probeResult{}, err
	}
	defer func() { _ = conn.Close() }()
	greeting := []byte{0x05, 0x01, 0x00}
	if endpoint.Username != "" {
		greeting = []byte{0x05, 0x02, 0x00, 0x02}
	}
	if _, err := conn.Write(greeting); err != nil {
		return probeResult{}, err
	}
	reply := make([]byte, 2)
	if _, err := readFull(conn, reply); err != nil {
		return probeResult{}, err
	}
	if reply[0] != 0x05 {
		return probeResult{}, fmt.Errorf("%w: not socks5", ErrProtocolMismatch)
	}
	switch reply[1] {
	case 0x00:
		return probeResult{OK: true}, nil
	case 0x02:
		if endpoint.Username == "" {
			return probeResult{OK: true, Auth: true}, nil
		}
		auth := []byte{0x01}
		auth = append(auth, byte(len(endpoint.Username)))
		auth = append(auth, endpoint.Username...)
		auth = append(auth, byte(len(endpoint.Password)))
		auth = append(auth, endpoint.Password...)
		if _, err := conn.Write(auth); err != nil {
			return probeResult{}, err
		}
		status := make([]byte, 2)
		if _, err := readFull(conn, status); err != nil {
			return probeResult{}, err
		}
		if status[0] != 0x01 {
			return probeResult{}, fmt.Errorf("%w: socks5 auth unsupported", ErrProtocolMismatch)
		}
		if status[1] != 0x00 {
			return probeResult{OK: true, Auth: true}, nil
		}
		return probeResult{OK: true}, nil
	default:
		return probeResult{}, fmt.Errorf("%w: socks5 method rejected", ErrProtocolMismatch)
	}
}

// probeHTTP 只看 CONNECT 的状态行：目标故意选必然被拒的地址，代理会回 4xx/5xx，
// 只要它以 HTTP/1.x 应答就证明这个端口在按 HTTP 代理解析。
func probeHTTP(ctx context.Context, endpoint Endpoint) (probeResult, error) {
	conn, err := dialEndpoint(ctx, endpoint)
	if err != nil {
		return probeResult{}, err
	}
	defer func() { _ = conn.Close() }()
	var request strings.Builder
	request.WriteString("CONNECT 127.0.0.1:1 HTTP/1.1\r\nHost: 127.0.0.1:1\r\n")
	if endpoint.Username != "" {
		request.WriteString("Proxy-Authorization: Basic ")
		request.WriteString(base64.StdEncoding.EncodeToString([]byte(endpoint.Username + ":" + endpoint.Password)))
		request.WriteString("\r\n")
	}
	request.WriteString("\r\n")
	if _, err := conn.Write([]byte(request.String())); err != nil {
		return probeResult{}, err
	}
	reader := bufio.NewReaderSize(conn, detectReadLimit)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return probeResult{}, err
	}
	if len(line) > detectReadLimit {
		return probeResult{}, fmt.Errorf("%w: oversized response", ErrProtocolMismatch)
	}
	if !strings.HasPrefix(line, "HTTP/1.") {
		return probeResult{}, fmt.Errorf("%w: not http", ErrProtocolMismatch)
	}
	return probeResult{OK: true, Auth: strings.Contains(line, " 407 ")}, nil
}

func dialEndpoint(ctx context.Context, endpoint Endpoint) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: detectAttemptTimeout}
	deadline := time.Now().Add(detectAttemptTimeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	conn, err := dialer.DialContext(ctx, "tcp", endpoint.Address())
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrUnavailable, endpoint.Address(), sanitizeDialError(err))
	}
	_ = conn.SetDeadline(deadline)
	return conn, nil
}

func readFull(conn net.Conn, buffer []byte) (int, error) {
	total := 0
	for total < len(buffer) {
		n, err := conn.Read(buffer[total:])
		total += n
		if err != nil {
			if total == 0 {
				return 0, err
			}
			return total, nil
		}
		if n == 0 {
			return total, io.ErrNoProgress
		}
	}
	return total, nil
}

// sanitizeDialError 只保留系统错误的类型信息，避免把用户凭证带进错误文本。
func sanitizeDialError(err error) string {
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Err != nil {
		return opErr.Err.Error()
	}
	return err.Error()
}

func isDialFailure(err error) bool {
	return err == nil || errors.Is(err, ErrUnavailable)
}

// Discovered 是通过端口发现得到的候选端点。
type Discovered struct {
	Endpoint  string `json:"endpoint"`
	Scheme    string `json:"scheme"`
	SOCKS5    bool   `json:"socks5"`
	HTTP      bool   `json:"http"`
	Auth      bool   `json:"auth"`
	Port      int    `json:"port"`
	LatencyMs int64  `json:"latencyMs"`
}

// Discover 并行探测候选端口，只返回真的在说 HTTP/SOCKS5 的端口。
func Discover(ctx context.Context, host string, ports []int) []Discovered {
	if host == "" {
		host = DefaultHost
	}
	if len(ports) == 0 {
		ports = CandidatePorts
	}
	ctx, cancel := context.WithTimeout(ctx, discoverBudget)
	defer cancel()
	results := make([]Discovered, len(ports))
	var wait sync.WaitGroup
	for index, port := range ports {
		wait.Add(1)
		go func(index, port int) {
			defer wait.Done()
			endpoint := Endpoint{Scheme: SchemeAuto, Host: host, Port: port}
			if !dialable(ctx, endpoint) {
				return
			}
			detection, err := Detect(ctx, endpoint)
			if err != nil || detection.Scheme == "" {
				return
			}
			results[index] = Discovered{
				Endpoint: endpoint.String(), Scheme: detection.Scheme, SOCKS5: detection.SOCKS5,
				HTTP: detection.HTTP, Auth: detection.Auth, Port: port, LatencyMs: detection.LatencyMs,
			}
		}(index, port)
	}
	wait.Wait()
	found := make([]Discovered, 0, len(results))
	for _, item := range results {
		if item.Port != 0 {
			found = append(found, item)
		}
	}
	return found
}

func dialable(ctx context.Context, endpoint Endpoint) bool {
	dialer := &net.Dialer{Timeout: discoverDialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", endpoint.Address())
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
