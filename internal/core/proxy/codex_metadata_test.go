package proxy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCodexMetadataKeepsV2AndFiltersUnknownFeatures(t *testing.T) {
	header := http.Header{"X-Codex-Beta-Features": {"unknown-feature, remote_compaction_v2"}, "X-Codex-Turn-Metadata": {`{"compaction":{"implementation":"responses_compaction_v2"}}`}}
	value, err := readCodexMetadata(header)
	if err != nil || value.BetaFeatures != codexCompactionV2 || value.TurnMetadata != header.Get("X-Codex-Turn-Metadata") {
		t.Fatal("V2 compaction protocol metadata lost")
	}
	header.Set("X-Codex-Beta-Features", "unknown-feature")
	value, err = readCodexMetadata(header)
	if err != nil || value.BetaFeatures != "" || value.TurnMetadata != "" {
		t.Fatal("unrecognized features or their metadata forwarded")
	}
}

func TestCodexMetadataRejectsInvalidHeaders(t *testing.T) {
	for _, raw := range []string{`[]`, `null`, `{"duplicate":1,"duplicate":2}`, `{"invalid":`, "{\n}", strings.Repeat("x", maxCodexMetadataBytes+1)} {
		if ValidCodexMetadata(CodexMetadata{BetaFeatures: codexCompactionV2, TurnMetadata: raw}) {
			t.Fatal("invalid protocol metadata accepted")
		}
	}
	for _, header := range []http.Header{
		{"X-Codex-Beta-Features": {strings.Repeat("x", 1025)}},
		{"X-Codex-Beta-Features": {"remote_compaction_v2\r\nInjected: yes"}},
		{"X-Codex-Beta-Features": {"remote_compaction_v2", "remote_compaction_v2"}},
		{"X-Codex-Beta-Features": {"remote_compaction_v2"}, "X-Codex-Turn-Metadata": {"{}", "{}"}},
	} {
		if _, err := readCodexMetadata(header); err == nil {
			t.Fatal("unsafe or ambiguous metadata header accepted")
		}
	}
	if ValidCodexMetadata(CodexMetadata{BetaFeatures: "unrecognized"}) || ValidCodexMetadata(CodexMetadata{TurnMetadata: "{}"}) {
		t.Fatal("untrusted metadata bypassed the feature allowlist")
	}
}

func TestWebSocketFrameMetadataUsesCurrentTurn(t *testing.T) {
	base := CodexMetadata{BetaFeatures: codexCompactionV2, TurnMetadata: `{"request_kind":"old"}`}
	frame, _ := json.Marshal(map[string]string{"x-codex-turn-metadata": `{"request_kind":"current"}`})
	value, err := frameCodexMetadata(base, frame)
	if err != nil || value.TurnMetadata != `{"request_kind":"current"}` {
		t.Fatal("current turn metadata was replaced by the handshake")
	}
	if _, err := frameCodexMetadata(base, []byte(`{"x-codex-turn-metadata":"not-json"}`)); err == nil {
		t.Fatal("invalid per-frame metadata accepted")
	}
}

func TestCodexMetadataHTTPRejectsAmbiguityBeforeForwarding(t *testing.T) {
	calls := 0
	metadata := `{"compaction":{"implementation":"responses_compaction_v2"}}`
	server := New(Options{Port: 0, Models: []string{"gpt-test"}, Upstream: upstreamFunc(func(_ context.Context, r Request, consume func(Response) error) error {
		calls++
		if r.CodexMetadata.BetaFeatures != codexCompactionV2 || r.CodexMetadata.TurnMetadata != metadata || r.Compaction {
			t.Error("V2 metadata was lost or confused with the legacy endpoint")
		}
		return consume(Response{StatusCode: 200, ContentType: "application/json", Body: strings.NewReader(`{"output":[{"type":"compaction","encrypted_content":"opaque-qa"}]}`)})
	})})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop(context.Background())
	for _, test := range []struct {
		name   string
		header http.Header
		status int
	}{
		{"valid V2", http.Header{"X-Codex-Beta-Features": {codexCompactionV2}, "X-Codex-Turn-Metadata": {metadata}}, 200},
		{"duplicate feature header", http.Header{"X-Codex-Beta-Features": {codexCompactionV2, codexCompactionV2}}, 400},
		{"duplicate metadata header", http.Header{"X-Codex-Beta-Features": {codexCompactionV2}, "X-Codex-Turn-Metadata": {metadata, metadata}}, 400},
		{"duplicate JSON field", http.Header{"X-Codex-Beta-Features": {codexCompactionV2}, "X-Codex-Turn-Metadata": {`{"compaction":{},"compaction":{}}`}}, 400},
		{"invalid JSON shape", http.Header{"X-Codex-Beta-Features": {codexCompactionV2}, "X-Codex-Turn-Metadata": {`[]`}}, 400},
		{"metadata too large", http.Header{"X-Codex-Beta-Features": {codexCompactionV2}, "X-Codex-Turn-Metadata": {`{"value":"` + strings.Repeat("a", maxCodexMetadataBytes) + `"}`}}, 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", server.Status().Address+"/v1/responses", strings.NewReader(`{"model":"gpt-test","input":[]}`))
			if err != nil {
				t.Fatal(err)
			}
			req.Header = test.header.Clone()
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil || resp.StatusCode != test.status {
				t.Fatal("unexpected metadata boundary response", resp.StatusCode)
			}
			if test.status != 200 && strings.Contains(string(raw), "compaction") {
				t.Fatal("invalid metadata leaked into the error response")
			}
		})
	}
	if calls != 1 {
		t.Fatal("invalid metadata reached the upstream")
	}
}
