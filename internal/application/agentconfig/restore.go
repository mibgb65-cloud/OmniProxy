package agentconfig

import (
	"context"
	"encoding/json"
	"errors"
)

// Restore 仅恢复没有外部改动的受管文件，防止覆盖用户在接入后新增的设置。
func (s *Service) Restore(ctx context.Context, client string) ([]Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path, err := s.path(client)
	if err != nil {
		return nil, err
	}
	var ref, hash string
	if s.db.QueryRowContext(ctx, `SELECT credential_ref,applied_hash FROM agent_backups WHERE client=?`, client).Scan(&ref, &hash) != nil {
		return nil, failure("backup_unavailable")
	}
	body, exists, err := read(path)
	if err != nil {
		return nil, err
	}
	revision := digest(body, exists)
	defer clear(body)
	if revision != hash {
		return nil, failure("config_changed")
	}
	if client == "codex" {
		if err := s.checkCatalogFile(ctx); err != nil {
			return nil, err
		}
	}
	data, err := s.store.Get(ref)
	if err != nil {
		return nil, failure("backup_unavailable")
	}
	defer clear(data)
	var original backup
	if json.Unmarshal(data, &original) != nil {
		return nil, failure("backup_unavailable")
	}
	defer clear(original.Body)
	var desktop *desktopChange
	if client == "claude-desktop" {
		desktop, err = s.prepareDesktopChange(ctx, false, false)
		if err != nil {
			return nil, err
		}
		defer desktop.clear()
		if desktop == nil {
			// 旧版只登记配置条目，没有模式备份；沿用原移除保护。
			if err := s.desktopEntry(false); err != nil {
				return nil, err
			}
		} else if err := desktop.write(); err != nil {
			return nil, err
		}
	}
	if original.Exists {
		err = writeAtomic(path, original.Body, hash)
	} else {
		err = removeOwned(path, hash)
	}
	if err != nil {
		return nil, errors.Join(err, desktop.rollback(3))
	}
	if client == "codex" {
		if err := s.restoreCatalog(ctx); err != nil {
			return nil, err
		}
	}
	if err := s.deleteRestoreRecords(ctx, client, desktop != nil); err != nil {
		return nil, errors.Join(err, rollbackFile(path, digest(original.Body, original.Exists), fileSnapshot{body, exists, revision}), desktop.rollback(3))
	}
	if err = s.store.Delete(ref); err != nil {
		return nil, failure("backup_failed")
	}
	if err = s.SetEnabled(ctx, client, false); err != nil {
		return nil, err
	}
	return s.Inspect(ctx)
}

func (s *Service) deleteRestoreRecords(ctx context.Context, client string, desktop bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return failure("backup_failed")
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM agent_backups WHERE client=?`, client); err != nil {
		return failure("backup_failed")
	}
	if desktop {
		if _, err := tx.ExecContext(ctx, `DELETE FROM agent_desktop_activation WHERE client='claude-desktop'`); err != nil {
			return failure("backup_failed")
		}
	}
	if tx.Commit() != nil {
		return failure("backup_failed")
	}
	return nil
}
