package modelcatalog

import "testing"

func TestVerifiedSubscriptionOffersOnlyLongContext(t *testing.T) {
	window := ClaudeLongContextWindow
	verified := Entry{ID: "gpt-6.1-sol", ProviderKind: "openai", Metadata: Metadata{ContextWindow: &window, ContextVerified: true}}
	unknown := Entry{ID: "gpt-6-astra", ProviderKind: "openai"}
	entries, err := ClaudeEntries([]Entry{verified, unknown})
	if err != nil || len(entries) != 2 {
		t.Fatal("verified directory invalid", err)
	}
	for _, entry := range entries {
		if entry.ID == verified.ID || entry.ID == unknown.ID+ClaudeLongContextSuffix {
			t.Fatal("confirmed and unknown contexts were confused")
		}
	}
	alias := DesktopAlias(verified.ID + ClaudeLongContextSuffix)
	if actual, found := ResolveClaudeModel([]Entry{verified, unknown}, alias); !found || actual != verified.ID {
		t.Fatal("verified desktop model lost its physical route")
	}
	for _, metadata := range []Metadata{{ContextVerified: true}, {ContextVerified: true, ContextWindow: int64Pointer(200000)}} {
		if ValidateMetadata(metadata) == nil {
			t.Fatal("invalid confirmation accepted")
		}
	}
}

func int64Pointer(value int64) *int64 { return &value }
