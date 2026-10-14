package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"omniproxy/internal/provider/kimi"
	"omniproxy/internal/provider/stepfun"
)

func (s *Service) get(ctx context.Context, id string) (Item, string, error) {
	var item Item
	var ref string
	var options stepfun.Options
	var kimiOptions kimi.Options
	err := s.db.QueryRowContext(ctx, `
		SELECT a.id, a.provider_id, a.name, a.enabled, a.created_at, a.updated_at, a.credential_ref, a.auth_method, a.auth_email, a.auth_plan, a.auth_state, a.auth_account_id,
		COALESCE(s.region,''),COALESCE(s.billing_mode,''),COALESCE(s.config_version,0),COALESCE(k.region,''),COALESCE(k.config_version,0)
		FROM accounts a LEFT JOIN stepfun_account_options s ON s.account_id=a.id LEFT JOIN kimi_account_options k ON k.account_id=a.id WHERE a.id = ?`, id).Scan(
		&item.ID, &item.ProviderID, &item.Name, &item.Enabled, &item.CreatedAt, &item.UpdatedAt, &ref, &item.AuthMethod, &item.AuthEmail, &item.AuthPlan, &item.AuthState, &item.AuthAccountID, &options.Region, &options.BillingMode, &options.Version, &kimiOptions.Region, &kimiOptions.Version,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, "", ErrNotFound
	}
	if err != nil {
		return Item{}, "", fmt.Errorf("account: read: %w", err)
	}
	if options.Region != "" {
		item.StepFun = &options
	}
	if kimiOptions.Region != "" {
		item.Kimi = &kimiOptions
	}
	return item, ref, nil
}

type scanner interface{ Scan(dest ...any) error }

func scanItem(row scanner) (Item, error) {
	var item Item
	var options stepfun.Options
	var kimiOptions kimi.Options
	if err := row.Scan(&item.ID, &item.ProviderID, &item.Name, &item.Enabled, &item.CreatedAt, &item.UpdatedAt, &item.AuthMethod, &item.AuthEmail, &item.AuthPlan, &item.AuthState, &item.AuthAccountID, &options.Region, &options.BillingMode, &options.Version, &kimiOptions.Region, &kimiOptions.Version); err != nil {
		return Item{}, fmt.Errorf("account: scan: %w", err)
	}
	if options.Region != "" {
		item.StepFun = &options
	}
	if kimiOptions.Region != "" {
		item.Kimi = &kimiOptions
	}
	return item, nil
}
