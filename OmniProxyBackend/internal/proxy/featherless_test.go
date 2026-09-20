package proxy

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"omniproxy/internal/config"
	"omniproxy/internal/logs"
	"omniproxy/internal/storage"
	"omniproxy/internal/token"
)

func TestRouterRoutesFeatherlessDirect(t *testing.T) {
	router := NewRouter(config.Config{FeatherlessBaseURL: "https://api.featherless.ai/v1"})
	route := router.Route(mustRouterTestURL(t, "/featherless/v1/chat/completions"), []byte(`{"model":"Qwen/Qwen3-32B"}`))
	if route.Provider != token.ProviderFeatherless || route.Protocol != "openai" || route.Path != "/v1/chat/completions" {
		t.Fatalf("unexpected Featherless route: %#v", route)
	}
	target, err := router.TargetURL(route, token.Token{Provider: token.ProviderFeatherless, CredentialType: token.CredentialTypeAPIKey})
	if err != nil {
		t.Fatal(err)
	}
	if target != "https://api.featherless.ai/v1/chat/completions" {
		t.Fatalf("unexpected target url: %s", target)
	}
}

func TestRouterKeepsExplicitFeatherlessRouteForRepositoryStyleModel(t *testing.T) {
	router := NewRouter(config.Config{
		FeatherlessBaseURL: "https://api.featherless.ai/v1",
		GatewayRoutes: config.GatewayRoutes{
			OpenAI: config.GatewayRouteConfig{Provider: token.ProviderFeatherless},
		},
	})
	route := router.Route(mustRouterTestURL(t, "/opencode-router/v1/chat/completions"), []byte(`{"model":"Qwen/Qwen3-32B"}`))
	if route.Provider != token.ProviderFeatherless {
		t.Fatalf("expected explicit Featherless route, got %s", route.Provider)
	}
	target, err := router.TargetURL(route, token.Token{Provider: token.ProviderFeatherless, CredentialType: token.CredentialTypeAPIKey})
	if err != nil {
		t.Fatal(err)
	}
	if target != "https://api.featherless.ai/v1/chat/completions" {
		t.Fatalf("unexpected target url: %s", target)
	}
}

func TestServiceRoutesFeatherlessRequests(t *testing.T) {
	var upstreamPath string
	var authorizations []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamPath = r.URL.Path
		authorizations = append(authorizations, r.Header.Get("Authorization"))
		if len(authorizations) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"usage":{"total_tokens":5}}`))
	}))
	defer upstream.Close()

	manager := newFeatherlessTestManager(t, "fl-primary-test-token", "fl-backup-test-token")
	service, err := NewService(config.Config{
		ProxyPort:          3000,
		ControlPort:        3890,
		FeatherlessBaseURL: upstream.URL + "/v1",
		SwitchThreshold:    15,
		MaxRetries:         1,
	}, manager, logs.NewRecorder(10))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/featherless/v1/chat/completions", stringsReader(`{"model":"Qwen/Qwen3-32B","messages":[]}`))
	req.Header.Set("Authorization", "Bearer caller")
	res := httptest.NewRecorder()
	service.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", res.Code, res.Body.String())
	}
	if upstreamPath != "/v1/chat/completions" {
		t.Fatalf("unexpected featherless route path=%q", upstreamPath)
	}
	want := map[string]bool{
		"Bearer fl-primary-test-token": true,
		"Bearer fl-backup-test-token":  true,
	}
	if len(authorizations) != 2 || authorizations[0] == authorizations[1] || !want[authorizations[0]] || !want[authorizations[1]] {
		t.Fatalf("expected Featherless 429 retry to rotate keys, got %#v", authorizations)
	}
}

func TestServiceBalancesSequentialFeatherlessRequestsAcrossKeys(t *testing.T) {
	var authorizations []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorizations = append(authorizations, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"usage":{"total_tokens":1}}`))
	}))
	defer upstream.Close()

	manager := newFeatherlessTestManager(t, "fl-first-balance-token", "fl-second-balance-token")
	service, err := NewService(config.Config{
		ProxyPort:          3000,
		ControlPort:        3890,
		SchedulingMode:     config.SchedulingModeBalanced,
		FeatherlessBaseURL: upstream.URL + "/v1",
		SwitchThreshold:    15,
		MaxRetries:         0,
	}, manager, logs.NewRecorder(10))
	if err != nil {
		t.Fatal(err)
	}

	for range 2 {
		req := httptest.NewRequest(http.MethodPost, "/featherless/v1/chat/completions", stringsReader(`{"model":"Qwen/Qwen3-32B","messages":[]}`))
		res := httptest.NewRecorder()
		service.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d body=%s", res.Code, res.Body.String())
		}
	}
	if len(authorizations) != 2 || authorizations[0] == authorizations[1] {
		t.Fatalf("expected balanced Featherless requests to rotate keys, got %#v", authorizations)
	}
}

func newFeatherlessTestManager(t *testing.T, values ...string) *token.Manager {
	t.Helper()
	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	for index, value := range values {
		_, err := manager.Add(token.UpsertRequest{
			Name:       "featherless-test-" + string(rune('a'+index)),
			Provider:   token.ProviderFeatherless,
			TokenValue: value,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return manager
}
