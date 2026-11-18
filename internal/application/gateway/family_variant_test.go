package gateway

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	routes "omniproxy/internal/application/routing"
	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/core/proxy"
)

// familyRouter 只认识目录里精确存在的模型 ID，用来模拟“家族名不在目录里”的真实路由。
type familyRouter struct {
	entries []catalog.Entry
	seen    []string
}

func (r *familyRouter) Catalog(context.Context) ([]catalog.Entry, error) { return r.entries, nil }
func (r *familyRouter) Aliases(context.Context) ([]string, error)        { return nil, nil }
func (r *familyRouter) Resolve(_ context.Context, model string) (routes.Preview, error) {
	r.seen = append(r.seen, model)
	for _, entry := range r.entries {
		if entry.ID == model {
			return routes.Preview{Routable: true, Model: model, ProviderID: entry.ProviderID, AccountID: "acct_google"}, nil
		}
	}
	return routes.Preview{Routable: false, Matched: false, ErrorCode: "unsupported_model"}, nil
}

func antigravityFamilyEntries() []catalog.Entry {
	return []catalog.Entry{
		{ID: "gemini-4-argon-high", ProviderKind: "google", ProviderID: "builtin_google"},
		{ID: "gemini-4-argon-medium", ProviderKind: "google", ProviderID: "builtin_google"},
		{ID: "gemini-4-argon-low", ProviderKind: "google", ProviderID: "builtin_google"},
		{ID: "gpt-oss-120b-medium", ProviderKind: "google", ProviderID: "builtin_google"},
	}
}

func familyBody(model, effort string) []byte {
	fields := map[string]any{"model": model, "messages": []any{map[string]any{"role": "user", "content": "hi"}}}
	if effort != "" {
		fields["reasoning_effort"] = effort
	}
	raw, _ := json.Marshal(fields)
	return raw
}

func TestFamilyVariantResolvesClientFamilyName(t *testing.T) {
	router := &familyRouter{entries: antigravityFamilyEntries()}
	service := &Service{router: router}

	// 客户端按档位发送家族名：换成对应变体。
	request, err := service.RouteRequest(context.Background(), proxy.Request{Protocol: "chat", Model: "gemini-4-argon", Body: familyBody("gemini-4-argon", "medium")})
	if err != nil {
		t.Fatal("family request rejected", err)
	}
	if request.Model != "gemini-4-argon-medium" || request.ProviderID != "builtin_google" {
		t.Fatalf("family resolved to %q/%q", request.Model, request.ProviderID)
	}
	var parsed struct{ Model string }
	if json.Unmarshal(request.Body, &parsed) != nil || parsed.Model != "gemini-4-argon-medium" {
		t.Fatal("upstream body kept the family name")
	}
	// 档位此时仍留在正文里，供请求明细记录与后续剥离。
	if request.Effort != "medium" {
		t.Fatalf("recorded effort = %q", request.Effort)
	}

	// 没有档位时落到 high，而不是随机取一个变体。
	request, err = service.RouteRequest(context.Background(), proxy.Request{Protocol: "chat", Model: "gemini-4-argon", Body: familyBody("gemini-4-argon", "")})
	if err != nil || request.Model != "gemini-4-argon-high" {
		t.Fatalf("family without effort resolved to %q, %v", request.Model, err)
	}

	// 只有单一变体的家族也接受家族名（GPT-OSS 只有 medium）。
	request, err = service.RouteRequest(context.Background(), proxy.Request{Protocol: "chat", Model: "gpt-oss-120b", Body: familyBody("gpt-oss-120b", "")})
	if err != nil || request.Model != "gpt-oss-120b-medium" {
		t.Fatalf("single variant family resolved to %q, %v", request.Model, err)
	}

	// 精确 ID 不经过家族回退，避免改写真实存在的模型。
	router.seen = nil
	request, err = service.RouteRequest(context.Background(), proxy.Request{Protocol: "chat", Model: "gemini-4-argon-low", Body: familyBody("gemini-4-argon-low", "")})
	if err != nil || request.Model != "gemini-4-argon-low" || len(router.seen) != 1 {
		t.Fatalf("exact id rewritten: %q %v", request.Model, err)
	}
}

