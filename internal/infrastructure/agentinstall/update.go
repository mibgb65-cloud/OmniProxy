package agentinstall

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrUpdateUnsupported 表示客户端尚无经过核实的版本查询渠道。
	ErrUpdateUnsupported = errors.New("agentinstall: update_unsupported")
	// ErrUpdateFailed 不包含请求地址、响应内容或本机路径。
	ErrUpdateFailed   = errors.New("agentinstall: update_check_failed")
	claudeReleasePath = regexp.MustCompile(`^/releases/win32/(x64|arm64)/([0-9.]{5,23})/Claude-[a-f0-9]{40,64}\.msix$`)
)

// UpdateCheck 仅返回可验证的公开版本；unknown 不能解释为没有更新。
type UpdateCheck struct {
	State          string `json:"state"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
}

// CheckUpdate 只读取官方最新 MSIX 入口的重定向，不下载安装包，也不触发更新。
func (b *Backend) CheckUpdate(ctx context.Context, id, current string) (UpdateCheck, error) {
	if _, err := Find(id); err != nil {
		return UpdateCheck{}, err
	}
	if id != "claude-desktop" {
		return UpdateCheck{}, ErrUpdateUnsupported
	}
	arch := map[string]string{"amd64": "x64", "arm64": "arm64"}[runtime.GOARCH]
	if runtime.GOOS != "windows" || arch == "" {
		return UpdateCheck{}, ErrUpdateUnsupported
	}
	result := UpdateCheck{State: "unknown"}
	installed, ok := desktopVersion(current)
	if !ok {
		return result, nil
	}
	result.CurrentVersion = current
	client := http.Client{Timeout: 15 * time.Second}
	if b.updateHTTP != nil {
		client = *b.updateHTTP
	}
	// 官方端点不支持 HEAD。GET 仅接收重定向响应头，禁止继续请求安装包。
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://claude.ai/api/desktop/win32/"+arch+"/msix/latest/redirect", nil)
	if err != nil {
		return UpdateCheck{}, ErrUpdateFailed
	}
	request.Header.Set("User-Agent", "OmniProxy")
	response, err := client.Do(request)
	if err != nil {
		return UpdateCheck{}, ErrUpdateFailed
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 300 || response.StatusCode > 399 {
		return UpdateCheck{}, ErrUpdateFailed
	}
	if len(response.Header.Get("Location")) > 1024 {
		return result, nil
	}
	location, err := response.Location()
	if err != nil || location.Scheme != "https" || location.Host != "downloads.claude.ai" || location.User != nil || location.RawQuery != "" || location.Fragment != "" || location.RawPath != "" {
		return result, nil
	}
	parts := claudeReleasePath.FindStringSubmatch(location.Path)
	if len(parts) != 3 || parts[1] != arch {
		return result, nil
	}
	latest, ok := desktopVersion(parts[2])
	if !ok {
		return result, nil
	}
	result.LatestVersion, result.State = parts[2], "current"
	// 官方下载采用三段版本，MSIX 检测采用四段；缺失段视为 0，禁止字符串排序。
	for index := range installed {
		if latest[index] > installed[index] {
			result.State = "available"
			break
		}
		if latest[index] < installed[index] {
			break
		}
	}
	return result, nil
}

func desktopVersion(value string) ([4]uint64, bool) {
	var version [4]uint64
	parts := strings.Split(value, ".")
	if len(value) > 23 || len(parts) < 3 || len(parts) > 4 {
		return version, false
	}
	for index, part := range parts {
		if part == "" || len(part) > 5 || strings.Trim(part, "0123456789") != "" {
			return version, false
		}
		number, err := strconv.ParseUint(part, 10, 16)
		if err != nil {
			return version, false
		}
		version[index] = number
	}
	return version, true
}
