package agentinstall

import (
	"context"
	"errors"
	"io"
	"net/http"
	"runtime"
	"strings"
	"testing"
)

type updateTransport func(*http.Request) (*http.Response, error)

func (f updateTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type unreadUpdateBody struct{ t *testing.T }

func (b unreadUpdateBody) Read([]byte) (int, error) {
	b.t.Error("release response body must not be read")
	return 0, io.EOF
}
func (b unreadUpdateBody) Close() error { return nil }

func TestUpdateCheckValidatesReleaseAndComparesNumericVersions(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows desktop release")
	}
	arch := map[string]string{"amd64": "x64", "arm64": "arm64"}[runtime.GOARCH]
	if arch == "" {
		t.Skip("unsupported architecture")
	}
	prefix := "https://downloads.claude.ai/releases/win32/" + arch + "/"
	file := "/Claude-" + strings.Repeat("a", 40) + ".msix"
	for _, tc := range []struct{ name, current, location, state, latest string }{
		{"new", "2.37337.1.0", prefix + "2.42304.0" + file, "available", "2.42304.0"},
		{"equal", "2.42304.0.0", prefix + "2.42304.0" + file, "current", "2.42304.0"},
		{"newer-installed", "2.42305.0.0", prefix + "2.42304.0" + file, "current", "2.42304.0"},
		{"numeric-order", "2.9.0.0", prefix + "2.10.0" + file, "available", "2.10.0"},
		{"patch", "2.10.0.0", prefix + "2.10.0.1" + file, "available", "2.10.0.1"},
		{"overflow", "2.9.0.0", prefix + "2.65536.0" + file, "unknown", ""},
		{"beta", "2.9.0.0", prefix + "2.10.0-beta" + file, "unknown", ""},
		{"wrong-host", "2.9.0.0", "https://downloads.claude.ai.evil.invalid/releases/win32/" + arch + "/2.10.0" + file, "unknown", ""},
		{"insecure", "2.9.0.0", strings.Replace(prefix, "https:", "http:", 1) + "2.10.0" + file, "unknown", ""},
		{"credentials", "2.9.0.0", strings.Replace(prefix, "https://", "https://user@", 1) + "2.10.0" + file, "unknown", ""},
		{"query", "2.9.0.0", prefix + "2.10.0" + file + "?token=private", "unknown", ""},
		{"encoded-path", "2.9.0.0", prefix + "2%2E10.0" + file, "unknown", ""},
		{"other-arch", "2.9.0.0", strings.Replace(prefix, arch+"/", "other/", 1) + "2.10.0" + file, "unknown", ""},
		{"relative", "2.9.0.0", "/private", "unknown", ""},
		{"oversize", "2.9.0.0", strings.Repeat("a", 1025), "unknown", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			b := &Backend{updateHTTP: &http.Client{Transport: updateTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != http.MethodGet || r.URL.String() != "https://claude.ai/api/desktop/win32/"+arch+"/msix/latest/redirect" {
					t.Fatal("unexpected request or package download")
				}
				return &http.Response{StatusCode: 307, Header: http.Header{"Location": []string{tc.location}}, Body: unreadUpdateBody{t}}, nil
			})}}
			result, err := b.CheckUpdate(context.Background(), "claude-desktop", tc.current)
			if err != nil || result.State != tc.state || result.LatestVersion != tc.latest || calls != 1 || result.CurrentVersion != tc.current {
				t.Fatalf("unexpected update: %+v, %v, calls %d", result, err, calls)
			}
		})
	}
}

func TestUpdateCheckRejectsInputsAndSanitizesFailures(t *testing.T) {
	b := &Backend{updateHTTP: &http.Client{Transport: updateTransport(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid or unsupported input reached network")
		return nil, nil
	})}}
	for _, id := range []string{"claude-desktop;bad", "https://private.invalid", strings.Repeat("x", 33), "claude-code"} {
		if _, err := b.CheckUpdate(context.Background(), id, "2.9.0.0"); err == nil {
			t.Fatal("invalid product accepted")
		}
	}
	if runtime.GOOS != "windows" {
		return
	}
	for _, current := range []string{"", "Unknown", "2.9.0-beta", "2.65536.0", "2.9", "2.9.0\nsecret"} {
		result, err := b.CheckUpdate(context.Background(), "claude-desktop", current)
		if err != nil || result.State != "unknown" || result.CurrentVersion != "" {
			t.Fatal("invalid version reported as current", result, err)
		}
	}
	for _, status := range []int{0, 200, 403, 500} {
		b.updateHTTP.Transport = updateTransport(func(r *http.Request) (*http.Response, error) {
			if status == 0 {
				return nil, errors.New("private transport detail")
			}
			return &http.Response{StatusCode: status, Header: http.Header{}, Body: unreadUpdateBody{t}}, nil
		})
		if _, err := b.CheckUpdate(context.Background(), "claude-desktop", "2.9.0.0"); !errors.Is(err, ErrUpdateFailed) {
			t.Fatal("failure not sanitized", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	b.updateHTTP.Transport = updateTransport(func(r *http.Request) (*http.Response, error) {
		return nil, r.Context().Err()
	})
	if _, err := b.CheckUpdate(ctx, "claude-desktop", "2.9.0.0"); !errors.Is(err, ErrUpdateFailed) {
		t.Fatal("cancellation not sanitized", err)
	}
}
