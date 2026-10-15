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
	"omniproxy/internal/provider/mimo"
)

type mimoPlanStore struct{ last []byte }

func (*mimoPlanStore) Put([]byte) (string, error) { return "protected-plan-ref", nil }
func (s *mimoPlanStore) Get(string) ([]byte, error) {
	s.last = []byte("ttp-placeholder-plan-key")
	return s.last, nil
}
func (*mimoPlanStore) Delete(string) error { return nil }

func TestMiMoPlanGatewayNativeProtocolsScopeAndStaleSelection(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "mimo-plan.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	store := &mimoPlanStore{}
	accounts := account.NewService(db, store)
	providers := provider.NewService(db)
	observer := activity.New(db)
	s := New(proxy.Options{Observer: observer}, providers, accounts, nil)
	router := routes.New(db, providers, accounts)
	s.SetRouter(router)
	ids := map[string]account.Item{}
	for _, region := range []string{"cn", "sgp", "ams"} {
		a, err := accounts.Create(ctx, account.CreateInput{ProviderID: "builtin_mimo", Name: region, APIKey: "ttp-placeholder-plan-key", MiMo: &mimo.Options{Region: region, BillingMode: "token_plan"}})
		if err != nil {
			t.Fatal(err)
		}
		ids[region] = a
	}
	protocol, expected, stream, calls := "", "", false, 0
	s.mimo = mimo.New(kimiRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != expected || r.Header.Get("Api-Key") != "ttp-placeholder-plan-key" {
			t.Fatal("wrong plan cluster or credential")
		}
		raw, err := io.ReadAll(r.Body)
		var fields map[string]json.RawMessage
		if err != nil || json.Unmarshal(raw, &fields) != nil || string(fields["model"]) != `"mimo-v2.6-pro"` {
			t.Fatal("channel alias sent upstream")
		}
		body, kind := atriaJSON(protocol), "application/json"
		if stream {
			first, last := atriaStream(protocol)
			body, kind = first+last, "text/event-stream"
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{kind}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	}))
	if _, err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop(ctx)
	for _, region := range []string{"cn", "sgp", "ams"} {
		for current, path := range map[string]string{"chat": "/v1/chat/completions", "responses": "/v1/responses", "messages": "/anthropic/v1/messages"} {
			protocol = current
			expected = "https://token-plan-" + region + ".xiaomimimo.com" + path
			for _, wantStream := range []bool{false, true} {
				stream = wantStream
				requestPath, body := atriaInput(protocol, stream)
				body = strings.ReplaceAll(body, `"research"`, `"mimo-`+region+`-plan/mimo-v2.6-pro"`)
				response, err := http.Post(s.Info().Runtime.Address+"/v1/"+requestPath, "application/json", strings.NewReader(body))
				if err != nil {
					t.Fatal(err)
				}
				_, readErr := io.Copy(io.Discard, response.Body)
				closeErr := response.Body.Close()
				if response.StatusCode != 200 || readErr != nil || closeErr != nil {
					t.Fatal("native plan request failed", response.StatusCode, readErr, closeErr)
				}
				item, err := observer.Request(ctx, response.Header.Get("X-Request-ID"))
				if err != nil || item.InputTokens != "12" || item.OutputTokens != "4" || !item.UsageComplete || item.ProviderRegion != region || item.BillingMode != "token_plan" {
					t.Fatalf("plan usage attribution lost: %+v %v", item, err)
				}
				for _, b := range store.last {
					if b != 0 {
						t.Fatal("protected key not cleared")
					}
				}
			}
		}
	}
	if calls != 18 {
		t.Fatalf("unexpected requests or retries: %d", calls)
	}
	stale, err := s.RouteRequest(ctx, proxy.Request{Model: "mimo-cn-plan/mimo-v2.6-pro", Protocol: "chat", Body: []byte(`{"model":"mimo-cn-plan/mimo-v2.6-pro","messages":[{"role":"user","content":"hello"}]}`)})
	if err != nil {
		t.Fatal(err)
	}
	a := ids["cn"]
	o := *a.MiMo
	o.Region = "sgp"
	if _, err := accounts.Update(ctx, account.UpdateInput{ID: a.ID, Name: a.Name, MiMo: &o}); err != nil {
		t.Fatal(err)
	}
	if err := s.Forward(ctx, stale, func(proxy.Response) error { t.Fatal("stale plan reached upstream"); return nil }); err == nil || calls != 18 {
		t.Fatal("stale selection used a new cluster")
	}
}
