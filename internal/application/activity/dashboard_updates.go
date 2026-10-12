package activity

import (
	"context"
	"database/sql"
	"time"
)

const dashboardBucketSeconds int64 = 15 * 60

func dashboardBucket(start time.Time) int64 {
	return start.Unix() / dashboardBucketSeconds * dashboardBucketSeconds
}

func beginDashboard(ctx context.Context, tx *sql.Tx, start time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO dashboard_buckets(bucket,requests) VALUES(?,1) ON CONFLICT(bucket) DO UPDATE SET requests=requests+1`, dashboardBucket(start))
	return err
}

func updateDashboard(ctx context.Context, tx *sql.Tx, start time.Time, input, output int64, complete int) error {
	_, err := tx.ExecContext(ctx, `UPDATE dashboard_buckets SET input_tokens=input_tokens+?,output_tokens=output_tokens+?,complete_usage=complete_usage+? WHERE bucket=?`, input, output, complete, dashboardBucket(start))
	return err
}

func finishDashboard(ctx context.Context, tx *sql.Tx, start time.Time, state string, latency int64) error {
	var success, failed, cancelled int
	switch state {
	case "success":
		success = 1
	case "failed":
		failed = 1
	case "cancelled":
		cancelled = 1
		latency = 0
	}
	_, err := tx.ExecContext(ctx, `UPDATE dashboard_buckets SET success=success+?,failed=failed+?,cancelled=cancelled+?,latency_ms=latency_ms+? WHERE bucket=?`, success, failed, cancelled, latency, dashboardBucket(start))
	return err
}

// Updates 是单一桌面转发器消费的有界变更信号，不在事件中携带账号或用量。
func (s *Service) Updates() <-chan struct{} { return s.updates }

func (s *Service) commitActivity(tx *sql.Tx) error {
	if tx.Commit() != nil {
		return ErrUnavailable
	}
	select {
	case s.updates <- struct{}{}:
	default:
	}
	return nil
}
