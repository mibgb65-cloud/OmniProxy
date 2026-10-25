package netproxy

import (
	"bufio"
	"context"
	"encoding/binary"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeProxy 是测试用的混合代理：按首字节区分 SOCKS5 与 HTTP CONNECT。
type fakeProxy struct {
	listener  net.Listener
	mode      string // mixed | http | socks5
	hits      int32
	closeOnce sync.Once
}

func newFakeProxy(t *testing.T, mode string) *fakeProxy {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	proxy := &fakeProxy{listener: listener, mode: mode}
	go proxy.accept()
	t.Cleanup(proxy.Close)
	return proxy
}

func (p *fakeProxy) Close() {
	p.closeOnce.Do(func() { _ = p.listener.Close() })
}

func (p *fakeProxy) Endpoint() Endpoint {
	address := p.listener.Addr().(*net.TCPAddr)
	return Endpoint{Scheme: SchemeAuto, Host: "127.0.0.1", Port: address.Port}
}

func (p *fakeProxy) Hits() int32 { return atomic.LoadInt32(&p.hits) }

func (p *fakeProxy) accept() {
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			return
		}
		go p.handle(conn)
	}
}

func (p *fakeProxy) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	reader := bufio.NewReader(conn)
	first, err := reader.Peek(1)
	if err != nil {
		return
	}
	switch {
	case first[0] == 0x05 && p.mode != "http":
		atomic.AddInt32(&p.hits, 1)
		p.serveSOCKS(conn, reader)
	default:
		if p.mode == "socks5" {
			return
		}
		atomic.AddInt32(&p.hits, 1)
		p.serveHTTP(conn, reader)
	}
}

// serveSOCKS 处理无认证 SOCKS5 CONNECT。
func (p *fakeProxy) serveSOCKS(conn net.Conn, reader *bufio.Reader) bool {
	head := make([]byte, 2)
	if _, err := io.ReadFull(reader, head); err != nil || head[0] != 0x05 {
		return false
	}
	if _, err := io.ReadFull(reader, make([]byte, int(head[1]))); err != nil {
		return false
	}
	if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
		return false
	}
	request := make([]byte, 4)
	if _, err := io.ReadFull(reader, request); err != nil || request[1] != 0x01 {
		return false
	}
	host, err := readSOCKSAddress(reader, request[3])
	if err != nil {
		return false
	}
	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(reader, portBytes); err != nil {
		return false
	}
	target, err := net.DialTimeout("tcp", net.JoinHostPort(host, itoa(int(binary.BigEndian.Uint16(portBytes)))), 2*time.Second)
	if err != nil {
		_, _ = conn.Write([]byte{0x05, 0x04, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return false
	}
	if _, err := conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 127, 0, 0, 1, 0, 0}); err != nil {
		_ = target.Close()
		return false
	}
	pipe(conn, reader, target)
	return true
}

func readSOCKSAddress(reader *bufio.Reader, kind byte) (string, error) {
	switch kind {
	case 0x01:
		buffer := make([]byte, 4)
		if _, err := io.ReadFull(reader, buffer); err != nil {
			return "", err
		}
		return net.IP(buffer).String(), nil
	case 0x03:
		length := make([]byte, 1)
		if _, err := io.ReadFull(reader, length); err != nil {
			return "", err
		}
		buffer := make([]byte, int(length[0]))
		if _, err := io.ReadFull(reader, buffer); err != nil {
			return "", err
		}
		return string(buffer), nil
	case 0x04:
		buffer := make([]byte, 16)
		if _, err := io.ReadFull(reader, buffer); err != nil {
			return "", err
		}
		return net.IP(buffer).String(), nil
	default:
		return "", io.ErrUnexpectedEOF
	}
}

// serveHTTP 处理 CONNECT 隧道。
func (p *fakeProxy) serveHTTP(conn net.Conn, reader *bufio.Reader) bool {
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	fields := strings.Fields(line)
	if len(fields) < 3 || !strings.EqualFold(fields[0], "CONNECT") {
		_, _ = conn.Write([]byte("HTTP/1.1 405 Method Not Allowed\r\n\r\n"))
		return false
	}
	for {
		header, err := reader.ReadString('\n')
		if err != nil {
			return false
		}
		if header == "\r\n" || header == "\n" {
			break
		}
	}
	target, err := net.DialTimeout("tcp", fields[1], 2*time.Second)
	if err != nil {
		_, _ = conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return false
	}
	if _, err := conn.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n")); err != nil {
		_ = target.Close()
		return false
	}
	pipe(conn, reader, target)
	return true
}

func pipe(client net.Conn, buffered *bufio.Reader, target net.Conn) {
	defer func() { _ = target.Close() }()
	_ = client.SetDeadline(time.Time{})
	_ = target.SetDeadline(time.Time{})
	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(target, buffered)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(client, target)
		done <- struct{}{}
	}()
	<-done
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits []byte
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}

// plainListener 是一个会说话但不是代理的端口，用于验证协议不匹配的判定。
type plainListener struct {
	listener net.Listener
}

func newPlainListener() (*plainListener, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	plain := &plainListener{listener: listener}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = conn.Close() }()
				_, _ = conn.Write([]byte("SSH-2.0-OpenSSH_9.0\r\n"))
			}()
		}
	}()
	return plain, nil
}

func (p *plainListener) Endpoint() Endpoint {
	address := p.listener.Addr().(*net.TCPAddr)
	return Endpoint{Scheme: SchemeAuto, Host: "127.0.0.1", Port: address.Port}
}

func (p *plainListener) Close() error { return p.listener.Close() }

type fakeResolver struct {
	mu       sync.Mutex
	snapshot Snapshot
	err      error
	detected []string
}

func (r *fakeResolver) Snapshot(context.Context) (Snapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.snapshot, r.err
}

func (r *fakeResolver) RememberDetectedScheme(_ context.Context, scheme string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.detected = append(r.detected, scheme)
	r.snapshot.Detected = scheme
	return nil
}

func (r *fakeResolver) Detections() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.detected...)
}

func (r *fakeResolver) Set(snapshot Snapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshot = snapshot
}
