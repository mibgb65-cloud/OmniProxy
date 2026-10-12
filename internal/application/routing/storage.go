package routing

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	core "omniproxy/internal/core/routing"
)

func (s *Service) rules(ctx context.Context) ([]core.Rule, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,alias,provider_id,model,account_id,enabled,created_at,updated_at FROM model_routes ORDER BY alias COLLATE BINARY LIMIT ?`, maxRules+1)
	if err != nil {
		return nil, fmt.Errorf("routing: configuration unavailable")
	}
	defer rows.Close()
	rules := make([]core.Rule, 0)
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("routing: configuration unavailable")
	}
	if len(rules) > maxRules {
		return nil, ErrLimitReached
	}
	return rules, nil
}

func (s *Service) get(ctx context.Context, id string) (core.Rule, error) {
	rule, err := scanRule(s.db.QueryRowContext(ctx, `SELECT id,alias,provider_id,model,account_id,enabled,created_at,updated_at FROM model_routes WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return core.Rule{}, ErrNotFound
	}
	return rule, err
}

type scanner interface{ Scan(...any) error }

func scanRule(row scanner) (core.Rule, error) {
	var rule core.Rule
	if err := row.Scan(&rule.ID, &rule.Alias, &rule.ProviderID, &rule.Model, &rule.AccountID, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return core.Rule{}, err
		}
		return core.Rule{}, fmt.Errorf("routing: configuration unavailable")
	}
	return rule, nil
}
