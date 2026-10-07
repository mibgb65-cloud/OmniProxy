package wails

import (
	"context"
	"fmt"
)

// AccountKeyPreview 返回密钥脱敏文本，不返回完整密钥或凭证引用。
func (s *AppService) AccountKeyPreview(id string) (string, error) {
	service, err := s.accountService()
	if err != nil {
		return "", err
	}
	return service.KeyPreview(context.Background(), id)
}

// CopyAccountKey 由桌面端直接写入剪贴板，完整密钥不经过前端或查询缓存。
func (s *AppService) CopyAccountKey(id string) error {
	service, err := s.accountService()
	if err != nil {
		return err
	}
	s.clipboardMu.Lock()
	defer s.clipboardMu.Unlock()
	return service.UseAPIKey(context.Background(), id, func(key []byte) error {
		if s.copyText == nil || !s.copyText(string(key)) {
			return fmt.Errorf("account: clipboard unavailable")
		}
		return nil
	})
}
