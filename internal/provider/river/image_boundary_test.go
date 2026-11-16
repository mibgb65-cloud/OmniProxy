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

// Codex 的 view_image 工具结果会带约 1 MB 的 base64 内联图片；此前按 8 KiB URL 上限
// 判断，合法的工具结果图片被本地拒成 unsupported_content。
func TestResponsesInlineToolResultImageIsAccepted(t *testing.T) {
	dataURL := "data:image/png;base64," + strings.Repeat("A", 1<<20)
	body, err := json.Marshal(map[string]any{
		"model": DeepSeekFlash,
		"input": []any{
			map[string]any{"role": "user", "content": "看看这张截图"},
			map[string]any{"type": "function_call", "call_id": "call_1", "name": "view_image", "arguments": "{}"},
			map[string]any{"type": "function_call_output", "call_id": "call_1", "output": []any{
				map[string]any{"type": "input_image", "image_url": dataURL, "detail": "high"},
			}},
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
		Model: DeepSeekFlash, Protocol: "responses", Body: body,
	}, func(response proxy.Response) error {
		_, err := io.Copy(io.Discard, response.Body)
		return err
	}); err != nil {
		t.Fatalf("inline tool result image was rejected: %v", err)
	}
	input := forwarded["input"].([]any)
	output := input[2].(map[string]any)["output"].([]any)
	image := output[0].(map[string]any)["image_url"]
	if image != dataURL {
		t.Fatal("inline tool result image changed before upstream")
	}
}

// 阈值按形式分档：内联图片与组件一致用 16 MiB，远程地址仍是有界 URL。
func TestImageURLValidBounds(t *testing.T) {
	inline := func(n int) string { return "data:image/png;base64," + strings.Repeat("A", n) }
	for _, tc := range []struct {
		name string
		url  string
		ok   bool
	}{
		{name: "inline 1MiB", url: inline(1 << 20), ok: true},
		{name: "inline at limit", url: inline(imageInlineLimit - len("data:image/png;base64,")), ok: true},
		{name: "inline over limit", url: inline(imageInlineLimit), ok: false},
		{name: "inline non image", url: "data:text/plain;base64,AAAA", ok: false},
		{name: "remote at limit", url: "https://example.invalid/" + strings.Repeat("a", imageRetrievableURLLimit-len("https://example.invalid/")), ok: true},
		{name: "remote over limit", url: "https://example.invalid/" + strings.Repeat("a", imageRetrievableURLLimit), ok: false},
		{name: "empty", url: "", ok: false},
		{name: "control char", url: "data:image/png;base64,AA\x00BB", ok: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := imageURLValid(tc.url); got != tc.ok {
				t.Fatalf("imageURLValid(%d chars) = %t, want %t", len(tc.url), got, tc.ok)
			}
		})
	}
}
