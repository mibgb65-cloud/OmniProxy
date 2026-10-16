package routing

import (
	"context"
	"slices"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/provider/river"
)

func TestRiverLogicalModelsSharePriorityAndKeepLegacyRoutes(t *testing.T) {
	s, db := fixture(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at) VALUES('river','builtin_river','River','unused','2026-10-17T00:00:00Z','now')`); err != nil {
		t.Fatal(err)
	}
	order := []string{"builtin_river", "builtin_deepseek"}
	if err := s.SetProviderOrder(ctx, "deepseek-flash", order); err != nil {
		t.Fatal(err)
	}
	items, err := s.DefaultRoutes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, item := range items {
		if item.Model == river.DeepSeekFlash || item.Model == river.GLMFlash || item.Model == river.NemotronVL {
			t.Fatal("raw River ID published twice", item.Model)
		}
		if item.Model != "deepseek-flash" {
			continue
		}
		count++
		if len(item.Providers) != 2 || item.Preview.ProviderID != "builtin_river" || item.Preview.Model != river.DeepSeekFlash || len(item.ProviderOrder) != 2 {
			t.Fatal("unified priority missing", item)
		}
	}
	if count != 1 {
		t.Fatal("Flash was not merged")
	}
	for _, raw := range river.Models() {
		p, err := s.Resolve(ctx, river.PublicModel(raw))
		if err != nil || !p.Routable || p.Model != raw || p.ProviderID != "builtin_river" {
			t.Fatal("logical model did not resolve to real upstream", p, err)
		}
		legacy, err := s.Resolve(ctx, raw)
		if err != nil || !legacy.Routable || legacy.Model != raw || legacy.ProviderID != "builtin_river" {
			t.Fatal("legacy model changed provider", legacy, err)
		}
	}
	entries, err := s.Catalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	variants, err := catalog.ClaudeEntries(entries)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"deepseek-flash", "deepseek-v4-pro", "NVIDIA-Nemotron-3.5", "GLM-5.3-Flash"} {
		for _, suffix := range []string{"", "[1m]"} {
			found := false
			for _, item := range variants {
				if item.ID == id+suffix {
					found = true
					if item.ID != item.Name && id != "deepseek-flash" {
						t.Fatal("short display name lost", item)
					}
				}
			}
			if !found {
				t.Fatal("ordinary/1M choice missing", id+suffix)
			}
		}
	}
	legacy, err := s.CompatibilityCatalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range river.Models() {
		for _, old := range []string{raw + "[1m]", catalog.DesktopAlias(raw), catalog.DesktopAlias(raw + "[1m]")} {
			if actual, ok := catalog.ResolveClaudeModel(legacy, old); !ok || actual != raw {
				t.Fatal("old Claude ID no longer resolves", old, actual)
			}
		}
	}
	if _, err := db.Exec(`UPDATE accounts SET enabled=0 WHERE id='river'`); err != nil {
		t.Fatal(err)
	}
	p, err := s.Resolve(ctx, "deepseek-flash")
	if err != nil || !p.Routable || p.ProviderID != "builtin_deepseek" || p.Model != "deepseek-flash" || p.ProviderPriority != 2 {
		t.Fatal("disabled River did not select direct DeepSeek", p, err)
	}
	if _, err := db.Exec(`UPDATE accounts SET enabled=1 WHERE id='river'`); err != nil {
		t.Fatal(err)
	}
	if err := s.SetProviderOrder(ctx, "deepseek-flash", []string{"builtin_deepseek", "builtin_river"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, Input{Alias: "pinned", ProviderID: "builtin_river", Model: "deepseek-flash", AccountID: "river", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	p, err = s.Resolve(ctx, "pinned")
	if err != nil || p.ProviderID != "builtin_river" || p.Model != river.DeepSeekFlash {
		t.Fatal("fixed route escaped scope", p, err)
	}
	if err := s.SetProviderOrder(ctx, river.DeepSeekFlash, []string{"builtin_river", "builtin_deepseek"}); err != nil {
		t.Fatal(err)
	}
	prefs, err := s.preferences(ctx)
	if err != nil || !slices.Equal(prefs["deepseek-flash"], order) || prefs[river.DeepSeekFlash] != nil {
		t.Fatal("legacy priority created a duplicate", prefs, err)
	}
}
