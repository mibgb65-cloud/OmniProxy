package token

import (
	"testing"
)

func TestNormalizeRequiresAtriaKeyPrefix(t *testing.T) {
	tests := []struct {
		name       string
		tokenValue string
		wantErr    bool
	}{
		{name: "valid atria key", tokenValue: "atr_1234567890abcdef", wantErr: false},
		{name: "missing prefix", tokenValue: "1234567890abcdef1234", wantErr: true},
		{name: "wrong prefix", tokenValue: "sk_1234567890abcdef", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, _, _, _, err := normalizeRequest(UpsertRequest{
				Name:       "atria",
				Provider:   ProviderAtria,
				TokenValue: tt.tokenValue,
			})
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for token value %q, got nil", tt.tokenValue)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for token value %q: %v", tt.tokenValue, err)
			}
		})
	}
}
