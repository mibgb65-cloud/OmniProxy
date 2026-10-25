// Package proxysettings 保存提供商出站代理的连接参数，并向 Core 提供配置快照。
//
// 密码只以 credential_ref 的形式落库（DPAPI 凭据文件），读取时按需解密。
package proxysettings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"omniproxy/internal/application/provider"
	"omniproxy/internal/core/netproxy"
)

const (
	cacheTTL       = 5 * time.Second
	maxNoProxyLen  = 512
	maxUsernameLen = 256
	testTimeout    = 5 * time.Second
)

var (
	// ErrInvalidInput 表示代理参数不合法。
	ErrInvalidInput = errors.New("proxysettings: invalid input")
	// ErrUnavailable 表示设置记录缺失。
	ErrUnavailable = errors.New("proxysettings: configuration_unavailable")
)

// SecretStore 与凭据库一致的最小接口，便于测试注入。
type SecretStore interface {
	Put(secret []byte) (string, error)
	Get(ref string) ([]byte, error)
	Delete(ref string) error
}

// Settings 是返回给 UI 的代理配置，不含明文密码。
type Settings struct {
	Enabled        bool   `json:"enabled"`
	Scheme         string `json:"scheme"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Username       string `json:"username"`
	HasPassword    bool   `json:"hasPassword"`
	NoProxy        string `json:"noProxy"`
	DetectedScheme string `json:"detectedScheme"`
	UpdatedAt      string `json:"updatedAt"`
	// Endpoint 是脱敏后的端点文本，UI 直接展示它，不自行拼接凭证。
	Endpoint string `json:"endpoint"`
}

// SaveInput 是保存代理设置的入参。
type SaveInput struct {
	Enabled       bool   `json:"enabled"`
	Scheme        string `json:"scheme"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	ClearPassword bool   `json:"clearPassword"`
	NoProxy       string `json:"noProxy"`
}

// TestResult 描述一次端点探测结果。
type TestResult struct {
	Reachable  bool   `json:"reachable"`
	SOCKS5     bool   `json:"socks5"`
	HTTP       bool   `json:"http"`
	Auth       bool   `json:"auth"`
	Scheme     string `json:"scheme"`
	ErrorCode  string `json:"errorCode"`
	Endpoint   string `json:"endpoint"`
	LatencyMs  int64  `json:"latencyMs"`
	DetectedAt string `json:"detectedAt"`
}

// Discovered 是本地端口发现结果。
type Discovered struct {
	Endpoint  string `json:"endpoint"`
	Scheme    string `json:"scheme"`
	SOCKS5    bool   `json:"socks5"`
	HTTP      bool   `json:"http"`
	Auth      bool   `json:"auth"`
	Port      int    `json:"port"`
	LatencyMs int64  `json:"latencyMs"`
}

// Service 读写代理设置，并实现 netproxy.Resolver。
type Service struct {
	db      *sql.DB
	secrets SecretStore
	now     func() time.Time

	mu      sync.Mutex
	cache   netproxy.Snapshot
	cached  time.Time
	hasCash bool
}

// NewService 创建独立于桌面适配的服务。
func NewService(db *sql.DB, secrets SecretStore) *Service {
	return &Service{db: db, secrets: secrets, now: time.Now}
}

// Read 返回不含密码的设置。
func (s *Service) Read(ctx context.Context) (Settings, error) {
	row, err := s.readRow(ctx)
	if err != nil {
		return Settings{}, err
	}
	endpoint := netproxy.Endpoint{Scheme: row.scheme, Host: row.host, Port: row.port}
	return Settings{
		Enabled: row.enabled, Scheme: row.scheme, Host: row.host, Port: row.port,
		Username: row.username, HasPassword: row.passwordRef != "", NoProxy: row.noProxy,
		DetectedScheme: row.detected, UpdatedAt: row.updatedAt, Endpoint: endpoint.String(),
	}, nil
}

