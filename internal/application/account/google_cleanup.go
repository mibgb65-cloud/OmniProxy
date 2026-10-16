package account

import (
	"encoding/json"
	"errors"

	"omniproxy/internal/infrastructure/credential"
	"omniproxy/internal/provider/antigravity"
)

type protectedGoogleCredential struct {
	antigravity.Credentials
	Retired []string `json:"retired,omitempty"`
}

func missingGoogleCredential(err error) bool {
	return errors.Is(err, credential.ErrNotFound) || errors.Is(err, ErrNotFound)
}

func (s *Service) googleRetired(ref string) ([]string, error) {
	if ref == "" {
		return nil, nil
	}
	data, err := s.store.Get(ref)
	if missingGoogleCredential(err) {
		return nil, nil
	}
	if err != nil {
		return nil, antigravity.Failure("cleanup_failed")
	}
	defer clear(data)
	var bundle protectedGoogleCredential
	if len(data) > 65536 || json.Unmarshal(data, &bundle) != nil || len(bundle.Retired) > 16 {
		return nil, antigravity.Failure("cleanup_failed")
	}
	seen := map[string]bool{ref: true}
	for _, retired := range bundle.Retired {
		if validateID(retired) != nil || seen[retired] {
			return nil, antigravity.Failure("cleanup_failed")
		}
		seen[retired] = true
	}
	return bundle.Retired, nil
}

func (s *Service) cleanupGoogleReferences(refs []string) error {
	for _, ref := range refs {
		if err := s.store.Delete(ref); err != nil && !missingGoogleCredential(err) {
			return antigravity.Failure("cleanup_failed")
		}
	}
	return nil
}
