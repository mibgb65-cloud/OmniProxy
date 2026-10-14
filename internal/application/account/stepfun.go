package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"omniproxy/internal/provider/stepfun"
)

func (s *Service) prepareStep(ctx context.Context, providerID string, input *stepfun.Options, current *Item) (*stepfun.Options, error) {
	var kind string
	if err := s.db.QueryRowContext(ctx, `SELECT kind FROM providers WHERE id=?`, providerID).Scan(&kind); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("account: read Step provider: %w", err)
	}
	if kind != stepfun.Kind {
		if input != nil {
			return nil, fmt.Errorf("%w: Step options for other provider", ErrInvalidInput)
		}
		return nil, nil
	}
	if current != nil && input == nil {
		return current.StepFun, nil
	}
	o := stepfun.Options{Region: "cn", BillingMode: "payg"}
	if input != nil {
		o = *input
	}
	if !o.Valid() {
		return nil, fmt.Errorf("%w: Step options", ErrInvalidInput)
	}
	expected := int64(0)
	if current != nil && current.StepFun != nil {
		expected = current.StepFun.Version
	}
	if o.Version != expected {
		return nil, ErrConflict
	}
	return &o, nil
}

func saveStep(ctx context.Context, tx *sql.Tx, id string, o *stepfun.Options) error {
	if o == nil {
		return nil
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO stepfun_account_options(account_id,region,billing_mode,config_version) VALUES(?,?,?,1)
ON CONFLICT(account_id) DO UPDATE SET region=excluded.region,billing_mode=excluded.billing_mode,config_version=config_version+1
WHERE stepfun_account_options.config_version=?`, id, o.Region, o.BillingMode, o.Version)
	if err != nil {
		return fmt.Errorf("account: save Step options: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("account: Step update result: %w", err)
	}
	if changed != 1 {
		return ErrConflict
	}
	o.Version++
	return nil
}

func validateStepKey(options *stepfun.Options, key []byte) error {
	if options != nil && !stepfun.ValidAPIKey(key) {
		return fmt.Errorf("%w: Step API Key format", ErrInvalidInput)
	}
	return nil
}
