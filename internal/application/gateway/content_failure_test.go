package gateway

import (
	"context"
	"errors"
	"testing"

	"omniproxy/internal/core/proxy"
)

func TestClaudeImageToolResultFailsBeforeProviderCredentialOrUpstreamAccess(t *testing.T) {
	request := proxy.Request{Protocol: "messages", Body: []byte(`{"model":"deepseek-flash","messages":[{"role":"user","content":[{"type":"tool_result","tool_use_id":"screenshot","content":[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"private-image-marker"}}]}]},{"role":"user","content":"text-only follow-up"}]}`)}
	for _, kind := range []string{"river", "deepseek"} {
		t.Run(kind, func(t *testing.T) {
			// 不安装凭证服务：若错误延迟至密钥读取／上游转发，本回归会直接失败。
			service := &Service{}
			err := service.forwardAgent(context.Background(), target{kind: kind}, request, func(proxy.Response) error {
				t.Fatal("image-bearing history was silently sent upstream")
				return nil
			})
			var failure *proxy.Error
			if !errors.As(err, &failure) || failure.Code != "unsupported_content" || failure.StatusCode != 400 {
				t.Fatal("image capability boundary lost", err)
			}
		})
	}
}
