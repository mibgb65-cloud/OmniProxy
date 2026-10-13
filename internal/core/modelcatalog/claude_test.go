package modelcatalog

import (
	"strings"
	"testing"
)

func TestClaudeVariantsAndDesktopSuffixShareBaseRoute(t *testing.T) {
	window := ClaudeLongContextWindow
	entries := []Entry{
		{ID: "deepseek-flash", Name: strings.Repeat("F", 200), Metadata: Metadata{ContextWindow: &window, IsDefault: true}},
		{ID: "coding-route", Alias: true, TargetModel: "deepseek-flash", Metadata: Metadata{ContextWindow: &window}},
		{ID: "unknown"},
		{ID: strings.Repeat("x", 125), Metadata: Metadata{ContextWindow: &window}},
	}
	variants, err := ClaudeEntries(entries)
	if err != nil || len(variants) != 6 || variants[0].ID != "deepseek-flash" {
		t.Fatal("incorrect variants or changed default", err)
	}
	for _, entry := range variants {
		if strings.HasSuffix(entry.ID, ClaudeLongContextSuffix) && (entry.Metadata.IsDefault || !strings.HasSuffix(entry.Name, ClaudeLongContextSuffix)) {
			t.Fatal("variant default or label is incorrect")
		}
	}
	for _, id := range []string{"deepseek-flash", "coding-route"} {
		alias := DesktopAlias(id)
		longAlias := DesktopAlias(id + ClaudeLongContextSuffix)
		longBase := strings.TrimSuffix(longAlias, ClaudeLongContextSuffix)
		if longBase == alias {
			t.Fatal("desktop normalization would collapse the two context choices")
		}
		for _, model := range []string{id + ClaudeLongContextSuffix, alias, alias + ClaudeLongContextSuffix, longAlias, longBase} {
			actual, found := ResolveClaudeModel(entries, model)
			if !found || actual != id {
				t.Fatal("variant did not resolve through original route", model)
			}
		}
		if actual, found := ResolveDesktopAlias(entries, longAlias); !found || actual != id {
			t.Fatal("long context desktop alias not resolved")
		}
	}
	for _, model := range []string{"unknown[1m]", "missing[1m]", DesktopAlias("unknown[1m]"), "deepseek-flash[1m][1m]"} {
		if _, found := ResolveClaudeModel(entries, model); found {
			t.Fatal("unsupported or malformed context option resolved", model)
		}
	}
	if _, found := ResolveClaudeModel(nil, DesktopAlias("deepseek-flash[1m]")); found {
		t.Fatal("removed target remained available")
	}
	again, err := ClaudeEntries(variants)
	if err != nil || len(again) != len(variants) {
		t.Fatal("variants duplicated on refresh", err)
	}
}

func TestClaudeModelExactIDPrecedesGeneratedVariants(t *testing.T) {
	window := ClaudeLongContextWindow
	entries := []Entry{{ID: "coding-route", Metadata: Metadata{ContextWindow: &window}}, {ID: "coding-route[1m]", TargetModel: "other"}, {ID: DesktopAlias("coding-route"), TargetModel: "exact"}}
	for _, entry := range entries {
		actual, found := ResolveClaudeModel(entries, entry.ID)
		if !found || actual != entry.ID {
			t.Fatal("existing exact ID was reinterpreted")
		}
	}
	variants, err := ClaudeEntries(entries)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range variants {
		if entry.ID == "coding-route[1m]" && entry.TargetModel != "other" {
			t.Fatal("existing variant ID was overwritten")
		}
	}
}
