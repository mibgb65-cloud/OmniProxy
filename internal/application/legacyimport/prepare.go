package legacyimport

import (
	"context"
	"errors"
	"omniproxy/internal/application/account"
	"omniproxy/internal/infrastructure/legacy"
	"strings"
)

func (s *Service) prepare(ctx context.Context, directory string) (Preview, []plan, error) {
	source, err := legacy.Read(directory)
	if err != nil {
		return Preview{}, nil, err
	}
	known, err := s.knownKeys(ctx)
	if err != nil {
		return Preview{}, nil, err
	}
	preview := Preview{Directory: source.Directory, Snapshot: source.Snapshot, Entries: make([]Entry, 0, len(source.Tokens))}
	plans := make([]plan, 0, len(source.Tokens))
	seenIDs := make(map[string]bool)
	service := account.NewService(s.db, s.store)
	for i, token := range source.Tokens {
		entry := Entry{Index: i + 1, Name: displayName(token.Name, i+1), Provider: displayProvider(token.Provider), State: "invalid", Enabled: !token.Disabled}
		input, state := mapToken(token, source.Endpoints)
		entry.State = state
		identity := token.Provider + ":" + token.CredentialType + ":" + token.ID
		if token.ID == "" || len(token.ID) > 128 || len(token.Name) > 256 {
			entry.State = "invalid"
		}
		if entry.State == "ready" && seenIDs[identity] {
			entry.State = "duplicate"
		}
		if entry.State == "ready" {
			if _, err := service.Metadata(ctx, account.LegacyID(identity)); err == nil {
				entry.State = "duplicate"
			} else if !errors.Is(err, account.ErrNotFound) {
				return Preview{}, nil, errors.New("legacy: target_unavailable")
			}
		}
		if entry.State == "ready" {
			secret, err := s.decode(token.Value)
			if err != nil {
				entry.State = "credential_unavailable"
			} else {
				input.APIKey = string(secret)
				clear(secret)
				if input.APIKey != "" && strings.Contains(entry.Name, input.APIKey) {
					entry.Name = displayName("", i+1)
				}
				input.Name = entry.Name
				fingerprint := keyFingerprint(input.ProviderID, input.APIKey)
				if known[fingerprint] {
					entry.State = "duplicate"
				} else if service.ValidateLegacyKey(ctx, input) != nil {
					entry.State = "invalid"
				} else {
					known[fingerprint] = true
					seenIDs[identity] = true
					plans = append(plans, plan{input: input, identity: identity, entry: entry})
					preview.Ready++
				}
			}
		}
		preview.Entries = append(preview.Entries, entry)
	}
	return preview, plans, nil
}
