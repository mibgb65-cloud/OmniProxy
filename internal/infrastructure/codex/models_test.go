package codex

import (
	"context"
	"errors"
	catalog "omniproxy/internal/core/modelcatalog"
	"reflect"
	"testing"
)

func TestModelsPreservesOfficialServiceTiersAndRejectsMalformedData(t *testing.T) {
	tiers := []catalog.ServiceTier{{ID: "priority", Name: "Fast", Description: "Official description"}}
	for _, test := range []struct {
		name  string
		tiers any
		valid bool
	}{
		{"confirmed", tiers, true},
		{"unknown", nil, true},
		{"empty", []catalog.ServiceTier{}, true},
		{"wrong type", "priority", false},
		{"bad field type", []map[string]any{{"id": 1, "name": "Fast"}}, false},
		{"missing id", []catalog.ServiceTier{{Name: "Fast"}}, false},
		{"too many", make([]catalog.ServiceTier, 9), false},
		{"duplicate", []catalog.ServiceTier{tiers[0], tiers[0]}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &rpcClient{pending: map[uint64]chan wireResponse{}, done: make(chan struct{})}
			item := map[string]any{"model": "gpt-test", "displayName": "GPT Test"}
			if test.tiers != nil {
				item["serviceTiers"] = test.tiers
			}
			writer := &identityWriter{client: client, result: map[string]any{"data": []any{item}, "nextCursor": nil}}
			client.input = writer
			models, err := client.Models(context.Background())
			if writer.method != "model/list" || (err == nil) != test.valid {
				t.Fatal("model service tiers contract mismatch", err)
			}
			if !test.valid {
				if !errors.Is(err, ErrProtocol) {
					t.Fatal("malformed tiers did not return protocol failure", err)
				}
				return
			}
			if len(models) != 1 {
				t.Fatal("model lost")
			}
			if test.name == "confirmed" && !reflect.DeepEqual(models[0].Metadata.ServiceTiers, tiers) {
				t.Fatal("official service tiers dropped")
			}
			if test.name != "confirmed" && len(models[0].Metadata.ServiceTiers) != 0 {
				t.Fatal("unknown service tiers invented")
			}
		})
	}
}
