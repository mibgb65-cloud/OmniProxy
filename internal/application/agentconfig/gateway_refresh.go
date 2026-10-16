package agentconfig

import (
	"context"
	"errors"
	catalog "omniproxy/internal/core/modelcatalog"
)

func (s *Service) refreshGatewayClient(ctx context.Context, client, path string, before fileSnapshot, entries []catalog.Entry) error {
	if len(entries) == 0 {
		return failure("no_models")
	}
	key, err := s.key(ctx, client)
	if err != nil {
		return err
	}
	defer clear(key)
	selected := existingModel(client, before.body)
	found := false
	for _, entry := range entries {
		if entry.ID == selected {
			found = true
		}
	}
	if !found {
		selected = entries[0].ID
	}
	body, _, err := s.buildPlan(client, before.body, key, Plan{Models: entries, DefaultModel: selected})
	if err != nil {
		return err
	}
	defer clear(body)
	if digest(body, true) == before.revision {
		return nil
	}
	if err := s.writeClient(client, path, body, before.revision); err != nil {
		return err
	}
	if err := s.commitFileVersions(ctx, client, digest(body, true), false, ""); err != nil {
		return errors.Join(err, s.rollbackClient(client, path, digest(body, true), before))
	}
	return nil
}
