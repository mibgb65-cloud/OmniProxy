package proxy

import (
	"net/url"
	"strings"
	"testing"

	"omniproxy/internal/config"
	"omniproxy/internal/token"
)

func TestRouterReadsRequestBodyModel(t *testing.T) {
	route := NewRouter(config.Config{}).Route(mustRouterTestURL(t, "/backend-api/codex/responses"), []byte(`{"model":"gpt-body","input":"hello"}`))

	if route.Model != "gpt-body" {
		t.Fatalf("expected body model, got %#v", route)
	}
}

func TestRouterReadsQueryModel(t *testing.T) {
	route := NewRouter(config.Config{}).Route(mustRouterTestURL(t, "/v1/responses?model=gpt-query"), []byte(`{}`))

	if route.Model != "gpt-query" {
		t.Fatalf("expected query model, got %#v", route)
	}
}

func TestRouterReadsModelFromGeminiPath(t *testing.T) {
	route := NewRouter(config.Config{}).Route(mustRouterTestURL(t, "/gemini/v1beta/models/gemini-3-pro-preview:generateContent"), []byte(`{}`))

	if route.Model != "gemini-3-pro-preview" {
		t.Fatalf("expected path model, got %#v", route)
	}
}

func TestRouterUsesRouteDefaultModelWhenRequestOmitsModel(t *testing.T) {
	cfg := config.Config{
		GatewayRoutes: config.GatewayRoutes{
			Claude: config.GatewayRouteConfig{Provider: token.ProviderAnthropic, Model: "claude-route-default"},
			OpenAI: config.GatewayRouteConfig{Provider: token.ProviderOpenAI, Model: "openai-route-default"},
		},
	}
	route := NewRouter(cfg).Route(mustRouterTestURL(t, "/anthropic-router/v1/messages"), []byte(`{"messages":[]}`))

	if route.Model != "claude-route-default" {
		t.Fatalf("expected Claude route default model, got %#v", route)
	}
}

func TestRouterInfersDirectGatewayProviderFromModel(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		body           string
		route          config.GatewayRouteConfig
		routeName      string
		wantProvider   string
		wantCredential string
		wantModel      string
	}{
		{
			name:         "claude deepseek model uses deepseek provider",
			path:         "/anthropic-router/v1/messages",
			body:         `{"model":"deepseek-v4-pro[1m]"}`,
			routeName:    config.GatewayRouteClaude,
			wantProvider: token.ProviderDeepSeek,
			wantModel:    "deepseek-v4-pro",
		},
		{
			name:         "claude direct route switches between supported model providers",
			path:         "/anthropic-router/v1/messages",
			body:         `{"model":"mimo-v2.5"}`,
			route:        config.GatewayRouteConfig{Provider: token.ProviderDeepSeek, CredentialType: token.CredentialTypeAPIKey},
			routeName:    config.GatewayRouteClaude,
			wantProvider: token.ProviderXiaomi,
			wantModel:    "mimo-v2.5",
		},
		{
			name:           "claude generic gateway keeps selected provider",
			path:           "/anthropic-router/v1/messages",
			body:           `{"model":"deepseek-v4-pro"}`,
			route:          config.GatewayRouteConfig{Provider: token.ProviderAnyRouter, CredentialType: token.CredentialTypeAPIKey},
			routeName:      config.GatewayRouteClaude,
			wantProvider:   token.ProviderAnyRouter,
			wantCredential: token.CredentialTypeAPIKey,
			wantModel:      "deepseek-v4-pro",
		},
		{
			name:         "openai router infers openrouter model",
			path:         "/opencode-router/v1/chat/completions",
			body:         `{"model":"openrouter/auto"}`,
			routeName:    config.GatewayRouteOpenAI,
			wantProvider: token.ProviderOpenRouter,
			wantModel:    "openrouter/auto",
		},
		{
			name:           "openai router keeps selected non-default provider for default-looking model",
			path:           "/opencode-router/v1/chat/completions",
			body:           `{"model":"gpt-5.4"}`,
			route:          config.GatewayRouteConfig{Provider: token.ProviderOpenRouter, CredentialType: token.CredentialTypeAPIKey},
			routeName:      config.GatewayRouteOpenAI,
			wantProvider:   token.ProviderOpenRouter,
			wantCredential: token.CredentialTypeAPIKey,
			wantModel:      "gpt-5.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{}
			switch tt.routeName {
			case config.GatewayRouteClaude:
				cfg.GatewayRoutes.Claude = tt.route
			case config.GatewayRouteOpenAI:
				cfg.GatewayRoutes.OpenAI = tt.route
			}
			route := NewRouter(cfg).Route(mustRouterTestURL(t, tt.path), []byte(tt.body))
			if route.Provider != tt.wantProvider || route.CredentialType != tt.wantCredential || route.Model != tt.wantModel {
				t.Fatalf("unexpected route: %#v", route)
			}
		})
	}
}

