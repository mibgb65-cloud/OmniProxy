package proxy

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"omniproxy/internal/config"
	"omniproxy/internal/history"
	"omniproxy/internal/logs"
	"omniproxy/internal/storage"
	"omniproxy/internal/token"
	"path/filepath"
	"testing"

	"github.com/andybalholm/brotli"
)

// TestServiceRecordsUsageFromEncodedStream covers clients (Claude Code, Codex)
// that send Accept-Encoding: gzip, deflate, br. Providers such as Atria answer
// with a brotli encoded event stream, so the captured body has to be decoded
// before usage can be parsed.
func TestServiceRecordsUsageFromEncodedStream(t *testing.T) {
	const stream = "event: message_start\n" +
		"data: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":210,\"output_tokens\":0}}}\n\n" +
		"event: message_delta\n" +
		"data: {\"type\":\"message_delta\",\"usage\":{\"input_tokens\":210,\"output_tokens\":90}}\n\n" +
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

	for _, encoding := range []string{"gzip", "deflate", "br"} {
		t.Run(encoding, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var encoded bytes.Buffer
				switch encoding {
				case "gzip":
					writer := gzip.NewWriter(&encoded)
					_, _ = writer.Write([]byte(stream))
					_ = writer.Close()
				case "deflate":
					writer, _ := flate.NewWriter(&encoded, flate.DefaultCompression)
					_, _ = writer.Write([]byte(stream))
					_ = writer.Close()
				default:
					writer := brotli.NewWriter(&encoded)
					_, _ = writer.Write([]byte(stream))
					_ = writer.Close()
				}
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("Content-Encoding", encoding)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(encoded.Bytes())
			}))
			defer upstream.Close()

			manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := manager.Add(token.UpsertRequest{Name: "atria", Provider: token.ProviderAtria, TokenValue: "atr_test_token"}); err != nil {
				t.Fatal(err)
			}
			recorder, err := history.NewRecorder(storage.NewJSONStore[[]history.Entry](filepath.Join(t.TempDir(), "history.json")), 100)
			if err != nil {
				t.Fatal(err)
			}
			service, err := NewService(config.Config{
				ProxyPort:       3000,
				ControlPort:     3890,
				AtriaBaseURL:    upstream.URL,
				SwitchThreshold: 15,
				MaxRetries:      0,
			}, manager, logs.NewRecorder(10), recorder)
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest(http.MethodPost, "/anthropic-router/v1/messages", stringsReader(`{"model":"Atria-Dawn-Preview","stream":true,"messages":[]}`))
			req.Header.Set("Authorization", "Bearer caller")
			req.Header.Set("Accept-Encoding", "gzip, deflate, br")
			res := httptest.NewRecorder()
			service.ServeHTTP(res, req)

			if res.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d body=%s", res.Code, res.Body.String())
			}
			if res.Header().Get("Content-Encoding") != encoding {
				t.Fatalf("expected client to still receive %q encoded body", encoding)
			}
			entry := lastHistoryEntry(recorder)
			if entry.Provider != token.ProviderAtria || entry.InputTokens != 210 || entry.OutputTokens != 90 || entry.TotalTokens != 300 {
				t.Fatalf("expected usage from encoded stream, got %#v", entry)
			}
		})
	}
}
