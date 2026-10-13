//go:build windows

package agentconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDesktopPathsKeepSharedModeWithMSIXCache(t *testing.T) {
	cache := t.TempDir()
	packageRoot := filepath.Join(cache, "Packages", "AnthropicPBC.Claude_fnn82j28hfe8t", "LocalCache")
	writeDesktopFixture(t, filepath.Join(packageRoot, "Roaming", "Claude", "claude_desktop_config.json"), `{}`)
	if err := os.MkdirAll(filepath.Join(packageRoot, "Local"), 0700); err != nil {
		t.Fatal(err)
	}
	profile, normal, thirdParty, err := desktopClientPaths(cache)
	if err != nil || normal != filepath.Join(cache, "Claude", "claude_desktop_config.json") || thirdParty != filepath.Join(cache, "Claude-3p", "claude_desktop_config.json") || profile != filepath.Join(cache, "Claude-3p", "configLibrary", DesktopProfileID+".json") {
		t.Fatal("MSIX cache replaced the shared startup configuration", err)
	}
}

func TestDesktopPathsRetainUnpackagedFallback(t *testing.T) {
	cache := t.TempDir()
	profile, normal, thirdParty, err := desktopClientPaths(cache)
	if err != nil || normal != filepath.Join(cache, "Claude", "claude_desktop_config.json") || thirdParty != filepath.Join(cache, "Claude-3p", "claude_desktop_config.json") || profile != filepath.Join(cache, "Claude-3p", "configLibrary", DesktopProfileID+".json") {
		t.Fatal("unpackaged fallback changed", err)
	}
}

func TestDesktopPathsUseExistingMSIXLocalLibrary(t *testing.T) {
	for _, family := range []string{"AnthropicPBC.Claude_fnn82j28hfe8t", "Claude_pzs8sxrjxfjjc"} {
		t.Run(family, func(t *testing.T) {
			cache := t.TempDir()
			root := filepath.Join(cache, "Packages", family, "LocalCache", "Local", "Claude-3p")
			writeDesktopFixture(t, filepath.Join(root, "configLibrary", "_meta.json"), `{"entries":[],"appliedId":""}`)
			profile, normal, thirdParty, err := desktopClientPaths(cache)
			if err != nil || profile != filepath.Join(root, "configLibrary", DesktopProfileID+".json") || thirdParty != filepath.Join(root, "claude_desktop_config.json") || normal != filepath.Join(cache, "Claude", "claude_desktop_config.json") {
				t.Fatal("existing virtual library was shadowed by shared config", err)
			}
		})
	}
}

func TestDesktopPathsRejectInvalidVirtualLibrary(t *testing.T) {
	cache := t.TempDir()
	metadata := filepath.Join(cache, "Packages", "AnthropicPBC.Claude_fnn82j28hfe8t", "LocalCache", "Local", "Claude-3p", "configLibrary", "_meta.json")
	if err := os.MkdirAll(metadata, 0700); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := desktopClientPaths(cache); err == nil {
		t.Fatal("invalid virtual library silently replaced by fallback")
	}
}
