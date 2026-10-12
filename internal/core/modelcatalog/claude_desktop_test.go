package modelcatalog

import (
	"strings"
	"testing"

	"omniproxy/internal/core/routing"
)

func TestDesktopAliasesResolveOnlyCurrentCatalog(t *testing.T) {
	entries := []Entry{{ID: "deepseek-flash"}, {ID: "gpt-6-sol"}, {ID: "coding-route", Alias: true, TargetModel: "deepseek-v4-pro"}}
	seen := map[string]bool{}
	for _, entry := range entries {
		alias := DesktopAlias(entry.ID)
		if !routing.ValidModel(alias) || !strings.HasPrefix(alias, "claude-") || seen[alias] {
			t.Fatal("desktop route is invalid or ambiguous")
		}
		seen[alias] = true
		if len(alias) != len(DesktopAlias(entries[0].ID)) || strings.Contains(alias, "deepseek") || strings.Contains(alias, "gpt") {
			t.Fatal("desktop route changes default selection or contains a rejected vendor name")
		}
		actual, found := ResolveDesktopAlias(entries, alias)
		if !found || actual != entry.ID {
			t.Fatal("route did not resolve to the original model or user alias")
		}
		if _, found := ResolveDesktopAlias(nil, alias); found {
			t.Fatal("removed catalog entries remained usable")
		}
	}
	if _, found := ResolveDesktopAlias(entries, DesktopAlias("missing-model")); found {
		t.Fatal("unknown route fell back to another model")
	}
}
