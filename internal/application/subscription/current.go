package subscription

import (
	"unicode"
	"unicode/utf8"
)

// Current 恢复同一账号操作的进行中授权；不存在时返回空状态。
func (s *Service) Current(providerID, accountID string) (State, error) {
	if providerID != "builtin_openai" || len(accountID) > 128 || !utf8.ValidString(accountID) {
		return State{}, failure("invalid_input")
	}
	for _, r := range accountID {
		if unicode.IsControl(r) {
			return State{}, failure("invalid_input")
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		return State{}, nil
	}
	if s.current.input.ProviderID != providerID || s.current.input.AccountID != accountID {
		return State{}, failure("busy")
	}
	return s.current.view, nil
}
