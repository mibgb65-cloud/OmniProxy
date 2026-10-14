package wails

import (
	"context"
	"omniproxy/internal/application/googlequota"
)

// GoogleResources 仅读取项目绑定与本机观察，不调用推理或冒充官方剩余查询。
func (s *AppService) GoogleResources() ([]googlequota.Resource, error) {
	if s.db == nil {
		return nil, googlequota.ErrUnavailable
	}
	return googlequota.New(s.db).List(context.Background())
}

// SetGoogleProject 保存用户声明的项目，不修改 Google Cloud 项目或 Key。
func (s *AppService) SetGoogleProject(input googlequota.BindingInput) error {
	if s.db == nil {
		return googlequota.ErrUnavailable
	}
	return googlequota.New(s.db).Bind(context.Background(), input)
}

// OpenGoogleConsole 仅打开固定官方页面，不接受任意 URL。
func (s *AppService) OpenGoogleConsole(page string) error {
	address := ""
	switch page {
	case "usage":
		address = "https://aistudio.google.com/usage"
	case "billing":
		address = "https://aistudio.google.com/billing"
	default:
		return googlequota.ErrInvalid
	}
	if s.openAgentURL == nil {
		return googlequota.ErrUnavailable
	}
	if s.openAgentURL(address) != nil {
		return googlequota.ErrUnavailable
	}
	return nil
}
