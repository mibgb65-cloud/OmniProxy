package activity

import (
	"context"
	"database/sql"
	"math"
	"time"

	"omniproxy/internal/core/proxy"
)

func finishFirstToken(ctx context.Context, tx *sql.Tx, start time.Time, event proxy.RequestEvent) error {
	// 失败／取消保留各自已测到的明细，但不混入成功流式首字均值。
	if event.State != "success" || event.FirstTokenMS == nil {
		return nil
	}
	value := *event.FirstTokenMS
	result, err := tx.ExecContext(ctx, `UPDATE dashboard_buckets SET first_token_ms=first_token_ms+?,first_token_samples=first_token_samples+1 WHERE bucket=? AND first_token_ms<=? AND first_token_samples<?`, value, dashboardBucket(start), math.MaxInt64-value, int64(math.MaxInt64))
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrUnavailable
	}
	return nil
}
