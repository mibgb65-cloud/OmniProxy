package agentconfig

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

func (change *desktopChange) build(activate bool) error {
	fields := []desktopField{change.next.Normal, change.next.ThirdParty, change.next.Applied}
	for index := range change.files {
		file := &change.files[index]
		key := "deploymentMode"
		if index == 2 {
			key = "appliedId"
			if err := desktopEntries(file.config, activate); err != nil {
				return err
			}
		}
		field := fields[index]
		delete(file.config, key)
		if field.Present {
			file.config[key] = field.Value
		}
		// 原本不存在的启动文件只在仍无其他设置时删除，保留客户端后来添加的设置。
		if !activate && index < 2 && !field.FileExists && len(file.config) == 0 {
			continue
		}
		body, err := json.MarshalIndent(file.config, "", "  ")
		if err != nil || len(body) >= maxConfigBytes {
			return failure("invalid_config")
		}
		file.after = append(body, '\n')
	}
	return nil
}

func desktopEntries(config map[string]any, add bool) error {
	entries := []any{}
	if value, exists := config["entries"]; exists {
		var ok bool
		entries, ok = value.([]any)
		if !ok {
			return failure("invalid_config")
		}
	}
	retained := []any{}
	for _, entry := range entries {
		item, ok := entry.(map[string]any)
		id, valid := item["id"].(string)
		if !ok || !valid || len(id) > 128 {
			return failure("invalid_config")
		}
		if id != DesktopProfileID {
			retained = append(retained, entry)
		}
	}
	if add {
		retained = append(retained, map[string]any{"id": DesktopProfileID, "name": "OmniProxy · 本地网关"})
	}
	config["entries"] = retained
	return nil
}

func (change *desktopChange) clear() {
	if change == nil {
		return
	}
	for index := range change.files {
		clear(change.files[index].before.body)
		clear(change.files[index].after)
	}
}

func (change *desktopChange) write() error {
	if change == nil {
		return nil
	}
	for index, file := range change.files {
		var err error
		if file.after == nil {
			err = removeOwned(file.path, file.before.revision)
		} else {
			err = writeAtomic(file.path, file.after, file.before.revision)
		}
		if err != nil {
			return errors.Join(err, change.rollback(index))
		}
	}
	return nil
}

func (change *desktopChange) rollback(count int) error {
	if change == nil {
		return nil
	}
	var result error
	for index := count - 1; index >= 0; index-- {
		file := change.files[index]
		result = errors.Join(result, rollbackFile(file.path, digest(file.after, file.after != nil), file.before))
	}
	return result
}

func commitDesktopChange(ctx context.Context, tx *sql.Tx, change *desktopChange) error {
	if change == nil {
		return nil
	}
	original, err := json.Marshal(change.original)
	if err != nil {
		return failure("backup_failed")
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO agent_desktop_activation(client,original_state,applied_hash,updated_at) VALUES('claude-desktop',?,?,?) ON CONFLICT(client) DO UPDATE SET original_state=excluded.original_state,applied_hash=excluded.applied_hash,updated_at=excluded.updated_at`, string(original), desktopStateHash(change.next), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return failure("backup_failed")
	}
	return nil
}
