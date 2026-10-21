package codex

import (
	"testing"

	"filippo.io/age"
)

func TestAuthScryptCostTrackingDoesNotRelaxVerification(t *testing.T) {
	for _, test := range []struct {
		factor string
		costly bool
	}{{"10", false}, {"18", true}, {"19", true}, {"20", true}, {"21", false}, {"invalid", false}} {
		t.Run(test.factor, func(t *testing.T) {
			identity, err := age.NewScryptIdentity("synthetic-passphrase")
			if err != nil {
				t.Fatal(err)
			}
			identity.SetMaxWorkFactor(20)
			tracked := &authScryptIdentity{ScryptIdentity: identity}
			// 非法盐保证不执行高内存 KDF，但仍经过官方 stanza 校验。
			_, err = tracked.Unwrap([]*age.Stanza{{Type: "scrypt", Args: []string{"invalid-salt", test.factor}}})
			if err == nil || tracked.highCost != test.costly {
				t.Fatal("cost tracking weakened age verification or missed the KDF bound")
			}
		})
	}
}
