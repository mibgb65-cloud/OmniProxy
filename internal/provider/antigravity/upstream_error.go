package antigravity

import (
	"bytes"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"omniproxy/internal/core/proxy"
)

// 稳定错误码：拒绝类 400 不可重试；额度类 429 带重置秒数交给 failover 休息。
const (
	codePromptTurnedAway  = "prompt_turned_away"
	codePromptRefusedOnce = "prompt_refused_before"
	codeQuotaExhausted    = "upstream_quota_exhausted"
	codeRateLimited       = "rate_limited"
)

// maxRest 与网关的额度阻断上限一致：更远的重置时间不当作休息依据。
const maxRest = 8 * 24 * time.Hour

// capacityReset 匹配「You have exhausted your capacity on this model. Your quota
// will reset after 3h19m30s.」里的重置时长（Go duration 形式）。
var capacityReset = regexp.MustCompile(`(?i)quota will reset after ([0-9hmsunµ.]+)`)

// upstreamError 是上游非 2xx 的白名单分类：状态、是否为内容拒绝、重置时长。
// 它不保存上游正文，Error() 只输出固定形态，禁止把正文带进日志或界面。
type upstreamError struct {
	status  int
	refusal bool
	reset   time.Duration
}

func (e *upstreamError) Error() string {
	return "google_subscription: upstream_" + strconv.Itoa(e.status)
}

// classifyFailure 把上游状态与（有界读取的）正文归类；只有 429 带语义。
func classifyFailure(status int, body []byte) *upstreamError {
	if status != 429 {
		return nil
	}
	classified := &upstreamError{status: status}
	low := bytes.ToLower(body)
	if match := capacityReset.FindSubmatch(low); match != nil {
		if delay, err := time.ParseDuration(strings.TrimRight(string(match[1]), ".")); err == nil && delay > 0 && delay <= maxRest {
			classified.reset = delay
		}
	}
	// 内容拒绝的正文固定为「Resource has been exhausted (e.g. check quota)」；
	// 容量耗尽用上面那句带时长的措辞，两者不可混用。
	if bytes.Contains(low, []byte("resource has been exhausted")) {
		classified.refusal = true
	}
	return classified
}

// failure 把传输错误转换为网关可用的稳定错误。
// system 是发送前的原文 system（未打散），用于确认拒绝是否由 prompt 行触发。
func (c *Client) failure(err error, cred Credentials, system string) error {
	var upstream *upstreamError
	if !errors.As(err, &upstream) {
		return ProxyError(err)
	}
	if upstream.refusal && turnsAway(system) {
		c.noteRefusal(cred, system)
		return &proxy.Error{Code: codePromptTurnedAway, StatusCode: 400}
	}
	if upstream.reset > 0 {
		seconds := int((upstream.reset + time.Second - 1) / time.Second)
		return &proxy.Error{Code: codeQuotaExhausted, StatusCode: 429, RetryAfter: strconv.Itoa(seconds)}
	}
	return &proxy.Error{Code: codeRateLimited, StatusCode: 429}
}
