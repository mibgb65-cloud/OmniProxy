package gateway

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/core/proxy"

	routes "omniproxy/internal/application/routing"
)

// 目录折叠后的形态：基础模型带档位到变体 ID 的映射。
func foldedHydrogenEntries() []catalog.Entry {
	return catalog.FoldEffortVariants([]catalog.Entry{
		{ID: "gemini-4-hydrogen", Name: "Gemini 4 Hydrogen", ProviderKind: "google", ProviderID: "builtin_google"},
		{ID: "gemini-4-hydrogen-low", Name: "Gemini 4 Hydrogen (Low)", ProviderKind: "google", ProviderID: "builtin_google"},
		{ID: "gemini-4-hydrogen-medium", Name: "Gemini 4 Hydrogen (Medium)", ProviderKind: "google", ProviderID: "builtin_google"},
		{ID: "gemini-4-hydrogen-high", Name: "Gemini 4 Hydrogen (High)", ProviderKind: "google", ProviderID: "builtin_google"},
		{ID: "gemini-4-hydrogen-xhigh", Name: "Gemini 4 Hydrogen (XHigh)", ProviderKind: "google", ProviderID: "builtin_google"},
	})
}

// 桌面端会把变体型号当模型发送，同时带一个档位字段；型号里的档位才是用户的选择。
func TestVariantModelEffortWinsOverClientField(t *testing.T) {
	entries := foldedHydrogenEntries()
	router := &variantRouter{entries: entries}
	service := &Service{router: router}
	request, err := service.RouteRequest(context.Background(), proxy.Request{
		Protocol: "chat",
		Model:    "gemini-4-hydrogen-high",
		Body:     familyBody("gemini-4-hydrogen-high", "low"),
	})
	if err != nil {
		t.Fatal("variant request rejected", err)
	}
	if request.Model != "gemini-4-hydrogen-high" {
		t.Fatalf("variant model rewritten to %q", request.Model)
	}
	if request.Effort != "high" {
		t.Fatalf("effort = %q, want high from the model id", request.Effort)
	}
	var parsed map[string]any
	if json.Unmarshal(request.Body, &parsed) != nil {
		t.Fatal("body unreadable")
	}
	if _, kept := parsed["reasoning_effort"]; kept {
		t.Fatal("conflicting reasoning_effort kept in body")
	}

	// 型号没档位时仍按客户端档位选变体：家族名 + low → -low。
	request, err = service.RouteRequest(context.Background(), proxy.Request{
		Protocol: "chat",
		Model:    "gemini-4-hydrogen",
		Body:     familyBody("gemini-4-hydrogen", "low"),
	})
	if err != nil || request.Model != "gemini-4-hydrogen-low" || request.Effort != "low" {
		t.Fatalf("family pick = %q/%q, %v", request.Model, request.Effort, err)
	}

	// 型号没档位且客户端也没指定档位时不猜测。
	request, err = service.RouteRequest(context.Background(), proxy.Request{
		Protocol: "chat",
		Model:    "gemini-4-hydrogen",
		Body:     familyBody("gemini-4-hydrogen", ""),
	})
	if err != nil || request.Effort != "" {
		t.Fatalf("unrequested effort = %q, %v", request.Effort, err)
	}
}

// variantRouter 让变体 ID 与基础 ID 都可路由，模拟真实目录里两种 ID 都存在的情况。
type variantRouter struct{ entries []catalog.Entry }

func (r *variantRouter) Catalog(context.Context) ([]catalog.Entry, error) { return r.entries, nil }
func (r *variantRouter) Aliases(context.Context) ([]string, error)        { return nil, nil }
func (r *variantRouter) Resolve(_ context.Context, model string) (routes.Preview, error) {
	for _, entry := range r.entries {
		if entry.ID == model || entry.TargetModel == model {
			return routes.Preview{Routable: true, Model: model, ProviderID: entry.ProviderID, AccountID: "acct_google"}, nil
		}
		for _, variant := range entry.Metadata.EffortModels {
			if variant == model {
				return routes.Preview{Routable: true, Model: model, ProviderID: entry.ProviderID, AccountID: "acct_google"}, nil
			}
		}
	}
	return routes.Preview{Routable: false, Matched: false, ErrorCode: "unsupported_model"}, nil
}

// variantOnlyEntries 复刻真实生产目录：上游只发布档位变体，没有基础型号，
// 目录折叠因此不会发生（Metadata.EffortModels 为空）。
func variantOnlyEntries() []catalog.Entry {
	return []catalog.Entry{
		{ID: "gemini-4-hydrogen-low", Name: "Gemini 4 Hydrogen (Low)", ProviderKind: "google", ProviderID: "builtin_google"},
		{ID: "gemini-4-hydrogen-medium", Name: "Gemini 4 Hydrogen (Medium)", ProviderKind: "google", ProviderID: "builtin_google"},
		{ID: "gemini-4-hydrogen-high", Name: "Gemini 4 Hydrogen (High)", ProviderKind: "google", ProviderID: "builtin_google"},
		{ID: "gemini-4-hydrogen-xhigh", Name: "Gemini 4 Hydrogen (XHigh)", ProviderKind: "google", ProviderID: "builtin_google"},
	}
}

