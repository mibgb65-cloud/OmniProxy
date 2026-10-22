package wails

import "errors"

var (
	errProjectResourceInvalid     = errors.New("project: invalid resource")
	errProjectResourceUnavailable = errors.New("project: browser unavailable")
)

// OpenProjectResource 仅允许固定项目入口，避免前端输入变成任意 URL 或本地协议。
func (s *AppService) OpenProjectResource(resource string) error {
	var address string
	switch resource {
	case "website":
		address = "https://omniproxy.aicnos.com/"
	case "repository":
		address = "https://github.com/mibgb65-cloud/OmniProxy"
	case "disclaimer":
		address = "https://omniproxy.aicnos.com/legal/disclaimer/"
	case "terms":
		address = "https://omniproxy.aicnos.com/legal/terms/"
	case "privacy":
		address = "https://omniproxy.aicnos.com/legal/privacy/"
	default:
		return errProjectResourceInvalid
	}
	if s.openAgentURL == nil || s.openAgentURL(address) != nil {
		// 浏览器底层错误可能包含本地路径或启动参数，不直接传给界面。
		return errProjectResourceUnavailable
	}
	return nil
}
