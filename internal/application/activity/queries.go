package activity

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const requestColumns = `r.id,r.started_at,r.finished_at,r.protocol,r.transport,r.input_model,r.model,r.stream,r.provider_id,r.provider_name,r.account_id,r.account_name,r.email,r.workspace_id,(r.account_id!='' AND a.id IS NULL),r.state,r.status_code,r.error_code,r.latency_ms,r.input_tokens,r.output_tokens,r.input_known,r.output_known,r.usage_complete,r.provider_region,r.billing_mode,r.first_token_ms,r.client_id,(SELECT COUNT(*) FROM request_attempts t WHERE t.request_id=r.id)`

func validateQuery(q RequestQuery) error {
	if len(q.Search) > 128 || !utf8.ValidString(q.Search) || q.Offset < 0 || q.Offset > MaxRetention || q.Limit < 1 || q.Limit > 200 {
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
func (s *Service) Requests(ctx context.Context, q RequestQuery) (RequestPage, error) {
	page := RequestPage{Items: []RequestItem{}, Limit: q.Limit}
	if err := validateQuery(q); err != nil {
		return page, err
	}
	where := `(?='' OR r.state=?) AND (?='' OR instr(lower(r.id || ' ' || r.input_model || ' ' || r.model || ' ' || r.account_name || ' ' || r.email || ' ' || r.workspace_id || ' ' || r.client_id),lower(?))>0)`
	args := []any{q.State, q.State, q.Search, q.Search}
	if s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM request_records r WHERE `+where, args...).Scan(&page.Total) != nil {
		return page, ErrUnavailable
	}
	if s.db.QueryRowContext(ctx, `SELECT max_records FROM request_retention WHERE id=1`).Scan(&page.MaxRecords) != nil {
		return page, ErrUnavailable
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+requestColumns+` FROM request_records r LEFT JOIN accounts a ON a.id=r.account_id WHERE `+where+` ORDER BY r.started_at DESC,r.id DESC LIMIT ? OFFSET ?`, append(args, q.Limit, q.Offset)...)
	if err != nil {
		return page, ErrUnavailable
	}
	defer rows.Close()
	for rows.Next() {
		item, err := scanRequest(rows)
		if err != nil {
			return page, ErrUnavailable
		}
		page.Items = append(page.Items, item)
	}
	if rows.Err() != nil {
		return page, ErrUnavailable
	}
	return page, nil
}
func (s *Service) Request(ctx context.Context, id string) (RequestItem, error) {
	if len(id) != 32 || strings.ContainsAny(id, "\r\n\x00") {
		return RequestItem{}, ErrInvalid
	}
	item, err := scanRequest(s.db.QueryRowContext(ctx, `SELECT `+requestColumns+` FROM request_records r LEFT JOIN accounts a ON a.id=r.account_id WHERE r.id=?`, id))
	if err == sql.ErrNoRows {
		return item, ErrInvalid
	}
	if err != nil {
		return item, ErrUnavailable
	}
	item.Attempts, err = s.attempts(ctx, id)
	if err != nil {
		return item, err
	}
	return item, nil
}

type scanner interface{ Scan(...any) error }

func scanRequest(row scanner) (RequestItem, error) {
	var item RequestItem
	var input, output int64
	err := row.Scan(&item.ID, &item.StartedAt, &item.FinishedAt, &item.Protocol, &item.Transport, &item.InputModel, &item.Model, &item.Stream, &item.ProviderID, &item.ProviderName, &item.AccountID, &item.AccountName, &item.Email, &item.WorkspaceID, &item.AccountDeleted, &item.State, &item.StatusCode, &item.ErrorCode, &item.LatencyMS, &input, &output, &item.InputKnown, &item.OutputKnown, &item.UsageComplete, &item.ProviderRegion, &item.BillingMode, &item.FirstTokenMS, &item.Client, &item.AttemptCount)
	item.InputTokens = strconv.FormatInt(input, 10)
	item.OutputTokens = strconv.FormatInt(output, 10)
	return item, err
}