func TestPiRouterOpenAIModelsUseAutomaticCredentialMatching(t *testing.T) {
	route := NewRouter(config.Config{}).Route(mustRouterTestURL(t, "/pi-router/v1/chat/completions"), []byte(`{"model":"gpt-5.4"}`))

	if route.Provider != token.ProviderOpenAI || route.CredentialType != "" {
		t.Fatalf("expected Pi OpenAI model to use automatic credential matching, got %#v", route)
	}
}

func TestRouterCodexPrefixUsesAutomaticCredentialMatching(t *testing.T) {
	route := NewRouter(config.Config{}).Route(mustRouterTestURL(t, "/codex/v1/chat/completions"), []byte(`{"model":"gpt-5.4"}`))

	if route.Provider != token.ProviderOpenAI || route.CredentialType != "" || route.Path != "/v1/chat/completions" {
		t.Fatalf("expected codex prefix to use automatic credential matching, got %#v", route)
	}
}

func TestRouterIncludesGatewayFallbackRoutes(t *testing.T) {
	router := NewRouter(config.Config{
		GatewayRoutes: config.GatewayRoutes{
			OpenAI: config.GatewayRouteConfig{
				Provider:       token.ProviderOpenAI,
				CredentialType: token.CredentialTypeAPIKey,
				Model:          "gpt-route",
				Fallbacks: []config.GatewayRouteConfig{
					{Provider: token.ProviderDeepSeek},
					{Provider: token.ProviderPrem},
				},
			},
		},
	})

	route := router.Route(mustRouterTestURL(t, "/opencode-router/v1/chat/completions"), []byte(`{"messages":[]}`))
	if route.Provider != token.ProviderOpenAI || route.Model != "gpt-route" || route.Path != "/v1/chat/completions" {
		t.Fatalf("unexpected primary route: %#v", route)
	}
	if len(route.Fallbacks) != 2 {
		t.Fatalf("expected two fallback routes, got %#v", route.Fallbacks)
	}
	if route.Fallbacks[0].Provider != token.ProviderDeepSeek || route.Fallbacks[0].Protocol != "openai" || route.Fallbacks[0].Path != "/v1/chat/completions" {
		t.Fatalf("unexpected DeepSeek fallback route: %#v", route.Fallbacks[0])
	}
	if route.Fallbacks[1].Provider != token.ProviderPrem || route.Fallbacks[1].Protocol != "openai" || route.Fallbacks[1].Path != "/openai/v1/chat/completions" {
		t.Fatalf("unexpected Prem fallback route: %#v", route.Fallbacks[1])
	}
}

