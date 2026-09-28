package wails

import (
	"context"

	"omniproxy/internal/application/provider"
)

// ProviderCatalog returns the built-in provider kinds and their auth options.
func (s *AppService) ProviderCatalog() []provider.Definition {
	return provider.Catalog()
}

// ListProviders returns saved provider configurations.
func (s *AppService) ListProviders() ([]provider.Configuration, error) {
	return provider.NewService(s.db).List(context.Background())
}

// CreateProvider saves one provider configuration.
func (s *AppService) CreateProvider(input provider.CreateInput) (provider.Configuration, error) {
	return provider.NewService(s.db).Create(context.Background(), input)
}

// UpdateProvider updates one provider configuration.
func (s *AppService) UpdateProvider(id string, input provider.UpdateInput) (provider.Configuration, error) {
	return provider.NewService(s.db).Update(context.Background(), id, input)
}

// SetProviderEnabled changes the enabled state of a built-in or custom provider.
func (s *AppService) SetProviderEnabled(id string, enabled bool) (provider.Configuration, error) {
	return provider.NewService(s.db).SetEnabled(context.Background(), id, enabled)
}

// DeleteProvider removes a provider configuration without accounts.
func (s *AppService) DeleteProvider(id string) error {
	return provider.NewService(s.db).Delete(context.Background(), id)
}

// ProbeProvider measures endpoint reachability and response latency without credentials.
func (s *AppService) ProbeProvider(id string) (provider.ProbeResult, error) {
	return provider.NewService(s.db).Probe(context.Background(), id)
}
