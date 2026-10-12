package routing

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExactMatchAndModelBoundaries(t *testing.T) {
	rules := []Rule{{ID: "one", Alias: "coding-fast", Model: "deepseek-flash", Enabled: true}, {ID: "off", Alias: "disabled", Enabled: false}}
	if rule, ok := Match("coding-fast", rules); !ok || rule.ID != "one" {
		t.Fatal("exact rule not matched")
	}
	for _, model := range []string{"Coding-fast", "coding-fast-more", "disabled"} {
		if _, ok := Match(model, rules); ok {
			t.Errorf("unexpected match: %s", model)
		}
	}
	for _, model := range []string{"", "bad name", "中文", strings.Repeat("x", 129), "bad\nname"} {
		if ValidModel(model) {
			t.Errorf("accepted invalid model: %q", model)
		}
	}
	if !ValidModel(strings.Repeat("x", 128)) {
		t.Fatal("valid boundary rejected")
	}
}

func TestRewriteOnlyTopModel(t *testing.T) {
	body := []byte(`{"model":"coding-fast","messages":[{"role":"user","content":"keep model"}],"stream":true,"extra":{"model":"nested"}}`)
	rewritten, err := RewriteModel(body, "deepseek-flash")
	if err != nil {
		t.Fatal(err)
	}
	var value struct {
		Model    string
		Stream   bool
		Extra    struct{ Model string }
		Messages []struct{ Content string }
	}
	if json.Unmarshal(rewritten, &value) != nil || value.Model != "deepseek-flash" || !value.Stream || value.Extra.Model != "nested" || value.Messages[0].Content != "keep model" {
		t.Fatal("non-target fields changed")
	}
	if string(body) != `{"model":"coding-fast","messages":[{"role":"user","content":"keep model"}],"stream":true,"extra":{"model":"nested"}}` {
		t.Fatal("input mutated")
	}
}

func TestAccountPoolAndPinnedScope(t *testing.T) {
	accounts := []Account{{ID: "old", ProviderID: "ds", Enabled: true, CreatedAt: "2026-01-01T00:00:00Z"}, {ID: "new", ProviderID: "ds", Enabled: true, CreatedAt: "2026-01-01T00:00:00.1Z"}, {ID: "other", ProviderID: "other", Enabled: true, CreatedAt: "2025-01-01T00:00:00Z"}, {ID: "off", ProviderID: "ds", Enabled: false, CreatedAt: "2025-01-01T00:00:00Z"}}
	allowed := map[string]bool{"ds": true}
	if selected, count, code := SelectAccount(accounts, allowed, ""); selected.ID != "old" || count != 2 || code != "" {
		t.Fatal("pool order or scope incorrect")
	}
	if selected, count, code := SelectAccount(accounts, allowed, "new"); selected.ID != "new" || count != 1 || code != "" {
		t.Fatal("fixed account not selected")
	}
	for id, code := range map[string]string{"other": "account_mismatch", "off": "account_disabled", "missing": "account_missing"} {
		if _, _, actual := SelectAccount(accounts, allowed, id); actual != code {
			t.Errorf("%s: %s", id, actual)
		}
	}
}

func TestPreferredProviderPrecedesAccountAgeAndRetainsFixedScope(t *testing.T) {
	accounts := []Account{
		{ID: "old", ProviderID: "first", Enabled: true, CreatedAt: "2026-01-01T00:00:00Z"},
		{ID: "new", ProviderID: "preferred", Enabled: true, CreatedAt: "2026-02-01T00:00:00Z"},
	}
	allowed := map[string]bool{"first": true, "preferred": true}
	if selected, count, code := SelectPreferredAccount(accounts, allowed, "", "preferred"); selected.ID != "new" || count != 2 || code != "" {
		t.Fatal("preferred provider was ignored")
	}
	if selected, _, code := SelectPreferredAccount(accounts, allowed, "old", "preferred"); selected.ID != "old" || code != "" {
		t.Fatal("preference escaped fixed scope")
	}
	if selected, _, code := SelectPreferredAccount(accounts, allowed, "", "missing"); selected.ID != "old" || code != "" {
		t.Fatal("unavailable preference changed automatic order")
	}
}
