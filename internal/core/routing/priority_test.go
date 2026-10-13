package routing

import "testing"

func TestProviderOrderPrecedesAccountAgeAndKeepsFixedScope(t *testing.T) {
	accounts := []Account{
		{ID: "old", ProviderID: "automatic", Enabled: true, CreatedAt: "2025-01-01T00:00:00Z"},
		{ID: "second", ProviderID: "second", Enabled: true, CreatedAt: "2026-01-01T00:00:00Z"},
		{ID: "first-new", ProviderID: "first", Enabled: true, CreatedAt: "2026-03-01T00:00:00Z"},
		{ID: "first-old", ProviderID: "first", Enabled: true, CreatedAt: "2026-02-01T00:00:00Z"},
	}
	for _, test := range []struct {
		name     string
		allowed  map[string]bool
		order    []string
		fixed    string
		expected string
	}{
		{"first priority and oldest account", map[string]bool{"automatic": true, "second": true, "first": true}, []string{"first", "second"}, "", "first-old"},
		{"skip unavailable first", map[string]bool{"automatic": true, "second": true}, []string{"first", "second"}, "", "second"},
		{"unranked fallback", map[string]bool{"automatic": true}, []string{"first", "second"}, "", "old"},
		{"restore automatic", map[string]bool{"automatic": true, "second": true, "first": true}, nil, "", "old"},
		{"fixed account wins", map[string]bool{"automatic": true, "second": true, "first": true}, []string{"first", "second"}, "second", "second"},
	} {
		t.Run(test.name, func(t *testing.T) {
			selected, _, code := SelectPrioritizedAccount(accounts, test.allowed, test.fixed, test.order)
			if code != "" || selected.ID != test.expected {
				t.Fatalf("selected %s, want %s; %s", selected.ID, test.expected, code)
			}
		})
	}
	accounts[2].Enabled, accounts[3].Enabled = false, false
	selected, count, code := SelectPrioritizedAccount(accounts, map[string]bool{"automatic": true, "second": true, "first": true}, "", []string{"first", "second"})
	if code != "" || count != 2 || selected.ID != "second" {
		t.Fatal("disabled accounts did not use second priority")
	}
}
