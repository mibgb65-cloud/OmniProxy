package openai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

func TestForwardUsesNativeResponsesAndBearerKey(t *testing.T) {
	var gotPath, gotAuth, gotAccept, gotBody string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth, gotAccept = r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("Accept")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_test"}`))
	}))
	defer upstream.Close()

	client := New(nil)
	request := proxy.Request{
		Protocol: "responses", Stream: true,
		Body: []byte(`{"model":"gpt-6.1-sol","input":"hi","stream":true}`),
	}
	status := 0
	err := client.Forward(context.Background(), upstream.URL+"/v1", []byte("sk-test"), request, func(response proxy.Response) error {
		status = response.StatusCode
		return nil
	})
	if err != nil || status != http.StatusOK {
		t.Fatalf("forward failed: status=%d err=%v", status, err)
	}
	if gotPath != "/v1/responses" || gotAuth != "Bearer sk-test" || gotAccept != "text/event-stream" {
		t.Fatalf("unexpected upstream request: path=%q auth=%q accept=%q", gotPath, gotAuth, gotAccept)
	}
	if gotBody != string(request.Body) {
		t.Fatalf("body rewritten: %q", gotBody)
	}
}

func TestForwardCompactionUsesCompactEndpoint(t *testing.T) {
	var gotPath, gotAccept string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAccept = r.URL.Path, r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"response.compaction","output":[]}`))
	}))
	defer upstream.Close()

	request := proxy.Request{Protocol: "responses", Compaction: true, Body: []byte(`{"model":"gpt-6.1-sol","input":[]}`)}
	if err := New(nil).Forward(context.Background(), upstream.URL+"/v1", []byte("sk-test"), request, func(proxy.Response) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/responses/compact" || gotAccept != "application/json" {
		t.Fatalf("unexpected compact request: path=%q accept=%q", gotPath, gotAccept)
	}
}

func TestDiscoverModelsFiltersNonChatFamilies(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" || r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Fatalf("unexpected discovery request: %s %q", r.URL.Path, r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"data":[
			{"id":"gpt-5.3-codex"},{"id":"gpt-6.1-sol"},{"id":"o3"},
			{"id":"chatgpt-4o-latest"},{"id":"codex-mini-latest"},
			{"id":"text-embedding-3-large"},{"id":"gpt-image-1"},
			{"id":"whisper-1"},{"id":"gpt-4o-mini-tts"},{"id":"gpt-4o-audio-preview"}
		]}`))
	}))
	defer upstream.Close()

	models, err := New(nil).DiscoverModels(context.Background(), upstream.URL+"/v1", []byte("sk-test"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"gpt-5.3-codex", "gpt-6.1-sol", "o3", "chatgpt-4o-latest", "codex-mini-latest"}
	if strings.Join(models, ",") != strings.Join(want, ",") {
		t.Fatalf("models = %v, want %v", models, want)
	}
}

func TestForwardRejectsNonResponsesProtocol(t *testing.T) {
	err := New(nil).Forward(context.Background(), "https://api.openai.com/v1", []byte("sk-test"), proxy.Request{Protocol: "chat"}, func(proxy.Response) error { return nil })
	var failure *proxy.Error
	if !errors.As(err, &failure) || failure.Code != "unsupported_protocol" {
		t.Fatalf("unexpected error: %v", err)
	}
}
