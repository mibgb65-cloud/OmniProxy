package activity

import (
	"context"
	"math"
	"omniproxy/internal/core/proxy"
	"time"
)

func (s *Service) Begin(ctx context.Context, event proxy.RequestEvent) error {
	if len(event.ID) != 32 || event.StartedAt.IsZero() {
		return ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrUnavailable
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO request_records(id,started_at,day,protocol,transport) VALUES(?,?,?,?,?)`, event.ID, timeText(event.StartedAt), event.StartedAt.In(time.Local).Format(time.DateOnly), event.Protocol, event.Transport)
	if err != nil {
		return ErrUnavailable
	}
	n, err := result.RowsAffected()
	if err != nil {
		return ErrUnavailable
	}
	if n > 0 {
		if _, err = tx.ExecContext(ctx, `UPDATE usage_totals SET requests=requests+1 WHERE id=1`); err != nil {
			return ErrUnavailable
		}
		if err := beginDashboard(ctx, tx, event.StartedAt); err != nil {
			return ErrUnavailable
		}
	}
	if err := prune(ctx, tx); err != nil {
		return ErrUnavailable
	}
	return s.commitActivity(tx)
}
func (s *Service) Update(ctx context.Context, event proxy.RequestEvent) error {
	key, err := subject(event.Target)
	if err != nil {
		return err
	}
	usage := event.Usage
	if usage.Input < 0 || usage.Output < 0 || usage.Input > 1e12 || usage.Output > 1e12 {
		return ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrUnavailable
	}
	defer tx.Rollback()
	var oldInput, oldOutput int64
	var oldComplete bool
	var oldSubject, state, startedAt string
	if tx.QueryRowContext(ctx, `SELECT input_tokens,output_tokens,usage_complete,subject,state,started_at FROM request_records WHERE id=?`, event.ID).Scan(&oldInput, &oldOutput, &oldComplete, &oldSubject, &state, &startedAt) != nil {
		return ErrUnavailable
	}
	if state != "pending" {
		return nil
	}
	if oldSubject != "" && key != oldSubject {
		return ErrInvalid
	}
	if key != "" && oldSubject == "" {
		_, err = tx.ExecContext(ctx, `INSERT INTO usage_buckets(day,subject,model,provider_name,account_name,email,workspace_id,requests) VALUES(?,?,?,?,?,?,?,1) ON CONFLICT(day,subject,model) DO UPDATE SET requests=requests+1`, event.StartedAt.In(time.Local).Format(time.DateOnly), key, event.Model, event.Target.ProviderName, event.Target.AccountName, event.Target.Email, event.Target.WorkspaceID)
		if err != nil {
			return ErrUnavailable
		}
	}
	input, output := max(oldInput, usage.Input), max(oldOutput, usage.Output)
	complete := oldComplete || usage.Complete
	completeDelta := 0
	if complete && !oldComplete {
		completeDelta = 1
	}
	if key == "" && (input > 0 || output > 0 || complete) {
		return ErrInvalid
	}
	if key != "" {
		_, err = tx.ExecContext(ctx, `UPDATE usage_buckets SET input_tokens=input_tokens+?,output_tokens=output_tokens+?,completed_usage=completed_usage+? WHERE day=? AND subject=? AND model=?`, input-oldInput, output-oldOutput, completeDelta, event.StartedAt.In(time.Local).Format(time.DateOnly), key, event.Model)
		if err != nil {
			return ErrUnavailable
		}
	}
	var totalInput, totalOutput int64
	if err := updateProviderModel(ctx, tx, event, oldSubject == "" && key != "", input-oldInput, output-oldOutput, completeDelta); err != nil {
		return ErrUnavailable
	}
	if tx.QueryRowContext(ctx, `SELECT input_tokens,output_tokens FROM usage_totals WHERE id=1`).Scan(&totalInput, &totalOutput) != nil || totalInput > math.MaxInt64-(input-oldInput) || totalOutput > math.MaxInt64-(output-oldOutput) {
		return ErrUnavailable
	}
	_, err = tx.ExecContext(ctx, `UPDATE usage_totals SET input_tokens=input_tokens+?,output_tokens=output_tokens+?,completed_usage=completed_usage+? WHERE id=1`, input-oldInput, output-oldOutput, completeDelta)
	if err != nil {
		return ErrUnavailable
	}
	_, err = tx.ExecContext(ctx, `UPDATE request_records SET provider_region=?,billing_mode=?,input_model=?,model=?,stream=?,provider_id=?,provider_name=?,account_id=?,account_name=?,email=?,workspace_id=?,subject=?,input_tokens=?,output_tokens=?,input_known=MAX(input_known,?),output_known=MAX(output_known,?),usage_complete=? WHERE id=?`, event.Target.ProviderRegion, event.Target.BillingMode, event.InputModel, event.Model, event.Stream, event.Target.ProviderID, event.Target.ProviderName, event.Target.AccountID, event.Target.AccountName, event.Target.Email, event.Target.WorkspaceID, key, input, output, usage.InputKnown, usage.OutputKnown, complete, event.ID)
	if err != nil {
		return ErrUnavailable
	}
	start, err := time.Parse(time.RFC3339Nano, startedAt)
	if err != nil {
		return ErrUnavailable
	}
	if err := updateDashboard(ctx, tx, start, input-oldInput, output-oldOutput, completeDelta); err != nil {
		return ErrUnavailable
	}
	if err := updateAttemptUsage(ctx, tx, event); err != nil {
		return ErrUnavailable
	}
	return s.commitActivity(tx)
}
func (s *Service) Finish(ctx context.Context, event proxy.RequestEvent) error {
	if event.State != "success" && event.State != "failed" && event.State != "cancelled" {
		return ErrInvalid
	}
	if event.LatencyMS < 0 || event.LatencyMS > int64((24*time.Hour)/time.Millisecond) {
		return ErrInvalid
	}
	if event.FirstTokenMS != nil && (!event.Stream || *event.FirstTokenMS < 0 || *event.FirstTokenMS > event.LatencyMS) {
		return ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrUnavailable
	}
	defer tx.Rollback()
	var currentSubject, model, day, state, startedAt string
	if tx.QueryRowContext(ctx, `SELECT subject,model,day,state,started_at FROM request_records WHERE id=?`, event.ID).Scan(&currentSubject, &model, &day, &state, &startedAt) != nil {
		return ErrUnavailable
	}
	if state != "pending" {
		return nil
	}
	if currentSubject == "" {
		if _, err = tx.ExecContext(ctx, `INSERT INTO usage_buckets(day,subject,model,provider_name,account_name,email,workspace_id,requests) VALUES(?,'unassigned',?,'','','','',1) ON CONFLICT(day,subject,model) DO UPDATE SET requests=requests+1`, day, model); err != nil {
			return ErrUnavailable
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE request_records SET finished_at=?,state=?,status_code=?,error_code=?,latency_ms=?,first_token_ms=? WHERE id=? AND state='pending'`, timeText(event.FinishedAt), event.State, event.StatusCode, event.ErrorCode, event.LatencyMS, event.FirstTokenMS, event.ID); err != nil {
		return ErrUnavailable
	}
	start, err := time.Parse(time.RFC3339Nano, startedAt)
	if err != nil || finishDashboard(ctx, tx, start, event.State, event.LatencyMS) != nil {
		return ErrUnavailable
	}
	if err := finishFirstToken(ctx, tx, start, event); err != nil {
		return ErrUnavailable
	}
	if err = prune(ctx, tx); err != nil {
		return ErrUnavailable
	}
	return s.commitActivity(tx)
}

func timeText(value time.Time) string { return value.UTC().Format("2006-01-02T15:04:05.000000000Z") }