// Save 校验并保存设置；密码为空表示保留原密码。
func (s *Service) Save(ctx context.Context, input SaveInput) (Settings, error) {
	normalized, err := normalize(input)
	if err != nil {
		return Settings{}, err
	}
	current, err := s.readRow(ctx)
	if err != nil {
		return Settings{}, err
	}
	passwordRef := current.passwordRef
	switch {
	case normalized.ClearPassword:
		if passwordRef != "" && s.secrets != nil {
			if deleteErr := s.secrets.Delete(passwordRef); deleteErr != nil {
				return Settings{}, fmt.Errorf("proxysettings: clear password: %w", deleteErr)
			}
		}
		passwordRef = ""
	case normalized.Password != "":
		if s.secrets == nil {
			return Settings{}, fmt.Errorf("%w: secret store unavailable", ErrInvalidInput)
		}
		ref, putErr := s.secrets.Put([]byte(normalized.Password))
		if putErr != nil {
			return Settings{}, fmt.Errorf("proxysettings: store password: %w", putErr)
		}
		if passwordRef != "" {
			_ = s.secrets.Delete(passwordRef)
		}
		passwordRef = ref
	}
	detected := current.detected
	if current.host != normalized.Host || current.port != normalized.Port || current.username != normalized.Username {
		// 端点变化后旧的协议识别结果不再成立。
		detected = ""
	}
	updatedAt := s.now().UTC().Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `
		UPDATE proxy_settings SET enabled=?, scheme=?, host=?, port=?, username=?, password_ref=?, no_proxy=?, detected_scheme=?, updated_at=?
		WHERE id=1`,
		normalized.Enabled, normalized.Scheme, normalized.Host, normalized.Port, normalized.Username,
		passwordRef, normalized.NoProxy, detected, updatedAt); err != nil {
		return Settings{}, fmt.Errorf("proxysettings: save: %w", err)
	}
	s.Invalidate()
	return s.Read(ctx)
}

// Test 探测给定端点，返回可展示的协议与错误码，不改变已保存设置。
func (s *Service) Test(ctx context.Context, input SaveInput) (TestResult, error) {
	normalized, err := normalize(input)
	if err != nil {
		return TestResult{}, err
	}
	endpoint := netproxy.Endpoint{Scheme: normalized.Scheme, Host: normalized.Host, Port: normalized.Port, Username: normalized.Username}
	if endpoint.Username != "" {
		password := normalized.Password
		if password == "" {
			if stored, _, readErr := s.storedCredentials(ctx); readErr == nil {
				password = stored
			}
		}
		endpoint.Password = password
	}
	ctx, cancel := context.WithTimeout(ctx, testTimeout)
	defer cancel()
	result := TestResult{Endpoint: endpoint.String()}
	detection, detectErr := netproxy.Detect(ctx, endpoint)
	result.SOCKS5, result.HTTP, result.Auth, result.LatencyMs = detection.SOCKS5, detection.HTTP, detection.Auth, detection.LatencyMs
	if detectErr != nil {
		result.ErrorCode = netproxy.CodeOf(detectErr)
		if result.ErrorCode == "" {
			result.ErrorCode = "proxy_protocol_mismatch"
		}
		return result, nil
	}
	result.Reachable = true
	result.Scheme = detection.Scheme
	result.DetectedAt = s.now().UTC().Format(time.RFC3339Nano)
	return result, nil
}

// Discover 探测本机常见代理端口。
func (s *Service) Discover(ctx context.Context) ([]Discovered, error) {
	found := netproxy.Discover(ctx, netproxy.DefaultHost, nil)
	result := make([]Discovered, 0, len(found))
	for _, item := range found {
		result = append(result, Discovered{
			Endpoint: item.Endpoint, Scheme: item.Scheme, SOCKS5: item.SOCKS5,
			HTTP: item.HTTP, Auth: item.Auth, Port: item.Port, LatencyMs: item.LatencyMs,
		})
	}
	return result, nil
}

// Snapshot 实现 netproxy.Resolver；5 秒内复用缓存，保存设置后立即失效。
func (s *Service) Snapshot(ctx context.Context) (netproxy.Snapshot, error) {
	s.mu.Lock()
	if s.hasCash && s.now().Sub(s.cached) < cacheTTL {
		snapshot := s.cache
		s.mu.Unlock()
		return snapshot, nil
	}
	s.mu.Unlock()

	snapshot, err := s.buildSnapshot(ctx)
	if err != nil {
		return netproxy.Snapshot{}, err
	}
	s.mu.Lock()
	s.cache, s.cached, s.hasCash = snapshot, s.now(), true
	s.mu.Unlock()
	return snapshot, nil
}

// Invalidate 清空快照缓存，保存设置后调用。
func (s *Service) Invalidate() {
	s.mu.Lock()
	s.hasCash = false
	s.cache = netproxy.Snapshot{}
	s.mu.Unlock()
}

