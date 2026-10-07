package account

import (
	"context"
	"fmt"
)

const (
	visibleKeyRunes     = 4
	keyPreviewThreshold = defaultNameRunes + 2*visibleKeyRunes
	keyMask             = "••••••••"
)

// UseAPIKey 只在受控回调内使用密钥，完成后清理读取的字节，不返回完整密钥。
func (s *Service) UseAPIKey(ctx context.Context, id string, consume func([]byte) error) error {
	if err := validateID(id); err != nil {
		return err
	}
	if consume == nil {
		return fmt.Errorf("%w: key consumer", ErrInvalidInput)
	}
	_, ref, err := s.get(ctx, id)
	if err != nil {
		return err
	}
	if ref == "" {
		return ErrNotFound
	}
	secret, err := s.store.Get(ref)
	if err != nil {
		return fmt.Errorf("account: read protected key: %w", err)
	}
	defer clear(secret)
	key, err := validateKey(string(secret))
	if err != nil {
		return fmt.Errorf("account: invalid protected key: %w", err)
	}
	defer clear(key)
	if err := ctx.Err(); err != nil {
		return err
	}
	return consume(key)
}

// KeyPreview 从真实凭证生成脱敏展示；较短的密钥全部遮掩，避免组合片段还原密钥。
func (s *Service) KeyPreview(ctx context.Context, id string) (string, error) {
	preview := ""
	err := s.UseAPIKey(ctx, id, func(key []byte) error {
		runes := []rune(string(key))
		defer clear(runes)
		preview = keyMask
		// 自动名称可能已显示前 8 个字符，短密钥不再追加尾部可识别片段。
		if len(runes) > keyPreviewThreshold {
			preview = string(runes[:visibleKeyRunes]) + keyMask + string(runes[len(runes)-visibleKeyRunes:])
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return preview, nil
}
