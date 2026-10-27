package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// resetCreditsPathSuffix 与官方客户端一致：重置卡汇总与额度地址同前缀。
const resetCreditsPathSuffix = "/rate-limit-reset-credits"

// ReadResetCredits 读取官方存下的重置卡汇总；实现不得返回令牌或原始响应。
// 失败只是一种补充信息缺失，调用方保持“暂未返回”，不得据此把数量当成零。
func (c *UsageClient) ReadResetCredits(ctx context.Context, accessToken []byte, accountID string) (*ResetCreditSummary, error) {
	if len(accessToken) == 0 || accountID == "" {
		return nil, ErrProtocol
	}
	target := c.resetCreditsTarget()
	if target == "" {
		return nil, ErrProtocol
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, ErrProtocol
	}
	request.Header.Set("Authorization", "Bearer "+string(accessToken))
	request.Header.Set("ChatGPT-Account-Id", accountID)
	request.Header.Set("User-Agent", "codex-cli")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("OpenAI-Beta", "codex-1")
	defer request.Header.Del("Authorization")
	response, err := c.http.Do(request)
	if err != nil {
		// 传输层失败（含超时与取消）只让重置卡信息保持未知。
		return nil, ErrUsageTransient
	}
	defer func() { _ = response.Body.Close() }()
	switch {
	case response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden:
		return nil, ErrUsageUnauthorized
	case response.StatusCode != http.StatusOK:
		return nil, ErrUsageTransient
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxUsageBytes))
	if err != nil {
		return nil, ErrUsageTransient
	}
	return parseResetCreditSummary(raw)
}

// resetCreditsTarget 只替换官方地址的路径段，不解析调用方输入，也不访问其他主机。
func (c *UsageClient) resetCreditsTarget() string {
	parsed, err := url.Parse(c.target())
	if err != nil || parsed.Host == "" {
		return ""
	}
	path := strings.TrimSuffix(strings.TrimSuffix(parsed.Path, "/"), "/usage")
	parsed.Path = path + resetCreditsPathSuffix
	parsed.RawQuery, parsed.Fragment = "", ""
	return parsed.String()
}

// parseResetCreditSummary 只保留公开的重置卡字段；credits:null 表示仅数量。
func parseResetCreditSummary(raw []byte) (*ResetCreditSummary, error) {
	var body map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&body) != nil || body == nil {
		return nil, ErrProtocol
	}
	if data, ok := body["data"].(map[string]any); ok {
		body = data
	}
	countValue, hasCount := firstResetField(body, "available_count", "availableCount")
	creditsValue, hasCredits := body["credits"]
	if !hasCount && !hasCredits {
		return nil, ErrProtocol
	}
	summary := &ResetCreditSummary{}
	if hasCount {
		count, ok := resetCreditInt(countValue)
		if !ok {
			return nil, ErrProtocol
		}
		summary.AvailableCount = &count
	}
	if hasCredits && creditsValue != nil {
		items, ok := creditsValue.([]any)
		if !ok {
			return nil, ErrProtocol
		}
		summary.Credits = make([]ResetCredit, 0, len(items))
		seen := map[string]bool{}
		for _, item := range items {
			card, ok := resetCreditFromAny(item)
			// 单张卡片字段不可用时只丢出该卡片，不否决整份汇总。
			if !ok || seen[card.ID] {
				continue
			}
			seen[card.ID] = true
			summary.Credits = append(summary.Credits, card)
		}
	}
	// 官方没有给出总数时不按明细长度推算，交给调用方保持未知。
	if summary.AvailableCount == nil || !validResetCredits(summary) {
		return nil, ErrProtocol
	}
	return summary, nil
}

func resetCreditFromAny(value any) (ResetCredit, bool) {
	record, ok := value.(map[string]any)
	if !ok {
		return ResetCredit{}, false
	}
	id := resetCreditString(record, "id", "credit_id", "creditId")
	if id == "" || !validMetadata(id, 256) {
		return ResetCredit{}, false
	}
	card := ResetCredit{
		ID:          id,
		Title:       resetCreditString(record, "title"),
		Description: resetCreditString(record, "description"),
		ResetType:   resetCreditString(record, "reset_type", "resetType", "type"),
		Status:      resetCreditString(record, "status", "state"),
	}
	if granted, ok := resetCreditTime(record, "granted_at", "created_at", "grantedAt"); ok {
		card.GrantedAt = &granted
	}
	if expires, ok := resetCreditTime(record, "expires_at", "expire_at", "expiresAt"); ok {
		card.ExpiresAt = &expires
	}
	// 官方可能不返回获得时间；缺失只由界面显示为未知，不否决该卡片。
	if !validMetadata(card.Title, 256) || !validMetadata(card.Description, 4096) || !validMetadata(card.ResetType, 80) || !validMetadata(card.Status, 80) {
		return ResetCredit{}, false
	}
	if card.GrantedAt != nil && !validUnixSeconds(*card.GrantedAt) || card.ExpiresAt != nil && !validUnixSeconds(*card.ExpiresAt) {
		return ResetCredit{}, false
	}
	return card, true
}

func firstResetField(source map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if value, ok := source[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func resetCreditString(source map[string]any, keys ...string) string {
	value, ok := firstResetField(source, keys...)
	if !ok {
		return ""
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func resetCreditInt(value any) (int64, bool) {
	var number int64
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		number = parsed
	case float64:
		number = int64(math.Round(typed))
	case int64:
		number = typed
	case int:
		number = int64(typed)
	case string:
		// 官方某些版本把计数序列化成字符串。
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, false
		}
		number = int64(math.Round(parsed))
	default:
		return 0, false
	}
	if number < 0 || number > 1000000 {
		return 0, false
	}
	return number, true
}

// resetCreditTime 接受秒、数字字符串或 RFC3339；无法识别的值按未知处理。
func resetCreditTime(source map[string]any, keys ...string) (int64, bool) {
	value, ok := firstResetField(source, keys...)
	if !ok {
		return 0, false
	}
	var seconds int64
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		seconds = parsed
	case float64:
		seconds = int64(math.Round(typed))
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return 0, false
		}
		if parsed, err := strconv.ParseInt(text, 10, 64); err == nil {
			seconds = parsed
			break
		}
		parsed, err := time.Parse(time.RFC3339Nano, text)
		if err != nil {
			return 0, false
		}
		seconds = parsed.Unix()
	default:
		return 0, false
	}
	if !validUnixSeconds(seconds) {
		return 0, false
	}
	return seconds, true
}

// validUnixSeconds 与 stdio 校验使用同一时间范围，避免越界时间进入界面。
func validUnixSeconds(value int64) bool {
	return value >= 0 && value <= 253402300799
}
