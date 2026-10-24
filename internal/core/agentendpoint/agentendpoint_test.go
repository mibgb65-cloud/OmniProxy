package agentendpoint

import "testing"

func TestPrefixMatchesBaseURL(t *testing.T) {
	for _, client := range []string{"codex", "claude", "claude-desktop", "opencode", "kilocode", "mimo-code", "qwen-code", "harness-web", "harness-desktop", "kimi-code", "pi", "factory-droid", "goose"} {
		prefix, ok := Prefix(client)
		if !ok || prefix == "" || prefix[0] != '/' {
			t.Fatalf("client %q has no valid prefix: %q %t", client, prefix, ok)
		}
		base, ok := BaseURL(client, 3001)
		if !ok || base != "http://127.0.0.1:3001"+prefix {
			t.Fatalf("client %q base url = %q, want prefix suffix", client, base)
		}
		allowed := Clients(prefix)
		found := false
		for _, candidate := range allowed {
			if candidate == client {
				found = true
			}
		}
		if !found {
			t.Fatalf("prefix %q does not admit its own client %q: %v", prefix, client, allowed)
		}
	}
}

func TestPrefixSharedByClaudeClients(t *testing.T) {
	if got := Clients("/claude"); len(got) != 2 || got[0] != "claude" || got[1] != "claude-desktop" {
		t.Fatalf("expected /claude to admit both Claude clients, got %v", got)
	}
}

func TestUnknownInputsAreRejected(t *testing.T) {
	if _, ok := Prefix("unknown"); ok {
		t.Fatal("expected unknown client to be rejected")
	}
	if _, ok := BaseURL("unknown", 3001); ok {
		t.Fatal("expected unknown client base url to be rejected")
	}
	if Clients("/unknown") != nil {
		t.Fatal("expected unknown prefix to admit no client")
	}
}

// TestRuntimeSeparatesCodexRuntimes 断言共用前缀的 Codex 运行形态按来源标识区分，
// 并且只认固定标记：未知来源保持 codex，其他客户端完全不受影响。
func TestRuntimeSeparatesCodexRuntimes(t *testing.T) {
	for _, test := range []struct {
		originator, userAgent, want string
	}{
		{"codex_cli_rs", "codex_cli_rs/0.163.0", "codex-cli"},
		{"Codex Desktop", "codex_cli_rs/0.163.0-alpha.10.2", "codex-desktop"},
		{"codex_desktop", "", "codex-desktop"},
		{"codex_vscode", "", "codex-vscode"},
		{"", "codex_cli_rs/0.163.0", "codex-cli"},
		{"omniproxy", "OmniProxy/2.0.0", "codex"},
		{"", "", "codex"},
	} {
		if got := Runtime("codex", test.originator, test.userAgent); got != test.want {
			t.Fatalf("Runtime(codex, %q, %q) = %q, want %q", test.originator, test.userAgent, got, test.want)
		}
	}
	if got := Runtime("claude-desktop", "codex_cli_rs", "codex_cli_rs/1.0"); got != "claude-desktop" {
		t.Fatalf("其他客户端被改写: %q", got)
	}
}
