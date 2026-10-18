package agentinstall

import "testing"

func TestGatewayAgentVersionParsing(t *testing.T) {
	for _, item := range []struct{ command, output, expected string }{
		{"kimi", "Kimi Code CLI 0.32.1", "0.32.1"},
		{"kimi", "kimi, version 1.42.0", "1.42.0"},
		{"pi", "0.71.0", "0.71.0"},
		{"pi", "pi v0.71.0", "0.71.0"},
		{"droid", "droid 0.229.0", "0.229.0"},
		{"goose", "goose version 1.41.0", "1.41.0"},
		{"goose", "goose 1.41.0", "1.41.0"},
		{"goose", "droid 1.0.0", ""},
		{"kimi", "private error 1.0.0", ""},
		{"pi", "pi 1.0.0\nprivate detail", ""},
	} {
		if got := gatewayAgentVersion(item.command, item.output); got != item.expected {
			t.Fatalf("incorrect version for %s", item.command)
		}
	}
}

func TestGatewayAgentFixedInstallationTargets(t *testing.T) {
	for _, id := range []string{"kimi-code-cli", "pi-cli", "factory-droid-cli", "goose-cli", "goose-desktop"} {
		product, err := Find(id)
		if err != nil || product.OfficialURL == "" || product.PackageID != "" {
			t.Fatal("missing fixed official entry or unverified automatic installer", id)
		}
	}
	if !codingDesktopMatches("goose-desktop", "goose", "Block, Inc.") || codingDesktopMatches("goose-desktop", "goose helper", "Block, Inc.") || codingDesktopMatches("goose-desktop", "goose", "Other") {
		t.Fatal("Goose desktop detection is not exact")
	}
}
