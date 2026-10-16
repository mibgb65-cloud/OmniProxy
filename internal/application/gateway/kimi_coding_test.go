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
	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/kimi"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKimiCodingHTTPBillingRegionIdentityStreamsAndNoRetry(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "kimi-code.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	store := &testStore{}
	accounts, providers := account.NewService(db, store), provider.NewService(db)
	ids := map[string]string{}
	for _, mode := range []string{"payg", "coding"} {
		for _, region := range []string{"cn", "global"} {
			a, err := accounts.Create(ctx, account.CreateInput{ProviderID: "builtin_kimi", Name: mode + region, APIKey: "placeholder-kimi-key", Kimi: &kimi.Options{Region: region, BillingMode: mode}})
			if err != nil {
				t.Fatal(err)
			}
			ids[mode+region] = a.ID
		}
	}
	observer := activity.New(db)
	router := routes.New(db, providers, accounts)
	s := New(proxy.Options{Observer: observer}, providers, accounts, nil)
	s.SetRouter(router)
	mode, region, protocol, stream, status, calls := "coding", "cn", "chat", false, 200, 0
	s.kimi = kimi.New(kimiRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		o := kimi.Options{Region: region, BillingMode: mode}
		base, _ := o.BaseURL()
		path := base + "/" + protocol
		if protocol == "chat" {
			path = base + "/chat/completions"
		}
		if protocol == "messages" && mode == "payg" {
			path = strings.TrimSuffix(base, "/v1") + "/anthropic/v1/messages"
		}
		if r.URL.String() != path || r.Header.Get("Authorization") != "Bearer placeholder-gateway-test-key" {
			t.Fatal("wrong actual billing/region credential target")
		}
		if mode == "coding" && r.Header.Get("User-Agent") != "real-integration-client/1.0" {
			t.Fatal("real client identity overwritten")
		}
		var fields map[string]json.RawMessage
		if json.NewDecoder(r.Body).Decode(&fields) != nil {
			t.Fatal("invalid upstream body")
		}
		want := "kimi-k3"
		if mode == "coding" {
			want = "k3"
		}
		if string(fields["model"]) != `"`+want+`"` {
			t.Fatal("wrong upstream model")
		}
		body, kind := atriaJSON(protocol), "application/json"
		if stream {
			first, last := atriaStream(protocol)
			body, kind = first+last, "text/event-stream"
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{kind}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	}))
	if _, err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop(ctx)
	for _, m := range []string{"coding", "payg"} {
		for _, r := range []string{"cn", "global"} {
			mode, region = m, r
			if _, err := router.SetKimiPreferredBillingMode(ctx, m); err != nil {
				t.Fatal(err)
			}
			if _, err := router.SetKimiPreferredRegion(ctx, r); err != nil {
				t.Fatal(err)
			}
			for _, p := range []string{"chat", "responses", "messages"} {
				protocol = p
				for _, useStream := range []bool{false, true} {
					stream = useStream
					path, body := atriaInput(p, stream)
					body = strings.ReplaceAll(body, `"research"`, `"kimi-k3"`)
					request, _ := http.NewRequest(http.MethodPost, s.Info().Runtime.Address+"/v1/"+path, strings.NewReader(body))
					request.Header.Set("Content-Type", "application/json")
					request.Header.Set("User-Agent", "real-integration-client/1.0")
					response, err := http.DefaultClient.Do(request)
					if err != nil {
						t.Fatal(err)
					}
					_, readErr := io.Copy(io.Discard, response.Body)
					closeErr := response.Body.Close()
					if response.StatusCode != 200 || readErr != nil || closeErr != nil {
						t.Fatal("native request failed", response.StatusCode, readErr, closeErr)
					}
					item, err := observer.Request(ctx, response.Header.Get("X-Request-ID"))
					if err != nil || item.BillingMode != mode || item.ProviderRegion != region || item.AccountID != ids[mode+region] {
						t.Fatal("actual billing attribution lost", err)
					}
				}
			}
		}
	}
	for _, old := range []string{"kimi-cn/kimi-k3[1m]", catalog.DesktopAlias("kimi-cn/kimi-k3[1m]")} {
		r, err := s.RouteRequest(ctx, proxy.Request{Model: old, Protocol: "messages", Body: []byte(`{"model":"` + old + `","messages":[{"role":"user","content":"hi"}]}`)})
		if err != nil || r.AccountID != ids["paygcn"] {
			t.Fatal("legacy fixed API region lost", err)
		}
	}
	models, err := s.AvailableModels(ctx)
	if err != nil || len(models) != 6 {
		t.Fatal("deduplicated models missing", models, err)
	}
	for _, id := range models {
		if strings.Contains(id, "/") {
			t.Fatal("region/billing alias still published")
		}
	}
	mode, region, protocol, stream, status = "coding", "cn", "chat", false, 503
	if _, err := router.SetKimiPreferredBillingMode(ctx, mode); err != nil {
		t.Fatal(err)
	}
	if _, err := router.SetKimiPreferredRegion(ctx, region); err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodPost, s.Info().Runtime.Address+"/v1/chat/completions", strings.NewReader(`{"model":"kimi-k3","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "real-integration-client/1.0")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode < 400 || calls != 25 {
		t.Fatal("failed subscription request changed pool/retried", response.StatusCode, calls)
	}
}
