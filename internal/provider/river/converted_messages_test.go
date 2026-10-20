package river

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

func TestConvertedMessagesBudgetIsExplicitAndStillBounded(t *testing.T) {
	for _, stream := range []bool{false, true} {
		calls := 0
		client := New(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("synthetic"))}, nil
		}))
		body := []byte(`{"model":"` + DeepSeekFlash1017 + `","stream":` + map[bool]string{false: "false", true: "true"}[stream] + `,"messages":[{"role":"user","content":"` + strings.Repeat("x", proxy.MaxRequestBytes) + `"}]}`)
		request := proxy.Request{Protocol: "chat", Model: DeepSeekFlash1017, Stream: stream, Body: body}
		if err := client.Forward(context.Background(), "", []byte("synthetic-key"), request, func(proxy.Response) error { return nil }); err == nil || calls != 0 {
			t.Fatal("unmarked external Chat bypassed size validation", err)
		}
		request.ConvertedFromMessages = true
		if err := client.Forward(context.Background(), "", []byte("synthetic-key"), request, func(proxy.Response) error { return nil }); err != nil || calls != 1 {
			t.Fatal("trusted Messages conversion retained a smaller limit", err)
		}
	}
}
