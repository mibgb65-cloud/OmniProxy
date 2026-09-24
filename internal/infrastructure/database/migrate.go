package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Migrate applies all pending migrations found in migrations (a flat FS of
// "<version>_<name>.sql" files, e.g. 001_initial.sql). Each migration runs
// inside a transaction and is recorded in schema_version. Migrations are
// append-only: never edit an applied migration, add a new one instead
// (技术架构选择.md §15).
func Migrate(ctx context.Context, db *sql.DB, migrations fs.FS) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_version (
			version    INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL
		)`); err != nil {
		return fmt.Errorf("database: ensure schema_version: %w", err)
	}

	applied, err := appliedVersions(ctx, db)
	if err != nil {
		return err
	}

	names, err := migrationNames(migrations)
	if err != nil {
		return err
	}

	for _, name := range names {
		version, err := parseVersion(name)
		if err != nil {
			return err
		}
		if _, done := applied[version]; done {
			continue
		}
		if err := applyMigration(ctx, db, migrations, name, version); err != nil {
			return err
		}
	}
	return nil
}

func appliedVersions(ctx context.Context, db *sql.DB) (map[int64]struct{}, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_version`)
	if err != nil {
		return nil, fmt.Errorf("database: read schema_version: %w", err)
	}
	defer func() { _ = rows.Close() }()

	applied := make(map[int64]struct{})
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("database: scan schema_version: %w", err)
		}
		applied[v] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("database: iterate schema_version: %w", err)
	}
	return applied, nil
}

// migrationNames returns the sorted names of all .sql files at the FS root.
func migrationNames(migrations fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(migrations, ".")
	if err != nil {
		return nil, fmt.Errorf("database: read migrations dir: %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func applyMigration(ctx context.Context, db *sql.DB, migrations fs.FS, name string, version int64) error {
	content, err := fs.ReadFile(migrations, name)
	if err != nil {
		return fmt.Errorf("database: read migration %s: %w", name, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("database: begin migration %s: %w", name, err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, string(content)); err != nil {
		return fmt.Errorf("database: exec migration %s: %w", name, err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_version (version, applied_at) VALUES (?, ?)`,
		version, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("database: record migration %s: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("database: commit migration %s: %w", name, err)
	}
	return nil
}

// parseVersion extracts the leading numeric version from a migration file
// name such as "001_initial.sql".
func parseVersion(name string) (int64, error) {
	base := strings.TrimSuffix(name, ".sql")
	sep := strings.IndexByte(base, '_')
	if sep <= 0 {
		return 0, fmt.Errorf("database: migration %q must be named <version>_<name>.sql", name)
	}
	v, err := strconv.ParseInt(base[:sep], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("database: migration %q has a non-numeric version: %w", name, err)
	}
	return v, nil
}
