package gateway

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"omniproxy/internal/application/activity"
	"omniproxy/internal/provider/chatgpt"
)

const compactGatewayBody = `{"object":"response.compaction","output":[{"type":"compaction","encrypted_content":"opaque-next"}],"usage":{"input_tokens":20,"output_tokens":2}}`

func TestSubscriptionCompactionEndToEndPreservesJSONAndUsage(t *testing.T) {
	s, _, observer, _ := subscriptionQuotaFixture(t)
	calls := 0
	s.chatgpt = chatgpt.New(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		raw, err := io.ReadAll(r.Body)
		if err != nil || !strings.Contains(string(raw), `"encrypted_content":"opaque-before"`) || strings.Contains(string(raw), `"stream":true`) || strings.Contains(string(raw), `"store":`) || r.URL.Path != "/backend-api/codex/responses/compact" {
			t.Error("native compaction was converted to ordinary inference")
		}
		return upstreamReply(200, compactGatewayBody, false), nil
	}))
	resp, err := http.Post(s.Info().Runtime.Address+"/v1/responses/compact", "application/json", strings.NewReader(`{"model":"gpt-test","input":[{"type":"compaction","encrypted_content":"opaque-before"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil || resp.StatusCode != 200 || string(raw) != compactGatewayBody || calls != 1 {
		t.Fatal("native compaction failed or opaque payload changed")
	}
	page, err := observer.Requests(context.Background(), activity.RequestQuery{Limit: 10})
	if err != nil || len(page.Items) != 1 {
		t.Fatal("compaction accounting missing")
	}
	item, err := observer.Request(context.Background(), page.Items[0].ID)
	if err != nil || item.State != "success" || item.InputTokens != "20" || item.OutputTokens != "2" || item.AttemptCount != 1 {
		t.Fatal("compaction usage or attempts incorrect")
	}
}

func TestCompactionCannotFabricateThirdPartySupport(t *testing.T) {
	s, _, _, _ := failoverFixture(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Error("unsupported compaction reached third-party adapter")
		return nil, nil
	}))
	resp, err := http.Post(s.Info().Runtime.Address+"/v1/responses/compact", "application/json", strings.NewReader(`{"model":"deepseek-flash","input":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 501 {
		t.Fatal("unsupported adapter pretended to compact", resp.StatusCode)
	}
}

func TestCompactionInvalidSuccessDoesNotReplayOrExposeBody(t *testing.T) {
	s, _, _, _ := subscriptionQuotaFixture(t)
	calls := 0
	s.chatgpt = chatgpt.New(roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return upstreamReply(200, `{"object":"response","output":[{"type":"message","content":"private-qa-marker"}]}`, false), nil
	}))
	resp, err := http.Post(s.Info().Runtime.Address+"/v1/responses/compact", "application/json", strings.NewReader(`{"model":"gpt-test","input":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 502 || calls != 1 || strings.Contains(string(raw), "private-qa-marker") {
		t.Fatal("invalid compaction success was replayed or leaked")
	}
}

func TestSubscriptionCompactionV2KeepsProtocolAndOpaqueSSE(t *testing.T) {
	s, _, _, _ := subscriptionQuotaFixture(t)
	metadata := `{"request_kind":"compaction","compaction":{"implementation":"responses_compaction_v2"}}`
	output := "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_qa\",\"output\":[{\"type\":\"compaction\",\"encrypted_content\":\"opaque-next\"}]}}\n\n"
	s.chatgpt = chatgpt.New(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/backend-api/codex/responses" || r.Header.Get("X-Codex-Beta-Features") != "remote_compaction_v2" || r.Header.Get("X-Codex-Turn-Metadata") != metadata {
			t.Error("V2 compact markers disappeared across gateway routing")
		}
		return upstreamReply(200, output, true), nil
	}))
	req, _ := http.NewRequest("POST", s.Info().Runtime.Address+"/v1/responses", strings.NewReader(`{"model":"gpt-test","input":[],"stream":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Codex-Beta-Features", "remote_compaction_v2")
	req.Header.Set("X-Codex-Turn-Metadata", metadata)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || string(raw) != output {
		t.Fatal("V2 compaction item was rewritten")
	}
}
