package googlesubscription

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/provider/antigravity"
)

// maxGateSnapshot 与额度快照落库时的大小上限一致：更大的行按不可信处理。
const maxGateSnapshot = 131072

// QuotaGate 只读已保存的额度快照，判断某账号的某模型是否已经被耗尽。
//
// 它不发起官方查询、不读凭据，也不改变账号状态：预筛只是优化，读取失败、快照不可信
// 或模型无法归类时一律按「不预筛」返回，请求仍由 429 的反应式冷却兜底。
type QuotaGate struct{ db *sql.DB }

func NewQuotaGate(db *sql.DB) *QuotaGate { return &QuotaGate{db: db} }

// BlockedUntil 返回该账号该模型所属额度分组的恢复时间；未耗尽或无法判断时返回 false。
// 账号版本（updated_at）与登录态必须与当前一致，旧版本或已退出的快照不参与预筛。
func (g *QuotaGate) BlockedUntil(ctx context.Context, accountID, updatedAt, model string, now time.Time) (time.Time, bool) {
	if g == nil || g.db == nil || accountID == "" || updatedAt == "" || model == "" {
		return time.Time{}, false
	}
	var raw []byte
	err := g.db.QueryRowContext(ctx, `SELECT q.snapshot_json FROM google_subscription_quota_snapshots q
		JOIN accounts a ON a.id=q.account_id
		WHERE q.account_id=? AND a.updated_at=? AND a.auth_method=? AND a.auth_state='authenticated'`,
		accountID, updatedAt, account.AuthSubscription).Scan(&raw)
	if err != nil || len(raw) == 0 || len(raw) > maxGateSnapshot {
		return time.Time{}, false
	}
	var value antigravity.Quota
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&value) != nil || decoder.Decode(new(any)) != io.EOF || !antigravity.ValidateQuotaSnapshot(value) {
		return time.Time{}, false
	}
	return antigravity.QuotaExhaustedUntil(value, model, now)
}
