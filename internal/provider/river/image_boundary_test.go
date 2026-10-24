package river

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

// 官方已公布 input_image；适配器必须原样转发，而不是本地拒绝。
func TestResponsesImageInputReachesUpstreamUnchanged(t *testing.T) {
	for _, model := range Models() {
		for _, imageURL := range []string{"https://example.invalid/image.png", "data:image/png;base64,cGxhY2Vob2xkZXI="} {
			t.Run(model+"/"+imageURL[:4], func(t *testing.T) {
				body, err := json.Marshal(map[string]any{
					"model": model,
					"input": []any{
						map[string]any{"role": "user", "content": []any{
							map[string]any{"type": "input_text", "text": "Image from earlier in the conversation"},
							map[string]any{"type": "input_image", "image_url": imageURL, "detail": "high"},
						}},
						map[string]any{"role": "assistant", "content": "Earlier reply"},
						map[string]any{"role": "user", "content": "Now a text-only question"},
					},
				})
				if err != nil {
					t.Fatal(err)
				}
				var forwarded map[string]any
				client := New(roundTripFunc(func(request *http.Request) (*http.Response, error) {
					if json.NewDecoder(request.Body).Decode(&forwarded) != nil {
						t.Error("upstream body changed")
					}
					return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"model":"x"}`))}, nil
				}))
				if err := client.Forward(context.Background(), "", []byte("test-key"), proxy.Request{
					Model: model, Protocol: "responses", Body: body,
				}, func(response proxy.Response) error {
					_, err := io.Copy(io.Discard, response.Body)
					return err
				}); err != nil {
					t.Fatalf("verified image input was rejected: %v", err)
				}
				input := forwarded["input"].([]any)
				content := input[0].(map[string]any)["content"].([]any)
				image := content[1].(map[string]any)
				if image["type"] != "input_image" || image["image_url"] != imageURL || image["detail"] != "high" {
					t.Fatalf("image content changed before upstream: %v", image)
				}
			})
		}
	}
}

// 未接入的内容类型仍须明确失败，不能静默删除。
func TestUnsupportedContentStillFailsExplicitly(t *testing.T) {
	for _, content := range []string{
		`[{"type":"input_image"}]`,
		`[{"type":"input_image","image_url":""}]`,
		`[{"type":"input_image","image_url":"https://example.invalid/a.png","detail":"extra"}]`,
		`[{"type":"input_file","file_id":"f1"}]`,
		`[{"type":"input_audio","input_audio":{"data":"AAAA"}}]`,
	} {
		body := []byte(`{"model":"` + DeepSeekFlash + `","input":[{"role":"user","content":` + content + `}]}`)
		called := false
		client := New(roundTripFunc(func(*http.Request) (*http.Response, error) {
			called = true
			return nil, errors.New("unexpected request")
		}))
		err := client.Forward(context.Background(), "", []byte("test-key"), proxy.Request{
			Model: DeepSeekFlash, Protocol: "responses", Body: body,
		}, func(proxy.Response) error { return nil })
		var failure *proxy.Error
		if !errors.As(err, &failure) || failure.Code != "unsupported_content" || called {
			t.Fatalf("unsupported content reached upstream or lost classification: %v", content)
		}
	}
}
