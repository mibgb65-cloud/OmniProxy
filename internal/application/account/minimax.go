package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"omniproxy/internal/provider/minimax"
)

func (s *Service) prepareMiniMax(ctx context.Context, providerID string, input *minimax.Options, current *Item) (*minimax.Options, error) {
	var kind string
	if err := s.db.QueryRowContext(ctx, `SELECT kind FROM providers WHERE id=?`, providerID).Scan(&kind); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("account: read MiniMax provider: %w", err)
	}
	if kind != minimax.Kind {
		if input != nil {
			return nil, fmt.Errorf("%w: MiniMax options for other provider", ErrInvalidInput)
		}
		return nil, nil
	}
	if current != nil && input == nil {
		return current.MiniMax, nil
	}
	o := minimax.Options{Region: "cn", BillingMode: "payg"}
	if input != nil {
		o = *input
	}
	if !o.Valid() {
		return nil, fmt.Errorf("%w: MiniMax options", ErrInvalidInput)
	}
	expected := int64(0)
	if current != nil && current.MiniMax != nil {
		expected = current.MiniMax.Version
	}
	if o.Version != expected {
		return nil, ErrConflict
	}
	return &o, nil
}

func saveMiniMax(ctx context.Context, tx *sql.Tx, id string, o *minimax.Options) error {
	if o == nil {
		return nil
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO minimax_account_options(account_id,region,billing_mode,config_version) VALUES(?,?,?,1)
ON CONFLICT(account_id) DO UPDATE SET region=excluded.region,billing_mode=excluded.billing_mode,config_version=config_version+1
WHERE minimax_account_options.config_version=?`, id, o.Region, o.BillingMode, o.Version)
	if err != nil {
		return fmt.Errorf("account: save MiniMax options: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("account: MiniMax update result: %w", err)
	}
	if changed != 1 {
		return ErrConflict
	}
	o.Version++
	return nil
}

func validateMiniMaxKey(options *minimax.Options, key []byte) error {
	if options != nil && !minimax.ValidKeyForOptions(*options, key) {
		return fmt.Errorf("%w: MiniMax API Key format", ErrInvalidInput)
	}
	return nil
}
