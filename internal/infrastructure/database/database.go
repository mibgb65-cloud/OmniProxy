// Package database opens the OmniProxy SQLite database.
//
// SQLite is accessed through database/sql with the pure-Go modernc.org/sqlite
// driver, so builds need no CGO toolchain. The schema itself only ever changes
// through migrations (see migrate.go and 技术架构选择.md §14-§15).
package database

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite" // register the "sqlite" driver
)

// driverName is the database/sql driver registered by modernc.org/sqlite.
const driverName = "sqlite"

// Config configures the database connection.
type Config struct {
	// Path is the SQLite database file. Parent directories are created.
	Path string
}

// Open opens (creating if necessary) the SQLite database at cfg.Path.
func Open(cfg Config) (*sql.DB, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("database: empty path")
	}
	if dir := filepath.Dir(cfg.Path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("database: create data dir: %w", err)
		}
	}

	dsn, err := sqliteDSN(cfg.Path, "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, err
	}
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("database: open %s: %w", cfg.Path, err)
	}
	// A local desktop app has a single writer; one connection keeps SQLite
	// contention predictable.
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("database: ping %s: %w", cfg.Path, err)
	}
	return db, nil
}

func sqliteDSN(path, query string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("database: resolve path: %w", err)
	}
	uriPath := filepath.ToSlash(abs)
	// Windows 盘符必须是 /C:/…，否则 URL 将 C: 解释为 authority。
	if !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	return (&url.URL{Scheme: "file", Path: uriPath, RawQuery: query}).String(), nil
}
