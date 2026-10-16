//go:build windows

package agentinstall

import (
	"context"
	"testing"
)

func TestLaunchRejectsUntrustedTargets(t *testing.T) {
	for _, id := range []string{"", "opencode", "../opencode-cli", "harness-cli;bad", "codex-cli", "https://example.invalid", "harness-cli --task bad"} {
		if Launch(context.Background(), id) == nil {
			t.Fatal("untrusted launch target accepted")
		}
	}
}
