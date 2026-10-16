package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"omniproxy/internal/application/account"
	"omniproxy/internal/application/activity"
	"omniproxy/internal/application/provider"
	routes "omniproxy/internal/application/routing"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/kimi"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type kimiRoundTrip func(*http.Request) (*http.Response, error)

func (f kimiRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestKimiRegionalGatewayNativeProtocolsAndCacheUsage(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "kimi.db")})
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
		a, err := accounts.Create(ctx, account.CreateInput{ProviderID: "builtin_kimi", Name: "Kimi " + region, APIKey: "placeholder-kimi-key", Kimi: &kimi.Options{Region: region}})
		if err != nil {
			t.Fatal(err)
		}
		ids[region] = a.ID
	}
	expected := ""
	calls := 0
	s.kimi = kimi.New(kimiRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != expected || r.Header.Get("Authorization") != "Bearer placeholder-gateway-test-key" {
			t.Fatal("wrong platform or protected key")
		}
		raw, _ := io.ReadAll(r.Body)
		var fields map[string]json.RawMessage
		if json.Unmarshal(raw, &fields) != nil || string(fields["model"]) != `"kimi-k3"` {
			t.Fatal("regional alias sent upstream")
		}
		body := `{"id":"chat","choices":[{"message":{"content":"OK"}}],"usage":{"prompt_tokens":10,"completion_tokens":3,"prompt_tokens_details":{"cached_tokens":2,"cache_write_tokens":3}}}`
		if strings.HasSuffix(r.URL.Path, "responses") {
			body = `{"id":"resp","object":"response","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":3,"input_tokens_details":{"cached_tokens":2,"cache_write_tokens":3}}}`
		}
		if strings.HasSuffix(r.URL.Path, "messages") {
			body = `{"id":"msg","type":"message","content":[{"type":"text","text":"OK"}],"usage":{"input_tokens":10,"cache_read_input_tokens":2,"cache_creation_input_tokens":3,"output_tokens":3}}`
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	}))
	info, err := s.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop(ctx)
	for _, region := range []string{"cn", "global"} {
		for _, protocol := range []string{"chat/completions", "responses", "messages"} {
			alias := "kimi-" + region + "/kimi-k3"
			body := `{"model":"` + alias + `","messages":[{"role":"user","content":"hi"}],"max_tokens":32}`
			if protocol == "responses" {
				body = `{"model":"` + alias + `","input":"hi","max_output_tokens":32}`
			}
			o := kimi.Options{Region: region}
			base, _ := o.BaseURL()
			expected = base + "/" + protocol
			if protocol == "messages" {
				expected = strings.TrimSuffix(base, "/v1") + "/anthropic/v1/messages"
			}
			response, err := http.Post(info.Runtime.Address+"/v1/"+protocol, "application/json", strings.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			raw, _ := io.ReadAll(response.Body)
			response.Body.Close()
			if response.StatusCode != 200 {
				t.Fatalf("native %s/%s: %d %s", region, protocol, response.StatusCode, raw)
			}
		}
	}
	if calls != 6 {
		t.Fatal("unexpected upstream retry")
	}
	decision, err := router.Resolve(ctx, "kimi-k3")
	if err != nil || !decision.Routable || decision.AccountID != ids["cn"] {
		t.Fatalf("default region priority not applied: %+v %v", decision, err)
	}
	page, err := observer.Requests(ctx, activity.RequestQuery{Limit: 100})
	if err != nil || len(page.Items) != 6 {
		t.Fatal("native requests not recorded")
	}
	for _, item := range page.Items {
		expectedInput := "10"
		if item.Protocol == "messages" {
			expectedInput = "15"
		}
		if item.ProviderRegion == "" || item.InputTokens != expectedInput || item.OutputTokens != "3" {
			t.Fatalf("wrong regional usage: %+v", item)
		}
	}
	entries, err := router.Catalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.TargetModel != "kimi-k3" {
			for _, protocol := range entry.Protocols {
				if protocol == "responses" {
					t.Fatal("K2 Responses invented")
				}
			}
		}
	}
	routed, err := s.RouteRequest(ctx, proxy.Request{Model: "kimi-cn/kimi-k3", Protocol: "chat", Body: []byte(`{"model":"kimi-cn/kimi-k3","messages":[{"role":"user","content":"hi"}]}`)})
	if err != nil {
		t.Fatal(err)
	}
	a, err := accounts.Metadata(ctx, ids["cn"])
	if err != nil {
		t.Fatal(err)
	}
	options := *a.Kimi
	options.Region = "global"
	if _, err := accounts.Update(ctx, account.UpdateInput{ID: a.ID, Name: a.Name, Kimi: &options}); err != nil {
		t.Fatal(err)
	}
	if err := s.Forward(ctx, routed, func(proxy.Response) error { return nil }); err == nil || calls != 6 {
		t.Fatal("stale request changed platform")
	}
}
