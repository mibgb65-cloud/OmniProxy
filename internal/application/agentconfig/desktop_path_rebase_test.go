package agentconfig

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDesktopPathCorrectionPreservesCacheAndRestoresSharedBaseline(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	cache := t.TempDir()
	sharedProfile, normal, thirdParty, err := desktopClientPaths(cache)
	if err != nil {
		t.Fatal(err)
	}
	packageRoot := filepath.Join(cache, "Packages", "AnthropicPBC.Claude_fnn82j28hfe8t", "LocalCache", "Local", "Claude-3p")
	cachedProfile := filepath.Join(packageRoot, "configLibrary", DesktopProfileID+".json")
	s.paths.ClaudeDesktop = cachedProfile
	s.paths.ClaudeDesktopNormal = normal
	s.paths.ClaudeDesktopThirdParty = filepath.Join(packageRoot, "claude_desktop_config.json")
	if _, err := s.Apply(ctx, Input{Client: "claude-desktop", Revision: digest(nil, false)}); err != nil {
		t.Fatal(err)
	}
	cached, err := os.ReadFile(cachedProfile)
	if err != nil {
		t.Fatal(err)
	}
	defer clear(cached)
	// 修复后必须以真实共享配置为恢复基准，不能把旧包缓存备份恢复到用户的生效位置。
	s.paths.ClaudeDesktop = sharedProfile
	s.paths.ClaudeDesktopThirdParty = thirdParty
	original := `{"custom":"retained"}`
	writeDesktopFixture(t, sharedProfile, original)
	writeDesktopFixture(t, normal, `{"deploymentMode":"1p"}`)
	writeDesktopFixture(t, thirdParty, `{"deploymentMode":"1p","custom":"retained"}`)
	meta := filepath.Join(filepath.Dir(sharedProfile), "_meta.json")
	writeDesktopFixture(t, meta, `{"appliedId":"original","entries":[{"id":"original","name":"Original"}]}`)
	input := Input{Client: "claude-desktop", Revision: digest([]byte(original), true)}
	if _, err := s.Apply(ctx, input); err == nil || err.Error() != "agent: config_changed" {
		t.Fatal("path correction overwrote shared configuration without reread", err)
	}
	input.AcceptCurrentConfig = true
	if _, err := s.Apply(ctx, input); err != nil {
		t.Fatal(err)
	}
	if active, err := s.DesktopGatewayActive(); err != nil || !active {
		t.Fatal("shared gateway configuration not activated", err)
	}
	if _, err := s.Restore(ctx, "claude-desktop"); err != nil {
		t.Fatal(err)
	}
	if restored, err := os.ReadFile(sharedProfile); err != nil || string(restored) != original {
		t.Fatal("restoration used the stale package backup", err)
	}
	if readDesktopFixture(t, normal)["deploymentMode"] != "1p" || readDesktopFixture(t, thirdParty)["deploymentMode"] != "1p" || readDesktopFixture(t, meta)["appliedId"] != "original" {
		t.Fatal("restoration used the stale package startup state")
	}
	if unchanged, err := os.ReadFile(cachedProfile); err != nil || string(unchanged) != string(cached) {
		t.Fatal("path correction modified the unused package configuration", err)
	}
}

func TestDesktopRereadRebasesActivationAtSameLogicalPath(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	change, err := s.desktopFiles()
	if err != nil {
		t.Fatal(err)
	}
	defer change.clear()
	normal, thirdParty, meta := change.files[0].path, change.files[1].path, change.files[2].path
	writeDesktopFixture(t, normal, `{"deploymentMode":"1p"}`)
	writeDesktopFixture(t, thirdParty, `{"deploymentMode":"1p"}`)
	writeDesktopFixture(t, meta, `{"appliedId":"first","entries":[{"id":"first","name":"First"}]}`)
	if _, err := s.Apply(ctx, Input{Client: "claude-desktop", Revision: digest(nil, false)}); err != nil {
		t.Fatal(err)
	}
	// 文件视图改变或重装后，逻辑路径可能相同；重新读取必须同步更新配置与激活状态的恢复基准。
	fresh := `{"custom":"current"}`
	writeDesktopFixture(t, s.paths.ClaudeDesktop, fresh)
	writeDesktopFixture(t, normal, `{"deploymentMode":"3p"}`)
	writeDesktopFixture(t, thirdParty, `{"deploymentMode":"3p","custom":"current"}`)
	writeDesktopFixture(t, meta, `{"appliedId":"old-project","entries":[{"id":"old-project","name":"Old project"}]}`)
	if _, err := s.Apply(ctx, Input{Client: "claude-desktop", Revision: digest([]byte(fresh), true), AcceptCurrentConfig: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Restore(ctx, "claude-desktop"); err != nil {
		t.Fatal(err)
	}
	if restored, err := os.ReadFile(s.paths.ClaudeDesktop); err != nil || string(restored) != fresh {
		t.Fatal("reread profile baseline not restored", err)
	}
	if readDesktopFixture(t, normal)["deploymentMode"] != "3p" || readDesktopFixture(t, thirdParty)["deploymentMode"] != "3p" || readDesktopFixture(t, meta)["appliedId"] != "old-project" {
		t.Fatal("reread retained the old activation baseline at the same logical path")
	}
}
