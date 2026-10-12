package agentconfig

import (
	"context"
	"errors"
	"time"

	catalog "omniproxy/internal/core/modelcatalog"
)

// Apply 自动提供完整目录；旧调用者显式传入 Model 时仍校验该默认模型。
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
		desktop, err = s.prepareDesktopChange(ctx, true)
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
	if err := writeAtomic(prepared.path, patched, input.Revision); err != nil {
		return nil, errors.Join(err, rollbackCatalog())
	}
	if err := desktop.write(); err != nil {
		return nil, errors.Join(err, rollbackFile(prepared.path, digest(patched, true), before), rollbackCatalog())
	}
	if err := s.commitFileVersions(ctx, input.Client, digest(patched, true), local, digest(catalogBody, true), fileVersionCommit{staged, desktop}); err != nil {
		return nil, errors.Join(err, desktop.rollback(3), rollbackFile(prepared.path, digest(patched, true), before), rollbackCatalog())
	}
	committed = true
	if err := s.SetEnabled(ctx, input.Client, true); err != nil {
		return nil, err
	}
	return s.Inspect(ctx)
}

type fileVersionCommit struct {
	rebase  backupRebase
	desktop *desktopChange
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
