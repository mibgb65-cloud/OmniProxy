package deepseek

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type modelsTransport func(*http.Request) (*http.Response, error)

func (f modelsTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestOfficialCatalogValidation(t *testing.T) {
	for _, tc := range []struct {
		body  string
		valid bool
	}{{`{"data":[{"id":"deepseek-chat"},{"id":"deepseek-reasoner"}]}`, true}, {`{"data":[{"id":"invalid model"}]}`, false}, {`{"data":null}`, false}, {`{"data":[{"id":"deepseek-chat"},{"id":"deepseek-chat"}]}`, true}} {
		client := New(modelsTransport(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != DefaultBaseURL+"/models" || r.Header.Get("Authorization") != "Bearer test-only-key" {
				t.Error("wrong official request")
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
		}))
		ids, err := client.DiscoverModels(context.Background(), []byte("test-only-key"))
		if (err == nil) != tc.valid {
			t.Fatal("invalid catalog accepted")
		}
		if tc.valid && strings.Count(tc.body, "deepseek-chat") == 2 && len(ids) != 1 {
			t.Fatal("duplicate not removed")
		}
	}
}
