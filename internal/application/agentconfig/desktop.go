package agentconfig

import (
	"encoding/json"
	"path/filepath"
)

// desktopEntry 只维护自己的配置库条目，不修改 appliedId 或 deploymentMode。
func (s *Service) desktopEntry(add bool) error {
	path := filepath.Join(filepath.Dir(s.paths.ClaudeDesktop), "_meta.json")
	body, exists, err := read(path)
	if err != nil {
		return err
	}
	defer clear(body)
	config := map[string]any{}
	if len(body) > 0 && json.Unmarshal(body, &config) != nil {
		return failure("invalid_config")
	}
	if config == nil {
		return failure("invalid_config")
	}
	if !add && config["appliedId"] == DesktopProfileID {
		return failure("desktop_profile_active")
	}
	if err := desktopEntries(config, add); err != nil {
		return err
	}
	result, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return failure("invalid_config")
	}
	return writeAtomic(path, append(result, '\n'), digest(body, exists))
}
