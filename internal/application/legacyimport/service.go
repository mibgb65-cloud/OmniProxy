// Package legacyimport 将旧账号转换为新版独立凭据，不触碰旧数据或客户端配置。
package legacyimport

import (
	"context"
	"database/sql"
	"errors"

	"omniproxy/internal/application/account"
	"omniproxy/internal/infrastructure/legacy"
)

// Entry 只含可展示的元数据和结果码。
type Entry struct {
	Index    int    `json:"index"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	State    string `json:"state"`
	Enabled  bool   `json:"enabled"`
}

// Preview 不包含密钥、密文或数据库凭据引用。
type Preview struct {
	Directory string  `json:"directory"`
	Snapshot  string  `json:"snapshot"`
	Entries   []Entry `json:"entries"`
	Ready     int     `json:"ready"`
}

// Result 逐项报告，已成功项目可在重试时安全跳过。
type Result struct {
	Entries  []Entry `json:"entries"`
	Imported int     `json:"imported"`
	Failed   int     `json:"failed"`
}

// Service 通过现有账号服务保存凭据，保留逐账号事务边界。
type Service struct {
	db     *sql.DB
	store  account.CredentialStore
	decode func(string) ([]byte, error)
}

// New 创建只在用户点击时执行的迁移服务。
func New(db *sql.DB, store account.CredentialStore) *Service {
	return &Service{db: db, store: store, decode: legacy.Unprotect}
}

type plan struct {
	input    account.CreateInput
	identity string
	entry    Entry
}

// Preview 校验源快照和账号，失败项不会阻止其他明确可迁移的项目。
func (s *Service) Preview(ctx context.Context, directory string) (Preview, error) {
	preview, plans, err := s.prepare(ctx, directory)
	for i := range plans {
		plans[i].input.APIKey = ""
	}
	return preview, err
}

// Import 确认源未变化后导入；单个账号失败不会撤销已成功且可幂等重试的项目。
func (s *Service) Import(ctx context.Context, directory, snapshot string) (Result, error) {
	if len(snapshot) != 64 {
		return Result{}, errors.New("legacy: invalid_snapshot")
	}
	preview, plans, err := s.prepare(ctx, directory)
	if err != nil {
		return Result{}, err
	}
	defer func() {
		for i := range plans {
			plans[i].input.APIKey = ""
		}
	}()
	if preview.Snapshot != snapshot {
		return Result{}, errors.New("legacy: source_changed")
	}
	result := Result{Entries: preview.Entries}
	service := account.NewService(s.db, s.store)
	for _, item := range plans {
		if err := ctx.Err(); err != nil {
			return result, errors.New("legacy: cancelled")
		}
		_, err := service.ImportLegacyKey(ctx, item.identity, item.input, item.entry.Enabled)
		if err != nil {
			result.Entries[item.entry.Index-1].State = "failed"
			result.Failed++
			continue
		}
		result.Entries[item.entry.Index-1].State = "imported"
		result.Imported++
	}
	return result, nil
}
