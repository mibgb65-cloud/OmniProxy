package river

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
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
		var data []map[string]string
		for _, id := range append(Models(), DeepSeekFlash1017, "unverified-model") {
			data = append(data, map[string]string{"id": id})
		}
		if err := json.NewEncoder(w).Encode(map[string]any{"data": data}); err != nil {
			t.Error(err)
		}
	}))
	defer upstream.Close()
	models, err := New(nil).DiscoverModels(context.Background(), upstream.URL+"/v1", []byte("test-key"))
	if err != nil || !reflect.DeepEqual(models, Models()) {
		t.Fatal("discovery lost exact ID", models, err)
	}
	for _, key := range [][]byte{nil, []byte("bad\nkey"), []byte(strings.Repeat("x", 8193))} {
		if _, err := New(nil).DiscoverModels(context.Background(), "", key); err == nil {
			t.Fatal("invalid credential accepted")
		}
	}
}
