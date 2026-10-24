package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// maxQuotaPrefixBytes 只够容纳首个额度拒绝事件：官方该拒绝是几十字节的 JSON 或一条
// 短 SSE 事件。更长的首事件（工具回显可达数百 KiB）不可能再被识别，必须尽早放行，
// 不能让判定拖慢真实流式回答的首字延迟。
const maxQuotaPrefixBytes = 4 << 10

// quotaPrefixChunkBytes 是单次预读的粒度，只影响判定时的系统调用次数。
const quotaPrefixChunkBytes = 512

// inspectPassthroughQuota 在写给客户端之前判定执行前额度拒绝。
// 命中时返回真实重置时间；未命中时把预读字节原样接回正文，客户端收到的内容不变。
//
// 这里不按上游声明的媒体类型过滤：官方可能用 200 或 429 包装同一个拒绝，媒体类型也不稳定
// （例如实际落成 text/plain）。判定强度来自正文证据本身，而预读上限只有 4 KiB，
// 不值得为一个不稳定的头部漏掉换号机会。
func inspectPassthroughQuota(response *http.Response, now time.Time) (time.Time, bool) {
	if response == nil || response.Body == nil {
		return time.Time{}, false
	}
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusTooManyRequests {
		return time.Time{}, false
	}
	prefix, body := readQuotaPrefix(response.Body)
	response.Body = body
	if len(prefix) == 0 {
		return time.Time{}, false
	}
	rejected, reset := QuotaRejected(prefix, now)
	return reset, rejected
}

// readQuotaPrefix 有界预读首个事件，并把预读字节接回正文。
// 读到的 EOF／错误不再重放：上游正文在这里最多只前进 4 KiB，后续读取按正常流程处理。
func readQuotaPrefix(body io.ReadCloser) ([]byte, io.ReadCloser) {
	prefix := make([]byte, 0, maxQuotaPrefixBytes)
	scratch := make([]byte, quotaPrefixChunkBytes)
	for len(prefix) < maxQuotaPrefixBytes {
		limit := min(len(scratch), maxQuotaPrefixBytes-len(prefix))
		n, err := body.Read(scratch[:limit])
		if n > 0 {
			prefix = append(prefix, scratch[:n]...)
			if quotaEventComplete(prefix) {
				break
			}
		}
		if err != nil {
			break
		}
	}
	if len(prefix) == 0 {
		return nil, body
	}
	return prefix, &prefixBody{reader: io.MultiReader(bytes.NewReader(prefix), body), closer: body}
}

// quotaEventComplete 判断首个事件是否已经完整，避免为判定而读完整个响应。
func quotaEventComplete(prefix []byte) bool {
	if json.Valid(bytes.TrimSpace(prefix)) {
		return true
	}
	return bytes.Contains(prefix, []byte("\n\n")) || bytes.Contains(prefix, []byte("\r\n\r\n"))
}

// prefixBody 把预读前缀与上游剩余正文接回去，Close 仍作用在上游正文上。
type prefixBody struct {
	reader io.Reader
	closer io.Closer
}

func (b *prefixBody) Read(p []byte) (int, error) { return b.reader.Read(p) }
func (b *prefixBody) Close() error               { return b.closer.Close() }
