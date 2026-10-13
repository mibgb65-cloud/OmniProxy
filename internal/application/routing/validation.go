package routing

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	core "omniproxy/internal/core/routing"
	"omniproxy/internal/provider/deepseek"
)

func validateID(id string, optional bool) error {
	if optional && id == "" {
		return nil
	}
	if len(id) == 0 || len(id) > 128 || !utf8.ValidString(id) || strings.TrimSpace(id) != id {
		return ErrInvalidInput
	}
	for _, char := range id {
		if unicode.IsControl(char) {
			return ErrInvalidInput
		}
	}
	return nil
}

func supportedModel(model string) bool {
	for _, id := range deepseek.Models() {
		if model == id {
			return true
		}
	}
	return false
}

func validateInput(input Input, data snapshot) error {
	if !core.ValidModel(input.Alias) || !core.ValidModel(input.Model) || validateID(input.ProviderID, false) != nil || validateID(input.AccountID, true) != nil {
		return ErrInvalidInput
	}
	found := false
	for _, provider := range data.providers {
		if provider.ID == input.ProviderID {
			found = supportedProvider(provider.Kind)
			break
		}
	}
	if !found || !targetSupports(data, input.ProviderID, input.AccountID, input.Model) {
		return fmt.Errorf("routing: invalid target")
	}
	if input.AccountID != "" {
		for _, account := range data.accounts {
			if account.ID == input.AccountID && account.ProviderID == input.ProviderID {
				return nil
			}
		}
		return fmt.Errorf("routing: invalid account")
	}
	return nil
}
