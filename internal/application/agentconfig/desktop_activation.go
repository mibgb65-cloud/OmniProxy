package agentconfig

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
)

type desktopField struct {
	FileExists bool    `json:"fileExists"`
	Present    bool    `json:"present"`
	Value      *string `json:"value"`
}

type desktopState struct {
	Normal, ThirdParty, Applied desktopField
	Scope                       string `json:"scope,omitempty"`
}

type desktopFileChange struct {
	path   string
	before fileSnapshot
	config map[string]any
	after  []byte
}

type desktopChange struct {
	files    [3]desktopFileChange
	original desktopState
	current  desktopState
	next     desktopState
}

func (s *Service) desktopFiles() (*desktopChange, error) {
	library := filepath.Dir(s.paths.ClaudeDesktop)
	thirdParty := filepath.Dir(library)
	paths := [3]string{
		filepath.Join(filepath.Dir(thirdParty), "Claude", "claude_desktop_config.json"),
		filepath.Join(thirdParty, "claude_desktop_config.json"),
		filepath.Join(library, "_meta.json"),
	}
	if s.paths.ClaudeDesktopNormal != "" {
		paths[0] = s.paths.ClaudeDesktopNormal
	}
	if s.paths.ClaudeDesktopThirdParty != "" {
		paths[1] = s.paths.ClaudeDesktopThirdParty
	}
	change := &desktopChange{}
	scope, _ := json.Marshal(paths)
	change.current.Scope = digest(scope, true)
	for index, path := range paths {
		file, err := snapshotFile(path)
		if err != nil {
			change.clear()
			return nil, err
		}
		change.files[index] = desktopFileChange{path: path, before: file, config: map[string]any{}}
		config := &change.files[index].config
		if len(file.body) > 0 && json.Unmarshal(file.body, config) != nil || *config == nil {
			change.clear()
			return nil, failure("invalid_config")
		}
	}
	var err error
	change.current.Normal, err = readDesktopField(change.files[0], "deploymentMode", true)
	if err == nil {
		change.current.ThirdParty, err = readDesktopField(change.files[1], "deploymentMode", true)
	}
	if err == nil {
		change.current.Applied, err = readDesktopField(change.files[2], "appliedId", false)
	}
	if err != nil {
		change.clear()
		return nil, err
	}
	return change, nil
}

func readDesktopField(file desktopFileChange, key string, mode bool) (desktopField, error) {
	value, present := file.config[key]
	field := desktopField{FileExists: file.before.exists, Present: present}
	if value == nil {
		return field, nil
	}
	text, ok := value.(string)
	if !ok || len(text) > 128 || mode && text != "1p" && text != "3p" {
		return desktopField{}, failure("invalid_config")
	}
	field.Value = &text
	return field, nil
}

func (s *Service) prepareDesktopChange(ctx context.Context, activate bool) (*desktopChange, error) {
	change, err := s.desktopFiles()
	if err != nil {
		return nil, err
	}
	var original, applied string
	err = s.db.QueryRowContext(ctx, `SELECT original_state,applied_hash FROM agent_desktop_activation WHERE client='claude-desktop'`).Scan(&original, &applied)
	if err == sql.ErrNoRows && !activate {
		change.clear()
		return nil, nil
	}
	change.original = change.current
	if err == nil {
		// 解码备份时不能复用当前状态的字符串指针，否则会篡改后续版本比较。
		change.original = desktopState{}
		if len(original) > 2048 || json.Unmarshal([]byte(original), &change.original) != nil || !validDesktopState(change.original) || len(applied) != 64 {
			change.clear()
			return nil, failure("backup_unavailable")
		}
		if change.original.Scope != change.current.Scope {
			if !activate {
				change.clear()
				return nil, failure("config_changed")
			}
			// 安装渠道或实际数据根目录变化后，以新位置的现状为模式恢复基准。
			change.original = change.current
		}
		if !activate && desktopStateHash(change.current) != applied {
			change.clear()
			return nil, failure("config_changed")
		}
	} else if err != sql.ErrNoRows {
		change.clear()
		return nil, failure("backup_failed")
	}
	change.next = change.original
	if activate {
		mode, profile := "3p", DesktopProfileID
		change.next = desktopState{
			Normal: desktopField{true, true, &mode}, ThirdParty: desktopField{true, true, &mode}, Applied: desktopField{true, true, &profile},
			Scope: change.current.Scope,
		}
	}
	if err := change.build(activate); err != nil {
		change.clear()
		return nil, err
	}
	return change, nil
}

func validDesktopState(state desktopState) bool {
	if state.Scope != "" && len(state.Scope) != 64 {
		return false
	}
	for index, field := range []desktopField{state.Normal, state.ThirdParty, state.Applied} {
		if field.Present && !field.FileExists || !field.Present && field.Value != nil {
			return false
		}
		if field.Value != nil && (len(*field.Value) > 128 || index < 2 && *field.Value != "1p" && *field.Value != "3p") {
			return false
		}
	}
	return true
}

func desktopStateHash(state desktopState) string {
	body, _ := json.Marshal(state)
	return digest(body, true)
}
