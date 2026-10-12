package agentconfig

import (
	"context"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
)

func TestClientDirectoryExcludesOnlyConfirmedIncompatibleModels(t *testing.T) {
	s, source := catalogFixture(t)
	noTools := false
	source.entries = []catalog.Entry{
		{ID: "unknown-capabilities"},
		{ID: "messages-only", Protocols: []string{"messages"}},
		{ID: "no-tools", Metadata: catalog.Metadata{ToolUse: &noTools}},
		{ID: "image-only", Metadata: catalog.Metadata{InputModalities: []string{"image"}}},
	}
	plan, err := s.PreviewPlan(context.Background(), Input{Client: "codex", Revision: digest(nil, false)})
	if err != nil || len(plan.Models) != 1 || plan.Models[0].ID != "unknown-capabilities" {
		t.Fatal("incorrect Codex compatibility filter", err)
	}
	plan, err = s.PreviewPlan(context.Background(), Input{Client: "claude-desktop", Revision: digest(nil, false)})
	if err != nil || len(plan.Models) != 2 {
		t.Fatal("incorrect Desktop compatibility filter", err)
	}
}
