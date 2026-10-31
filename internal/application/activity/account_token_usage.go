package activity

import (
	"context"
	"database/sql"
	"strconv"

	"omniproxy/internal/application/account"
	"omniproxy/internal/core/proxy"
)

// AccountTokenUsage 是该记账身份的终身累计：明细被保留上限淘汰、账号或提供商删除、应用重启都不减少。
// 输入包含缓存，不能把三项相加。空字符串代表该项明细未知，区别于上游明确返回的零。
type AccountTokenUsage struct {
	InputTokens          string `json:"inputTokens"`
	OutputTokens         string `json:"outputTokens"`
	CachedInputTokens    string `json:"cachedInputTokens"`
	Requests             int64  `json:"requests"`
	UnknownCacheRequests int64  `json:"unknownCacheRequests"`
	IncompleteRequests   int64  `json:"incompleteRequests"`
	// PendingRequests 是此刻仍在进行中的记录数，用于「正在使用」状态；只取留存明细，不影响 Token 累计。
	PendingRequests int64 `json:"pendingRequests"`
}

// usageIdentity 只含公开身份字段，必须与网关记账时的快照取值一致。
func usageIdentity(item account.Item, kind string) proxy.TargetSnapshot {
	target := proxy.TargetSnapshot{ProviderKind: kind, AccountID: item.ID, AuthMethod: item.AuthMethod,
		Email: item.AuthEmail, WorkspaceID: item.AuthAccountID}
	target.ProviderRegion, target.BillingMode = item.ProviderScope()
	return target
}

// accountUsageDelta 是一条记账快照对终身累计的差值：重复快照只按增量记一次。
type accountUsageDelta struct {
	requests, completed, inputKnown, outputKnown, cachedKnown int
	input, output, cached                                     int64
}

// updateAccountUsage 与 usage_totals／usage_buckets／usage_provider_models 在同一次记账事务里按差值更新。
// 明细之后被淘汰或账号被删除都不会回退这里的累计，因此账号卡片不会再随保留上限变小。
func updateAccountUsage(ctx context.Context, tx *sql.Tx, event proxy.RequestEvent, key string, delta accountUsageDelta) error {
	if key == "" {
		return nil
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO usage_accounts(subject,provider_name,account_name,email,workspace_id,requests,completed_usage,input_tokens,output_tokens,input_known,output_known,cached_input_tokens,cached_known)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(subject) DO UPDATE SET provider_name=excluded.provider_name,account_name=excluded.account_name,email=excluded.email,workspace_id=excluded.workspace_id,
		requests=requests+excluded.requests,completed_usage=completed_usage+excluded.completed_usage,
		input_tokens=input_tokens+excluded.input_tokens,output_tokens=output_tokens+excluded.output_tokens,
		input_known=input_known+excluded.input_known,output_known=output_known+excluded.output_known,
		cached_input_tokens=cached_input_tokens+excluded.cached_input_tokens,cached_known=cached_known+excluded.cached_known`,
		key, event.Target.ProviderName, event.Target.AccountName, event.Target.Email, event.Target.WorkspaceID,
		delta.requests, delta.completed, delta.input, delta.output, delta.inputKnown, delta.outputKnown, delta.cached, delta.cachedKnown)
	return err
}

func flagCount(value bool) int {
	if value {
		return 1
	}
	return 0
}

// AccountUsage 读取任意账号的本机累计用量，用于没有官方额度接口的账号展示已用 Token。
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
	var input, output, cached, inputKnown, outputKnown, cacheKnown, completed int64
	err = s.db.QueryRowContext(ctx, `SELECT input_tokens,output_tokens,cached_input_tokens,input_known,output_known,cached_known,requests,completed_usage
		FROM usage_accounts WHERE subject=?`, key,
	).Scan(&input, &output, &cached, &inputKnown, &outputKnown, &cacheKnown, &result.Requests, &completed)
	switch {
	case err == sql.ErrNoRows:
		// 从未记账的身份是已知零，与「上游没有返回明细」的未知区分。
	case err != nil:
		return nil, ErrUnavailable
	}
	if result.Requests > cacheKnown {
		result.UnknownCacheRequests = result.Requests - cacheKnown
	}
	if result.Requests > completed {
		result.IncompleteRequests = result.Requests - completed
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
	// 进行中的请求只来自仍在保留期内的明细：卡片用它显示「正在使用」圆环，不参与 Token 口径。
	if s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM request_records WHERE subject=? AND state='pending'`, key).Scan(&result.PendingRequests) != nil {
		return nil, ErrUnavailable
	}
	return result, nil
}
