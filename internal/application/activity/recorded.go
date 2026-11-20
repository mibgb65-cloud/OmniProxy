package activity

import (
	"context"
	"unicode"
	"unicode/utf8"
)

const recordedSearchColumns = `r.id || ' ' || r.model || ' ' || r.account_name || ' ' || r.email || ' ' || r.client_id`

// RecordedQuery 是「记录」页的搜索与分页参数；只匹配元数据，不检索完整正文。
type RecordedQuery struct {
	Search string `json:"search"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}

// RecordedItem 是已保存正文的一条摘要；完整正文通过 RequestContent 按需读取。
type RecordedItem struct {
	ID            string `json:"id"`
	StartedAt     string `json:"startedAt"`
	CapturedAt    string `json:"capturedAt"`
	Model         string `json:"model"`
	ProviderName  string `json:"providerName"`
	AccountName   string `json:"accountName"`
	Client        string `json:"client"`
	State         string `json:"state"`
	Protocol      string `json:"protocol"`
	Transport     string `json:"transport"`
	Preview       string `json:"preview"`
	OriginalBytes int64  `json:"originalBytes"`
	StoredBytes   int64  `json:"storedBytes"`
	Truncated     bool   `json:"truncated"`
}

// RecordedPage 是按保存时间倒序的一页记录。
type RecordedPage struct {
	Items []RecordedItem `json:"items"`
	Total int            `json:"total"`
	Limit int            `json:"limit"`
}

// RecordedRequests 返回已保存正文的请求摘要；未记录正文的请求不会出现。
func (s *Service) RecordedRequests(ctx context.Context, q RecordedQuery) (RecordedPage, error) {
	page := RecordedPage{Items: []RecordedItem{}, Limit: q.Limit}
	if err := validateRecordedQuery(q); err != nil {
		return page, err
	}
	where := `(?='' OR instr(lower(` + recordedSearchColumns + `),lower(?))>0)`
	args := []any{q.Search, q.Search}
	if s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM request_contents c JOIN request_records r ON r.id=c.request_id WHERE `+where, args...).Scan(&page.Total) != nil {
		return page, ErrUnavailable
	}
	rows, err := s.db.QueryContext(ctx, `SELECT r.id,r.started_at,c.captured_at,r.model,r.provider_name,r.account_name,r.client_id,r.state,r.protocol,r.transport,c.original_bytes,c.stored_bytes,c.truncated,c.preview,CASE WHEN c.preview='' THEN c.data ELSE NULL END,c.format
		FROM request_contents c JOIN request_records r ON r.id=c.request_id WHERE `+where+`
		ORDER BY c.captured_at DESC,r.id DESC LIMIT ? OFFSET ?`, append(args, q.Limit, q.Offset)...)
	if err != nil {
		return page, ErrUnavailable
	}
	defer rows.Close()
	for rows.Next() {
		var item RecordedItem
		var truncated int
		var preview, format string
		var legacyData []byte
		if rows.Scan(&item.ID, &item.StartedAt, &item.CapturedAt, &item.Model, &item.ProviderName, &item.AccountName,
			&item.Client, &item.State, &item.Protocol, &item.Transport, &item.OriginalBytes, &item.StoredBytes,
			&truncated, &preview, &legacyData, &format) != nil {
			return page, ErrUnavailable
		}
		item.Truncated = truncated == 1
		// 迁移前的明文行没有预览，按需解出正文并生成一次；新行直接使用采集时的预览。
		if preview == "" && len(legacyData) > 0 {
			if content, err := decodeContent(legacyData, format); err == nil {
				preview = contentPreview(content)
			}
		}
		item.Preview = preview
		page.Items = append(page.Items, item)
	}
	if rows.Err() != nil {
		return page, ErrUnavailable
	}
	return page, nil
}

func validateRecordedQuery(q RecordedQuery) error {
	if len(q.Search) > 128 || !utf8.ValidString(q.Search) || q.Offset < 0 || q.Offset > MaxRetention || q.Limit < 1 || q.Limit > 50 {
		return ErrInvalid
	}
	for _, char := range q.Search {
		if unicode.IsControl(char) {
			return ErrInvalid
		}
	}
	return nil
}
