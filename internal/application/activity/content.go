package activity

import (
	"context"
	"database/sql"
	"errors"
	"time"
	"unicode/utf8"

	"omniproxy/internal/core/proxy"
)

// RequestContent 是请求详情里的正文快照；Available 为假表示未开启记录或历史记录。
type RequestContent struct {
	Content       string `json:"content"`
	OriginalBytes int64  `json:"originalBytes"`
	StoredBytes   int64  `json:"storedBytes"`
	Available     bool   `json:"available"`
	Truncated     bool   `json:"truncated"`
}

// ContentCaptureEnabled 报告是否保存新请求的正文；未迁移的数据库按不记录处理。
func (s *Service) ContentCaptureEnabled(ctx context.Context) (bool, error) {
	var enabled int
	err := s.db.QueryRowContext(ctx, `SELECT record_content FROM request_retention WHERE id=1`).Scan(&enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrUnavailable
	}
	if err != nil {
		return false, ErrUnavailable
	}
	return enabled == 1, nil
}

// SetContentCapture 保存开关；关闭后只停止记录新正文，已有内容保留到淘汰或手动清理。
func (s *Service) SetContentCapture(ctx context.Context, enabled bool) (Retention, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value := 0
	if enabled {
		value = 1
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE request_retention SET record_content=? WHERE id=1`, value); err != nil {
		return Retention{}, ErrUnavailable
	}
	return s.Retention(ctx)
}

// RecordRequestContent 保存一份已由 Core 校验并截断的请求正文；重复写入保留首份。
func (s *Service) RecordRequestContent(ctx context.Context, event proxy.RequestContentEvent) error {
	if len(event.RequestID) != 32 || event.Content == "" || len(event.Content) > proxy.MaxRequestContentBytes ||
		event.OriginalBytes < 1 || event.OriginalBytes > proxy.MaxPassthroughBytes || !utf8.ValidString(event.Content) {
		return ErrInvalid
	}
	truncated := 0
	if event.Truncated {
		truncated = 1
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO request_contents(request_id,content,original_bytes,stored_bytes,truncated,captured_at) VALUES(?,?,?,?,?,?)`,
		event.RequestID, event.Content, event.OriginalBytes, len(event.Content), truncated, timeText(time.Now().UTC()))
	if err != nil {
		return ErrUnavailable
	}
	return nil
}

// RequestContent 读取一条请求正文；不存在时返回 Available=false，而不是错误。
func (s *Service) RequestContent(ctx context.Context, id string) (RequestContent, error) {
	if len(id) != 32 {
		return RequestContent{}, ErrInvalid
	}
	var item RequestContent
	var truncated int
	err := s.db.QueryRowContext(ctx, `SELECT content,original_bytes,stored_bytes,truncated FROM request_contents WHERE request_id=?`, id).
		Scan(&item.Content, &item.OriginalBytes, &item.StoredBytes, &truncated)
	if errors.Is(err, sql.ErrNoRows) {
		return RequestContent{}, nil
	}
	if err != nil {
		return RequestContent{}, ErrUnavailable
	}
	item.Available, item.Truncated = true, truncated == 1
	return item, nil
}

// ClearRequestContents 删除全部已保存正文，返回删除条数；请求明细本身不受影响。
func (s *Service) ClearRequestContents(ctx context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result, err := s.db.ExecContext(ctx, `DELETE FROM request_contents`)
	if err != nil {
		return 0, ErrUnavailable
	}
	removed, err := result.RowsAffected()
	if err != nil {
		return 0, ErrUnavailable
	}
	return removed, nil
}
