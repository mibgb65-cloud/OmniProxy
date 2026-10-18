package activity

import (
	"context"
	"time"
)

// Recover 仅处理超出最长代理期限的旧请求，不改写其他实例正在执行的调用。
func (s *Service) Recover(ctx context.Context) error {
	threshold := timeText(time.Now().Add(-10 * time.Minute))
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrUnavailable
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE request_attempts SET state='cancelled',finished_at=?,status_code=499,error_code='process_interrupted' WHERE state='pending' AND request_id IN (SELECT id FROM request_records WHERE state!='pending' OR started_at<?)`, timeText(time.Now()), threshold); err != nil {
		return ErrUnavailable
	}
	_, err = tx.ExecContext(ctx, `UPDATE dashboard_buckets SET cancelled=cancelled+(SELECT COUNT(*) FROM request_records r WHERE r.state='pending' AND r.started_at<? AND (CAST(strftime('%s',r.started_at) AS INTEGER)/900)*900=dashboard_buckets.bucket) WHERE bucket IN (SELECT (CAST(strftime('%s',started_at) AS INTEGER)/900)*900 FROM request_records WHERE state='pending' AND started_at<?)`, threshold, threshold)
	if err != nil {
		return ErrUnavailable
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO usage_buckets(day,subject,model,provider_name,account_name,email,workspace_id,requests) SELECT day,'unassigned',model,'','','','',COUNT(*) FROM request_records WHERE state='pending' AND started_at<? AND subject='' GROUP BY day,model ON CONFLICT(day,subject,model) DO UPDATE SET requests=requests+excluded.requests`, threshold)
	if err != nil {
		return ErrUnavailable
	}
	_, err = tx.ExecContext(ctx, `UPDATE request_records SET state='cancelled',finished_at=?,status_code=499,error_code='process_interrupted' WHERE state='pending' AND started_at<?`, timeText(time.Now()), threshold)
	if err != nil || prune(ctx, tx) != nil {
		return ErrUnavailable
	}
	return s.commitActivity(tx)
}