func TestRouterUsesModelRoutesBeforeClientGatewayRoutes(t *testing.T) {
	router := NewRouter(config.Config{
		GatewayRoutes: config.GatewayRoutes{
			Codex: config.GatewayRouteConfig{
				Provider:       token.ProviderOpenAI,
				CredentialType: token.CredentialTypeCodexAuthJSON,
				Model:          "gpt-5.4",
			},
		},
		ModelRoutes: config.ModelRoutes{
			"deepseek-v4-pro": {
				Provider: token.ProviderDeepSeek,
				Model:    "deepseek-v4-pro",
				Fallbacks: []config.GatewayRouteConfig{
					{Provider: token.ProviderPrem, Model: "deepseek-v4-pro"},
				},
			},
		},
	})

	route := router.Route(mustRouterTestURL(t, "/codex/v1/responses"), []byte(`{"model":"deepseek-v4-pro","input":"hi"}`))
	if route.Provider != token.ProviderDeepSeek || route.CredentialType != "" || route.Model != "deepseek-v4-pro" {
		t.Fatalf("expected Codex request model to use DeepSeek model route, got %#v", route)
	}
	if len(route.Fallbacks) != 1 || route.Fallbacks[0].Provider != token.ProviderPrem || route.Fallbacks[0].Path != "/openai/v1/responses" {
		t.Fatalf("unexpected model fallback route: %#v", route.Fallbacks)
	}
}

