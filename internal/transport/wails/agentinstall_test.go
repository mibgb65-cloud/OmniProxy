package wails

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"omniproxy/internal/application/agentconfig"
)

func TestInstallationBindingsRejectMissingServiceAndArbitraryURL(t *testing.T) {
	s := &AppService{}
	if _, err := s.AgentInstallations(); err == nil {
		t.Fatal("missing service accepted")
	}
	if _, err := s.AgentInstall("claude-code"); err == nil {
		t.Fatal("missing service installed")
	}
	if _, err := s.AgentInstallationCheckUpdate("claude-desktop"); err == nil {
		t.Fatal("missing service checked for updates")
	}
	if _, err := s.AgentInstallationUpdate("claude-desktop"); err == nil {
		t.Fatal("missing service updated software")
	}
	var opened string
	s.openAgentURL = func(url string) error { opened = url; return nil }
	if err := s.AgentInstallationOpenOfficial("https://malicious.invalid"); err == nil || opened != "" {
		t.Fatal("arbitrary URL opened")
	}
	if err := s.AgentInstallationOpenOfficial("claude-code"); err != nil || !strings.HasPrefix(opened, "https://code.claude.com/") {
		t.Fatal("official source not opened")
	}
}

func TestClaudeCodeLauncherUsesOnlyOfficialCodeDeepLink(t *testing.T) {
	s := &AppService{}
	if s.AgentOpenClaudeCode() == nil {
		t.Fatal("missing desktop launcher accepted")
	}
	var opened string
	s.openAgentURL = func(url string) error { opened = url; return nil }
	root := t.TempDir()
	profilePath := filepath.Join(root, "Claude-3p", "configLibrary", agentconfig.DesktopProfileID+".json")
	s.agents = agentconfig.New(nil, nil, agentconfig.Paths{ClaudeDesktop: profilePath}, 3001, nil)
	if err := s.AgentOpenClaudeCode(); err == nil || opened != "" {
		t.Fatal("inactive gateway opened official login")
	}
	for path, body := range map[string]string{
		filepath.Join(root, "Claude", "claude_desktop_config.json"):    `{"deploymentMode":"3p"}`,
		filepath.Join(root, "Claude-3p", "claude_desktop_config.json"): `{"deploymentMode":"3p"}`,
		filepath.Join(filepath.Dir(profilePath), "_meta.json"):         `{"appliedId":"` + agentconfig.DesktopProfileID + `"}`,
		profilePath: `{"inferenceProvider":"gateway","inferenceGatewayBaseUrl":"http://127.0.0.1:3001","inferenceGatewayApiKey":"synthetic-local-key"}`,
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.AgentOpenClaudeCode(); err != nil || opened != "claude://code/new" {
		t.Fatal("Code programming entry not opened", err)
	}
	if err := s.AgentInstallationOpenOfficial("claude-desktop"); err != nil || opened != "https://code.claude.com/docs/en/desktop" {
		t.Fatal("Claude Code Desktop guide not opened", err)
	}
}
