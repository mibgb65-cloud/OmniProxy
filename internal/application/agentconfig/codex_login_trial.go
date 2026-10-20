package agentconfig

import (
	"encoding/json"
	"errors"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"omniproxy/internal/infrastructure/codex"
)

const loginTrialName = "omniproxy-login-trial.json"

type loginTrialBackup struct {
	Version       int    `json:"version"`
	ConfigPath    string `json:"configPath"`
	Config        backup `json:"config"`
	Auth          backup `json:"auth"`
	ConfigApplied string `json:"configApplied"`
	AuthApplied   string `json:"authApplied"`
}

type loginTrialManifest struct {
	Version   int    `json:"version"`
	Reference string `json:"credentialRef"`
}

// TryCodexFileLogin 试用旧版登录方式；必须显式调用，默认客户端配置流程不会触发。
func (s *Service) TryCodexFileLogin(auth []byte, expectedID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	normalized, err := codex.ClientAuthFile(auth, expectedID)
	if err != nil {
		return failure("invalid_client_auth")
	}
	defer clear(normalized)
	config, err := snapshotFile(s.paths.Codex)
	if err != nil {
		return err
	}
	defer clear(config.body)
	authPath, manifestPath := s.loginTrialPaths()
	originalAuth, err := snapshotFile(authPath)
	if err != nil {
		return err
	}
	defer clear(originalAuth.body)
	manifest, err := snapshotFile(manifestPath)
	if err != nil {
		return err
	}
	defer clear(manifest.body)
	if manifest.exists {
		return failure("login_trial_active")
	}
	patched, err := legacyLoginConfig(config.body, expectedID, s.port)
	if err != nil {
		return err
	}
	defer clear(patched)
	payload := loginTrialBackup{
		Version: 1, ConfigPath: filepath.Clean(s.paths.Codex), Config: backup{Body: config.body, Exists: config.exists},
		Auth:          backup{Body: originalAuth.body, Exists: originalAuth.exists},
		ConfigApplied: digest(patched, true), AuthApplied: digest(normalized, true),
	}
	bundle, err := json.Marshal(payload)
	if err != nil {
		return failure("backup_failed")
	}
	defer clear(bundle)
	ref, err := s.store.Put(bundle)
	if err != nil {
		return failure("backup_failed")
	}
	receipt, err := json.Marshal(loginTrialManifest{Version: 1, Reference: ref})
	if err != nil {
		return errors.Join(failure("backup_failed"), s.store.Delete(ref))
	}
	if err := writeAtomic(manifestPath, receipt, manifest.revision); err != nil {
		return errors.Join(err, s.store.Delete(ref))
	}
	if err := writeAtomic(authPath, normalized, originalAuth.revision); err != nil {
		return errors.Join(err, s.restoreLoginTrial())
	}
	if err := writeAtomic(s.paths.Codex, patched, config.revision); err != nil {
		return errors.Join(err, rollbackFile(authPath, digest(normalized, true), originalAuth), s.restoreLoginTrial())
	}
	return nil
}

func legacyLoginConfig(body []byte, expectedID string, port int) ([]byte, error) {
	var config map[string]any
	if len(body) == 0 || toml.Unmarshal(body, &config) != nil {
		return nil, failure("invalid_config")
	}
	if provider, exists := config["model_provider"]; exists && provider != "openai" {
		return nil, failure("provider_conflict")
	}
	if profile, exists := config["profile"]; exists && profile != "" {
		return nil, failure("profile_conflict")
	}
	base, ok := config["openai_base_url"].(string)
	endpoint, err := url.Parse(base)
	if !ok || len(base) > 2048 || strings.ContainsAny(base, "\r\n\x00") || err != nil ||
		endpoint.Scheme != "http" || endpoint.Hostname() != "127.0.0.1" || endpoint.Port() != strconv.Itoa(port) ||
		endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return nil, failure("client_not_configured")
	}
	if workspace, exists := config["forced_chatgpt_workspace_id"]; exists && workspace != expectedID {
		return nil, failure("workspace_conflict")
	}
	providers, _ := config["model_providers"].(map[string]any)
	if providers["openai"] != nil {
		return nil, failure("provider_conflict")
	}
	return patchTOML(body, "openai", map[string]string{
		"model_provider": quote("openai"), "forced_login_method": quote("chatgpt"), "cli_auth_credentials_store": quote("file"),
	})
}

func (s *Service) loginTrialPaths() (string, string) {
	root := filepath.Dir(s.paths.Codex)
	return filepath.Join(root, "auth.json"), filepath.Join(root, loginTrialName)
}

func (s *Service) rollbackLoginTrial(ref, path, revision string) error {
	if err := removeOwned(path, revision); err != nil {
		return err
	}
	return s.store.Delete(ref)
}
