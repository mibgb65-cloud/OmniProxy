package proxy

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/coder/websocket"
)

func TestImageWarmupHistoryPreservesDataAndBoundsMergedRequest(t *testing.T) {
	for _, overflow := range []bool{false, true} {
		t.Run(map[bool]string{false: "preserved", true: "merged-limit"}[overflow], func(t *testing.T) {
			n := 3 << 20
			if overflow {
				n = 8 << 20
			}
			image := "data:image/png;base64," + strings.Repeat("A", n)
			var calls atomic.Int32
			socket, ctx := regressionSocket(t, Options{Upstream: upstreamFunc(func(_ context.Context, r Request, consume func(Response) error) error {
				calls.Add(1)
				var v struct {
					Input []json.RawMessage `json:"input"`
				}
				var first struct {
					Content []struct {
						URL string `json:"image_url"`
					} `json:"content"`
				}
				if json.Unmarshal(r.Body, &v) != nil || len(v.Input) < 1 || json.Unmarshal(v.Input[0], &first) != nil || len(first.Content) < 1 || first.Content[0].URL != image {
					t.Error("image history changed")
				}
				return consume(Response{StatusCode: 200, ContentType: "text/event-stream", Body: strings.NewReader("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"done\",\"output\":[]}}\n\n")})
			})})
			input := []any{map[string]any{"role": "user", "content": []any{map[string]string{"type": "input_image", "image_url": image}}}}
			frame := map[string]any{"type": "response.create", "generate": false, "model": "gpt-test", "input": input}
			raw, _ := json.Marshal(frame)
			if err := socket.Write(ctx, websocket.MessageText, raw); err != nil {
				t.Fatal(err)
			}
			_, created, err := socket.Read(ctx)
			if err != nil {
				t.Fatal(err)
			}
			var event struct {
				Response struct {
					ID string `json:"id"`
				} `json:"response"`
			}
			if json.Unmarshal(created, &event) != nil || event.Response.ID == "" {
				t.Fatal("warmup response missing")
			}
			if _, _, err = socket.Read(ctx); err != nil {
				t.Fatal(err)
			}
			if !overflow {
				input = []any{map[string]any{"role": "user", "content": "describe the picture"}}
			}
			frame = map[string]any{"type": "response.create", "model": "gpt-test", "previous_response_id": event.Response.ID, "input": input}
			raw, _ = json.Marshal(frame)
			if err = socket.Write(ctx, websocket.MessageText, raw); err != nil {
				t.Fatal(err)
			}
			_, result, err := socket.Read(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if overflow {
				var failure struct {
					Status int `json:"status"`
					Error  struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				if json.Unmarshal(result, &failure) != nil || failure.Status != 413 || failure.Error.Code != "request_too_large" || calls.Load() != 0 {
					t.Fatal("merged image history was not bounded")
				}
			} else if !strings.Contains(string(result), "response.completed") || calls.Load() != 1 {
				t.Fatal("image warmup was forwarded or request failed")
			}
		})
	}
}

func TestOversizedResponsesSocketFrameNeverReachesUpstream(t *testing.T) {
	var calls atomic.Int32
	socket, ctx := regressionSocket(t, Options{Upstream: upstreamFunc(func(context.Context, Request, func(Response) error) error { calls.Add(1); return nil })})
	body := `{"type":"response.create","model":"gpt-test","input":[]}` + strings.Repeat(" ", MaxResponsesRequestBytes)
	err := socket.Write(ctx, websocket.MessageText, []byte(body))
	if err == nil {
		_, _, err = socket.Read(ctx)
	}
	if websocket.CloseStatus(err) != websocket.StatusMessageTooBig || calls.Load() != 0 {
		t.Fatal("oversized socket frame not rejected with 1009", websocket.CloseStatus(err), calls.Load())
	}
}

type oversizedImageRouter struct{ upstreamFunc }

func (r oversizedImageRouter) RouteRequest(_ context.Context, request Request) (Request, error) {
	request.Body = append(request.Body, []byte(strings.Repeat(" ", MaxResponsesRequestBytes))...)
	return request, nil
}
func TestResponsesSocketRejectsRoutedImageEnvelopeAboveLimit(t *testing.T) {
	for _, generate := range []string{"true", "false"} {
		t.Run(generate, func(t *testing.T) {
			upstream := oversizedImageRouter{upstreamFunc(func(context.Context, Request, func(Response) error) error {
				t.Error("oversized routed image reached upstream")
				return nil
			})}
			socket, ctx := regressionSocket(t, Options{Upstream: upstream})
			result := sendSocket(t, ctx, socket, `{"type":"response.create","generate":`+generate+`,"model":"gpt-test","input":[]}`)
			if string(result["status"]) != "413" || !strings.Contains(string(result["error"]), "request_too_large") {
				t.Fatal("routed size limit not enforced")
			}
		})
	}
}
