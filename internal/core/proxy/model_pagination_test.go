package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestModelPaginationStopsOnMissingCursor(t *testing.T) {
	server := New(Options{Models: []string{"a-model", "b-model"}})
	for _, test := range []struct {
		query string
		count int
	}{
		{"limit=1&after_id=a-model", 1},
		{"limit=1&after_id=missing-model", 0},
		{"before_id=missing-model", 0},
	} {
		request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:0/v1/models?"+test.query, nil)
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		var body struct {
			Data    []listedModel `json:"data"`
			HasMore bool          `json:"has_more"`
		}
		if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &body) != nil || len(body.Data) != test.count || body.HasMore {
			t.Fatal("pagination repeated missing cursor", test.query)
		}
	}
}
