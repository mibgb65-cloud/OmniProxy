package stepfun

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"omniproxy/internal/core/proxy"
	"strings"
	"testing"
)

func TestSSEIsDeliveredBeforeUpstreamCompletes(t *testing.T) {
	reader, writer := io.Pipe()
	release := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer writer.Close()
		if _, err := io.WriteString(writer, "data: {\"choices\":[{\"delta\":{\"content\":\"first\"}}]}\n\n"); err != nil {
			return
		}
		<-release
		io.WriteString(writer, "data: [DONE]\n\n")
	}()
	client := New(roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Error("SSE accept missing")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: reader}, nil
	}))
	err := client.Forward(context.Background(), Options{Region: "cn", BillingMode: "step_plan"}, []byte("placeholder-step-key"), proxy.Request{Model: "step-5-preview", Protocol: "chat", Stream: true, Body: []byte(`{"model":"step-5-preview","stream":true,"messages":[{"role":"user","content":"hi"}]}`)}, func(response proxy.Response) error {
		buffer := bufio.NewReader(response.Body)
		line, err := buffer.ReadString('\n')
		if err != nil {
			return err
		}
		if !strings.Contains(line, "first") {
			t.Error("first event lost")
		}
		close(release)
		rest, err := io.ReadAll(buffer)
		if !strings.Contains(string(rest), "[DONE]") {
			t.Error("final event lost")
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	<-done
}

func TestCancellationStopsUpstream(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := New(roundTrip(func(r *http.Request) (*http.Response, error) { return nil, r.Context().Err() }))
	err := client.Forward(ctx, Options{Region: "cn", BillingMode: "payg"}, []byte("placeholder-step-key"), proxy.Request{Model: "step-5-preview", Protocol: "chat", Body: []byte(`{"model":"step-5-preview","messages":[{"role":"user","content":"hi"}]}`)}, func(proxy.Response) error { t.Fatal("cancelled response consumed"); return nil })
	if err == nil {
		t.Fatal("cancelled upstream accepted")
	}
}
