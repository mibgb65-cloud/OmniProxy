package activity

import (
	"context"
	"database/sql"
	"strconv"
	"unicode"
	"unicode/utf8"
)

// MaxExportRows 限制单次 XLSX 导出的行数，避免一次导出占满内存与磁盘。
const MaxExportRows = 100000

const exportColumns = `r.id,r.started_at,r.state,r.protocol,r.transport,r.input_model,r.model,r.reasoning_effort,r.provider_name,r.account_name,r.email,r.workspace_id,r.client_id,r.input_tokens,r.output_tokens,r.input_known,r.output_known,r.cached_input_tokens,r.usage_complete,r.status_code,r.error_code,r.latency_ms,r.first_token_ms,COALESCE(c.content,''),COALESCE(c.original_bytes,0),COALESCE(c.stored_bytes,0),COALESCE(c.truncated,0),(c.request_id IS NOT NULL)`

// ExportRequest 是导出的一行：请求元数据加可选正文快照。
type ExportRequest struct {
	Item    RequestItem
	Content RequestContent
}

// ExportRequests 按筛选条件顺序遍历匹配的请求；visit 返回错误时立即停止。
// 返回值 truncated 表示命中行数超过 MaxExportRows，只导出了前 MaxExportRows 行。
func (s *Service) ExportRequests(ctx context.Context, q RequestQuery, visit func(ExportRequest) error) (int, bool, error) {
	if err := validateExportQuery(q); err != nil {
		return 0, false, err
	}
	where := `(?='' OR r.state=?) AND (?='' OR instr(lower(r.id || ' ' || r.input_model || ' ' || r.model || ' ' || r.account_name || ' ' || r.email || ' ' || r.workspace_id || ' ' || r.client_id),lower(?))>0)`
	args := []any{q.State, q.State, q.Search, q.Search}
	rows, err := s.db.QueryContext(ctx, `SELECT `+exportColumns+` FROM request_records r LEFT JOIN request_contents c ON c.request_id=r.id WHERE `+where+` ORDER BY r.started_at ASC,r.id ASC LIMIT ?`, append(args, MaxExportRows+1)...)
	if err != nil {
		return 0, false, ErrUnavailable
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		if count == MaxExportRows {
			return count, true, nil
		}
		item, err := scanExportRow(rows)
		if err != nil {
			return count, false, ErrUnavailable
		}
		if err := visit(item); err != nil {
			return count, false, err
		}
		count++
	}
	if rows.Err() != nil {
		return count, false, ErrUnavailable
	}
	return count, false, nil
}

func validateExportQuery(q RequestQuery) error {
	if len(q.Search) > 128 || !utf8.ValidString(q.Search) {
		return ErrInvalid
	}
	for _, r := range q.Search {
		if unicode.IsControl(r) {
			return ErrInvalid
		}
	}
	switch q.State {
	case "", "success", "failed", "cancelled", "pending":
		return nil
	}
	return ErrInvalid
}

func scanExportRow(row scanner) (ExportRequest, error) {
	var result ExportRequest
	item := &result.Item
	var input, output int64
	var cached sql.NullInt64
	var bytesOriginal, bytesStored int64
	var truncated, available int
	if err := row.Scan(
		&item.ID, &item.StartedAt, &item.State, &item.Protocol, &item.Transport,
		&item.InputModel, &item.Model, &item.Effort, &item.ProviderName, &item.AccountName,
		&item.Email, &item.WorkspaceID, &item.Client, &input, &output, &item.InputKnown,
		&item.OutputKnown, &cached, &item.UsageComplete, &item.StatusCode, &item.ErrorCode,
		&item.LatencyMS, &item.FirstTokenMS, &result.Content.Content, &bytesOriginal, &bytesStored,
		&truncated, &available,
	); err != nil {
		return ExportRequest{}, err
	}
	item.InputTokens = strconv.FormatInt(input, 10)
	item.OutputTokens = strconv.FormatInt(output, 10)
	if cached.Valid {
		item.CachedInputTokens = strconv.FormatInt(cached.Int64, 10)
		item.CachedInputKnown = true
	}
	item.attachCost(input, output, cached)
	result.Content.Available = available != 0
	result.Content.Truncated = truncated != 0
	result.Content.OriginalBytes = bytesOriginal
	result.Content.StoredBytes = bytesStored
	return result, nil
}
