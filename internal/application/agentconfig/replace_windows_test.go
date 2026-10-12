//go:build windows

package agentconfig

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestExclusiveConfigWritePreservesVersionChecks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	original := []byte(`{"theme":"dark","other":"retained"}`)
	if err := writeEncryptedConfig(path, original, digest(nil, false)); err != nil {
		t.Fatal(err)
	}
	updated := []byte(`{"theme":"light"}`)
	if err := writeEncryptedConfig(path, updated, digest(original, true)); err != nil {
		t.Fatal(err)
	}
	if err := writeEncryptedConfig(path, []byte(`{}`), digest(original, true)); err == nil || err.Error() != "agent: config_changed" {
		t.Fatal("stale revision accepted", err)
	}
	current, err := os.ReadFile(path)
	if err != nil || string(current) != string(updated) {
		t.Fatal("current file modified or old tail retained")
	}
}

func TestExclusiveConfigWriteRejectsLockedFileAndCleansStaleCreation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := writeEncryptedConfig(path, []byte(`{}`), digest([]byte(`old`), true)); err == nil {
		t.Fatal("stale creation accepted")
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("failed creation left a file", err)
	}
	original := []byte(`{"retained":true}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(handle)
	if err := writeEncryptedConfig(path, []byte(`{}`), digest(original, true)); err == nil || err.Error() != "agent: config_locked" {
		t.Fatal("locked file not reported", err)
	}
	current, err := os.ReadFile(path)
	if err != nil || string(current) != string(original) {
		t.Fatal("locked file modified")
	}
}

func TestExclusiveConfigWriteRejectsUnsafeInputs(t *testing.T) {
	root := t.TempDir()
	if err := writeEncryptedConfig(root, []byte(`{}`), digest(nil, false)); err == nil || err.Error() != "agent: unsafe_config" {
		t.Fatal("directory accepted as configuration", err)
	}
	path := filepath.Join(root, "config.json")
	if err := writeEncryptedConfig(path, []byte(strings.Repeat("x", maxConfigBytes+1)), digest(nil, false)); err == nil || err.Error() != "agent: unsafe_config" {
		t.Fatal("oversized configuration accepted", err)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("unsafe configuration created a file", err)
	}
}

func TestClaudeDesktopEncryptedDirectoryApplyAndRestore(t *testing.T) {
	parent := os.Getenv("OMNIPROXY_AGENTCONFIG_TEST_ENCRYPTED_PARENT")
	if parent == "" {
		t.Skip("requires an explicit encrypted parent for synthetic desktop configuration")
	}
	name, err := windows.UTF16PtrFromString(parent)
	if err != nil || !filepath.IsAbs(parent) {
		t.Fatal("invalid encrypted fixture parent")
	}
	attributes, err := windows.GetFileAttributes(name)
	if err != nil || attributes&windows.FILE_ATTRIBUTE_ENCRYPTED == 0 {
		t.Fatal("fixture parent is not encrypted")
	}
	root, err := os.MkdirTemp(parent, ".omniproxy-validation-")
	if err != nil {
		t.Fatal(err)
	}
	s := fixture(t)
	s.paths.ClaudeDesktop = filepath.Join(root, "Claude-3p", "configLibrary", DesktopProfileID+".json")
	t.Cleanup(func() {
		for _, path := range []string{
			s.paths.ClaudeDesktop,
			filepath.Join(root, "Claude-3p", "configLibrary", "_meta.json"),
			filepath.Join(root, "Claude", "claude_desktop_config.json"),
			filepath.Join(root, "Claude-3p", "claude_desktop_config.json"),
			filepath.Join(root, "Claude-3p", "configLibrary"),
			filepath.Join(root, "Claude-3p"), filepath.Join(root, "Claude"), root,
		} {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				t.Error("synthetic fixture cleanup failed")
			}
		}
	})
	ctx := context.Background()
	if _, err := s.Apply(ctx, Input{Client: "claude-desktop", Revision: digest(nil, false)}); err != nil {
		t.Fatal("first encrypted apply", err)
	}
	body, err := os.ReadFile(s.paths.ClaudeDesktop)
	if err != nil {
		t.Fatal(err)
	}
	defer clear(body)
	if _, err := s.Apply(ctx, Input{Client: "claude-desktop", Revision: digest(body, true)}); err != nil {
		t.Fatal("repeated encrypted apply", err)
	}
	path, err := windows.UTF16PtrFromString(s.paths.ClaudeDesktop)
	if err != nil {
		t.Fatal(err)
	}
	attributes, err = windows.GetFileAttributes(path)
	if err != nil || attributes&windows.FILE_ATTRIBUTE_ENCRYPTED == 0 {
		t.Fatal("configuration encryption lost")
	}
	if _, err := s.Restore(ctx, "claude-desktop"); err != nil {
		t.Fatal("encrypted restore", err)
	}
}