func TestRouterMapsNewProviderPrefixes(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		body     string
		provider string
		protocol string
		outPath  string
	}{
		{
			name:     "anthropic router zhipu",
			path:     "/anthropic-router/v1/messages",
			body:     `{"model":"glm-5.1"}`,
			provider: token.ProviderZhipu,
			protocol: "anthropic",
			outPath:  "/v1/messages",
		},
		{
			name:     "anthropic router minimax",
			path:     "/anthropic-router/v1/messages",
			body:     `{"model":"MiniMax-M2.7"}`,
			provider: token.ProviderMiniMax,
			protocol: "anthropic",
			outPath:  "/v1/messages",
		},
		{
			name:     "anthropic router zo claude",
			path:     "/anthropic-router/v1/messages",
			body:     `{"model":"claude-opus-4-7"}`,
			provider: token.ProviderZo,
			protocol: "anthropic",
			outPath:  "/v1/messages",
		},
		{
			name:     "claude desktop gateway",
			path:     "/claude-desktop/v1/messages",
			body:     `{"model":"deepseek-v4-pro[1m]"}`,
			provider: token.ProviderDeepSeek,
			protocol: "anthropic",
			outPath:  "/v1/messages",
		},
		{
			name:     "claude desktop gateway zo claude",
			path:     "/claude-desktop/v1/messages",
			body:     `{"model":"claude-sonnet-4-6"}`,
			provider: token.ProviderZo,
			protocol: "anthropic",
			outPath:  "/v1/messages",
		},
		{
			name:     "opencode router defaults openai",
			path:     "/opencode-router/v1/chat/completions",
			body:     `{"model":"gpt-5.4"}`,
			provider: token.ProviderOpenAI,
			protocol: "openai",
			outPath:  "/v1/chat/completions",
		},
		{
			name:     "opencode router openrouter",
			path:     "/opencode-router/v1/chat/completions",
			body:     `{"model":"openai/gpt-test"}`,
			provider: token.ProviderOpenRouter,
			protocol: "openai",
			outPath:  "/v1/chat/completions",
		},
		{
			name:     "opencode router tokenrouter",
			path:     "/opencode-router/v1/chat/completions",
			body:     `{"model":"auto:balance"}`,
			provider: token.ProviderTokenRouter,
			protocol: "openai",
			outPath:  "/v1/chat/completions",
		},
		{
			name:     "opencode router custom",
			path:     "/opencode-router/v1/chat/completions",
			body:     `{"model":"custom-model"}`,
			provider: token.ProviderCustom,
			protocol: "openai",
			outPath:  "/v1/chat/completions",
		},
		{
			name:     "pi router kimi",
			path:     "/pi-router/v1/chat/completions",
			body:     `{"model":"kimi-for-coding"}`,
			provider: token.ProviderKimi,
			protocol: "openai",
			outPath:  "/v1/chat/completions",
		},
		{
			name:     "gemini direct",
			path:     "/gemini/v1beta/models/gemini-3-pro-preview:generateContent",
			body:     `{}`,
			provider: token.ProviderGemini,
			protocol: "gemini",
			outPath:  "/v1beta/models/gemini-3-pro-preview:generateContent",
		},
		{
			name:     "openrouter direct",
			path:     "/openrouter/v1/chat/completions",
			body:     `{"model":"anthropic/claude-test"}`,
			provider: token.ProviderOpenRouter,
			protocol: "openai",
			outPath:  "/v1/chat/completions",
		},
		{
			name:     "tokenrouter direct",
			path:     "/tokenrouter/v1/chat/completions",
			body:     `{"model":"openai/gpt-test"}`,
			provider: token.ProviderTokenRouter,
			protocol: "openai",
			outPath:  "/v1/chat/completions",
		},
		{
			name:     "sub2api direct",
			path:     "/sub2api/v1/responses",
			body:     `{"model":"gpt-5.4"}`,
			provider: token.ProviderSub2API,
			protocol: "openai",
			outPath:  "/v1/responses",
		},
		{
			name:     "sub2api codex responses without version",
			path:     "/sub2api/responses",
			body:     `{"model":"gpt-5.5"}`,
			provider: token.ProviderSub2API,
			protocol: "openai",
			outPath:  "/v1/responses",
		},
		{
			name:     "sub2api anthropic direct",
			path:     "/sub2api/anthropic/v1/messages",
			body:     `{"model":"claude-sonnet-4-5"}`,
			provider: token.ProviderSub2API,
			protocol: "anthropic",
			outPath:  "/v1/messages",
		},
		{
			name:     "sub2api gemini direct",
			path:     "/sub2api/gemini/v1beta/models/gemini-3-pro-preview:generateContent",
			body:     `{}`,
			provider: token.ProviderSub2API,
			protocol: "gemini",
			outPath:  "/v1beta/models/gemini-3-pro-preview:generateContent",
		},
		{
			name:     "newapi direct",
			path:     "/newapi/v1/responses",
			body:     `{"model":"gpt-5.5"}`,
			provider: token.ProviderNewAPI,
			protocol: "openai",
			outPath:  "/v1/responses",
		},
		{
			name:     "newapi codex responses without version",
			path:     "/newapi/responses",
			body:     `{"model":"gpt-5.5"}`,
			provider: token.ProviderNewAPI,
			protocol: "openai",
			outPath:  "/v1/responses",
		},
		{
			name:     "newapi anthropic direct",
			path:     "/newapi/anthropic/v1/messages",
			body:     `{"model":"claude-sonnet-4-5"}`,
			provider: token.ProviderNewAPI,
			protocol: "anthropic",
			outPath:  "/v1/messages",
		},
		{
			name:     "newapi gemini direct",
			path:     "/newapi/gemini/v1beta/models/gemini-3-pro-preview:generateContent",
			body:     `{}`,
			provider: token.ProviderNewAPI,
			protocol: "gemini",
			outPath:  "/v1beta/models/gemini-3-pro-preview:generateContent",
		},
		{
			name:     "anyrouter direct",
			path:     "/anyrouter/v1/responses",
			body:     `{"model":"gpt-5-codex"}`,
			provider: token.ProviderAnyRouter,
			protocol: "openai",
			outPath:  "/v1/responses",
		},
		{
			name:     "anyrouter responses without version",
			path:     "/anyrouter/responses",
			body:     `{"model":"gpt-5-codex"}`,
			provider: token.ProviderAnyRouter,
			protocol: "openai",
			outPath:  "/v1/responses",
		},
		{
			name:     "anyrouter anthropic direct",
			path:     "/anyrouter/anthropic/v1/messages",
			body:     `{"model":"claude-opus-4-5-20251101"}`,
			provider: token.ProviderAnyRouter,
			protocol: "anthropic",
			outPath:  "/v1/messages",
		},
		{
			name:     "zo openai direct",
			path:     "/zo/v1/chat/completions",
			body:     `{"model":"gpt-5.5"}`,
			provider: token.ProviderZo,
			protocol: "openai",
			outPath:  "/v1/chat/completions",
		},
		{
			name:     "zo anthropic direct",
			path:     "/zo/v1/messages",
			body:     `{"model":"claude-sonnet-4-5"}`,
			provider: token.ProviderZo,
			protocol: "anthropic",
			outPath:  "/v1/messages",
		},
		{
			name:     "zo explicit anthropic prefix",
			path:     "/zo/anthropic/v1/messages",
			body:     `{"model":"claude-sonnet-4-5"}`,
			provider: token.ProviderZo,
			protocol: "anthropic",
			outPath:  "/v1/messages",
		},
		{
			name:     "zo responses direct",
			path:     "/zo/v1/responses",
			body:     `{"model":"gpt-5.5"}`,
			provider: token.ProviderZo,
			protocol: "openai",
			outPath:  "/v1/responses",
		},
		{
			name:     "prem direct",
			path:     "/prem/v1/chat/completions",
			body:     `{"model":"qwen3.5"}`,
			provider: token.ProviderPrem,
			protocol: "openai",
			outPath:  "/openai/v1/chat/completions",
		},
		{
			name:     "prem without version",
			path:     "/prem/chat/completions",
			body:     `{"model":"deepseek-v4-pro"}`,
			provider: token.ProviderPrem,
			protocol: "openai",
			outPath:  "/openai/v1/chat/completions",
		},
		{
			name:     "prem anthropic direct",
			path:     "/prem/anthropic/v1/messages",
			body:     `{"model":"deepseek-v4-pro"}`,
			provider: token.ProviderPrem,
			protocol: "anthropic",
			outPath:  "/anthropic/v1/messages",
		},
		{
			name:     "prem anthropic messages without protocol prefix",
			path:     "/prem/v1/messages",
			body:     `{"model":"deepseek-v4-pro"}`,
			provider: token.ProviderPrem,
			protocol: "anthropic",
			outPath:  "/anthropic/v1/messages",
		},
		{
			name:     "atria direct",
			path:     "/atria/v1/chat/completions",
			body:     `{"model":"Atria-Dawn-Preview"}`,
			provider: token.ProviderAtria,
			protocol: "openai",
			outPath:  "/v1/chat/completions",
		},
		{
			name:     "atria responses direct",
			path:     "/atria/v1/responses",
			body:     `{"model":"Atria-Dawn-Preview"}`,
			provider: token.ProviderAtria,
			protocol: "openai",
			outPath:  "/v1/responses",
		},
		{
			name:     "atria anthropic direct",
			path:     "/atria/anthropic/v1/messages",
			body:     `{"model":"Atria-Dawn-Preview"}`,
			provider: token.ProviderAtria,
			protocol: "anthropic",
			outPath:  "/v1/messages",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{}
			switch {
			case strings.HasPrefix(tt.path, "/anthropic-router") || strings.HasPrefix(tt.path, "/claude-desktop"):
				cfg.GatewayRoutes.Claude = config.GatewayRouteConfig{Provider: tt.provider}
			case strings.HasPrefix(tt.path, "/opencode-router") || strings.HasPrefix(tt.path, "/pi-router"):
				cfg.GatewayRoutes.OpenAI = config.GatewayRouteConfig{Provider: tt.provider}
			}
			router := NewRouter(cfg)
			route := router.Route(mustRouterTestURL(t, tt.path), []byte(tt.body))
			if route.Provider != tt.provider || route.Protocol != tt.protocol || route.Path != tt.outPath {
				t.Fatalf("unexpected route: %#v", route)
			}
		})
	}
}

