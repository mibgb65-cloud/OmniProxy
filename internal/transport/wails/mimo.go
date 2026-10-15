package wails

import (
	"context"
	"errors"
)

// OpenMiMoConsole 只打开固定官方套餐页面，不接收任意地址或读取 Key。
func (s *AppService) OpenMiMoConsole(id string) error {
	accounts, err := s.accountService()
	if err != nil {
		return err
	}
	item, err := accounts.Metadata(context.Background(), id)
	if err != nil || item.MiMo == nil || !item.MiMo.Valid() || item.MiMo.BillingMode != "token_plan" {
		return errors.New("mimo: invalid_configuration")
	}
	if s.openAgentURL == nil {
		return errors.New("mimo: browser_unavailable")
	}
	return s.openAgentURL("https://platform.xiaomimimo.com/token-plan")
}
