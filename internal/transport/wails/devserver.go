//go:build !production

package wails

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	devServerWaitTimeout    = 120 * time.Second
	devServerRequestTimeout = 2 * time.Second
	devServerRetryInterval  = 500 * time.Millisecond
	devServerURLMaxLength   = 2048
)

func waitForFrontend(ctx context.Context, log *slog.Logger) error {
	rawURL := os.Getenv("FRONTEND_DEVSERVER_URL")
	if rawURL == "" {
		return nil
	}
	serverURL, err := parseDevServerURL(rawURL)
	if err != nil {
		return err
	}
	log.Info("等待前端开发服务就绪", "url", serverURL.String(), "timeout", devServerWaitTimeout)
	if err := waitForDevServer(ctx, serverURL); err != nil {
		return err
	}
	log.Info("前端开发服务已就绪", "url", serverURL.String())
	return nil
}

func parseDevServerURL(rawURL string) (*url.URL, error) {
	// 开发地址来自环境变量；拒绝凭据、远端地址和查询参数，避免等待逻辑访问外部服务或输出敏感信息。
	invalidURL := errors.New("前端开发服务地址必须是有效的本机 HTTP/HTTPS 地址，且不含凭据、路径、查询参数或片段")
	if len(rawURL) > devServerURLMaxLength {
		return nil, invalidURL
	}
	serverURL, err := url.Parse(rawURL)
	if err != nil || (serverURL.Scheme != "http" && serverURL.Scheme != "https") || serverURL.User != nil ||
		(serverURL.Path != "" && serverURL.Path != "/") || serverURL.RawQuery != "" || serverURL.ForceQuery || serverURL.Fragment != "" {
		return nil, invalidURL
	}
	host := serverURL.Hostname()
	if !strings.EqualFold(host, "localhost") {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return nil, invalidURL
		}
	}
	if portText := serverURL.Port(); portText != "" {
		port, err := strconv.Atoi(portText)
		if err != nil || port < 1 || port > 65535 {
			return nil, invalidURL
		}
	}
	return serverURL, nil
}

func waitForDevServer(ctx context.Context, serverURL *url.URL) error {
	// 应用层等待保留地址校验、取消与时限控制；不依赖 CLI 的探测配置，并与生产构建隔离。
	ctx, cancel := context.WithTimeout(ctx, devServerWaitTimeout)
	defer cancel()
	transport := &http.Transport{
		DialContext:         (&net.Dialer{Timeout: devServerRequestTimeout}).DialContext,
		TLSHandshakeTimeout: devServerRequestTimeout,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport: transport,
		Timeout:   devServerRequestTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	ticker := time.NewTicker(devServerRetryInterval)
	defer ticker.Stop()
	lastState := "尚未连接"
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("等待前端开发服务失败（最后状态：%s）: %w", lastState, err)
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL.String(), nil)
		if err != nil {
			return fmt.Errorf("创建前端开发服务检查请求失败: %w", err)
		}
		response, err := client.Do(request)
		if err != nil {
			lastState = "连接失败"
		} else if err := response.Body.Close(); err != nil {
			lastState = "响应关闭失败"
		} else if response.StatusCode == http.StatusOK {
			return nil
		} else {
			lastState = fmt.Sprintf("HTTP %d", response.StatusCode)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("等待前端开发服务失败（最后状态：%s）: %w", lastState, ctx.Err())
		case <-ticker.C:
		}
	}
}
