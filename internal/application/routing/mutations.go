package routing

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	core "omniproxy/internal/core/routing"
)

// Create 保存精确映射，SQLite 唯一约束处理并发创建同名规则。
func (s *Service) Create(ctx context.Context, input Input) (core.Rule, error) {
	data, err := s.snapshot(ctx)
	if err != nil {
		return core.Rule{}, err
	}
	if err := validateInput(input, data); err != nil {
		return core.Rule{}, err
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return core.Rule{}, fmt.Errorf("routing: cannot allocate id")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	rule := core.Rule{ID: "route_" + hex.EncodeToString(random[:]), Alias: input.Alias, ProviderID: input.ProviderID, Model: input.Model, AccountID: input.AccountID, Enabled: input.Enabled, CreatedAt: now, UpdatedAt: now}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return core.Rule{}, fmt.Errorf("routing: save failed")
	}
	defer tx.Rollback()
	var count int
	if tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM model_routes`).Scan(&count) != nil {
		return core.Rule{}, fmt.Errorf("routing: save failed")
	}
	if count >= maxRules {
		return core.Rule{}, ErrLimitReached
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO model_routes(id,alias,provider_id,model,account_id,enabled,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, rule.ID, rule.Alias, rule.ProviderID, rule.Model, rule.AccountID, rule.Enabled, rule.CreatedAt, rule.UpdatedAt)
	if err != nil {
		return core.Rule{}, s.writeError(ctx, tx, input.Alias, "")
	}
	if tx.Commit() != nil {
		return core.Rule{}, fmt.Errorf("routing: save failed")
	}
	return rule, nil
}

// Update 修改目标与别名，保留规则 ID 和创建时间。
func (s *Service) Update(ctx context.Context, id string, input Input) (core.Rule, error) {
	if validateID(id, false) != nil {
		return core.Rule{}, ErrInvalidInput
	}
	data, err := s.snapshot(ctx)
	if err != nil {
		return core.Rule{}, err
	}
	if err := validateInput(input, data); err != nil {
		return core.Rule{}, err
	}
	rule, err := s.get(ctx, id)
	if err != nil {
		return core.Rule{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := s.db.ExecContext(ctx, `UPDATE model_routes SET alias=?,provider_id=?,model=?,account_id=?,enabled=?,updated_at=? WHERE id=?`, input.Alias, input.ProviderID, input.Model, input.AccountID, input.Enabled, now, id)
	if err != nil {
		return core.Rule{}, s.writeError(ctx, s.db, input.Alias, id)
	}
	if affected, err := result.RowsAffected(); err != nil || affected == 0 {
		return core.Rule{}, ErrNotFound
	}
	rule.Alias, rule.ProviderID, rule.Model, rule.AccountID, rule.Enabled, rule.UpdatedAt = input.Alias, input.ProviderID, input.Model, input.AccountID, input.Enabled, now
	return rule, nil
}

// SetEnabled 修改开关；目标变化由列表和解析实时诊断。
func (s *Service) SetEnabled(ctx context.Context, id string, enabled bool) (core.Rule, error) {
	if validateID(id, false) != nil {
		return core.Rule{}, ErrInvalidInput
	}
	rule, err := s.get(ctx, id)
	if err != nil {
		return core.Rule{}, err
	}
	rule.Enabled = enabled
	rule.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	result, err := s.db.ExecContext(ctx, `UPDATE model_routes SET enabled=?,updated_at=? WHERE id=?`, enabled, rule.UpdatedAt, id)
	if err != nil {
		return core.Rule{}, fmt.Errorf("routing: save failed")
	}
	if affected, err := result.RowsAffected(); err != nil || affected == 0 {
		return core.Rule{}, ErrNotFound
	}
	return rule, nil
}

// Delete 只删除映射，不删除提供商、账号或凭证。
func (s *Service) Delete(ctx context.Context, id string) error {
	if validateID(id, false) != nil {
		return ErrInvalidInput
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM model_routes WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("routing: delete failed")
	}
	if affected, err := result.RowsAffected(); err != nil || affected == 0 {
		return ErrNotFound
	}
	return nil
}

type queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (s *Service) writeError(ctx context.Context, db queryer, alias, id string) error {
	var count int
	if db.QueryRowContext(ctx, `SELECT COUNT(*) FROM model_routes WHERE alias=? AND id<>?`, alias, id).Scan(&count) == nil && count > 0 {
		return ErrDuplicateAlias
	}
	return fmt.Errorf("routing: save failed")
}
