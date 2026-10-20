package agentconfig

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

// RestoreCodexFileLogin 恢复试验前的配置和 auth.json，拒绝覆盖试验后的外部修改。
func (s *Service) RestoreCodexFileLogin() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.restoreLoginTrial()
}

func (s *Service) restoreLoginTrial() error {
	authPath, manifestPath := s.loginTrialPaths()
	manifest, err := snapshotFile(manifestPath)
	if err != nil {
		return err
	}
	defer clear(manifest.body)
	var receipt loginTrialManifest
	if !manifest.exists || json.Unmarshal(manifest.body, &receipt) != nil || receipt.Version != 1 {
		return failure("backup_unavailable")
	}
	bundle, err := s.store.Get(receipt.Reference)
	if err != nil {
		return failure("backup_unavailable")
	}
	defer clear(bundle)
	var original loginTrialBackup
	if json.Unmarshal(bundle, &original) != nil || original.Version != 1 ||
		!strings.EqualFold(original.ConfigPath, filepath.Clean(s.paths.Codex)) {
		return failure("backup_unavailable")
	}
	defer clear(original.Config.Body)
	defer clear(original.Auth.Body)
	config, err := snapshotFile(s.paths.Codex)
	if err != nil {
		return err
	}
	defer clear(config.body)
	auth, err := snapshotFile(authPath)
	if err != nil {
		return err
	}
	defer clear(auth.body)
	configBefore := digest(original.Config.Body, original.Config.Exists)
	authBefore := digest(original.Auth.Body, original.Auth.Exists)
	if config.revision != configBefore && config.revision != original.ConfigApplied ||
		auth.revision != authBefore && auth.revision != original.AuthApplied {
		return failure("config_changed")
	}
	if config.revision != configBefore {
		if err := rollbackFile(s.paths.Codex, config.revision, fileSnapshot{body: original.Config.Body, exists: original.Config.Exists}); err != nil {
			return err
		}
	}
	if auth.revision != authBefore {
		if err := rollbackFile(authPath, auth.revision, fileSnapshot{body: original.Auth.Body, exists: original.Auth.Exists}); err != nil {
			return err
		}
	}
	return s.rollbackLoginTrial(receipt.Reference, manifestPath, manifest.revision)
}
