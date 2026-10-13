//go:build !production

package wails

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseDevServerURL(t *testing.T) {
	for _, rawURL := range []string{"http://localhost:9245", "https://127.0.0.1:9245/", "http://[::1]:9245", "http://127.0.0.2"} {
		if _, err := parseDevServerURL(rawURL); err != nil {
			t.Errorf("本机开发地址被拒绝：%s: %v", rawURL, err)
		}
	}
	for _, rawURL := range []string{
		"", "http://example.com:9245", "file:///frontend", "http://user:password@localhost:9245",
		"http://localhost:0", "http://localhost:65536", "http://localhost:abc", "http://localhost:9245/app",
		"http://localhost:9245/?token=sensitive", "http://localhost:9245/?", "http://localhost:9245/#fragment",
		"http://localhost:9245\n", strings.Repeat("a", devServerURLMaxLength+1),
	} {
		_, err := parseDevServerURL(rawURL)
		if err == nil {
			t.Error("无效开发地址未被拒绝")
		} else if rawURL != "" && strings.Contains(err.Error(), rawURL) {
			t.Error("校验错误泄露了原始地址")
		}
	}
}

func TestWaitForFrontendWithoutDevServer(t *testing.T) {
	t.Setenv("FRONTEND_DEVSERVER_URL", "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := waitForFrontend(ctx, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("未配置开发服务时应直接启动：%v", err)
	}
}

func TestWaitForDevServerSlowStartup(t *testing.T) {
	// 就绪时间超过 Wails 的 10 次、每次 500ms 重试窗口，回归检查不能仅覆盖立即成功。
	readyAt := time.Now().Add(5500 * time.Millisecond)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if time.Now().Before(readyAt) {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("FRONTEND_DEVSERVER_URL", server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := waitForFrontend(ctx, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("慢启动后应连接成功：%v", err)
	}
	if time.Now().Before(readyAt) {
		t.Fatal("前端尚未就绪就结束了等待")
	}
}

func TestWaitForDevServerTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	serverURL, err := parseDevServerURL(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = waitForDevServer(ctx, serverURL)
	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "HTTP 503") {
		t.Fatalf("超时应保留最后的 HTTP 状态：%v", err)
	}
}

func TestWaitForDevServerConnectionFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.Close()
	serverURL, err := parseDevServerURL(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = waitForDevServer(ctx, serverURL)
	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "连接失败") {
		t.Fatalf("连接失败应有界等待并报告失败类型：%v", err)
	}
}

func TestWaitForDevServerDoesNotFollowRedirects(t *testing.T) {
	var redirected atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		redirected.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer server.Close()
	serverURL, err := parseDevServerURL(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = waitForDevServer(ctx, serverURL)
	if !errors.Is(err, context.DeadlineExceeded) || redirected.Load() {
		t.Fatalf("开发探测不应跟随重定向：%v，已重定向=%v", err, redirected.Load())
	}
}

func TestWaitForDevServerCanceledRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	serverURL, err := parseDevServerURL(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := waitForDevServer(ctx, serverURL); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("请求卡住时也应遵循等待截止时间：%v", err)
	}
}
