package gateway

import (
	"context"
	"database/sql"

	"omniproxy/internal/core/proxy"
)

type RiverSettings struct {
	AccountRotationMode string `json:"accountRotationMode"`
}

type RiverSettingsService struct{ db *sql.DB }

func NewRiverSettingsService(db *sql.DB) *RiverSettingsService { return &RiverSettingsService{db: db} }

func (s *RiverSettingsService) Read(ctx context.Context) (RiverSettings, error) {
	var value RiverSettings
	if s.db == nil || s.db.QueryRowContext(ctx, `SELECT account_rotation_mode FROM river_settings WHERE id=1`).Scan(&value.AccountRotationMode) != nil ||
		!validRotationMode(value.AccountRotationMode) {
		return RiverSettings{}, proxy.Failure("configuration_unavailable", 503)
	}
	return value, nil
}

func (s *RiverSettingsService) SetAccountRotationMode(ctx context.Context, mode string) (RiverSettings, error) {
	if !validRotationMode(mode) {
		return RiverSettings{}, proxy.Failure("invalid_rotation_mode", 400)
	}
	if s.db == nil {
		return RiverSettings{}, proxy.Failure("configuration_unavailable", 503)
	}
	result, err := s.db.ExecContext(ctx, `UPDATE river_settings SET account_rotation_mode=? WHERE id=1`, mode)
	if err != nil {
		return RiverSettings{}, proxy.Failure("settings_save_failed", 503)
	}
	if count, err := result.RowsAffected(); err != nil || count != 1 {
		return RiverSettings{}, proxy.Failure("settings_save_failed", 503)
	}
	return s.Read(ctx)
}

func (s *RiverSettingsService) AccountRotationMode(ctx context.Context) (string, error) {
	value, err := s.Read(ctx)
	return value.AccountRotationMode, err
}
