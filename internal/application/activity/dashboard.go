package activity

import (
	"context"
	"database/sql"
	"math"
	"math/big"
	"strconv"
	"time"
)

type dashboardCounts struct{ requests, success, failed, cancelled, latency, input, output, complete, firstToken, firstTokenSamples int64 }

func (c *dashboardCounts) add(other dashboardCounts) error {
	for _, pair := range [][2]*int64{{&c.requests, &other.requests}, {&c.success, &other.success}, {&c.failed, &other.failed}, {&c.cancelled, &other.cancelled}, {&c.latency, &other.latency}, {&c.input, &other.input}, {&c.output, &other.output}, {&c.complete, &other.complete}, {&c.firstToken, &other.firstToken}, {&c.firstTokenSamples, &other.firstTokenSamples}} {
		if *pair[1] < 0 || *pair[0] > math.MaxInt64-*pair[1] {
			return ErrUnavailable
		}
		*pair[0] += *pair[1]
	}
	return nil
}

func (c dashboardCounts) summary() DashboardSummary {
	result := DashboardSummary{Requests: c.requests, Success: c.success, Failed: c.failed, Cancelled: c.cancelled, Pending: max(0, c.requests-c.success-c.failed-c.cancelled), InputTokens: strconv.FormatInt(c.input, 10), OutputTokens: strconv.FormatInt(c.output, 10), TotalTokens: new(big.Int).Add(big.NewInt(c.input), big.NewInt(c.output)).String(), CompleteUsage: c.complete}
	if completed := c.success + c.failed; completed > 0 {
		rate, latency := float64(c.success)/float64(completed)*100, float64(c.latency)/float64(completed)
		result.SuccessRate, result.AverageLatencyMS = &rate, &latency
	}
	result.FirstTokenSamples = c.firstTokenSamples
	if c.firstTokenSamples > 0 {
		value := float64(c.firstToken) / float64(c.firstTokenSamples)
		result.AverageFirstTokenMS = &value
	}
	return result
}

func validateDashboard(query DashboardQuery, now time.Time) error {
	b := query.Boundaries
	if len(b) < 2 || len(b) > 33 {
		return ErrInvalid
	}
	if b[0] < now.Add(-32*24*time.Hour).Unix() || b[0] > now.Unix() || b[len(b)-1] > now.Add(26*time.Hour).Unix() || b[len(b)-1]-b[0] > int64((32*24*time.Hour)/time.Second) {
		return ErrInvalid
	}
	for i, value := range b {
		if value%dashboardBucketSeconds != 0 || i > 0 && (value <= b[i-1] || value-b[i-1] > int64((26*time.Hour)/time.Second)) {
			return ErrInvalid
		}
	}
	return nil
}

// Dashboard 在一致的只读快照中汇总已保存的时间桶，最近列表不参与计算指标。
func (s *Service) Dashboard(ctx context.Context, query DashboardQuery) (DashboardInfo, error) {
	result := DashboardInfo{Points: []DashboardPoint{}, Recent: []RequestItem{}, UpdatedAt: timeText(time.Now())}
	if err := validateDashboard(query, time.Now()); err != nil {
		return result, err
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return result, ErrUnavailable
	}
	defer tx.Rollback()
	counts, err := readDashboardCounts(ctx, tx, query.Boundaries)
	if err != nil {
		return result, err
	}
	var all dashboardCounts
	for i, c := range counts {
		if err := all.add(c); err != nil {
			return result, err
		}
		result.Points = append(result.Points, DashboardPoint{Start: query.Boundaries[i], End: query.Boundaries[i+1], Summary: c.summary()})
	}
	result.Summary = all.summary()
	var incomplete bool
	if tx.QueryRowContext(ctx, `SELECT captured_from,history_incomplete FROM dashboard_meta WHERE id=1`).Scan(&result.CapturedFrom, &incomplete) != nil {
		return result, ErrUnavailable
	}
	result.HistoryIncomplete = incomplete && query.Boundaries[0] < result.CapturedFrom
	result.Recent, err = readDashboardRecent(ctx, tx, query.Boundaries[0], query.Boundaries[len(query.Boundaries)-1])
	if err != nil {
		return result, ErrUnavailable
	}
	result.ProviderDistribution, err = readDashboardProviders(ctx, tx)
	var pending int64
	// 使用已有全局桶计算未结束请求，避免保留百万条明细时每次刷新扫描整表。
	if err != nil || tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(MAX(0,requests-success-failed-cancelled)),0) FROM dashboard_buckets`).Scan(&pending) != nil {
		return result, ErrUnavailable
	}
	result.PendingNow = &pending
	if err != nil || tx.Commit() != nil {
		return result, ErrUnavailable
	}
	return result, nil
}

func readDashboardCounts(ctx context.Context, tx *sql.Tx, bounds []int64) ([]dashboardCounts, error) {
	counts := make([]dashboardCounts, len(bounds)-1)
	rows, err := tx.QueryContext(ctx, `SELECT bucket,requests,success,failed,cancelled,latency_ms,input_tokens,output_tokens,complete_usage,first_token_ms,first_token_samples FROM dashboard_buckets WHERE bucket>=? AND bucket<? ORDER BY bucket`, bounds[0], bounds[len(bounds)-1])
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	index := 0
	for rows.Next() {
		var bucket int64
		var c dashboardCounts
		if rows.Scan(&bucket, &c.requests, &c.success, &c.failed, &c.cancelled, &c.latency, &c.input, &c.output, &c.complete, &c.firstToken, &c.firstTokenSamples) != nil {
			return nil, ErrUnavailable
		}
		for index+1 < len(counts) && bucket >= bounds[index+1] {
			index++
		}
		if err := counts[index].add(c); err != nil {
			return nil, err
		}
	}
	if rows.Err() != nil {
		return nil, ErrUnavailable
	}
	return counts, nil
}

func readDashboardRecent(ctx context.Context, tx *sql.Tx, from, to int64) ([]RequestItem, error) {
	rows, err := tx.QueryContext(ctx, `SELECT `+requestColumns+` FROM request_records r LEFT JOIN accounts a ON a.id=r.account_id WHERE r.started_at>=? AND r.started_at<? ORDER BY r.started_at DESC,r.id DESC LIMIT 8`, timeText(time.Unix(from, 0)), timeText(time.Unix(to, 0)))
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	items := []RequestItem{}
	for rows.Next() {
		item, err := scanRequest(rows)
		if err != nil {
			return nil, ErrUnavailable
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, ErrUnavailable
	}
	return items, nil
}
