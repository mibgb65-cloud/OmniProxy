package proxy

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// 固定入口只改变路径形状，协议处理仍以折叠后的 /v1/... 为准。
func TestResolveAgentPathFoldsFixedPrefixes(t *testing.T) {
	key := "opk_" + strings.Repeat("a", 64)
	cases := []struct {
		raw     string
		path    string
		prefix  string
		clients []string
		valid   bool
	}{
		{raw: "/codex/v1/responses", path: "/v1/responses", prefix: "/codex", clients: []string{"codex"}, valid: true},
		{raw: "/codex/v1/responses/compact", path: "/v1/responses/compact", prefix: "/codex", clients: []string{"codex"}, valid: true},
		{raw: "/claude/v1/messages", path: "/v1/messages", prefix: "/claude", clients: []string{"claude", "claude-desktop"}, valid: true},
		{raw: "/opencode/v1/chat/completions", path: "/v1/chat/completions", prefix: "/opencode", clients: []string{"opencode"}, valid: true},
		{raw: "/goose/v1/chat/completions", path: "/v1/chat/completions", prefix: "/goose", clients: []string{"goose"}, valid: true},
		{raw: "/codex", path: "/", prefix: "/codex", clients: []string{"codex"}, valid: true},
		// 近似路径不做前缀匹配，继续按未知路径 404，避免变成通用转发。
		{raw: "/codexanything/v1/responses", path: "/codexanything/v1/responses", prefix: "", clients: nil, valid: true},
		// 升级前写入的客户端配置继续可用，归属只由接入 Key 决定。
		{raw: "/v1/responses", path: "/v1/responses", prefix: "", clients: nil, valid: true},
		{raw: "/agent/codex/" + key + "/v1/responses", path: "/v1/responses", prefix: "", clients: []string{"codex"}, valid: true},
		{raw: "/agent/codex/short/v1/responses", valid: false},
	}
	for _, tc := range cases {
		req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:1"+tc.raw, nil)
		if err != nil {
			t.Fatal(err)
		}
		path, _, prefix, clients, valid := resolveAgentPath(req)
		if valid != tc.valid {
			t.Fatalf("%s validity = %t, want %t", tc.raw, valid, tc.valid)
		}
		if !valid {
			continue
		}
		if path != tc.path || prefix != tc.prefix || len(clients) != len(tc.clients) {
			t.Fatalf("%s = (%s, %s, %v), want (%s, %s, %v)", tc.raw, path, prefix, clients, tc.path, tc.prefix, tc.clients)
		}
		for index := range clients {
			if clients[index] != tc.clients[index] {
				t.Fatalf("%s clients = %v, want %v", tc.raw, clients, tc.clients)
			}
		}
	}
}

// 前缀不能放宽请求体上限：Responses／compact 仍取 16 MiB，Chat 仍是 2 MiB。
func TestPrefixedEntriesKeepProtocolLimits(t *testing.T) {
	for _, prefix := range []string{"/codex", "/claude", "/opencode", "/goose"} {
		for path, want := range map[string]int{
			"/v1/chat/completions":  MaxRequestBytes,
			"/v1/responses":         MaxResponsesRequestBytes,
			"/v1/responses/compact": MaxResponsesRequestBytes,
			"/v1/messages":          MaxMessagesRequestBytes,
		} {
			req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:1"+prefix+path, nil)
			if err != nil {
				t.Fatal(err)
			}
			folded, _, _, _, valid := resolveAgentPath(req)
			if !valid || requestBodyLimit(folded) != want {
				t.Fatalf("%s%s limit = %d, want %d", prefix, path, requestBodyLimit(folded), want)
			}
		}
	}
}

func TestCodexEntryAcceptsOfficialLoginWithoutAccessKey(t *testing.T) {
	protocol := ""
	upstream := upstreamFunc(func(_ context.Context, request Request, consume func(Response) error) error {
		protocol = request.Protocol
		return consume(Response{StatusCode: 200, ContentType: "application/json", Body: strings.NewReader(`{"choices":[{"message":{"content":"OK"}}]}`)})
	})
	authorized := 0
	server := New(Options{
		Upstream: upstream,
		Models:   []string{"deepseek-flash"},
		Authorize: func(context.Context, string) (string, bool) {
			authorized++
			return "", false
		},
		PrefixAccess: func(_ context.Context, prefix string) (string, bool) {
			return "codex", prefix == "/codex"
		},
	})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop(context.Background())
	req, _ := http.NewRequest(http.MethodPost, server.Status().Address+"/codex/v1/chat/completions", strings.NewReader(validBody))
	// Codex 官方登录令牌是超长 JWT：既不是接入 Key，也不能进入接入 Key 比较。
	req.Header.Set("Authorization", "Bearer "+strings.Repeat("j", 1400))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || protocol != "chat" {
		t.Fatalf("codex entry rejected: %d %s protocol=%s", resp.StatusCode, body, protocol)
	}
	if authorized != 0 {
		t.Fatalf("oversized token reached access key comparison %d time(s)", authorized)
	}
}

func TestFixedEntryRejectsForeignKey(t *testing.T) {
	server := New(Options{
		Upstream:  upstreamFunc(successfulUpstream),
		Models:    []string{"deepseek-flash"},
		Authorize: func(_ context.Context, key string) (string, bool) { return "claude", key == "claude-key" },
	})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop(context.Background())
	req, _ := http.NewRequest(http.MethodPost, server.Status().Address+"/codex/v1/chat/completions", strings.NewReader(validBody))
	req.Header.Set("Authorization", "Bearer claude-key")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 403 || !strings.Contains(string(body), "client_mismatch") {
		t.Fatalf("foreign key accepted on codex entry: %d %s", resp.StatusCode, body)
	}
}

func TestPrefixedEntryRequiresAccessKeyWhenPrefixAccessAbsent(t *testing.T) {
	server := New(Options{
		Upstream:  upstreamFunc(successfulUpstream),
		Models:    []string{"deepseek-flash"},
		Authorize: func(context.Context, string) (string, bool) { return "", false },
	})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop(context.Background())
	resp, err := http.Post(server.Status().Address+"/opencode/v1/chat/completions", "application/json", strings.NewReader(validBody))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("unkeyed prefixed entry accepted: %d", resp.StatusCode)
	}
}

// Claude Code 实际请求 /v1/messages?beta=true；折叠后查询参数校验必须仍然放行。
func TestPrefixedClaudeMessagesAcceptsBetaQuery(t *testing.T) {
	server := New(Options{
		Upstream: upstreamFunc(func(_ context.Context, request Request, consume func(Response) error) error {
			if request.Protocol != "messages" {
				t.Errorf("protocol = %s, want messages", request.Protocol)
			}
			return consume(Response{StatusCode: 200, ContentType: "application/json", Body: strings.NewReader(`{"type":"message","content":[]}`)})
		}),
		Models:    []string{"deepseek-flash"},
		Authorize: func(_ context.Context, key string) (string, bool) { return "claude", key == "claude-key" },
	})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop(context.Background())
	req, _ := http.NewRequest(http.MethodPost, server.Status().Address+"/claude/v1/messages?beta=true", strings.NewReader(`{"model":"deepseek-flash","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("Authorization", "Bearer claude-key")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("prefixed messages rejected: %d %s", resp.StatusCode, body)
	}
}
