package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/core/proxy"
)

func TestClaudeBetaContextVariantsReachDeepSeek(t *testing.T) {
	seen := 0
	service, db, _ := setupService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model     string
			Stream    bool
			MaxTokens int              `json:"max_tokens"`
			Tools     []map[string]any `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if r.URL.RawQuery != "" || body.Model != "deepseek-flash" && body.Model != "deepseek-v4-pro" || body.Stream || body.MaxTokens != 32000 || len(body.Tools) != 1 {
			t.Error("Claude request did not convert to the actual upstream protocol")
		}
		seen++
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"test","model":"`+body.Model+`","choices":[{"message":{"content":"OK"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`)
	}))
	request := func(model string, expected int) {
		t.Helper()
		// 保留本机 CLI 的关键字段形态，不使用真实提示词或会话元数据。
		body, _ := json.Marshal(map[string]any{
			"model": model, "stream": true, "max_tokens": 32000,
			"system":   []any{map[string]any{"type": "text", "text": "Reply OK.", "cache_control": map[string]string{"type": "ephemeral"}}},
			"messages": []any{map[string]any{"role": "user", "content": "hello"}, map[string]any{"role": "system", "content": []any{map[string]string{"type": "text", "text": "reminder"}}}},
			"tools":    []any{map[string]any{"name": "Read", "input_schema": map[string]any{"type": "object", "properties": map[string]any{"path": map[string]string{"type": "string"}}}}},
			"thinking": map[string]string{"type": "adaptive"}, "context_management": map[string]any{"edits": []any{}}, "output_config": map[string]string{"effort": "high"},
		})
		response, err := http.Post(service.Info().Runtime.Address+"/v1/messages?beta=true", "application/json", strings.NewReader(string(body)))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		result, err := io.ReadAll(response.Body)
		if err != nil || response.StatusCode != expected {
			t.Fatalf("status=%d expected=%d error=%v", response.StatusCode, expected, err)
		}
		if expected == 200 && (!strings.Contains(string(result), "message_stop") || !strings.Contains(string(result), "OK")) {
			t.Fatal("Claude streaming reply incomplete")
		}
	}
	for _, id := range []string{"deepseek-flash", "deepseek-v4-pro"} {
		for _, model := range []string{id, id + "[1m]", catalog.DesktopAlias(id), catalog.DesktopAlias(id + "[1m]"), strings.TrimSuffix(catalog.DesktopAlias(id+"[1m]"), "[1m]"), catalog.DesktopAlias(id) + "[1m]"} {
			request(model, 200)
		}
	}
	request("unknown[1m]", 400)
	request(catalog.DesktopAlias("unknown[1m]"), 400)
	if _, err := db.Exec(`UPDATE accounts SET enabled=0`); err != nil {
		t.Fatal(err)
	}
	request("deepseek-flash[1m]", 503)
	if seen != 12 {
		t.Fatal("unavailable models reached upstream")
	}
}

func TestClaudeLongContextPreservesSelectedUserRoute(t *testing.T) {
	window := catalog.ClaudeLongContextWindow
	router := &desktopAliasRouter{entries: []catalog.Entry{{ID: "coding-route", Alias: true, TargetModel: "deepseek-flash", Metadata: catalog.Metadata{ContextWindow: &window}}}}
	service := &Service{router: router}
	for _, model := range []string{"coding-route[1m]", catalog.DesktopAlias("coding-route[1m]"), strings.TrimSuffix(catalog.DesktopAlias("coding-route[1m]"), "[1m]")} {
		body, _ := json.Marshal(map[string]string{"model": model})
		request, err := service.RouteRequest(context.Background(), proxy.Request{Protocol: "messages", Model: model, Body: body})
		if err != nil || router.input != "coding-route" || request.Model != "deepseek-flash" || request.AccountID != "acct_first" {
			t.Fatal("context variant bypassed original route", err)
		}
	}
	router.entries[0].Metadata.ContextWindow = nil
	if _, err := service.RouteRequest(context.Background(), proxy.Request{Protocol: "messages", Model: "coding-route[1m]"}); err == nil {
		t.Fatal("unknown target context accepted")
	}
}

func TestClaudeLongContextBodyBeyondLegacyLimit(t *testing.T) {
	text := strings.Repeat("x", proxy.MaxRequestBytes+1)
	seen := false
	service, _, _ := setupService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model    string
			Messages []struct{ Content string }
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body.Model != "deepseek-flash" || len(body.Messages) != 1 || body.Messages[0].Content != text {
			t.Error("long context content was truncated or misrouted")
		}
		seen = true
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"content":"OK"}}]}`)
	}))
	body, _ := json.Marshal(map[string]any{"model": catalog.DesktopAlias("deepseek-flash[1m]"), "max_tokens": 1, "messages": []map[string]string{{"role": "user", "content": text}}})
	response, err := http.Post(service.Info().Runtime.Address+"/v1/messages?beta=true", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if _, err := io.Copy(io.Discard, response.Body); err != nil || response.StatusCode != 200 || !seen {
		t.Fatal("long context hit the legacy 2 MiB request limit", response.StatusCode, err)
	}
}
