package river

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiscoveryBoundsAndCredentialValidation(t *testing.T) {
	for _, body := range []string{`{}`, `{"data":null}`, `{"data":[]}`, `{"data":[{"id":"unsupported"}]}`, `{"data":[{"id":"bad key"}]}`, strings.Repeat("x", 131073)} {
		c := New(roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
		}))
		if _, err := c.DiscoverModels(context.Background(), "", []byte("test-key")); err == nil {
			t.Fatal("invalid catalog accepted")
		}
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" || r.Method != "GET" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("invalid discovery contract")
		}
		io.WriteString(w, `{"data":[{"id":"`+DeepSeekFlash+`"},{"id":"`+DeepSeekFlash+`"},{"id":"unverified-model"}]}`)
	}))
	defer upstream.Close()
	models, err := New(nil).DiscoverModels(context.Background(), upstream.URL+"/v1", []byte("test-key"))
	if err != nil || len(models) != 1 || models[0] != DeepSeekFlash {
		t.Fatal("discovery lost exact ID", models, err)
	}
	for _, key := range [][]byte{nil, []byte("bad\nkey"), []byte(strings.Repeat("x", 8193))} {
		if _, err := New(nil).DiscoverModels(context.Background(), "", key); err == nil {
			t.Fatal("invalid credential accepted")
		}
	}
}
