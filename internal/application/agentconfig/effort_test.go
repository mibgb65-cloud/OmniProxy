package agentconfig

import (
	"context"
	"database/sql"
	"os"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
)

// TestClaudeEffortSelectionPersistsAndValidates 固定模型固定档位的写入、读回、清除与校验。
func TestClaudeEffortSelectionPersistsAndValidates(t *testing.T) {
	s, source := catalogFixture(t)
	source.entries = []catalog.Entry{
		{ID: "gpt-default", Metadata: catalog.Metadata{IsDefault: true}},
		{
			ID:           "river-effort",
			Name:         "River Effort",
			ProviderKind: "river",
			Metadata: catalog.Metadata{
				ReasoningEfforts: []catalog.Effort{{Value: "none"}, {Value: "high"}, {Value: "max"}},
				DefaultEffort:    "high",
			},
		},
	}
	ctx := context.Background()
	if _, err := s.Apply(ctx, Input{
		Client:   "claude",
		Revision: digest(nil, false),
		Model:    "river-effort",
		Efforts:  map[string]string{"river-effort": "max"},
	}); err != nil {
		t.Fatal(err)
	}
	pinned := func() (string, error) {
		var effort string
		err := s.db.QueryRow(`SELECT effort FROM model_effort_pins WHERE model_id='river-effort'`).Scan(&effort)
		return effort, err
	}
	if effort, err := pinned(); err != nil || effort != "max" {
		t.Fatalf("pin not saved: %q %v", effort, err)
	}
	applied, err := os.ReadFile(s.paths.Claude)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := s.PreviewPlan(ctx, Input{Client: "claude", Revision: digest(applied, true)})
	if err != nil || plan.Efforts["river-effort"] != "max" {
		t.Fatalf("saved pin not reported to the interface: %+v %v", plan.Efforts, err)
	}

	// 未确认的档位被拒绝，且不会留下任何写入。
	if _, err := s.PreviewPlan(ctx, Input{
		Client:   "claude",
		Revision: digest(applied, true),
		Efforts:  map[string]string{"river-effort": "xhigh"},
	}); err == nil {
		t.Fatal("unconfirmed effort accepted")
	}

	// 空档位回到跟随客户端，并删除固定值。
	if _, err := s.Apply(ctx, Input{
		Client:   "claude",
		Revision: digest(applied, true),
		Model:    "river-effort",
		Efforts:  map[string]string{"river-effort": ""},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := pinned(); err != sql.ErrNoRows {
		t.Fatal("cleared pin remained stored")
	}
}

func TestEffortSelectionRejectsUnknownAndUnsupportedModels(t *testing.T) {
	entries := []catalog.Entry{
		{ID: "plain-model"},
		{ID: "river-effort", Metadata: catalog.Metadata{ReasoningEfforts: []catalog.Effort{{Value: "high"}}}},
	}
	if err := validateEfforts(entries, map[string]string{"missing-model": "high"}); err == nil {
		t.Fatal("unknown model accepted")
	}
	if err := validateEfforts(entries, map[string]string{"plain-model": "high"}); err == nil {
		t.Fatal("model without confirmed levels accepted")
	}
	if err := validateEfforts(entries, map[string]string{"river-effort": "high"}); err != nil {
		t.Fatal(err)
	}
}
