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

func TestDesktopPathsIgnoreExistingMSIXLocalLibrary(t *testing.T) {
	for _, family := range []string{"AnthropicPBC.Claude_fnn82j28hfe8t", "Claude_pzs8sxrjxfjjc"} {
		t.Run(family, func(t *testing.T) {
			for _, sharedExists := range []bool{false, true} {
				t.Run(map[bool]string{false: "first_setup", true: "existing_shared"}[sharedExists], func(t *testing.T) {
					cache := t.TempDir()
					root := filepath.Join(cache, "Packages", family, "LocalCache", "Local", "Claude-3p")
					writeDesktopFixture(t, filepath.Join(root, "configLibrary", "_meta.json"), `{"entries":[],"appliedId":"cached"}`)
					shared := filepath.Join(cache, "Claude-3p")
					if sharedExists {
						writeDesktopFixture(t, filepath.Join(shared, "configLibrary", "_meta.json"), `{"entries":[],"appliedId":"original"}`)
					}
					profile, normal, thirdParty, err := desktopClientPaths(cache)
					if err != nil || profile != filepath.Join(shared, "configLibrary", DesktopProfileID+".json") || thirdParty != filepath.Join(shared, "claude_desktop_config.json") || normal != filepath.Join(cache, "Claude", "claude_desktop_config.json") {
						t.Fatal("stale package library replaced the client shared configuration", err)
					}
				})
			}
		})
	}
}

func TestDesktopPathsIgnoreInvalidVirtualLibrary(t *testing.T) {
	cache := t.TempDir()
	metadata := filepath.Join(cache, "Packages", "AnthropicPBC.Claude_fnn82j28hfe8t", "LocalCache", "Local", "Claude-3p", "configLibrary", "_meta.json")
	if err := os.MkdirAll(metadata, 0700); err != nil {
		t.Fatal(err)
	}
	profile, normal, thirdParty, err := desktopClientPaths(cache)
	if err != nil || profile != filepath.Join(cache, "Claude-3p", "configLibrary", DesktopProfileID+".json") || normal != filepath.Join(cache, "Claude", "claude_desktop_config.json") || thirdParty != filepath.Join(cache, "Claude-3p", "claude_desktop_config.json") {
		t.Fatal("unused invalid package cache blocked shared configuration", err)
	}
}
