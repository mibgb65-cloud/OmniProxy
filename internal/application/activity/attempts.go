package activity

import (
	"context"
	"database/sql"
	"encoding/hex"
	"omniproxy/internal/core/proxy"
	"strconv"
)

// AttemptItem 是请求详情中的单次上游尝试；Tokens 未知时必须结合 Known 标志展示。
type AttemptItem struct {
	Index          int    `json:"index"`
	StartedAt      string `json:"startedAt"`
	FinishedAt     string `json:"finishedAt"`
	ProviderID     string `json:"providerId"`
	ProviderName   string `json:"providerName"`
	AccountID      string `json:"accountId"`
	AccountName    string `json:"accountName"`
	ProviderRegion string `json:"providerRegion"`
	BillingMode    string `json:"billingMode"`
	Model          string `json:"model"`
	State          string `json:"state"`
	StatusCode     int    `json:"statusCode"`
	ErrorCode      string `json:"errorCode"`
	LatencyMS      int64  `json:"latencyMs"`
	InputTokens    string `json:"inputTokens"`
	OutputTokens   string `json:"outputTokens"`
	InputKnown     bool   `json:"inputKnown"`
	OutputKnown    bool   `json:"outputKnown"`
	UsageComplete  bool   `json:"usageComplete"`
}

// RecordAttempt 保留首次目标与完成结果，不修改逻辑请求数或账号累计。
func (s *Service) RecordAttempt(ctx context.Context, e proxy.AttemptEvent) error {
	if _, err := hex.DecodeString(e.RequestID); err != nil || len(e.RequestID) != 32 || e.Index < 1 || e.Index > 3 || e.StartedAt.IsZero() || len(e.Model) > 256 || len(e.ErrorCode) > 128 || e.LatencyMS < 0 || e.LatencyMS > 86400000 {
		return ErrInvalid
	}
	if _, err := subject(e.Target); err != nil || e.Target.AccountID == "" {
		return ErrInvalid
	}
	if e.State != "pending" && e.State != "success" && e.State != "failed" && e.State != "cancelled" {
		return ErrInvalid
	}
	if e.Usage.Input < 0 || e.Usage.Output < 0 || e.Usage.Input > 1e12 || e.Usage.Output > 1e12 {
		return ErrInvalid
	}
	if e.State != "pending" && (e.FinishedAt.IsZero() || e.FinishedAt.Before(e.StartedAt)) {
		return ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrUnavailable
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO request_attempts(request_id,attempt_index,started_at,provider_id,provider_name,account_id,account_name,provider_region,billing_mode,model) SELECT ?,?,?,?,?,?,?,?,?,? WHERE EXISTS(SELECT 1 FROM request_records WHERE id=? AND state='pending')`, e.RequestID, e.Index, timeText(e.StartedAt), e.Target.ProviderID, e.Target.ProviderName, e.Target.AccountID, e.Target.AccountName, e.Target.ProviderRegion, e.Target.BillingMode, e.Model, e.RequestID)
	if err != nil {
		return ErrUnavailable
	}
	var account, model, state string
	if tx.QueryRowContext(ctx, `SELECT account_id,model,state FROM request_attempts WHERE request_id=? AND attempt_index=?`, e.RequestID, e.Index).Scan(&account, &model, &state) != nil {
		return ErrUnavailable
	}
	if account != e.Target.AccountID || model != e.Model {
		return ErrInvalid
	}
	if state != "pending" {
		return nil
	}
	if e.State != "pending" {
		_, err = tx.ExecContext(ctx, `UPDATE request_attempts SET finished_at=?,state=?,status_code=?,error_code=?,latency_ms=?,input_tokens=MAX(input_tokens,?),output_tokens=MAX(output_tokens,?),input_known=MAX(input_known,?),output_known=MAX(output_known,?),usage_complete=MAX(usage_complete,?) WHERE request_id=? AND attempt_index=?`, timeText(e.FinishedAt), e.State, e.StatusCode, e.ErrorCode, e.LatencyMS, e.Usage.Input, e.Usage.Output, e.Usage.InputKnown, e.Usage.OutputKnown, e.Usage.Complete, e.RequestID, e.Index)
		if err != nil {
			return ErrUnavailable
		}
	}
	return s.commitActivity(tx)
}

func updateAttemptUsage(ctx context.Context, tx *sql.Tx, e proxy.RequestEvent) error {
	if e.AttemptIndex == 0 {
		return nil
	}
	result, err := tx.ExecContext(ctx, `UPDATE request_attempts SET input_tokens=?,output_tokens=?,input_known=?,output_known=?,usage_complete=? WHERE request_id=? AND attempt_index=? AND account_id=? AND state='pending'`, e.Usage.Input, e.Usage.Output, e.Usage.InputKnown, e.Usage.OutputKnown, e.Usage.Complete, e.ID, e.AttemptIndex, e.Target.AccountID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	// 终止的明确拒绝仅在所有尝试结束后绑定父记录，没有已报告用量。
	if n != 1 && (e.Usage.InputKnown || e.Usage.OutputKnown) {
		return ErrInvalid
	}
	return nil
}

func (s *Service) attempts(ctx context.Context, id string) ([]AttemptItem, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT attempt_index,started_at,finished_at,provider_id,provider_name,account_id,account_name,provider_region,billing_mode,model,state,status_code,error_code,latency_ms,input_tokens,output_tokens,input_known,output_known,usage_complete FROM request_attempts WHERE request_id=? ORDER BY attempt_index LIMIT 3`, id)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	items := []AttemptItem{}
	for rows.Next() {
		var item AttemptItem
		var input, output int64
		if rows.Scan(&item.Index, &item.StartedAt, &item.FinishedAt, &item.ProviderID, &item.ProviderName, &item.AccountID, &item.AccountName, &item.ProviderRegion, &item.BillingMode, &item.Model, &item.State, &item.StatusCode, &item.ErrorCode, &item.LatencyMS, &input, &output, &item.InputKnown, &item.OutputKnown, &item.UsageComplete) != nil {
			return nil, ErrUnavailable
		}
		item.InputTokens, item.OutputTokens = strconv.FormatInt(input, 10), strconv.FormatInt(output, 10)
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, ErrUnavailable
	}
	return items, nil
}
