package proxy

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNativeFetchPreservesOriginAndKeyProtection(t *testing.T) {
	const key = "placeholder-native-fetch-test-key"
	server := New(Options{
		Upstream: upstreamFunc(successfulUpstream), Models: []string{"deepseek-flash"},
		Authorize: func(_ context.Context, value string) bool { return value == key },
	})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if _, err := server.Stop(ctx); err != nil {
			t.Error(err)
		}
	})
	for _, test := range []struct {
		name, origin, site, key string
		status                  int
	}{
		{"native Electron", "", "none", key, 200},
		{"CLI", "", "", key, 200},
		{"same origin", "", "same-origin", key, 200},
		{"browser origin", "https://attacker.example", "none", key, 403},
		{"opaque origin", "null", "none", key, 403},
		{"cross site", "", "cross-site", key, 403},
		{"same site", "", "same-site", key, 403},
		{"wrong key", "", "none", "placeholder-invalid-key", 401},
		{"missing key", "", "none", "", 401},
	} {
		t.Run(test.name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodGet, server.Status().Address+"/v1/models?limit=1000", nil)
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Origin", test.origin)
			request.Header.Set("Sec-Fetch-Site", test.site)
			request.Header.Set("Anthropic-Version", "2023-06-01")
			if test.key != "" {
				request.Header.Set("Authorization", "Bearer "+test.key)
			}
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil || response.StatusCode != test.status {
				t.Fatalf("status=%d, expected=%d, read error=%v", response.StatusCode, test.status, err)
			}
			if test.status == 200 && !strings.Contains(string(body), `"id":"deepseek-flash"`) {
				t.Fatal("authorized native request did not receive the model catalog")
			}
		})
	}
}
