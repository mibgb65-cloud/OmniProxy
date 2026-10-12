package agentconfig

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

type backupRebase struct{ oldRef, oldHash, newRef string }

// stageBackupRebase 只暂存当前文件的加密快照；应用成功前不改变原恢复基准。
func (s *Service) stageBackupRebase(ctx context.Context, input Input, file fileSnapshot) (backupRebase, error) {
	var staged backupRebase
	err := s.db.QueryRowContext(ctx, `SELECT credential_ref,applied_hash FROM agent_backups WHERE client=?`, input.Client).Scan(&staged.oldRef, &staged.oldHash)
	if err == sql.ErrNoRows {
		return backupRebase{}, nil
	}
	if err != nil {
		return backupRebase{}, failure("backup_failed")
	}
	if staged.oldHash == input.Revision {
		return backupRebase{}, nil
	}
	if !input.AcceptCurrentConfig {
		return backupRebase{}, failure("config_changed")
	}
	bundle, err := json.Marshal(backup{Body: file.body, Exists: file.exists})
	if err != nil {
		return backupRebase{}, failure("backup_failed")
	}
	defer clear(bundle)
	staged.newRef, err = s.store.Put(bundle)
	if err != nil {
		return backupRebase{}, failure("backup_failed")
	}
	return staged, nil
}

func commitBackupRebase(ctx context.Context, tx *sql.Tx, client string, staged backupRebase) error {
	if staged.newRef == "" {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `INSERT INTO agent_backup_history(client,credential_ref,applied_hash,archived_at) VALUES(?,?,?,?)`, client, staged.oldRef, staged.oldHash, now); err != nil {
		return failure("backup_failed")
	}
	result, err := tx.ExecContext(ctx, `UPDATE agent_backups SET credential_ref=? WHERE client=? AND credential_ref=? AND applied_hash=?`, staged.newRef, client, staged.oldRef, staged.oldHash)
	if err != nil {
		return failure("backup_failed")
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return failure("config_changed")
	}
	return nil
}

func (s *Service) discardBackupRebase(staged backupRebase) error {
	if staged.newRef == "" {
		return nil
	}
	if err := s.store.Delete(staged.newRef); err != nil {
		return errors.Join(failure("backup_failed"), err)
	}
	return nil
}
