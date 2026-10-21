package chatgpt

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"omniproxy/internal/core/proxy"
	"strings"
	"testing"
)

func TestResponsesPreservesSelectedServiceTierWithoutEnablingFastByDefault(t *testing.T) {
	for _, tier := range []string{"priority", "default", ""} {
		t.Run("tier="+tier, func(t *testing.T) {
			body := `{"model":"gpt-test","input":[],"stream":true}`
			if tier != "" {
				body = strings.TrimSuffix(body, "}") + `,"service_tier":"` + tier + `"}`
			}
			request, err := proxy.ParseProtocolRequest([]byte(body), "/v1/responses")
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			client := New(transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				var forwarded map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&forwarded); err != nil {
					t.Fatal(err)
				}
				var actual string
				raw, present := forwarded["service_tier"]
				if present && json.Unmarshal(raw, &actual) != nil || actual != tier || present != (tier != "") {
					t.Error("Fast selection was changed or forced")
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader("data: [DONE]\n\n"))}, nil
			}))
			err = client.Forward(context.Background(), []byte("qa-only"), "qa-account", request, func(response proxy.Response) error {
				_, err := io.Copy(io.Discard, response.Body)
				return err
			})
			if err != nil || calls != 1 {
				t.Fatal("request did not reach isolated transport", err)
			}
		})
	}
}
