package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
	"omniproxy/internal/provider/river"
)

// Claude Code／Desktop 在 Messages 与 Responses 里携带的图片，必须经网关转换后
// 以 image_url 块到达 River；本地不能再以 unsupported_content 拒绝。
func TestRiverForwardsClaudeImagesToUpstream(t *testing.T) {
	const png = "iVBORw0KGgoAAAANSUhEUg=="
	for _, protocol := range []string{"messages", "responses"} {
		for _, model := range []string{river.DeepSeekFlash, river.DeepSeekFlash1017} {
			t.Run(protocol+"/"+model, func(t *testing.T) {
				var forwarded map[string]any
				upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Messages 转成 Chat；Responses 由 River 原生转发，路径保持 /v1/responses。
					want := "/v1/chat/completions"
					if protocol == "responses" {
						want = "/v1/responses"
					}
					if r.URL.Path != want {
						t.Errorf("%s forwarded to %s", protocol, r.URL.Path)
					}
					if err := json.NewDecoder(r.Body).Decode(&forwarded); err != nil {
						t.Error(err)
					}
					w.Header().Set("Content-Type", "application/json")
					if protocol == "responses" {
						_, _ = io.WriteString(w, `{"id":"r1","object":"response","status":"completed","model":"x","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"a diagram"}]}],"usage":{"input_tokens":1280,"output_tokens":4}}`)
						return
					}
					_, _ = io.WriteString(w, `{"id":"r1","choices":[{"finish_reason":"stop","message":{"content":"a diagram"}}],"usage":{"prompt_tokens":1280,"completion_tokens":4}}`)
				}))
				defer upstream.Close()

				db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "gateway.db")})
				if err != nil {
					t.Fatal(err)
				}
				defer db.Close()
				if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
					t.Fatal(err)
				}
				if _, err := db.Exec(`UPDATE providers SET base_url=? WHERE id='builtin_river'`, upstream.URL+"/v1"); err != nil {
					t.Fatal(err)
				}
				if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at) VALUES('river-account','builtin_river','account','test-ref','2026-10-13T00:00:00Z','2026-10-13T00:00:00Z')`); err != nil {
					t.Fatal(err)
				}
				store := &testStore{}
				providers, accounts := provider.NewService(db), account.NewService(db, store)
				observer := activity.New(db)
				service := New(proxy.Options{Observer: observer}, providers, accounts, nil)
				router := routes.New(db, providers, accounts)
				if err := router.SetProviderOrder(context.Background(), river.PublicModel(model), []string{"builtin_river"}); err != nil {
					t.Fatal(err)
				}
				service.SetRouter(router)
				if _, err := service.Start(); err != nil {
					t.Fatal(err)
				}
				defer service.Stop(context.Background())

				public := river.PublicModel(model)
				path, body := "/v1/chat/completions", ""
				if protocol == "messages" {
					path = "/v1/messages"
					body = `{"model":"` + public + `","stream":false,"max_tokens":256,"messages":[
						{"role":"user","content":[
							{"type":"text","text":"what is in this screenshot"},
							{"type":"image","source":{"type":"base64","media_type":"image/png","data":"` + png + `"}}]},
						{"role":"assistant","content":[{"type":"tool_use","id":"call_shot","name":"screenshot","input":{}}]},
						{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_shot","content":[
							{"type":"text","text":"captured"},
							{"type":"image","source":{"type":"url","url":"https://example.invalid/shot.png"}}]}]}]}`
				} else {
					path = "/v1/responses"
					body = `{"model":"` + public + `","stream":false,"store":false,"input":[
						{"role":"user","content":[
							{"type":"input_text","text":"what is in this screenshot"},
							{"type":"input_image","image_url":"data:image/png;base64,` + png + `","detail":"high"}]}]}`
				}
				response, err := http.Post(service.Info().Runtime.Address+path, "application/json", strings.NewReader(body))
				if err != nil {
					t.Fatal(err)
				}
				defer response.Body.Close()
				data, _ := io.ReadAll(response.Body)
				if response.StatusCode != 200 {
					t.Fatalf("image request rejected locally: %d %s", response.StatusCode, data)
				}

				messages, _ := forwarded["messages"].([]any)
				if protocol == "responses" {
					// 原生转发：图片必须仍在 input 的 input_image 块里。
					input, _ := forwarded["input"].([]any)
					if len(input) != 1 {
						t.Fatalf("responses input lost: %v", forwarded["input"])
					}
					content, _ := input[0].(map[string]any)["content"].([]any)
					image := map[string]any{}
					for _, entry := range content {
						if block, _ := entry.(map[string]any); block["type"] == "input_image" {
							image = block
						}
					}
					if image["image_url"] != "data:image/png;base64,"+png || image["detail"] != "high" {
						t.Fatalf("native responses image changed: %v", image)
					}
				} else {
					images := collectImages(messages)
					if len(images) == 0 {
						t.Fatalf("image did not reach upstream: %v", forwarded["messages"])
					}
					if images[0]["url"] != "data:image/png;base64,"+png {
						t.Fatalf("base64 image changed: %v", images[0])
					}
				}
				for _, value := range store.last {
					if value != 0 {
						t.Fatal("credential bytes not cleared")
					}
				}
			})
		}
	}
}

func collectImages(messages []any) []map[string]any {
	images := []map[string]any{}
	for _, raw := range messages {
		message, _ := raw.(map[string]any)
		blocks, _ := message["content"].([]any)
		for _, entry := range blocks {
			block, _ := entry.(map[string]any)
			if block["type"] != "image_url" {
				continue
			}
			image, _ := block["image_url"].(map[string]any)
			images = append(images, image)
		}
	}
	return images
}