func TestRouterTargetsAnyRouterBaseURL(t *testing.T) {
	router := NewRouter(config.Config{
		AnyRouterBaseURL: "https://anyrouter.top",
	})
	selected := token.Token{Provider: token.ProviderAnyRouter, CredentialType: token.CredentialTypeAPIKey}

	route := router.Route(mustRouterTestURL(t, "/anyrouter/v1/responses"), []byte(`{"model":"gpt-5-codex"}`))
	target, err := router.TargetURL(route, selected)
	if err != nil {
		t.Fatal(err)
	}
	if target != "https://anyrouter.top/v1/responses" {
		t.Fatalf("unexpected AnyRouter OpenAI-compatible target url: %s", target)
	}

	selected.BaseURL = "https://mirror.example/v1"
	route = router.Route(mustRouterTestURL(t, "/anyrouter/anthropic/v1/messages"), []byte(`{"model":"claude-opus-4-5-20251101"}`))
	target, err = router.TargetURL(route, selected)
	if err != nil {
		t.Fatal(err)
	}
	if target != "https://mirror.example/v1/messages" {
		t.Fatalf("unexpected AnyRouter Anthropic-compatible target url: %s", target)
	}
}

func TestRouterTargetsPremBaseURL(t *testing.T) {
	router := NewRouter(config.Config{
		PremBaseURL: "http://127.0.0.1:3100/v1",
	})
	selected := token.Token{Provider: token.ProviderPrem, CredentialType: token.CredentialTypeAPIKey}

	route := router.Route(mustRouterTestURL(t, "/prem/v1/chat/completions"), []byte(`{"model":"qwen3.5"}`))
	target, err := router.TargetURL(route, selected)
	if err != nil {
		t.Fatal(err)
	}
	if target != "http://127.0.0.1:3100/openai/v1/chat/completions" {
		t.Fatalf("unexpected Prem OpenAI-compatible target url: %s", target)
	}

	selected.BaseURL = "http://127.0.0.1:3101/v1"
	route = router.Route(mustRouterTestURL(t, "/prem/chat/completions"), []byte(`{"model":"deepseek-v4-pro"}`))
	target, err = router.TargetURL(route, selected)
	if err != nil {
		t.Fatal(err)
	}
	if target != "http://127.0.0.1:3100/openai/v1/chat/completions" {
		t.Fatalf("expected Prem to use global pcci-proxy base url, got %s", target)
	}

	route = router.Route(mustRouterTestURL(t, "/prem/anthropic/v1/messages"), []byte(`{"model":"deepseek-v4-pro"}`))
	target, err = router.TargetURL(route, selected)
	if err != nil {
		t.Fatal(err)
	}
	if target != "http://127.0.0.1:3100/anthropic/v1/messages" {
		t.Fatalf("unexpected Prem Anthropic-compatible target url: %s", target)
	}
}

