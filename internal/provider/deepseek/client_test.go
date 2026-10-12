package deepseek

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

func TestForwardContract(t *testing.T) {
	key := []byte("placeholder-test-key")
	body := []byte(`{"model":"deepseek-flash","messages":[{"role":"user","content":"hello"}],"stream":true}`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Method != http.MethodPost {
			t.Errorf("unexpected endpoint: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer placeholder-test-key" || r.Header.Get("Accept") != "text/event-stream" {
			t.Error("missing protected authorization or SSE accept")
		}
		data, err := io.ReadAll(r.Body)
		if err != nil || string(data) != string(body) {
			t.Error("body not preserved")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer server.Close()
	client := New(nil)
	err := client.Forward(context.Background(), server.URL+"/v1", key, proxy.Request{Body: body, Stream: true}, func(response proxy.Response) error {
		if response.StatusCode != 200 || response.ContentType != "text/event-stream" {
			t.Errorf("unexpected response metadata: %+v", response)
		}
		_, err := io.ReadAll(response.Body)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRedirectIsNotFollowed(t *testing.T) {
	called := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(200) }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	err := New(nil).Forward(context.Background(), source.URL, []byte("placeholder-key"), proxy.Request{Body: []byte("{}")}, func(response proxy.Response) error {
		if response.StatusCode != 307 {
			t.Errorf("redirect changed status: %d", response.StatusCode)
		}
		return nil
	})
	if err != nil || called {
		t.Fatalf("redirect followed: %v %v", called, err)
	}
}

func TestEndpointAndCredentialValidation(t *testing.T) {
	for _, url := range []string{"http://remote.example", "https://user:password@example.com", "https://example.com?key=secret", "https://example.com/#secret", strings.Repeat("a", 2049)} {
		if _, err := completionURL(url); err == nil {
			t.Errorf("accepted unsafe URL: %.80s", url)
		}
	}
	for _, key := range [][]byte{nil, []byte("bad\r\nkey"), []byte("bad key"), []byte(strings.Repeat("x", 8193))} {
		err := New(nil).Forward(context.Background(), "https://api.deepseek.com", key, proxy.Request{}, func(proxy.Response) error { return nil })
		if err == nil {
			t.Error("accepted unsafe credential")
		}
	}
}
