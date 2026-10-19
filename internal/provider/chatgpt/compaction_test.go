package chatgpt

import (
	"context"
	"io"
	"net/http"
	"omniproxy/internal/core/proxy"
	"strings"
	"testing"
)

func TestCompactUsesFixedJSONEndpointAndPreservesOpaqueHistory(t *testing.T) {
	body := `{"model":"gpt-test","instructions":"qa","input":[{"type":"compaction","encrypted_content":"opaque-previous"}]}`
	output := `{"object":"response.compaction","output":[{"type":"compaction","encrypted_content":"opaque-next"}]}`
	calls := 0
	client := New(transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		raw, err := io.ReadAll(r.Body)
		if err != nil || string(raw) != body || r.URL.String() != "https://chatgpt.com/backend-api/codex/responses/compact" || r.Method != "POST" {
			t.Error("compaction body or endpoint was rewritten")
		}
		if r.Header.Get("Accept") != "application/json" || r.Header.Get("Authorization") != "Bearer qa-only" || r.Header.Get("ChatGPT-Account-ID") != "qa-account" {
			t.Error("compaction headers incorrect")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(output))}, nil
	}))
	err := client.Compact(context.Background(), []byte("qa-only"), "qa-account", proxy.Request{Compaction: true, Protocol: "responses", Body: []byte(body)}, func(response proxy.Response) error {
		raw, err := io.ReadAll(response.Body)
		if err != nil || string(raw) != output || response.ContentType != "application/json" {
			t.Error("opaque output changed")
		}
		return err
	})
	if err != nil || calls != 1 {
		t.Fatal("compact request failed")
	}
}

func TestCompactCannotBeUsedAsStreamingOrCrossProtocolForward(t *testing.T) {
	client := New(transportFunc(func(*http.Request) (*http.Response, error) {
		t.Error("invalid operation reached transport")
		return nil, nil
	}))
	for _, request := range []proxy.Request{{Protocol: "responses", Body: []byte(`{}`)}, {Compaction: true, Protocol: "messages", Body: []byte(`{}`)}, {Compaction: true, Protocol: "responses", Stream: true, Body: []byte(`{}`)}, {Compaction: true, Protocol: "responses", Body: []byte(`{`)}} {
		if err := client.Compact(context.Background(), nil, "", request, func(proxy.Response) error { return nil }); err == nil {
			t.Fatal("invalid compaction accepted")
		}
	}
}

func TestResponsesDoesNotForwardInjectedProtocolMetadata(t *testing.T) {
	client := New(transportFunc(func(*http.Request) (*http.Response, error) {
		t.Error("invalid metadata reached upstream")
		return nil, nil
	}))
	for _, metadata := range []proxy.CodexMetadata{{BetaFeatures: "unrecognized"}, {BetaFeatures: "remote_compaction_v2", TurnMetadata: "{}\r\nInjected: true"}, {BetaFeatures: "remote_compaction_v2", TurnMetadata: `{"duplicate":1,"duplicate":2}`}} {
		if err := client.Forward(context.Background(), []byte("qa-only"), "qa", proxy.Request{Protocol: "responses", Body: []byte(`{"model":"gpt-test","input":[]}`), CodexMetadata: metadata}, func(proxy.Response) error { return nil }); err == nil {
			t.Fatal("injected metadata accepted")
		}
	}
}