func TestFamilyVariantLeavesUnknownModelsAlone(t *testing.T) {
	router := &familyRouter{entries: antigravityFamilyEntries()}
	service := &Service{router: router}
	if _, err := service.RouteRequest(context.Background(), proxy.Request{Protocol: "chat", Model: "gemini-4-carbon", Body: familyBody("gemini-4-carbon", "")}); err == nil {
		t.Fatal("unknown family accepted")
	}
}

// relayFamilyEntries 复现真实目录：裸型号只在中转站，档位变体只在官方订阅。
func relayFamilyEntries(base string, variants ...string) []catalog.Entry {
	entries := []catalog.Entry{{ID: base, ProviderKind: "workbuddy_ai", ProviderID: "builtin_workbuddy_ai"}}
	for _, variant := range variants {
		entries = append(entries, catalog.Entry{ID: variant, ProviderKind: "google", ProviderID: "builtin_google"})
	}
	return entries
}

// 家族名 + 明确档位必须先解析到官方变体，不能被发布裸型号的中转站截获。
func TestRequestedFamilyVariantBeatsRelayBaseModel(t *testing.T) {
	carbon := relayFamilyEntries("gemini-4-carbon", "gemini-4-carbon-high", "gemini-4-carbon-medium", "gemini-4-carbon-low")
	for _, test := range []struct {
		name, effort, wantModel, wantProvider string
	}{
		{"explicit effort", "high", "gemini-4-carbon-high", "builtin_google"},
		{"middle effort", "medium", "gemini-4-carbon-medium", "builtin_google"},
		{"no effort keeps exact match", "", "gemini-4-carbon", "builtin_workbuddy_ai"},
		{"unknown level is not substituted", "xhigh", "gemini-4-carbon", "builtin_workbuddy_ai"},
	} {
		t.Run(test.name, func(t *testing.T) {
			router := &familyRouter{entries: carbon}
			service := &Service{router: router}
			request, err := service.RouteRequest(context.Background(), proxy.Request{
				Protocol: "chat", Model: "gemini-4-carbon", Body: familyBody("gemini-4-carbon", test.effort),
			})
			if err != nil {
				t.Fatal("request rejected", err)
			}
			if request.Model != test.wantModel || request.ProviderID != test.wantProvider {
				t.Fatalf("routed %q/%q, want %q/%q", request.Model, request.ProviderID, test.wantModel, test.wantProvider)
			}
			// 路由必须直接看到真实型号：裸型号根本不该参与服务商选择。
			if len(router.seen) == 0 || router.seen[0] != test.wantModel {
				t.Fatalf("routing resolved %v", router.seen)
			}
			if test.effort != "" && test.wantModel != "gemini-4-carbon" {
				// 档位已经由型号表达：正文里的档位字段必须剥离，记录仍保留档位。
				if request.Effort != test.effort || strings.Contains(string(request.Body), "reasoning_effort") {
					t.Fatalf("effort %q kept in body: %s", request.Effort, request.Body)
				}
			}
		})
	}
}

// 只有一个档位后缀时不改写：后缀可能只是型号名的一部分，不能凭一个变体判定家族。
func TestRequestedFamilyVariantRequiresSiblings(t *testing.T) {
	router := &familyRouter{entries: relayFamilyEntries("gemini-4-x", "gemini-4-x-high")}
	service := &Service{router: router}
	request, err := service.RouteRequest(context.Background(), proxy.Request{
		Protocol: "chat", Model: "gemini-4-x", Body: familyBody("gemini-4-x", "high"),
	})
	if err != nil {
		t.Fatal("request rejected", err)
	}
	if request.Model != "gemini-4-x" || request.ProviderID != "builtin_workbuddy_ai" {
		t.Fatalf("single sibling family rewritten: %q/%q", request.Model, request.ProviderID)
	}
}
