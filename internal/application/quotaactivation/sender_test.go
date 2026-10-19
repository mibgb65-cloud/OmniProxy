package quotaactivation

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"omniproxy/internal/application/subscription"
	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/codex"
	"omniproxy/internal/provider/chatgpt"
)

type testAccess struct {
	quota      subscription.Quota
	readHook   func() subscription.Quota
	expectedID string
	officialID string
	token      []byte
	tokenCalls int
}

func (a *testAccess) ReadQuota(context.Context, string) (subscription.Quota, error) {
	if a.readHook != nil {
		return a.readHook(), nil
	}
	return a.quota, nil
}
func (a *testAccess) Models(_ context.Context, id string) ([]codex.Model, error) {
	if id != a.expectedID {
		return nil, errors.New("wrong account")
	}
	return []codex.Model{{ID: "other"}, {ID: "default-model", Metadata: catalog.Metadata{IsDefault: true, ReasoningEfforts: []catalog.Effort{{Value: "low"}, {Value: "high"}}}}}, nil
}
func (a *testAccess) UseToken(_ context.Context, id string, callback func([]byte, string) error) error {
	if id != a.expectedID {
		return errors.New("wrong account")
	}
	a.tokenCalls++
	a.token = []byte("test-only-token")
	defer clear(a.token)
	return callback(a.token, a.officialID)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestMinimalMessageUsesOriginalAccountAndNoTools(t *testing.T) {
	f := newFixture(t)
	access := &testAccess{quota: unstartedQuota(), expectedID: f.accounts.item.ID, officialID: f.accounts.item.AuthAccountID}
	calls := 0
	client := chatgpt.New(roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.URL.String() != "https://chatgpt.com/backend-api/codex/responses" || request.Header.Get("ChatGPT-Account-ID") != "official-one" {
			t.Fatal("wrong account or endpoint")
		}
		var body struct {
			Model        string
			Instructions string
			Input        []struct{ Content []struct{ Text string } }
			Tools        []any
			ToolChoice   string `json:"tool_choice"`
			Store        bool
			Reasoning    struct{ Effort string }
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != "default-model" || body.Instructions != "Reply only OK." || len(body.Input) != 1 || body.Input[0].Content[0].Text != "Hi" || len(body.Tools) != 0 || body.ToolChoice != "none" || body.Store || body.Reasoning.Effort != "low" {
			t.Fatal("message was not minimal or used wrong model")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\"}}\n\n"))}, nil
	}))
	sender := NewMessageSender(f.accounts, f.providers, access, client)
	if err := sender.Send(context.Background(), f.accounts.item, SendPlan{Claimed: []string{"five_hour", "weekly"}}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || access.tokenCalls != 1 {
		t.Fatal("request count incorrect")
	}
	for _, value := range access.token {
		if value != 0 {
			t.Fatal("credential bytes retained")
		}
	}
	access.quota = activeQuota()
	if err := sender.Send(context.Background(), f.accounts.item, SendPlan{Claimed: []string{"five_hour", "weekly"}}); !errors.Is(err, ErrAlreadyActive) {
		t.Fatal("active account not skipped")
	}
	if calls != 1 || access.tokenCalls != 1 {
		t.Fatal("active account used credentials or sent again")
	}
	access.quota = unstartedQuota()
	access.officialID = "other"
	if err := sender.Send(context.Background(), f.accounts.item, SendPlan{Claimed: []string{"five_hour", "weekly"}}); err == nil || calls != 1 {
		t.Fatal("mismatched official identity used")
	}
}

func TestActivationResponseMustBeCompleteAndBounded(t *testing.T) {
	for _, body := range []string{
		"data: {\"type\":\"response.failed\"}\n\n", "data: {\"type\":\"error\"}\n\n",
		"data: {\"type\":\"response.completed\",\"response\":{\"status\":\"incomplete\"}}\n\n",
		"data: broken\n\n", "data: [DONE]\n\n", strings.Repeat("a", 65<<10),
	} {
		if err := consumeActivation(proxy.Response{StatusCode: 200, Body: strings.NewReader(body)}); err == nil {
			t.Fatal("invalid or unbounded response accepted")
		}
	}
	if err := consumeActivation(proxy.Response{StatusCode: 403, Body: strings.NewReader("private error")}); err == nil || strings.Contains(err.Error(), "private") {
		t.Fatal("unsafe rejected response")
	}
}
