package wails

import (
	"context"
	"errors"
)

// OpenStepConsole 从账号公开区域解析固定官方页面，不接受用户 URL 或密钥。
func (s *AppService) OpenStepConsole(id string) error {
	service, err := s.accountService()
	if err != nil {
		return err
	}
	item, err := service.Metadata(context.Background(), id)
	if err != nil || item.StepFun == nil || !item.StepFun.Valid() {
		return errors.New("stepfun: invalid_configuration")
	}
	address := "https://platform.stepfun.com/step-plan"
	if item.StepFun.Region == "global" {
		address = "https://platform.stepfun.ai/step-plan"
	}
	if s.openAgentURL == nil {
		return errors.New("stepfun: browser_unavailable")
	}
	return s.openAgentURL(address)
}
