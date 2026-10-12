package subscription

import (
	"context"
	"omniproxy/internal/application/account"
	"omniproxy/internal/infrastructure/codex"
	"strings"
)

func sameEmail(expected, actual string) bool {
	return strings.TrimSpace(expected) != "" && strings.EqualFold(strings.TrimSpace(expected), strings.TrimSpace(actual))
}

// 工作区 ID 在 Team 多成员之间共享，授权邮箱也必须匹配。
func verifyMember(ctx context.Context, client codex.Client, item account.Item) error {
	identity, err := client.Account(ctx, false)
	if err != nil || identity == nil {
		return failure("identity_unavailable")
	}
	if !sameEmail(item.AuthEmail, identity.Email) {
		return failure("identity_changed")
	}
	return nil
}
