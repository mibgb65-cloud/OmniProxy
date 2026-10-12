package codex

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type identityWriter struct {
	client  *rpcClient
	result  any
	method  string
	request []byte
}

func (w *identityWriter) Write(data []byte) (int, error) {
	var request struct {
		ID     uint64 `json:"id"`
		Method string `json:"method"`
	}
	if err := json.Unmarshal(data, &request); err != nil {
		return 0, err
	}
	w.method = request.Method
	w.request = append([]byte(nil), data...)
	result, err := json.Marshal(w.result)
	if err != nil {
		return 0, err
	}
	w.client.pending[request.ID] <- wireResponse{result: result}
	return len(data), nil
}

func (w *identityWriter) Close() error { return nil }

func TestOfficialAccountIDProtocolAndValidation(t *testing.T) {
	for _, test := range []struct {
		name  string
		id    any
		valid bool
	}{
		{"uuid", "11111111-2222-3333-4444-555555555555", true},
		{"opaque", "org_test-123", true},
		{"missing", nil, false},
		{"empty", "", false},
		{"wrong type", 123, false},
		{"too long", strings.Repeat("a", 129), false},
		{"control", "test\nprivate", false},
		{"path", "test/private", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &rpcClient{pending: map[uint64]chan wireResponse{}, done: make(chan struct{})}
			writer := &identityWriter{client: client, result: map[string]any{"accountId": test.id}}
			client.input = writer
			id, err := client.AccountID(context.Background())
			if writer.method != "account/rateLimits/read" {
				t.Fatal("unexpected protocol method")
			}
			if test.valid && (err != nil || id != test.id) || !test.valid && (err == nil || id != "") {
				t.Fatal("account ID boundary not enforced")
			}
		})
	}
}
