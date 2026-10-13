package atria

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

func testRequest(protocol string, stream bool) proxy.Request {
	input := `"messages":[{"role":"user","content":"hello"}]`
	if protocol == "messages" {
		input += `,"max_tokens":16`
	}
	if protocol == "responses" {
		input = `"input":[{"role":"user","content":[{"type":"input_text","text":"hello"}]}]`
	}
	return proxy.Request{Model: ModelID, Protocol: protocol, Stream: stream, Body: []byte(fmt.Sprintf(`{"model":%q,%s,"stream":%t}`, ModelID, input, stream))}
}

func TestNativeProtocolAndAuthenticationContracts(t *testing.T) {
	for _, protocol := range []string{"chat", "responses", "messages"} {
		for _, stream := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", protocol, stream), func(t *testing.T) {
				request := testRequest(protocol, stream)
				upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					path, _ := protocolPath(protocol)
					if r.URL.Path != "/v1"+path || r.Method != http.MethodPost {
						t.Errorf("unexpected endpoint: %s %s", r.Method, r.URL.Path)
					}
					if protocol == "messages" {
						if r.Header.Get("X-Api-Key") != "test-key" || r.Header.Get("Anthropic-Version") != "2023-06-01" || r.Header.Get("Authorization") != "" {
							t.Error("incorrect Messages authentication")
						}
					} else if r.Header.Get("Authorization") != "Bearer test-key" || r.Header.Get("X-Api-Key") != "" {
						t.Error("incorrect Bearer authentication")
					}
					body, _ := io.ReadAll(r.Body)
					if protocol != "chat" || !stream {
						if string(body) != string(request.Body) {
							t.Error("native request was transformed")
						}
					} else {
						var fields struct {
							Options struct {
								Usage bool `json:"include_usage"`
							} `json:"stream_options"`
						}
						if json.Unmarshal(body, &fields) != nil || !fields.Options.Usage {
							t.Error("stream usage not requested")
						}
					}
					w.Header().Set("Content-Type", "application/json")
					if stream {
						w.Header().Set("Content-Type", "text/event-stream")
					}
					w.Header().Set("Retry-After", "42")
					w.WriteHeader(http.StatusTooManyRequests)
					_, _ = io.WriteString(w, "native-response")
				}))
				defer upstream.Close()
				err := New(nil).Forward(context.Background(), upstream.URL+"/v1/", []byte("test-key"), request, func(response proxy.Response) error {
					body, err := io.ReadAll(response.Body)
					if response.StatusCode != http.StatusTooManyRequests || response.RetryAfter != "42" || string(body) != "native-response" {
						t.Error("native response metadata lost")
					}
					return err
				})
				if err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestEndpointAndCredentialSafety(t *testing.T) {
	for _, base := range []string{"http://remote.example/v1", "https://user:pass@example.test/v1", "https://example.test/v1?key=private", "https://example.test:0/v1", "https://example.test/v1#fragment"} {
		if _, err := requestURL(base, "/responses"); err == nil {
			t.Fatalf("unsafe URL accepted: %s", base)
		}
	}
	endpoint, err := requestURL("", "/responses")
	if err != nil || endpoint != DefaultBaseURL+"/responses" {
		t.Fatal("incorrect default URL", endpoint, err)
	}
	for _, key := range []string{"", "test\nkey", strings.Repeat("k", 8193)} {
		err := New(nil).Forward(context.Background(), "", []byte(key), testRequest("responses", false), func(proxy.Response) error { t.Fatal("invalid credential used"); return nil })
		if err == nil {
			t.Fatal("invalid credential accepted")
		}
	}
}

func TestRedirectIsNotFollowedAndCancellationStopsRequest(t *testing.T) {
	redirectTarget := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("credential followed redirect") }))
	defer redirectTarget.Close()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectTarget.URL, http.StatusTemporaryRedirect)
	}))
	defer upstream.Close()
	err := New(nil).Forward(context.Background(), upstream.URL, []byte("test-key"), testRequest("responses", false), func(response proxy.Response) error {
		if response.StatusCode != http.StatusTemporaryRedirect {
			t.Error("redirect status lost")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := New(nil).Forward(ctx, upstream.URL, []byte("test-key"), testRequest("responses", false), func(proxy.Response) error { t.Error("cancelled request consumed"); return nil }); err == nil {
		t.Fatal("cancel ignored")
	}
}
