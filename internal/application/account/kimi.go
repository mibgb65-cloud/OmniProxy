package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"omniproxy/internal/provider/kimi"
)

func (s *Service) prepareKimi(ctx context.Context, providerID string, input *kimi.Options, current *Item) (*kimi.Options, error) {
	var kind string
	if err := s.db.QueryRowContext(ctx, `SELECT kind FROM providers WHERE id=?`, providerID).Scan(&kind); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("account: read Kimi provider: %w", err)
	}
	if kind != kimi.Kind {
		if input != nil {
			return nil, fmt.Errorf("%w: Kimi options for other provider", ErrInvalidInput)
		}
		return nil, nil
	}
	if current != nil && input == nil {
		return current.Kimi, nil
	}
	o := kimi.Options{Region: "cn"}
	if input != nil {
		o = *input
	}
	if !o.Valid() {
		return nil, fmt.Errorf("%w: Kimi platform", ErrInvalidInput)
	}
	expected := int64(0)
	if current != nil && current.Kimi != nil {
		expected = current.Kimi.Version
	}
	if o.Version != expected {
		return nil, ErrConflict
	}
	return &o, nil
}

func saveKimi(ctx context.Context, tx *sql.Tx, id string, o *kimi.Options) error {
	if o == nil {
		return nil
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO kimi_account_options(account_id,region,config_version)VALUES(?,?,1)
ON CONFLICT(account_id) DO UPDATE SET region=excluded.region,config_version=config_version+1
WHERE kimi_account_options.config_version=?`, id, o.Region, o.Version)
	if err != nil {
		return fmt.Errorf("account: save Kimi platform: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("account: Kimi update result: %w", err)
	}
	if count != 1 {
		return ErrConflict
	}
	o.Version++
	return nil
}

func validateKimiKey(o *kimi.Options, key []byte) error {
	if o != nil && !kimi.ValidAPIKey(key) {
		return fmt.Errorf("%w: Kimi API Key format", ErrInvalidInput)
	}
	return nil
}
