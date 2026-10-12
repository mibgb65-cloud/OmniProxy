package agentinstall

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDesktopPackageDetection(t *testing.T) {
	for _, test := range []struct {
		name, product, packages, state, version string
	}{
		{
			name: "claude_store_current", product: "claude-desktop",
			packages: `[{"Name":"AnthropicPBC.Claude","Version":"2.31226.1.0","InstallLocation":"C:\\Program Files\\WindowsApps\\AnthropicPBC.Claude","Status":"Ok"}]`,
			state:    "installed", version: "2.31226.1.0",
		},
		{
			name: "claude_store_legacy", product: "claude-desktop",
			packages: `[{"Name":"Claude","Version":"1.2.3.0","Status":"Ok"}]`,
			state:    "installed", version: "1.2.3.0",
		},
		{
			name: "healthy_package_after_legacy_remnant", product: "claude-desktop",
			packages: `[{"Name":"Claude","Version":"1.2.3.0","Status":"Modified"},{"Name":"AnthropicPBC.Claude","Version":"2.31226.1.0","Status":"Ok"}]`,
			state:    "installed", version: "2.31226.1.0",
		},
		{
			name: "codex_store", product: "codex-desktop",
			packages: `[{"Name":"OpenAI.Codex","Version":"26.1007.2314.0","Status":"Ok"}]`,
			state:    "installed", version: "26.1007.2314.0",
		},
		{name: "missing", product: "claude-desktop", packages: `[]`, state: "not_installed"},
		{name: "invalid_json", product: "claude-desktop", packages: `[`, state: "unknown"},
		{name: "unrelated_package", product: "claude-desktop", packages: `[{"Name":"Unrelated.Claude","Version":"1.2.3.0","Status":"Ok"}]`, state: "unknown"},
		{name: "wrong_product", product: "codex-desktop", packages: `[{"Name":"AnthropicPBC.Claude","Version":"1.2.3.0","Status":"Ok"}]`, state: "unknown"},
		{name: "damaged_package", product: "claude-desktop", packages: `[{"Name":"AnthropicPBC.Claude","Version":"1.2.3.0","Status":"Modified"}]`, state: "unknown"},
		{name: "invalid_version", product: "claude-desktop", packages: `[{"Name":"AnthropicPBC.Claude","Version":"private raw output","Status":"Ok"}]`, state: "unknown"},
	} {
		t.Run(test.name, func(t *testing.T) {
			product, err := Find(test.product)
			if err != nil {
				t.Fatal(err)
			}
			detected := inspectDesktopPackages([]byte(test.packages), product.AppxNames)
			if detected.State != test.state || detected.Version != test.version {
				t.Fatalf("unexpected detection: %+v", detected)
			}
			if test.state == "unknown" && detected.ErrorCode != "detection_failed" {
				t.Fatal("detection error was not reported safely")
			}
		})
	}
}

func TestDesktopPackageMetadataIsBounded(t *testing.T) {
	product, err := Find("claude-desktop")
	if err != nil {
		t.Fatal(err)
	}
	for _, packages := range []any{
		[]map[string]string{{"Name": "AnthropicPBC.Claude", "Version": "2.1.0.0", "Status": "Ok", "InstallLocation": strings.Repeat("x", 4097)}},
		make([]map[string]string, 9),
	} {
		output, err := json.Marshal(packages)
		if err != nil {
			t.Fatal(err)
		}
		if detected := inspectDesktopPackages(output, product.AppxNames); detected.State != "unknown" || detected.ErrorCode != "detection_failed" {
			t.Fatalf("invalid metadata accepted: %+v", detected)
		}
	}
}
