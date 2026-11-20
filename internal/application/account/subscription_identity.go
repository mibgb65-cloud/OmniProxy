package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// normalizeSubscriptionEmail 统一订阅身份的邮箱比较口径；界面仍显示官方返回值。
func normalizeSubscriptionEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// IdentityAccount 返回与订阅身份相同的已有账号：提供商 + 规范化邮箱 + 官方 account_id。
// 前端「添加账号」不携带 accountId，重新授权同一账号时必须更新这条记录，不能再插入一条，
// 否则网关会把同一账号当成两个候选，目录与额度快照也会各存一份。
// 旧记录可能尚未补全 account_id，此时同提供商同邮箱仍视为同一身份；
// 双方 account_id 都存在且不同（如 Team 工作区的不同成员）时不匹配。
func (s *Service) IdentityAccount(ctx context.Context, providerID, email, accountID string) (Item, bool, error) {
	normalized := normalizeSubscriptionEmail(email)
	identity := strings.TrimSpace(accountID)
	if normalized == "" || identity == "" || validateID(providerID) != nil {
		return Item{}, false, nil
	}
	var id string
	err := s.db.QueryRowContext(ctx, `
		SELECT id FROM accounts
		WHERE provider_id = ? AND auth_method = ? AND lower(trim(auth_email)) = ?
		  AND (auth_account_id = '' OR auth_account_id = ?)
		ORDER BY CASE WHEN auth_account_id = ? THEN 0 ELSE 1 END, updated_at DESC, id DESC
		LIMIT 1`, providerID, AuthSubscription, normalized, identity, identity).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, false, nil
	}
	if err != nil {
		return Item{}, false, fmt.Errorf("account: find subscription identity: %w", err)
	}
	item, err := s.Metadata(ctx, id)
	if err != nil {
		return Item{}, false, err
	}
	return item, true, nil
}
