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
