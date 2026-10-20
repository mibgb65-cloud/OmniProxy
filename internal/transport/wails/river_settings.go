package wails

import (
	"context"

	"omniproxy/internal/application/gateway"
)

func (s *AppService) GetRiverSettings() (gateway.RiverSettings, error) {
	return gateway.NewRiverSettingsService(s.db).Read(context.Background())
}

func (s *AppService) SetRiverAccountRotationMode(mode string) (gateway.RiverSettings, error) {
	return gateway.NewRiverSettingsService(s.db).SetAccountRotationMode(context.Background(), mode)
}
