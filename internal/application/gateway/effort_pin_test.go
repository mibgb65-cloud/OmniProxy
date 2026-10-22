package gateway

import (
	"context"
	"encoding/json"
	"testing"

	routes "omniproxy/internal/application/routing"
	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/core/proxy"
)

type pinnedRouter struct {
	desktopAliasRouter
	pins map[string]string
}

func (r *pinnedRouter) EffortPins(context.Context) (map[string]string, error) { return r.pins, nil }

func TestPinnedEffortOverridesClientSelection(t *testing.T) {
	entries := []catalog.Entry{{
		ID:       "deepseek-flash",
		Metadata: catalog.Metadata{ReasoningEfforts: []catalog.Effort{{Value: "high"}, {Value: "max"}}, DefaultEffort: "high"},
	}}
	router := &pinnedRouter{desktopAliasRouter: desktopAliasRouter{entries: entries}}
	service := &Service{router: router}
	body := func(effort string) []byte {
		raw, _ := json.Marshal(map[string]any{
			"model":         catalog.DesktopAlias("deepseek-flash"),
			"output_config": map[string]string{"effort": effort},
			"messages":      []any{map[string]any{"role": "user", "content": "hello"}},
		})
		return raw
	}
	read := func(t *testing.T, raw []byte) (string, string) {
		t.Helper()
		var parsed struct {
			OutputConfig struct {
				Effort string `json:"effort"`
			} `json:"output_config"`
			Thinking struct {
				Type string `json:"type"`
			} `json:"thinking"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			t.Fatal(err)
		}
		return parsed.OutputConfig.Effort, parsed.Thinking.Type
	}

	// 没有固定档位时原样保留客户端选择。
	request, err := service.RouteRequest(context.Background(), proxy.Request{Protocol: "messages", Model: catalog.DesktopAlias("deepseek-flash"), Body: body("high")})
	if err != nil || request.Protocol != "messages" {
		t.Fatal("unpinned request failed", err)
	}
	if effort, _ := read(t, request.Body); effort != "high" {
		t.Fatalf("client effort changed without a pin: %q", effort)
	}

	// 固定档位覆盖客户端选择，并保持思考开启。
	router.pins = map[string]string{"deepseek-flash": "max"}
	request, err = service.RouteRequest(context.Background(), proxy.Request{Protocol: "messages", Model: catalog.DesktopAlias("deepseek-flash"), Body: body("high")})
	if err != nil {
		t.Fatal(err)
	}
	if effort, mode := read(t, request.Body); effort != "max" || mode != "adaptive" {
		t.Fatalf("pinned effort not applied: %q/%q", effort, mode)
	}

	// 其他协议不使用 Messages 的 effort 契约，固定档位不参与改写。
	chatBody, _ := json.Marshal(map[string]any{"model": "deepseek-flash", "messages": []any{}})
	chat, err := service.RouteRequest(context.Background(), proxy.Request{Protocol: "chat", Model: "deepseek-flash", Body: chatBody})
	if err != nil || string(chat.Body) != string(chatBody) {
		t.Fatal("chat request body changed by a messages-only pin")
	}
}

func TestPinnedEffortIgnoresUnavailableConfiguration(t *testing.T) {
	service := &Service{router: &desktopAliasRouter{entries: []catalog.Entry{{ID: "deepseek-flash"}}}}
	body, _ := json.Marshal(map[string]any{"model": "deepseek-flash", "output_config": map[string]string{"effort": "high"}, "messages": []any{}})
	request, err := service.RouteRequest(context.Background(), proxy.Request{Protocol: "messages", Model: "deepseek-flash", Body: body})
	if err != nil {
		t.Fatal("request failed when the router exposes no pins", err)
	}
	var parsed struct {
		OutputConfig struct {
			Effort string `json:"effort"`
		} `json:"output_config"`
	}
	if json.Unmarshal(request.Body, &parsed) != nil || parsed.OutputConfig.Effort != "high" {
		t.Fatal("client effort lost without a pin source")
	}
	_ = routes.Preview{}
}
