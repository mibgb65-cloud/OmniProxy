package account

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// LegacyID 不含凭据，以旧账号身份确定迁移后的 ID，重试不会覆盖现有账号。
func LegacyID(identity string) string {
	digest := sha256.Sum256([]byte(identity))
	return "acct_v1_" + hex.EncodeToString(digest[:16])
}

// ValidateLegacyKey 复用创建账号的关键输入和计费通道验证，不写入凭据或数据库。
func (s *Service) ValidateLegacyKey(ctx context.Context, input CreateInput) error {
	key, err := validateKey(input.APIKey)
	if err != nil {
		return err
	}
	defer clear(key)
	if _, err := resolveName(input.Name, key); err != nil {
		return err
	}
	step, err := s.prepareStep(ctx, input.ProviderID, input.StepFun, nil)
	if err != nil {
		return err
	}
	if err := validateStepKey(step, key); err != nil {
		return err
	}
	kimi, err := s.prepareKimi(ctx, input.ProviderID, input.Kimi, nil)
	if err != nil {
		return err
	}
	if err := validateKimiKey(kimi, key); err != nil {
		return err
	}
	mimo, err := s.prepareMiMo(ctx, input.ProviderID, input.MiMo, nil)
	if err != nil {
		return err
	}
	if err := validateMiMoKey(mimo, key); err != nil {
		return err
	}
	minimax, err := s.prepareMiniMax(ctx, input.ProviderID, input.MiniMax, nil)
	if err != nil {
		return err
	}
	return validateMiniMaxKey(minimax, key)
}

// ImportLegacyKey 保留名称和启停；事务及失败清理沿用普通创建流程。
func (s *Service) ImportLegacyKey(ctx context.Context, identity string, input CreateInput, enabled bool) (Item, error) {
	if len(identity) == 0 || len(identity) > 4096 {
		return Item{}, ErrInvalidInput
	}
	id := LegacyID(identity)
	item, err := s.Metadata(ctx, id)
	if err == nil {
		return item, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Item{}, err
	}
	return s.create(ctx, input, id, enabled)
}
