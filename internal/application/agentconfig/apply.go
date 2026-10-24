package agentconfig

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"omniproxy/internal/application/modeleffort"
	catalog "omniproxy/internal/core/modelcatalog"
)

// Apply 未指定 ModelIDs 时提供完整目录；显式选择只写入所选模型。
func (s *Service) Apply(ctx context.Context, input Input) (items []Item, resultErr error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	prepared, err := s.prepare(ctx, input)
	if err != nil {
		return nil, err
	}
	defer clear(prepared.body)
	key, err := s.key(ctx, input.Client)
	if err != nil {
		return nil, err
	}
	defer clear(key)
	patched, _, err := s.buildPlan(input.Client, prepared.body, key, prepared.plan)
	if err != nil {
		return nil, err
	}
	defer clear(patched)
	before := fileSnapshot{prepared.body, prepared.exists, input.Revision}
	staged, err := s.stageBackupRebase(ctx, input, before)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			resultErr = errors.Join(resultErr, s.discardBackupRebase(staged))
		}
	}()
	if err := s.saveBackup(ctx, "agent_backups", input.Client, before); err != nil {
		return nil, err
	}
	var desktop *desktopChange
	if input.Client == "claude-desktop" {
		desktop, err = s.prepareDesktopChange(ctx, true, staged.newRef != "")
		if err != nil {
			return nil, err
		}
		defer desktop.clear()
	}
	local := prepared.plan.Mode == "local-catalog"
	var oldCatalog fileSnapshot
	var catalogBody []byte
	if local {
		oldCatalog, err = snapshotFile(s.catalogPath())
		if err != nil {
			return nil, err
		}
		defer clear(oldCatalog.body)
		catalogBody, err = catalog.CodexJSON(prepared.plan.Models)
		if err != nil {
			return nil, failure("models_unavailable")
		}
		if err := s.saveBackup(ctx, "agent_catalog_backups", "codex", oldCatalog); err != nil {
			return nil, err
		}
		if err := writeAtomic(s.catalogPath(), catalogBody, oldCatalog.revision); err != nil {
			return nil, err
		}
	}
	rollbackCatalog := func() error {
		if local {
			return rollbackFile(s.catalogPath(), digest(catalogBody, true), oldCatalog)
		}
		return nil
	}
	var login *codexLoginChange
	if input.Client == "codex" {
		login, err = s.applyCodexLogin(ctx)
		if err != nil {
			return nil, err
		}
		defer login.clear()
	}
	if err := s.writeClient(input.Client, prepared.path, patched, input.Revision); err != nil {
		return nil, errors.Join(err, login.rollback(), rollbackCatalog())
	}
	if err := desktop.write(); err != nil {
		return nil, errors.Join(err, login.rollback(), s.rollbackClient(input.Client, prepared.path, digest(patched, true), before), rollbackCatalog())
	}
	if err := s.commitFileVersions(ctx, input.Client, digest(patched, true), local, digest(catalogBody, true), fileVersionCommit{rebase: staged, desktop: desktop, saveSelection: true, modelIDs: input.ModelIDs, saveEfforts: effortClient(input.Client) && input.Efforts != nil, efforts: input.Efforts}); err != nil {
		return nil, errors.Join(err, login.rollback(), desktop.rollback(3), s.rollbackClient(input.Client, prepared.path, digest(patched, true), before), rollbackCatalog())
	}
	committed = true
	if err := s.SetEnabled(ctx, input.Client, true); err != nil {
		return nil, err
	}
	return s.Inspect(ctx)
}

type fileVersionCommit struct {
	rebase        backupRebase
	desktop       *desktopChange
	saveSelection bool
	modelIDs      []string
	saveEfforts   bool
	efforts       map[string]string
}

func (s *Service) commitFileVersions(ctx context.Context, client, configHash string, catalogWritten bool, catalogHash string, options ...fileVersionCommit) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return failure("backup_failed")
	}
	defer tx.Rollback()
	if len(options) > 0 {
		if err := commitBackupRebase(ctx, tx, client, options[0].rebase); err != nil {
			return err
		}
		if err := commitDesktopChange(ctx, tx, options[0].desktop); err != nil {
			return err
		}
		if options[0].saveSelection {
			var selection any
			if options[0].modelIDs != nil {
				encoded, err := json.Marshal(options[0].modelIDs)
				if err != nil {
					return failure("invalid_model_selection")
				}
				selection = string(encoded)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE agent_backups SET model_ids=? WHERE client=?`, selection, client); err != nil {
				return failure("backup_failed")
			}
		}
		// 固定档位与配置版本一起提交：文件没写成功就不会留下只生效一半的档位。
		if options[0].saveEfforts {
			if err := modeleffort.Save(ctx, tx, options[0].efforts); err != nil {
				return failure("effort_unavailable")
			}
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `UPDATE agent_backups SET applied_hash=?,updated_at=? WHERE client=?`, configHash, now, client); err != nil {
		return failure("backup_failed")
	}
	if catalogWritten {
		if _, err := tx.ExecContext(ctx, `UPDATE agent_catalog_backups SET applied_hash=?,updated_at=? WHERE client='codex'`, catalogHash, now); err != nil {
			return failure("backup_failed")
		}
	}
	if tx.Commit() != nil {
		return failure("backup_failed")
	}
	return nil
}
