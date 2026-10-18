package legacyimport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

func keyFingerprint(provider, key string) string {
	digest := sha256.Sum256([]byte(provider + "\x00" + strings.TrimSpace(key)))
	return hex.EncodeToString(digest[:])
}

func (s *Service) knownKeys(ctx context.Context) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT provider_id,credential_ref FROM accounts WHERE auth_method='api_key' LIMIT 10001`)
	if err != nil {
		return nil, errors.New("legacy: target_unavailable")
	}
	defer rows.Close()
	known := make(map[string]bool)
	for rows.Next() {
		var provider, ref string
		if rows.Scan(&provider, &ref) != nil || len(known) > 10000 {
			return nil, errors.New("legacy: target_unavailable")
		}
		key, err := s.store.Get(ref)
		if err != nil {
			return nil, errors.New("legacy: target_credential_unavailable")
		}
		known[keyFingerprint(provider, string(key))] = true
		clear(key)
	}
	if rows.Err() != nil {
		return nil, errors.New("legacy: target_unavailable")
	}
	return known, nil
}
