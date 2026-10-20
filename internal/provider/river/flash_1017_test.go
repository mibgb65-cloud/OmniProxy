package river

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"omniproxy/internal/core/proxy"
)

func TestFlash1017NativeRequestsPreserveVersion(t *testing.T) {
	for _, protocol := range []string{"chat", "responses"} {
		t.Run(protocol, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var fields map[string]json.RawMessage
				path, _ := protocolPath(protocol)
				if r.URL.Path != "/v1"+path || json.NewDecoder(r.Body).Decode(&fields) != nil || string(fields["model"]) != `"`+DeepSeekFlash1017+`"` {
					t.Error("versioned model changed in upstream request")
				}
				w.Header().Set("Content-Type", "application/json")
				if _, err := io.WriteString(w, `{"model":"`+DeepSeekFlash1017+`"}`); err != nil {
					t.Error(err)
				}
			}))
			defer upstream.Close()
			fields := map[string]any{"model": DeepSeekFlash1017}
			if protocol == "chat" {
				fields["messages"] = []any{map[string]any{"role": "user", "content": "hello"}}
				fields["max_tokens"] = 384000
			} else {
				fields["input"] = "hello"
				fields["max_output_tokens"] = 384000
			}
			body, err := json.Marshal(fields)
			if err != nil {
				t.Fatal(err)
			}
			err = New(nil).Forward(context.Background(), upstream.URL+"/v1", []byte("test-key"), proxy.Request{Model: DeepSeekFlash1017, Protocol: protocol, Body: body}, func(response proxy.Response) error {
				if response.StatusCode != http.StatusOK {
					t.Error("upstream request failed", response.StatusCode)
				}
				_, err := io.Copy(io.Discard, response.Body)
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
