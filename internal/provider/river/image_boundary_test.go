package river

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"omniproxy/internal/core/proxy"
)

func TestResponsesImageHistoryFailsExplicitlyWithoutDroppingContent(t *testing.T) {
	for _, model := range Models() {
		for _, imageURL := range []string{"https://example.invalid/image.png", "data:image/png;base64,cGxhY2Vob2xkZXI="} {
			t.Run(model+"/"+imageURL[:4], func(t *testing.T) {
				body, err := json.Marshal(map[string]any{
					"model": model,
					"input": []any{
						map[string]any{"role": "user", "content": []any{
							map[string]any{"type": "input_text", "text": "Image from earlier in the conversation"},
							map[string]any{"type": "input_image", "image_url": imageURL},
						}},
						map[string]any{"role": "assistant", "content": "Earlier reply"},
						map[string]any{"role": "user", "content": "Now a text-only question"},
					},
				})
				if err != nil {
					t.Fatal(err)
				}
				client := New(roundTripFunc(func(*http.Request) (*http.Response, error) {
					t.Fatal("unsupported image history reached upstream")
					return nil, nil
				}))
				err = client.Forward(context.Background(), "", []byte("test-key"), proxy.Request{
					Model: model, Protocol: "responses", Body: body,
				}, func(proxy.Response) error {
					t.Fatal("unsupported input was silently converted")
					return nil
				})
				var failure *proxy.Error
				if !errors.As(err, &failure) || failure.Code != "unsupported_content" || failure.StatusCode != http.StatusBadRequest {
					t.Fatalf("image rejection lost its classification: %v", err)
				}
			})
		}
	}
}
