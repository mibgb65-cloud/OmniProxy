package chatgpt

import (
	"fmt"
	"io"
	"omniproxy/internal/core/proxy"
	"strings"
	"testing"
	"time"
)

func TestQuotaDenialRequiresCompleteFirstPreExecutionEvidence(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	raw := fmt.Sprintf(`{"error":{"type":"usage_limit_reached","message":"private marker","resets_at":%d}}`, now.Add(7*24*time.Hour).Unix())
	event := "data: " + raw + "\n\n"
	created := "data: {\"type\":\"response.created\",\"response\":{\"id\":\"one\"}}\n\n"
	for _, tt := range []struct {
		name, body string
		want       bool
	}{
		{"json", raw, true}, {"sse", event, true}, {"crlf", strings.ReplaceAll(event, "\n", "\r\n"), true},
		{"flat-sse", "event: error\ndata: {\"type\":\"error\",\"code\":\"usage_limit_reached\"}\n\n", true},
		{"after-created", created + event, false}, {"truncated", strings.TrimSuffix(event, "\n"), false},
		{"leading-blank-truncated", "\n\n" + strings.TrimSuffix(event, "\n"), false},
		{"wrong-event", "event: response.created\n" + event, false},
		{"message-only", `{"error":{"message":"usage_limit_reached"}}`, false},
		{"generic-error", `{"error":{"type":"invalid_request_error"}}`, false},
		{"response-present", `{"type":"error","code":"usage_limit_reached","response":{"id":"one"}}`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			denied, reset := quotaRejection([]byte(tt.body), now)
			if denied != tt.want {
				t.Fatal("unsafe quota evidence classification", denied)
			}
			if denied && strings.Contains(tt.body, "resets_at") && !reset.Equal(now.Add(7*24*time.Hour)) {
				t.Fatal("weekly retry boundary lost", reset)
			}
		})
	}
	for _, kind := range []string{"", "text/event-stream", "application/json"} {
		response := inferResponseType(proxy.Response{StatusCode: 200, ContentType: kind, Body: strings.NewReader(raw)})
		if !response.PreExecutionQuotaRejected || response.StatusCode != 200 || response.ContentTypeInferenceReason != "quota_rejected" {
			t.Fatal("quota evidence or wire status lost", kind)
		}
		body, err := io.ReadAll(response.Body)
		if err != nil || string(body) != raw {
			t.Fatal("quota diagnostics changed response")
		}
	}
	if denied, reset := quotaRejection([]byte(`{"error":{"type":"usage_limit_reached","resets_at":9999999999999}}`), now); !denied || !reset.IsZero() {
		t.Fatal("untrusted reset time accepted")
	}
}
