package codex

import (
	"context"
	"testing"
)

func TestQuotaProtocolRejectsMalformedWindowData(t *testing.T) {
	for _, test := range []struct {
		name   string
		window map[string]any
		valid  bool
	}{
		{"zero usage", map[string]any{"usedPercent": 0, "windowDurationMins": 300, "resetsAt": nil}, true},
		{"over limit", map[string]any{"usedPercent": 120, "windowDurationMins": 10080}, true},
		{"missing usage", map[string]any{"windowDurationMins": 300}, false},
		{"negative usage", map[string]any{"usedPercent": -1}, false},
		{"fractional usage", map[string]any{"usedPercent": 0.5}, false},
		{"invalid duration", map[string]any{"usedPercent": 0, "windowDurationMins": 0}, false},
		{"invalid reset", map[string]any{"usedPercent": 0, "resetsAt": -1}, false},
		{"oversized reset", map[string]any{"usedPercent": 0, "resetsAt": 253402300800}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &rpcClient{pending: map[uint64]chan wireResponse{}, done: make(chan struct{})}
			writer := &identityWriter{client: client, result: map[string]any{"rateLimits": map[string]any{"primary": test.window}}}
			client.input = writer
			_, err := client.RateLimits(context.Background())
			if (err == nil) != test.valid || writer.method != "account/rateLimits/read" {
				t.Fatal("invalid quota response accepted or valid response rejected")
			}
		})
	}
}
