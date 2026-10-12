package proxy

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"omniproxy/internal/infrastructure/logger"
	"omniproxy/internal/platform/profile"
)

func TestRequestLogsDoNotContainCredentialsOrContent(t *testing.T) {
	var output bytes.Buffer
	log, files, err := logger.Setup(logger.Options{Profile: profile.Test, Console: &output})
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	secret := "sk-placeholder-only-for-log-test"
	server := New(Options{Logger: log, Models: []string{"deepseek-flash"}, Upstream: upstreamFunc(func(ctx context.Context, req Request, consume func(Response) error) error {
		return consume(Response{StatusCode: 401, ContentType: "application/json", Body: strings.NewReader(secret)})
	})})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop(context.Background())
	status, body := request(t, server, `{"model":"deepseek-flash","messages":[{"role":"user","content":"private prompt marker"}]}`)
	if status != 401 || strings.Contains(body, secret) || strings.Contains(output.String(), secret) || strings.Contains(output.String(), "private prompt marker") {
		t.Fatal("request or credential data reached response/logs")
	}
	if !strings.Contains(output.String(), "upstream_authentication_failed") {
		t.Fatal("failure classification not logged")
	}
}
