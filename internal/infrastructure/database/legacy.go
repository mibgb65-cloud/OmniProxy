package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

// ResolveLegacyPath 在迁移前识别历史统计库，返回独立的新库路径。
// 已存在的 v2 库保持原路径；绝不改写旧库或把不兼容的表视为已迁移。
func ResolveLegacyPath(ctx context.Context, path string) (string, error) {
	legacy, err := isLegacyDatabase(ctx, path)
	if err != nil || !legacy {
		return path, err
	}
	target := filepath.Join(filepath.Dir(path), "v2", filepath.Base(path))
	foreign, err := isLegacyDatabase(ctx, target)
	if err != nil {
		return "", err
	}
	if foreign {
		return "", fmt.Errorf("database: isolated v2 path also contains legacy data")
	}
	return target, nil
}

func isLegacyDatabase(ctx context.Context, path string) (bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("database: inspect existing file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("database: existing path is not a regular file")
	}
	// URI 转义保证路径中的 #、? 不会改变只读参数；保留 WAL 可见性。
	dsn, err := sqliteDSN(path, "mode=ro&_pragma=busy_timeout(5000)")
	if err != nil {
		return false, err
	}
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return false, fmt.Errorf("database: open schema probe: %w", err)
	}
	defer func() { _ = db.Close() }()
	var columns, legacyColumns, v2Columns int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*),
		COALESCE(SUM(name IN ('client','client_model','provider_model','success')),0),
		COALESCE(SUM(name IN ('day','protocol','input_model','provider_id')),0)
		FROM pragma_table_info('request_records')`).Scan(&columns, &legacyColumns, &v2Columns)
	if err != nil {
		return false, fmt.Errorf("database: inspect request schema: %w", err)
	}
	if columns == 0 || (v2Columns == 4 && legacyColumns == 0) {
		return false, nil
	}
	if legacyColumns != 4 || v2Columns != 0 {
		return false, fmt.Errorf("database: unrecognised request_records schema; existing data preserved")
	}
	// Beta.1 可能已应用前七个迁移。若混有新版账号，不能自动改目录隐藏它们。
	var accountTable, accounts int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='accounts'`).Scan(&accountTable); err != nil {
		return false, fmt.Errorf("database: inspect account schema: %w", err)
	}
	if accountTable != 0 {
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM accounts`).Scan(&accounts); err != nil {
			return false, fmt.Errorf("database: inspect account count: %w", err)
		}
		if accounts != 0 {
			return false, fmt.Errorf("database: legacy statistics mixed with accounts; manual recovery required")
		}
	}
	var versionTable, migrated int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_version'`).Scan(&versionTable); err != nil {
		return false, fmt.Errorf("database: inspect migration schema: %w", err)
	}
	if versionTable != 0 {
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_version WHERE version >= 8`).Scan(&migrated); err != nil {
			return false, fmt.Errorf("database: inspect migration version: %w", err)
		}
		if migrated != 0 {
			return false, fmt.Errorf("database: legacy statistics mixed with v2 migration history; manual recovery required")
		}
	}
	return true, nil
}
