package account

import (
	"context"
	"testing"

	"omniproxy/internal/core/netproxy"
)

// 额度、目录与续期可能没有账号之外的上下文；账号项必须能给出提供商身份。
func TestOutboundContextCarriesProvider(t *testing.T) {
	ctx := Item{ProviderID: "builtin_google"}.OutboundContext(context.Background())
	scope, ok := netproxy.ScopeFrom(ctx)
	if !ok || scope.ProviderID != "builtin_google" {
		t.Fatalf("outbound scope = %+v, ok = %v", scope, ok)
	}
	// 空提供商标识不应伪造一个作用域。
	if _, ok := netproxy.ScopeFrom((Item{}).OutboundContext(context.Background())); ok {
		t.Fatal("empty provider created a scope")
	}
}
