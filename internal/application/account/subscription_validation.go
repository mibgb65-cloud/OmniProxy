package account

import (
	"unicode"
	"unicode/utf8"
)

func validateSubscription(meta SubscriptionMetadata) error {
	if len(meta.AccountID) > 128 {
		return ErrInvalidInput
	}
	for _, char := range meta.AccountID {
		if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '-' || char == '_') {
			return ErrInvalidInput
		}
	}
	if len(meta.Email) > 320 || len(meta.Plan) > 80 || !utf8.ValidString(meta.Email+meta.Plan) {
		return ErrInvalidInput
	}
	for _, char := range meta.Email + meta.Plan {
		if unicode.IsControl(char) {
			return ErrInvalidInput
		}
	}
	if meta.State != "authenticated" && meta.State != "logged_out" && meta.State != "reauth_required" && meta.State != "unknown" {
		return ErrInvalidInput
	}
	return nil
}
