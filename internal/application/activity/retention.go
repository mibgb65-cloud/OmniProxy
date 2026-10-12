package activity

import (
	"context"
	"database/sql"
)

func prune(ctx context.Context, tx *sql.Tx) error {
	var limit int
	if err := tx.QueryRowContext(ctx, `SELECT max_records FROM request_retention WHERE id=1`).Scan(&limit); err != nil {
		return err
	}
	var total int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM request_records`).Scan(&total); err != nil {
		return err
	}
	if total <= limit {
		return nil
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM request_records WHERE id IN (SELECT id FROM request_records WHERE state!='pending' ORDER BY started_at,id LIMIT ?)`, total-limit)
	return err
}
func (s *Service) Retention(ctx context.Context) (Retention, error) {
	result := Retention{LogCapacity: 5000}
	if s.db.QueryRowContext(ctx, `SELECT max_records FROM request_retention WHERE id=1`).Scan(&result.MaxRecords) != nil {
		return result, ErrUnavailable
	}
	if s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM request_records`).Scan(&result.StoredRecords) != nil {
		return result, ErrUnavailable
	}
	return result, nil
}
func (s *Service) SetRetention(ctx context.Context, limit int) (Retention, error) {
	if limit < MinRetention || limit > MaxRetention {
		return Retention{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Retention{}, ErrUnavailable
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE request_retention SET max_records=? WHERE id=1`, limit); err != nil {
		return Retention{}, ErrUnavailable
	}
	if prune(ctx, tx) != nil || tx.Commit() != nil {
		return Retention{}, ErrUnavailable
	}
	return s.Retention(ctx)
}
