package agentconfig

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"time"
)

type fileSnapshot struct {
	body     []byte
	exists   bool
	revision string
}

func snapshotFile(path string) (fileSnapshot, error) {
	body, exists, err := read(path)
	if err != nil {
		return fileSnapshot{}, err
	}
	return fileSnapshot{body, exists, digest(body, exists)}, nil
}

func (s *Service) checkCatalogFile(ctx context.Context) error {
	file, err := snapshotFile(s.catalogPath())
	if err != nil {
		return err
	}
	defer clear(file.body)
	var hash string
	err = s.db.QueryRowContext(ctx, `SELECT applied_hash FROM agent_catalog_backups WHERE client='codex'`).Scan(&hash)
	if err == sql.ErrNoRows {
		if file.exists {
			return failure("catalog_conflict")
		}
		return nil
	}
	if err != nil {
		return failure("backup_failed")
	}
	if hash != file.revision {
		return failure("config_changed")
	}
	return nil
}

func (s *Service) saveBackup(ctx context.Context, table, client string, file fileSnapshot) error {
	if table != "agent_backups" && table != "agent_catalog_backups" {
		return failure("backup_failed")
	}
	var ref string
	err := s.db.QueryRowContext(ctx, "SELECT credential_ref FROM "+table+" WHERE client=?", client).Scan(&ref)
	if err == nil {
		return nil
	}
	if err != sql.ErrNoRows {
		return failure("backup_failed")
	}
	bundle, err := json.Marshal(backup{Body: file.body, Exists: file.exists})
	if err != nil {
		return failure("backup_failed")
	}
	defer clear(bundle)
	ref, err = s.store.Put(bundle)
	if err != nil {
		return failure("backup_failed")
	}
	if _, err = s.db.ExecContext(ctx, "INSERT INTO "+table+" VALUES(?,?,?,?)", client, ref, file.revision, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return errors.Join(failure("backup_failed"), s.store.Delete(ref))
	}
	return nil
}

// removeOwned 在删除前重新核对受管版本，不能删除外部修改或同名替换文件。
func removeOwned(path, revision string) error {
	current, err := snapshotFile(path)
	if err != nil {
		return err
	}
	defer clear(current.body)
	if current.revision != revision {
		return failure("config_changed")
	}
	if !current.exists {
		return nil
	}
	if os.Remove(path) != nil {
		return failure("write_failed")
	}
	return nil
}

func rollbackFile(path, revision string, before fileSnapshot) error {
	if before.exists {
		return writeAtomic(path, before.body, revision)
	}
	return removeOwned(path, revision)
}

func (s *Service) restoreCatalog(ctx context.Context) error {
	var ref, hash string
	err := s.db.QueryRowContext(ctx, `SELECT credential_ref,applied_hash FROM agent_catalog_backups WHERE client='codex'`).Scan(&ref, &hash)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return failure("backup_unavailable")
	}
	if err := s.checkCatalogFile(ctx); err != nil {
		return err
	}
	bundle, err := s.store.Get(ref)
	if err != nil {
		return failure("backup_unavailable")
	}
	defer clear(bundle)
	var original backup
	if json.Unmarshal(bundle, &original) != nil {
		return failure("backup_unavailable")
	}
	defer clear(original.Body)
	if err := rollbackFile(s.catalogPath(), hash, fileSnapshot{body: original.Body, exists: original.Exists}); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM agent_catalog_backups WHERE client='codex'`); err != nil {
		return failure("backup_failed")
	}
	if s.store.Delete(ref) != nil {
		return failure("backup_failed")
	}
	return nil
}