// RememberDetectedScheme 回写协议识别结果，下次启动无需再次探测。
func (s *Service) RememberDetectedScheme(ctx context.Context, scheme string) error {
	if scheme != netproxy.SchemeHTTP && scheme != netproxy.SchemeSOCKS5 {
		return fmt.Errorf("%w: scheme", ErrInvalidInput)
	}
	if s.db == nil {
		return ErrUnavailable
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE proxy_settings SET detected_scheme=? WHERE id=1`, scheme); err != nil {
		return fmt.Errorf("proxysettings: remember scheme: %w", err)
	}
	s.Invalidate()
	return nil
}

type row struct {
	enabled     bool
	scheme      string
	host        string
	port        int
	username    string
	passwordRef string
	noProxy     string
	detected    string
	updatedAt   string
}

func (s *Service) readRow(ctx context.Context) (row, error) {
	if s.db == nil {
		return row{}, ErrUnavailable
	}
	var item row
	var enabled int
	err := s.db.QueryRowContext(ctx, `
		SELECT enabled, scheme, host, port, username, password_ref, no_proxy, detected_scheme, updated_at
		FROM proxy_settings WHERE id=1`).Scan(
		&enabled, &item.scheme, &item.host, &item.port, &item.username, &item.passwordRef, &item.noProxy, &item.detected, &item.updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return row{}, ErrUnavailable
	}
	if err != nil {
		return row{}, fmt.Errorf("proxysettings: read: %w", err)
	}
	if enabled != 0 && enabled != 1 {
		return row{}, fmt.Errorf("%w: enabled flag", ErrUnavailable)
	}
	item.enabled = enabled == 1
	return item, nil
}

func (s *Service) storedCredentials(ctx context.Context) (string, string, error) {
	current, err := s.readRow(ctx)
	if err != nil {
		return "", "", err
	}
	if current.passwordRef == "" || s.secrets == nil {
		return "", current.username, nil
	}
	secret, err := s.secrets.Get(current.passwordRef)
	if err != nil {
		return "", current.username, fmt.Errorf("proxysettings: read password: %w", err)
	}
	defer clear(secret)
	return string(secret), current.username, nil
}

// buildSnapshot 汇总代理参数与每个提供商的路由策略。
func (s *Service) buildSnapshot(ctx context.Context) (netproxy.Snapshot, error) {
	current, err := s.readRow(ctx)
	if errors.Is(err, ErrUnavailable) || isMissingSchema(err) {
		// 还没有迁移到 044 的数据库按"未启用代理"处理，不能因此阻断所有出站。
		return netproxy.Snapshot{Version: "unavailable"}, nil
	}
	if err != nil {
		return netproxy.Snapshot{}, err
	}
	password, _, err := s.storedCredentials(ctx)
	if err != nil {
		// 密码读不出来时仍返回端点：认证失败会以代理错误暴露，不静默改成直连。
		password = ""
	}
	modes, hostModes, err := s.providerModes(ctx)
	if err != nil {
		if isMissingSchema(err) {
			return netproxy.Snapshot{Version: "unavailable"}, nil
		}
		return netproxy.Snapshot{}, err
	}
	return netproxy.Snapshot{
		Version:   current.updatedAt + "|" + hostModesVersion(hostModes),
		Enabled:   current.enabled,
		Endpoint:  netproxy.Endpoint{Scheme: current.scheme, Host: current.host, Port: current.port, Username: current.username, Password: password},
		Detected:  current.detected,
		NoProxy:   splitList(current.noProxy),
		Modes:     modes,
		HostModes: hostModes,
	}, nil
}

func (s *Service) providerModes(ctx context.Context) (map[string]string, map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, kind, proxy_mode, base_url FROM providers`)
	if err != nil {
		return nil, nil, fmt.Errorf("proxysettings: provider modes: %w", err)
	}
	defer func() { _ = rows.Close() }()
	modes := make(map[string]string)
	hostModes := make(map[string]string)
	for rows.Next() {
		var id, kind, mode, baseURL string
		if err := rows.Scan(&id, &kind, &mode, &baseURL); err != nil {
			return nil, nil, fmt.Errorf("proxysettings: scan provider mode: %w", err)
		}
		if !netproxy.ValidMode(mode) {
			mode = netproxy.ModeInherit
		}
		modes[id] = mode
		// 内置提供商常常没有自定义 base URL：用目录默认端点参与主机匹配，
		// 让没有请求上下文的额度/余额查询也能找到正确的提供商策略。
		candidates := []string{baseURL}
		if baseURL == "" {
			if definition, ok := provider.DefinitionFor(kind); ok {
				candidates = append(candidates, definition.DefaultBaseURL)
			}
		}
		for _, candidate := range candidates {
			host := hostOf(candidate)
			if host == "" {
				continue
			}
			if existing, exists := hostModes[host]; !exists || existing == netproxy.ModeInherit || mode != netproxy.ModeInherit {
				hostModes[host] = mode
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("proxysettings: iterate provider modes: %w", err)
	}
	return modes, hostModes, nil
}

func hostOf(baseURL string) string {
	if baseURL == "" {
		return ""
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" {
		return ""
	}
	return strings.ToLower(parsed.Host)
}

// isMissingSchema 识别"还没有迁移到 044"的数据库。
func isMissingSchema(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "no such table") || strings.Contains(text, "no such column")
}

func hostModesVersion(hostModes map[string]string) string {
	if len(hostModes) == 0 {
		return ""
	}
	var builder strings.Builder
	for host, mode := range hostModes {
		builder.WriteString(host)
		builder.WriteByte('=')
		builder.WriteString(mode)
		builder.WriteByte(';')
	}
	return builder.String()
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

// normalize 校验并规范化入参；代理端口没有默认值，必须显式给出。
func normalize(input SaveInput) (SaveInput, error) {
	scheme := strings.TrimSpace(input.Scheme)
	if scheme == "" {
		scheme = netproxy.SchemeAuto
	}
	if !netproxy.ValidScheme(scheme) {
		return SaveInput{}, fmt.Errorf("%w: scheme", ErrInvalidInput)
	}
	host := strings.TrimSpace(input.Host)
	if host == "" {
		host = netproxy.DefaultHost
	}
	if len(host) > 255 || strings.ContainsAny(host, " \t\r\n/\\@") {
		return SaveInput{}, fmt.Errorf("%w: host", ErrInvalidInput)
	}
	if net.ParseIP(host) == nil {
		for _, char := range host {
			if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '-' || char == '.') {
				return SaveInput{}, fmt.Errorf("%w: host", ErrInvalidInput)
			}
		}
		if host[0] == '-' || host[len(host)-1] == '-' || host[0] == '.' {
			return SaveInput{}, fmt.Errorf("%w: host", ErrInvalidInput)
		}
	}
	if input.Port < 1 || input.Port > 65535 {
		return SaveInput{}, fmt.Errorf("%w: port", ErrInvalidInput)
	}
	username := input.Username
	if len(username) > maxUsernameLen {
		return SaveInput{}, fmt.Errorf("%w: username", ErrInvalidInput)
	}
	if len(input.Password) > 256 {
		return SaveInput{}, fmt.Errorf("%w: password", ErrInvalidInput)
	}
	for _, value := range []string{username, input.Password} {
		for _, char := range value {
			if char < 33 || char > 126 {
				return SaveInput{}, fmt.Errorf("%w: credentials", ErrInvalidInput)
			}
		}
	}
	if input.Password != "" && username == "" {
		return SaveInput{}, fmt.Errorf("%w: password requires username", ErrInvalidInput)
	}
	noProxy := strings.TrimSpace(input.NoProxy)
	if noProxy == "" {
		noProxy = "localhost,127.0.0.1,::1"
	}
	if len(noProxy) > maxNoProxyLen {
		return SaveInput{}, fmt.Errorf("%w: no_proxy too long", ErrInvalidInput)
	}
	for _, entry := range splitList(noProxy) {
		if strings.ContainsAny(entry, " \t\r\n") || len(entry) > 255 {
			return SaveInput{}, fmt.Errorf("%w: no_proxy entry", ErrInvalidInput)
		}
	}
	return SaveInput{
		Enabled: input.Enabled, Scheme: scheme, Host: host, Port: input.Port,
		Username: username, Password: input.Password, ClearPassword: input.ClearPassword, NoProxy: noProxy,
	}, nil
}

// EndpointOf 把保存的字段还原成 Core 端点，供其他调用方复用。
func EndpointOf(settings Settings) netproxy.Endpoint {
	return netproxy.Endpoint{Scheme: settings.Scheme, Host: settings.Host, Port: settings.Port, Username: settings.Username}
}
