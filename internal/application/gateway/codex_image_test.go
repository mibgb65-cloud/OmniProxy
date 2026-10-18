package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"omniproxy/internal/application/activity"
	routes "omniproxy/internal/application/routing"
	"omniproxy/internal/provider/chatgpt"
)

func TestNativeResponsesImageSurvivesGatewayRouting(t *testing.T) {
	for _, transport := range []string{"http", "websocket"} {
		t.Run(transport, func(t *testing.T) {
			s, db, observer, router := failoverFixture(t, roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("wrong adapter"); return nil, nil }))
			_, err := db.Exec(`UPDATE accounts SET provider_id='builtin_openai',auth_method='subscription_login',auth_email=id||'@example.invalid',auth_account_id='workspace',auth_state='authenticated'; INSERT INTO account_models(account_id,model,name,fetched_at,account_version,metadata_json) SELECT id,'gpt-test','GPT','2026-01-01T00:00:00Z',version,'{"inputModalities":["text","image"]}' FROM account_catalog_versions`)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = router.Create(context.Background(), routes.Input{Alias: "vision-route", ProviderID: "builtin_openai", Model: "gpt-test", Enabled: true}); err != nil {
				t.Fatal(err)
			}
			image := "data:image/png;base64," + strings.Repeat("A", 3<<20)
			var calls atomic.Int32
			s.SetSubscription(failoverSubscription{})
			s.chatgpt = chatgpt.New(roundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls.Add(1)
				var body struct {
					Model string
					Input []struct {
						Content []struct {
							Type, Text, Detail string
							URL                string `json:"image_url"`
						} `json:"content"`
					} `json:"input"`
				}
				if json.NewDecoder(req.Body).Decode(&body) != nil || body.Model != "gpt-test" || len(body.Input) != 1 || len(body.Input[0].Content) != 2 {
					t.Error("image envelope or routing changed")
				} else {
					part := body.Input[0].Content[1]
					if part.Type != "input_image" || part.Detail != "high" || part.URL != image {
						t.Error("image data URI or detail changed")
					}
				}
				// 同时覆盖 Beta.4 的缺失类型头兼容，不能因新增图片能力破坏流统计。
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(missingTypeStream))}, nil
			}))
			frame := map[string]any{"model": "vision-route", "stream": true, "input": []any{map[string]any{"role": "user", "content": []any{map[string]string{"type": "input_text", "text": "describe"}, map[string]string{"type": "input_image", "image_url": image, "detail": "high"}}}}}
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			address := s.Info().Runtime.Address + "/v1/responses"
			if transport == "http" {
				raw, _ := json.Marshal(frame)
				response, e := http.Post(address, "application/json", strings.NewReader(string(raw)))
				if e != nil {
					t.Fatal(e)
				}
				body, e := io.ReadAll(response.Body)
				response.Body.Close()
				if e != nil || response.StatusCode != 200 || !strings.Contains(string(body), "response.completed") {
					t.Fatal("HTTP image request failed", response.StatusCode, e)
				}
			} else {
				socket, _, e := websocket.Dial(ctx, strings.Replace(address, "http:", "ws:", 1), nil)
				if e != nil {
					t.Fatal(e)
				}
				defer socket.CloseNow()
				frame["type"] = "response.create"
				raw, _ := json.Marshal(frame)
				if e = socket.Write(ctx, websocket.MessageText, raw); e != nil {
					t.Fatal(e)
				}
				for {
					_, raw, e = socket.Read(ctx)
					if e != nil {
						t.Fatal(e)
					}
					var event struct{ Type string }
					if json.Unmarshal(raw, &event) != nil {
						t.Fatal("invalid event")
					}
					if event.Type == "response.completed" {
						break
					}
					if event.Type == "error" {
						t.Fatal("socket image request rejected")
					}
				}
			}
			if calls.Load() != 1 {
				t.Fatal("image request replayed", calls.Load())
			}
			deadline := time.Now().Add(2 * time.Second)
			for {
				page, e := observer.Requests(ctx, activity.RequestQuery{Limit: 10})
				if e != nil || len(page.Items) != 1 {
					t.Fatal("image request record missing", e)
				}
				item, e := observer.Request(ctx, page.Items[0].ID)
				if e != nil {
					t.Fatal(e)
				}
				if item.State != "pending" {
					if item.State != "success" || item.StatusCode != 200 || !item.UsageComplete || item.FirstTokenMS == nil {
						t.Fatal("image completion or metrics lost")
					}
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("image record did not finish")
				}
				time.Sleep(time.Millisecond)
			}
		})
	}
}