func TestRouterAvoidsDuplicateOpenAIVersionForVersionedProviderBaseURL(t *testing.T) {
	router := NewRouter(config.Config{
		ZhipuBaseURL: "https://open.bigmodel.cn/api/paas/v4",
		GatewayRoutes: config.GatewayRoutes{
			OpenAI: config.GatewayRouteConfig{Provider: token.ProviderZhipu},
		},
	})
	route := router.Route(mustRouterTestURL(t, "/opencode-router/v1/chat/completions"), []byte(`{"model":"glm-5.1"}`))
	target, err := router.TargetURL(route, token.Token{Provider: token.ProviderZhipu, CredentialType: token.CredentialTypeAPIKey})
	if err != nil {
		t.Fatal(err)
	}
	if target != "https://open.bigmodel.cn/api/paas/v4/chat/completions" {
		t.Fatalf("unexpected target url: %s", target)
	}
}

func TestRouterTargetsXiaomiTokenPlanAnthropicSGPBaseURL(t *testing.T) {
	router := NewRouter(config.Config{
		XiaomiTokenPlanBaseURL:             "https://token-plan-cn.xiaomimimo.com/v1",
		XiaomiTokenPlanSGPBaseURL:          "https://token-plan-sgp.xiaomimimo.com/v1",
		XiaomiTokenPlanAMSBaseURL:          "https://token-plan-ams.xiaomimimo.com/v1",
		XiaomiTokenPlanAnthropicBaseURL:    "https://token-plan-cn.xiaomimimo.com/anthropic",
		XiaomiTokenPlanSGPAnthropicBaseURL: "https://token-plan-sgp.xiaomimimo.com/anthropic",
		XiaomiTokenPlanAMSAnthropicBaseURL: "https://token-plan-ams.xiaomimimo.com/anthropic",
		GatewayRoutes: config.GatewayRoutes{
			Claude: config.GatewayRouteConfig{Provider: token.ProviderXiaomi},
		},
	})
	selected := token.Token{Provider: token.ProviderXiaomi, CredentialType: token.CredentialTypeMimoTokenPlan, Region: token.MimoRegionSGP}
	cases := []struct {
		name string
		path string
		body string
	}{
		{
			name: "direct xiaomi anthropic prefix",
			path: "/xiaomi/anthropic/v1/messages",
			body: `{}`,
		},
		{
			name: "anthropic model router",
			path: "/anthropic-router/v1/messages",
			body: `{"model":"mimo-v2.5"}`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			route := router.Route(mustRouterTestURL(t, tt.path), []byte(tt.body))
			target, err := router.TargetURL(route, selected)
			if err != nil {
				t.Fatal(err)
			}
			if target != "https://token-plan-sgp.xiaomimimo.com/anthropic/v1/messages" {
				t.Fatalf("unexpected target url: %s", target)
			}
		})
	}

	route := router.Route(mustRouterTestURL(t, "/xiaomi/v1/chat/completions"), []byte(`{}`))
	target, err := router.TargetURL(route, selected)
	if err != nil {
		t.Fatal(err)
	}
	if target != "https://token-plan-sgp.xiaomimimo.com/v1/chat/completions" {
		t.Fatalf("unexpected openai-compatible target url: %s", target)
	}
}

