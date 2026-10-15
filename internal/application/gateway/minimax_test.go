package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/activity"
	"omniproxy/internal/application/provider"
	routes "omniproxy/internal/application/routing"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/minimax"
)

type minimaxTransport func(*http.Request) (*http.Response, error)

func (f minimaxTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestMiniMaxFourPoolsRouteAndRecordUsage(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "step.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	store := &testStore{}
	accounts := account.NewService(db, store)
	providers := provider.NewService(db)
	observer := activity.New(db)
	s := New(proxy.Options{Observer: observer}, providers, accounts, nil)
	router := routes.New(db, providers, accounts)
	s.SetRouter(router)
	ids := map[string]string{}
	for _, region := range []string{"cn", "global"} {
		for _, mode := range []string{"payg", "subscription"} {
			a, err := accounts.Create(ctx, account.CreateInput{ProviderID: "builtin_minimax", Name: "MiniMax " + region + " " + mode, APIKey: "placeholder-step-key", MiniMax: &minimax.Options{Region: region, BillingMode: mode}})
			if err != nil {
				t.Fatal(err)
			}
			ids[region+"/"+mode] = a.ID
		}
	}
	expected := ""
	calls := 0
	s.minimax = minimax.New(minimaxTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != expected {
			t.Errorf("upstream=%s expected=%s", r.URL, expected)
		}
		raw, _ := io.ReadAll(r.Body)
		var fields map[string]json.RawMessage
		if json.Unmarshal(raw, &fields) != nil || string(fields["model"]) != `"MiniMax-M3"` {
			t.Error("channel alias leaked upstream")
		}
		if r.Header.Get("Authorization") != "Bearer placeholder-gateway-test-key" {
			t.Error("wrong protected credential")
		}
		response := `{"id":"ok","choices":[{"message":{"content":"OK"}}],"usage":{"prompt_tokens":10,"completion_tokens":3}}`
		if r.URL.Path == "/v1/responses" || r.URL.Path == "/subscription/v1/responses" {
			response = `{"id":"resp_ok","object":"response","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":3}}`
		}
		if strings.HasSuffix(r.URL.Path, "/messages") {
			response = `{"id":"msg_ok","type":"message","role":"assistant","content":[{"type":"text","text":"OK"}],"usage":{"input_tokens":10,"output_tokens":3}}`
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(response))}, nil
	}))
	info, err := s.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop(ctx)
	for _, region := range []string{"cn", "global"} {
		for _, mode := range []string{"payg", "subscription"} {
			for _, protocol := range []string{"chat/completions", "responses", "messages"} {
				channel := "api"
				if mode == "subscription" {
					channel = "plan"
				}
				alias := "minimax-" + region + "-" + channel + "/MiniMax-M3"
				body := `{"model":"` + alias + `","messages":[{"role":"user","content":"hi"}],"max_tokens":32}`
				if protocol == "responses" {
					body = `{"model":"` + alias + `","input":"hi","max_output_tokens":32}`
				}
				o := minimax.Options{Region: region, BillingMode: mode}
				base, _ := o.BaseURL()
				expected = base + "/" + protocol
				if protocol == "messages" {
					expected = strings.TrimSuffix(base, "/v1") + "/anthropic/v1/messages"
				}
				response, err := http.Post(info.Runtime.Address+"/v1/"+protocol, "application/json", strings.NewReader(body))
				if err != nil {
					t.Fatal(err)
				}
				_, err = io.ReadAll(response.Body)
				response.Body.Close()
				if err != nil || response.StatusCode != 200 {
					t.Fatalf("native %s/%s/%s: %d %v", region, mode, protocol, response.StatusCode, err)
				}
			}
		}
	}
	if calls != 12 {
		t.Fatalf("native requests=%d", calls)
	}
	decision, err := router.Resolve(ctx, "MiniMax-M3")
	if err != nil || decision.Routable || decision.ErrorCode != "ambiguous_minimax_pool" {
		t.Fatalf("mixed billing did not reject ambiguity: %+v %v", decision, err)
	}
	page, err := observer.Requests(ctx, activity.RequestQuery{Limit: 100})
	if err != nil || len(page.Items) != 12 {
		t.Fatalf("history=%+v %v", page, err)
	}
	for _, item := range page.Items {
		if item.ProviderRegion == "" || item.BillingMode == "" || item.InputTokens != "10" || item.OutputTokens != "3" {
			t.Fatalf("wrong scoped usage: %+v", item)
		}
	}
	// 单个账号改变通道，已经解析的请求不能偷偷跟随去另一计费池。
	routed, err := s.RouteRequest(ctx, proxy.Request{Model: "minimax-cn-api/MiniMax-M3", Protocol: "chat", Body: []byte(`{"model":"minimax-cn-api/MiniMax-M3","messages":[{"role":"user","content":"hi"}]}`)})
	if err != nil {
		t.Fatal(err)
	}
	a, err := accounts.Metadata(ctx, ids["cn/payg"])
	if err != nil {
		t.Fatal(err)
	}
	o := *a.MiniMax
	o.BillingMode = "subscription"
	if _, err := accounts.Update(ctx, account.UpdateInput{ID: a.ID, Name: a.Name, MiniMax: &o}); err != nil {
		t.Fatal(err)
	}
	if err := s.Forward(ctx, routed, func(proxy.Response) error { return nil }); err == nil || calls != 12 {
		t.Fatal("stale route changed billing channel")
	}
}
