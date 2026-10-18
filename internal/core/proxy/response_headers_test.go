package proxy

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"omniproxy/internal/infrastructure/logger"
	"omniproxy/internal/platform/profile"
)

func TestResponseHeaderDiagnosticsAreEnumerated(t *testing.T) {
	for _, tt := range []struct{ value, want string }{
		{"", "missing"}, {"  ", "missing"}, {"text/event-stream; charset=utf-8", "sse"},
		{"application/json", "json"}, {"text/html", "html"}, {"bad; header", "invalid"},
		{"private-body-marker", "other"}, {"application/private-token-marker", "other"},
	} {
		if got := responseMediaType(tt.value); got != tt.want {
			t.Errorf("want %s got %s", tt.want, got)
		}
	}
	var out bytes.Buffer
	log, files, err := logger.Setup(logger.Options{Profile: profile.Test, Console: &out})
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	trace := &Trace{}
	ObserveResponse(context.Background(), trace, Response{StatusCode: 200, ContentType: "application/private-token-marker"}, "responses")
	trace.completeLogged(502, "invalid_upstream_response", log)
	if strings.Contains(out.String(), "private-token-marker") || !strings.Contains(out.String(), `upstream_media_type=other`) {
		t.Fatal("raw header leaked or classification lost")
	}
	out.Reset()
	ObserveResponse(context.Background(), trace, Response{StatusCode: 200, ContentType: "text/event-stream", ContentTypeInferred: true}, "responses")
	trace.completeLogged(200, "", log)
	if !strings.Contains(out.String(), `upstream_media_type=missing`) || !strings.Contains(out.String(), `upstream_content_type_inferred=true`) {
		t.Fatal("inferred header provenance lost")
	}
}
