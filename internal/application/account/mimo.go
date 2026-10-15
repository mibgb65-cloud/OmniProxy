package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"omniproxy/internal/provider/mimo"
)

func (s *Service) prepareMiMo(ctx context.Context, providerID string, input *mimo.Options, current *Item) (*mimo.Options, error) {
	var kind string
	if err := s.db.QueryRowContext(ctx, `SELECT kind FROM providers WHERE id=?`, providerID).Scan(&kind); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("account: read MiMo provider: %w", err)
	}
	if kind != mimo.Kind {
		if input != nil {
			return nil, fmt.Errorf("%w: MiMo options for other provider", ErrInvalidInput)
		}
		return nil, nil
	}
	if current != nil && input == nil {
		return current.MiMo, nil
	}
	o := mimo.Options{Region: "", BillingMode: "payg"}
	if input != nil {
		o = *input
	}
	if !o.Valid() {
		return nil, fmt.Errorf("%w: MiMo options", ErrInvalidInput)
	}
	expected := int64(0)
	if current != nil && current.MiMo != nil {
		expected = current.MiMo.Version
	}
	if o.Version != expected {
		return nil, ErrConflict
	}
	return &o, nil
}

func saveMiMo(ctx context.Context, tx *sql.Tx, id string, o *mimo.Options) error {
	if o == nil {
		return nil
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO mimo_account_options(account_id,region,billing_mode,config_version) VALUES(?,?,?,1)
ON CONFLICT(account_id) DO UPDATE SET region=excluded.region,billing_mode=excluded.billing_mode,config_version=config_version+1
WHERE mimo_account_options.config_version=?`, id, o.Region, o.BillingMode, o.Version)
	if err != nil {
		return fmt.Errorf("account: save MiMo options: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("account: MiMo update result: %w", err)
	}
	if changed != 1 {
		return ErrConflict
	}
	o.Version++
	return nil
}

func validateMiMoKey(options *mimo.Options, key []byte) error {
	if options != nil && !mimo.ValidKeyForOptions(*options, key) {
		return fmt.Errorf("%w: MiMo API Key format", ErrInvalidInput)
	}
	return nil
}