func TestRouterTargetsXiaomiTokenPlanAMSBaseURL(t *testing.T) {
	router := NewRouter(config.Config{
		XiaomiTokenPlanBaseURL:             "https://token-plan-cn.xiaomimimo.com/v1",
		XiaomiTokenPlanAMSBaseURL:          "https://token-plan-ams.xiaomimimo.com/v1",
		XiaomiTokenPlanAnthropicBaseURL:    "https://token-plan-cn.xiaomimimo.com/anthropic",
		XiaomiTokenPlanAMSAnthropicBaseURL: "https://token-plan-ams.xiaomimimo.com/anthropic",
		GatewayRoutes: config.GatewayRoutes{
			Claude: config.GatewayRouteConfig{Provider: token.ProviderXiaomi},
		},
	})
	selected := token.Token{Provider: token.ProviderXiaomi, CredentialType: token.CredentialTypeMimoTokenPlan, Region: token.MimoRegionAMS}

	route := router.Route(mustRouterTestURL(t, "/anthropic-router/v1/messages"), []byte(`{"model":"mimo-v2.5"}`))
	target, err := router.TargetURL(route, selected)
	if err != nil {
		t.Fatal(err)
	}
	if target != "https://token-plan-ams.xiaomimimo.com/anthropic/v1/messages" {
		t.Fatalf("unexpected anthropic-compatible target url: %s", target)
	}

	route = router.Route(mustRouterTestURL(t, "/xiaomi/v1/chat/completions"), []byte(`{}`))
	target, err = router.TargetURL(route, selected)
	if err != nil {
		t.Fatal(err)
	}
	if target != "https://token-plan-ams.xiaomimimo.com/v1/chat/completions" {
		t.Fatalf("unexpected openai-compatible target url: %s", target)
	}
}

func mustRouterTestURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
