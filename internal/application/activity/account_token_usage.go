package activity

import (
	"context"
	"strconv"
	"strings"

	"omniproxy/internal/application/account"
	"omniproxy/internal/core/proxy"
)

// AccountTokenUsage 仅统计当前身份的留存代理记录；输入包含缓存，不能把三项相加。
// 空字符串代表该项明细未知，区别于上游明确返回的零。
type AccountTokenUsage struct {
	InputTokens          string `json:"inputTokens"`
	OutputTokens         string `json:"outputTokens"`
	CachedInputTokens    string `json:"cachedInputTokens"`
	Requests             int64  `json:"requests"`
	UnknownCacheRequests int64  `json:"unknownCacheRequests"`
	IncompleteRequests   int64  `json:"incompleteRequests"`
	// PendingRequests 是此刻仍在进行中的记录数，用于「正在使用」状态；历史未完整记录不算进行中。
	PendingRequests int64 `json:"pendingRequests"`
}

// usageIdentity 只含公开身份字段，必须与网关记账时的快照取值一致。
func usageIdentity(item account.Item, kind string) proxy.TargetSnapshot {
	target := proxy.TargetSnapshot{ProviderKind: kind, AccountID: item.ID, AuthMethod: item.AuthMethod,
		Email: item.AuthEmail, WorkspaceID: item.AuthAccountID}
	target.ProviderRegion, target.BillingMode = item.ProviderScope()
	return target
}

// AccountUsage 读取任意账号的本机留存用量，用于没有官方额度接口的账号展示已用 Token。
// 提供商类型取自当前提供商配置，与记账身份同源；不访问凭据或官方接口。
func (s *Service) AccountUsage(ctx context.Context, item account.Item) (*AccountTokenUsage, error) {
	if item.ID == "" || item.ProviderID == "" {
		return nil, ErrInvalid
	}
	var kind string
	if s.db.QueryRowContext(ctx, `SELECT kind FROM providers WHERE id=?`, item.ProviderID).Scan(&kind) != nil {
		return nil, ErrUnavailable
	}
	if kind == "" {
		return nil, ErrInvalid
	}
	return s.tokenUsage(ctx, item, kind)
}

func (s *Service) accountTokenUsage(ctx context.Context, item account.Item) (*AccountTokenUsage, error) {
	return s.tokenUsage(ctx, item, "openai")
}

func (s *Service) tokenUsage(ctx context.Context, item account.Item, kind string) (*AccountTokenUsage, error) {
	// 按线路构造与记账一致的身份索引，避免混入其他提供商／账号以及扫描全部历史。
	key, err := subject(usageIdentity(item, kind))
	if err != nil {
		return nil, err
	}
	result := &AccountTokenUsage{}
	var input, output, cached, inputKnown, outputKnown, cacheKnown int64
	err = s.db.QueryRowContext(ctx, `SELECT
		COALESCE(SUM(input_tokens),0),COALESCE(SUM(output_tokens),0),COALESCE(SUM(cached_input_tokens),0),
		COALESCE(SUM(input_known),0),COALESCE(SUM(output_known),0),
		COUNT(cached_input_tokens),COUNT(*),
		COALESCE(SUM(CASE WHEN cached_input_tokens IS NULL AND (input_known=0 OR input_tokens>0) THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN usage_complete=0 OR state='pending' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN state='pending' THEN 1 ELSE 0 END),0)
		FROM request_records WHERE subject=? AND provider_id=? AND account_id=? AND lower(trim(email))=? AND workspace_id=?
		AND (state IN ('success','pending') OR input_known=1 OR output_known=1)`,
		key, item.ProviderID, item.ID, strings.ToLower(strings.TrimSpace(item.AuthEmail)), item.AuthAccountID,
	).Scan(&input, &output, &cached, &inputKnown, &outputKnown, &cacheKnown,
		&result.Requests, &result.UnknownCacheRequests, &result.IncompleteRequests, &result.PendingRequests)
	if err != nil {
		return nil, ErrUnavailable
	}
	if inputKnown > 0 || result.Requests == 0 {
		result.InputTokens = strconv.FormatInt(input, 10)
	}
	if outputKnown > 0 || result.Requests == 0 {
		result.OutputTokens = strconv.FormatInt(output, 10)
	}
	if cacheKnown > 0 || result.Requests == 0 || (inputKnown == result.Requests && input == 0) {
		result.CachedInputTokens = strconv.FormatInt(cached, 10)
	}
	return result, nil
}
