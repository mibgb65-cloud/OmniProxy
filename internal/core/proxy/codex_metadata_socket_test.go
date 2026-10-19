package proxy

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestCodexV2MetadataFlowsAcrossSocketWithoutStaleTurn(t *testing.T) {
	calls := 0
	server := New(Options{Port: 0, Models: []string{"gpt-test"}, Upstream: upstreamFunc(func(_ context.Context, r Request, c func(Response) error) error {
		calls++
		want := `{"request_kind":"compact"}`
		if calls == 2 {
			want = ""
		}
		if calls == 3 {
			want = `{"request_kind":"turn"}`
		}
		if r.CodexMetadata.BetaFeatures != codexCompactionV2 || r.CodexMetadata.TurnMetadata != want {
			t.Error("socket lost V2 capability or reused previous turn metadata")
		}
		return c(Response{StatusCode: 200, ContentType: "text/event-stream", Body: strings.NewReader("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_qa\",\"output\":[]}}\n\n")})
	})})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	socket, _, err := websocket.Dial(ctx, strings.Replace(server.Status().Address, "http:", "ws:", 1)+"/v1/responses", &websocket.DialOptions{HTTPHeader: http.Header{"X-Codex-Beta-Features": {"remote_compaction_v2"}, "X-Codex-Turn-Metadata": {`{"request_kind":"compact"}`}}})
	if err != nil {
		t.Fatal(err)
	}
	defer socket.CloseNow()
	for i := 0; i < 3; i++ {
		body := map[string]any{"type": "response.create", "model": "gpt-test", "input": []any{}}
		if i == 2 {
			body["client_metadata"] = map[string]string{"x-codex-turn-metadata": `{"request_kind":"turn"}`}
		}
		raw, _ := json.Marshal(body)
		if socket.Write(ctx, websocket.MessageText, raw) != nil {
			t.Fatal("socket write failed")
		}
		if _, _, err := socket.Read(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 3 {
		t.Fatal("socket forwarding incomplete")
	}
}
