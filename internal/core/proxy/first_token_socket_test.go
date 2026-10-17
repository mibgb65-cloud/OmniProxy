package proxy

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestSocketFirstTokenUsesDeliveredContent(t *testing.T) {
	finished := make(chan RequestEvent, 1)
	server := New(Options{Port: 0, Models: []string{"deepseek-flash"}, Observer: firstTokenObserver{finished}, Upstream: upstreamFunc(func(_ context.Context, _ Request, consume func(Response) error) error {
		return consume(Response{StatusCode: 200, ContentType: "text/event-stream", Body: strings.NewReader("data: {\"type\":\"response.created\"}\n\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"hi\"}\n\ndata: {\"type\":\"response.completed\",\"response\":{\"id\":\"r\",\"output\":[]}}\n\n")})
	})})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	socket, _, err := websocket.Dial(ctx, strings.Replace(server.Status().Address, "http:", "ws:", 1)+"/v1/responses", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer socket.CloseNow()
	if err := socket.Write(ctx, websocket.MessageText, []byte(`{"type":"response.create","model":"deepseek-flash","input":[{"role":"user","content":"hi"}]}`)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, _, err := socket.Read(ctx); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case event := <-finished:
		if event.FirstTokenMS == nil || event.Transport != "websocket" || *event.FirstTokenMS > event.LatencyMS {
			t.Fatal("socket timing missing", event)
		}
	case <-ctx.Done():
		t.Fatal("socket request not recorded")
	}
}
