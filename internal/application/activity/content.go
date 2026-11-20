package activity

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"errors"
	"io"
	"time"
	"unicode/utf8"

	"omniproxy/internal/core/proxy"
)

// ContentStoreBudget 是正文压缩数据的默认总预算；超出后删除最旧正文。
const ContentStoreBudget int64 = 512 << 20

// contentStoreBudget 可在测试中调小，生产保持 ContentStoreBudget。
var contentStoreBudget = ContentStoreBudget

// RequestContent 是请求详情里的完整正文快照；Available 为假表示未开启记录或历史记录。
// Truncated 只对迁移前的历史记录可能为真，新记录始终完整保存。
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

// RecordRequestContent 完整保存请求正文：gzip 压缩后入库，并保存列表预览。
// 重复写入保留首份；超出总预算时删除最旧正文，不触碰请求明细与累计。
func (s *Service) RecordRequestContent(ctx context.Context, event proxy.RequestContentEvent) error {
	if len(event.RequestID) != 32 || event.Content == "" || len(event.Content) > proxy.MaxPassthroughBytes ||
		event.OriginalBytes < 1 || event.OriginalBytes > proxy.MaxPassthroughBytes || !utf8.ValidString(event.Content) {
		return ErrInvalid
	}
	data, format, err := packContent(event.Content)
	if err != nil || len(data) == 0 || len(data) > proxy.MaxPassthroughBytes {
		return ErrUnavailable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrUnavailable
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO request_contents(request_id,data,format,preview,original_bytes,stored_bytes,truncated,captured_at) VALUES(?,?,?,?,?,?,0,?)`,
		event.RequestID, data, format, contentPreview(event.Content), event.OriginalBytes, len(data), timeText(time.Now().UTC()))
	if err != nil {
		return ErrUnavailable
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return ErrUnavailable
	}
	if inserted > 0 {
		if err := enforceContentBudget(ctx, tx); err != nil {
			return ErrUnavailable
		}
	}
	if err := tx.Commit(); err != nil {
		return ErrUnavailable
	}
	return nil
}

// RequestContent 读取并解压一条完整正文；不存在时返回 Available=false，而不是错误。
func (s *Service) RequestContent(ctx context.Context, id string) (RequestContent, error) {
	if len(id) != 32 {
		return RequestContent{}, ErrInvalid
	}
	var item RequestContent
	var data []byte
	var format string
	var truncated int
	err := s.db.QueryRowContext(ctx, `SELECT data,format,original_bytes,stored_bytes,truncated FROM request_contents WHERE request_id=?`, id).
		Scan(&data, &format, &item.OriginalBytes, &item.StoredBytes, &truncated)
	if errors.Is(err, sql.ErrNoRows) {
		return RequestContent{}, nil
	}
	if err != nil {
		return RequestContent{}, ErrUnavailable
	}
	content, err := decodeContent(data, format)
	if err != nil {
		return RequestContent{}, ErrUnavailable
	}
	item.Content, item.Available, item.Truncated = content, true, truncated == 1
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

// enforceContentBudget 删除最旧正文直到压缩总占用不超过预算。
func enforceContentBudget(ctx context.Context, tx *sql.Tx) error {
	var total int64
	if tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(stored_bytes),0) FROM request_contents`).Scan(&total) != nil {
		return ErrUnavailable
	}
	for total > contentStoreBudget {
		var id string
		var size int64
		if tx.QueryRowContext(ctx, `SELECT request_id,stored_bytes FROM request_contents ORDER BY captured_at ASC LIMIT 1`).Scan(&id, &size) != nil {
			return ErrUnavailable
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM request_contents WHERE request_id=?`, id); err != nil {
			return ErrUnavailable
		}
		total -= size
	}
	return nil
}

// packContent 优先 gzip；压缩反而更大时按明文保存，保证单条不超过请求体上限。
func packContent(content string) ([]byte, string, error) {
	compressed, err := compressContent(content)
	if err != nil {
		return nil, "", err
	}
	if len(compressed) < len(content) {
		return compressed, "gzip", nil
	}
	return []byte(content), "text", nil
}

func compressContent(content string) ([]byte, error) {
	var buffer bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buffer, gzip.BestSpeed)
	if err != nil {
		return nil, err
	}
	if _, err := writer.Write([]byte(content)); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// decodeContent 兼容迁移前的明文行与新写入的 gzip 行。
func decodeContent(data []byte, format string) (string, error) {
	if format == "text" {
		if !utf8.Valid(data) {
			return "", errors.New("activity: invalid text content")
		}
		return string(data), nil
	}
	if format != "gzip" {
		return "", errors.New("activity: unknown content format")
	}
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer reader.Close()
	content, err := io.ReadAll(io.LimitReader(reader, proxy.MaxPassthroughBytes+1))
	if err != nil {
		return "", err
	}
	if len(content) > proxy.MaxPassthroughBytes || !utf8.Valid(content) {
		return "", errors.New("activity: invalid compressed content")
	}
	return string(content), nil
}