// 只有变体的目录下也必须以型号为准：真实记录里 gemini-4-hydrogen-xhigh 仍是 low。
func TestVariantOnlyCatalogUsesModelLevel(t *testing.T) {
	entries := variantOnlyEntries()
	service := &Service{router: &variantRouter{entries: entries}}
	for _, test := range []struct{ model, clientLevel, want string }{
		{"gemini-4-hydrogen-xhigh", "low", "xhigh"},
		{"gemini-4-hydrogen-high", "low", "high"},
		{"gemini-4-hydrogen-low", "high", "low"},
		{"gemini-4-hydrogen-xhigh", "", "xhigh"},
	} {
		request, err := service.RouteRequest(context.Background(), proxy.Request{
			Protocol: "chat",
			Model:    test.model,
			Body:     familyBody(test.model, test.clientLevel),
		})
		if err != nil {
			t.Fatalf("%s rejected: %v", test.model, err)
		}
		if request.Model != test.model {
			t.Fatalf("%s rewritten to %q", test.model, request.Model)
		}
		if request.Effort != test.want {
			t.Fatalf("%s effort = %q, want %q", test.model, request.Effort, test.want)
		}
		var parsed map[string]any
		if json.Unmarshal(request.Body, &parsed) != nil {
			t.Fatalf("%s body unreadable", test.model)
		}
		if _, kept := parsed["reasoning_effort"]; kept {
			t.Fatalf("%s kept a conflicting reasoning_effort", test.model)
		}
	}
}

// 只有一个变体、没有同族兄弟时不猜测：恰好以 “-high” 结尾的普通型号保持客户端档位。
func TestLoneVariantSuffixDoesNotOverrideClientLevel(t *testing.T) {
	entries := []catalog.Entry{
		{ID: "gemini-9-solo-high", Name: "Gemini 9 Solo High", ProviderKind: "google", ProviderID: "builtin_google"},
	}
	service := &Service{router: &variantRouter{entries: entries}}
	request, err := service.RouteRequest(context.Background(), proxy.Request{
		Protocol: "chat",
		Model:    "gemini-9-solo-high",
		Body:     familyBody("gemini-9-solo-high", "low"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if request.Effort != "low" {
		t.Fatalf("effort = %q, want the client value", request.Effort)
	}
}

func TestRewriteEffortVariantReplacesModelAndDropsLevel(t *testing.T) {
	body, err := rewriteEffortVariant("messages", []byte(`{"model":"base","output_config":{"effort":"high","format":"kept"},"thinking":{"type":"adaptive"},"messages":[]}`), "base-high")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{`"model":"base-high"`, `"format":"kept"`, `"thinking"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("body missing %s: %s", want, text)
		}
	}
	if strings.Contains(text, `"effort"`) {
		t.Fatalf("effort field kept: %s", text)
	}

	body, err = rewriteEffortVariant("responses", []byte(`{"model":"base","reasoning":{"effort":"low"},"input":[]}`), "base-low")
	if err != nil || strings.Contains(string(body), `"effort"`) || !strings.Contains(string(body), `"model":"base-low"`) {
		t.Fatalf("responses rewrite failed: %s %v", body, err)
	}
	if strings.Contains(string(body), `"reasoning"`) {
		t.Fatalf("empty reasoning object kept: %s", body)
	}

	body, err = rewriteEffortVariant("chat", []byte(`{"model":"base","reasoning_effort":"medium","messages":[]}`), "base-medium")
	if err != nil || strings.Contains(string(body), `"reasoning_effort"`) || !strings.Contains(string(body), `"model":"base-medium"`) {
		t.Fatalf("chat rewrite failed: %s %v", body, err)
	}
}

// 目录折叠后，旧的档位变体 ID 必须还原成基础模型 + 档位，而不是 400。
func TestSetEffortVariantLevelRestoresBaseAndLevel(t *testing.T) {
	for _, tc := range []struct{ protocol, body string }{
		{"messages", `{"model":"gemini-3.8-flash-low","output_config":{"format":"kept"},"messages":[]}`},
		{"responses", `{"model":"gemini-3.8-flash-low","input":[]}`},
		{"chat", `{"model":"gemini-3.8-flash-low","messages":[]}`},
	} {
		body, err := setEffortVariantLevel(tc.protocol, []byte(tc.body), "gemini-3.8-flash", "low")
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		if !strings.Contains(text, `"model":"gemini-3.8-flash"`) || !strings.Contains(text, `"low"`) {
			t.Fatalf("%s: base model or level missing: %s", tc.protocol, text)
		}
		if tc.protocol == "messages" && !strings.Contains(text, `"format":"kept"`) {
			t.Fatalf("messages output_config lost other keys: %s", text)
		}
	}
}
