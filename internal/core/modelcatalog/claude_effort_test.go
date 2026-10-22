package modelcatalog

import (
	"strings"
	"testing"
)

func TestClaudeExplicitEffortsPreserveRealIdentityAndLongContext(t *testing.T) {
	window := int64(1048576)
	source := Entry{ID: "DeepSeek-V4.1-Flash-1017", Name: "DeepSeek-V4.1-Flash-1017", ProviderKind: "river",
		Metadata: Metadata{ContextWindow: &window, ReasoningEfforts: []Effort{{Value: "none"}, {Value: "high"}, {Value: "max"}}, DefaultEffort: "high", IsDefault: true}}
	entries, err := ClaudeEntries([]Entry{source})
	if err != nil || len(entries) != 8 {
		t.Fatal("missing plain, 1m or effort choices", len(entries), err)
	}
	count := 0
	for _, e := range entries {
		if e.Metadata.IsDefault {
			count++
		}
		if !strings.HasPrefix(e.ID, ClaudeEffortPrefix) {
			continue
		}
		for _, id := range []string{e.ID, DesktopAlias(e.ID), strings.TrimSuffix(DesktopAlias(e.ID), ClaudeLongContextSuffix)} {
			model, effort, found := ResolveClaudeEffort([]Entry{source}, id)
			if !found || model != source.ID || effort == "" {
				t.Fatal("virtual choice did not resolve", id, model, effort)
			}
		}
	}
	if count != 1 {
		t.Fatal("effort choice unexpectedly changed default", count)
	}
	again, err := ClaudeEntries(entries)
	if err != nil || len(again) != len(entries) {
		t.Fatal("effort choices are not idempotent", len(again), err)
	}
	if _, _, ok := ResolveClaudeEffort(nil, claudeEffortID(source.ID, "high")); ok {
		t.Fatal("removed target stayed routable")
	}
}

func TestClaudeEffortDoesNotInventCapabilitiesOrShadowExistingModels(t *testing.T) {
	source := Entry{ID: "route", ProviderKind: "river", Metadata: Metadata{ReasoningEfforts: []Effort{{Value: "high"}}}}
	id := claudeEffortID(source.ID, "high")
	for _, entries := range [][]Entry{
		{{ID: "unknown", ProviderKind: "river"}},
		{{ID: "unimplemented", ProviderKind: "other", Metadata: source.Metadata}},
		{source, {ID: id, ProviderKind: "other"}},
	} {
		got, err := ClaudeEntries(entries)
		if err != nil || len(got) != len(entries) {
			t.Fatal("invented effort choice or shadowed real ID", got, err)
		}
		if _, _, ok := ResolveClaudeEffort(entries, id); ok {
			t.Fatal("unknown or colliding effort choice accepted")
		}
	}
	if _, _, ok := ResolveClaudeEffort([]Entry{source}, id+ClaudeLongContextSuffix); ok {
		t.Fatal("invented 1m support")
	}
}
