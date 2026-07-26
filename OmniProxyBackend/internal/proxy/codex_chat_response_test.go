package proxy

import (
	"bufio"
	"io"
	"strings"
	"testing"
	"time"
)

// The upstream request always asks for stream:true, so a chunk has to reach the
// client while the generation is still running. Buffering the whole body first
// made time-to-first-token equal to time-to-last-token.
func TestCodexChatSSEDeliversChunksBeforeUpstreamCloses(t *testing.T) {
	src, upstream := io.Pipe()
	body := codexChatSSEBody(src, "gpt-5.5")
	t.Cleanup(func() {
		_ = body.Close()
		_ = upstream.Close()
	})

	go func() {
		_, _ = upstream.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"hello\"}\n"))
		// The upstream deliberately stays open afterwards.
	}()

	lines := make(chan string, 8)
	go func() {
		reader := bufio.NewReader(body)
		for {
			line, err := reader.ReadString('\n')
			if line != "" {
				lines <- line
			}
			if err != nil {
				close(lines)
				return
			}
		}
	}()

	deadline := time.After(5 * time.Second)
	seen := []string{}
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				t.Fatalf("stream ended before delivering content: %#v", seen)
			}
			seen = append(seen, line)
			if strings.Contains(line, `"hello"`) {
				return
			}
		case <-deadline:
			t.Fatalf("no content chunk while the upstream was still open: %#v", seen)
		}
	}
}

func TestCodexChatSSEConversionShape(t *testing.T) {
	src := io.NopCloser(strings.NewReader(strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_1","model":"gpt-5.5"}}`,
		`data: {"type":"response.output_text.delta","delta":"he"}`,
		`data: {"type":"response.output_text.delta","delta":"llo"}`,
		`data: {"type":"response.completed","response":{"id":"resp_1"}}`,
		`data: [DONE]`,
		``,
	}, "\n")))

	out, err := io.ReadAll(codexChatSSEBody(src, "requested-model"))
	if err != nil {
		t.Fatal(err)
	}
	converted := string(out)

	for _, want := range []string{
		`"role":"assistant"`,
		`"content":"he"`,
		`"content":"llo"`,
		`"finish_reason"`,
		"data: [DONE]",
	} {
		if !strings.Contains(converted, want) {
			t.Fatalf("expected %q in the converted stream, got:\n%s", want, converted)
		}
	}
	// The role chunk must precede content, and [DONE] must terminate the stream.
	if strings.Index(converted, `"role":"assistant"`) > strings.Index(converted, `"content":"he"`) {
		t.Fatalf("role chunk must come first, got:\n%s", converted)
	}
	if !strings.HasSuffix(strings.TrimSpace(converted), "data: [DONE]") {
		t.Fatalf("expected the stream to end with [DONE], got:\n%s", converted)
	}
}

func TestCodexChatSSEFinalizesUnterminatedStreams(t *testing.T) {
	src := io.NopCloser(strings.NewReader(`data: {"type":"response.output_text.delta","delta":"partial"}` + "\n"))

	out, err := io.ReadAll(codexChatSSEBody(src, "requested-model"))
	if err != nil {
		t.Fatal(err)
	}
	converted := string(out)
	// An upstream that stops without a terminal event still has to produce a
	// finish reason, or the client waits forever.
	if !strings.Contains(converted, `"finish_reason":"stop"`) {
		t.Fatalf("expected a synthesised finish reason, got:\n%s", converted)
	}
	if !strings.Contains(converted, "data: [DONE]") {
		t.Fatalf("expected [DONE], got:\n%s", converted)
	}
}
