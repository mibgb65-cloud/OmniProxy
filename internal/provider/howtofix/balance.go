package howtofix

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Balance 是 OpenAI 兼容计费接口的展示值。金额按上游返回的美元数值四舍五入到分，
// 只用于界面展示，不参与记账。
//
// 该账号实测：GET /dashboard/billing/subscription 的 hard_limit_usd 就是剩余美元余额
// （soft／system_hard_limit_usd 与它同值，因此不再单独展示）；GET /dashboard/billing/usage
// 的 total_usage 在本部署中始终为 0，不反映真实支出，所以不读取、也不显示「已用」。
type Balance struct {
	Currency  string `json:"currency"`
	Remaining string `json:"remaining"`
}

func balanceFailure(code string) error { return errors.New("balance: " + code) }

// ReadBalance 用账号 Key 读取预付余额。
//
// 依据 handbook §6「Programmatic Billing API」：GET /dashboard/billing/subscription
// 返回 hard limit／current spend／active balance（USD），只在受控凭据回调内调用。
func (c *Client) ReadBalance(ctx context.Context, baseURL string, key []byte) (result Balance, err error) {
	if !validKey(key) {
		return Balance{}, balanceFailure("credential_unavailable")
	}
	subscription, err := c.getJSON(ctx, baseURL, "/dashboard/billing/subscription", key)
	if err != nil {
		return Balance{}, err
	}
	remaining, ok := amount(subscription["hard_limit_usd"])
	if !ok {
		return Balance{}, balanceFailure("invalid_response")
	}
	return Balance{Currency: "USD", Remaining: remaining}, nil
}

func (c *Client) getJSON(ctx context.Context, baseURL, path string, key []byte) (map[string]json.RawMessage, error) {
	address, err := endpointURL(baseURL, path)
	if err != nil {
		return nil, balanceFailure("configuration_unavailable")
	}
	requestCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, address, nil)
	if err != nil {
		return nil, balanceFailure("network_failed")
	}
	request.Header.Set("Authorization", "Bearer "+string(key))
	request.Header.Set("Accept", "application/json")
	defer request.Header.Del("Authorization")
	response, err := c.http.Do(request)
	if err != nil {
		return nil, balanceFailure("network_failed")
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, balanceFailure(statusCode(response.StatusCode))
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, maxBalanceBytes+1))
	defer clear(body)
	if readErr != nil || len(body) > maxBalanceBytes {
		return nil, balanceFailure("invalid_response")
	}
	fields := map[string]json.RawMessage{}
	if json.Unmarshal(body, &fields) != nil || fields == nil {
		return nil, balanceFailure("invalid_response")
	}
	return fields, nil
}

func statusCode(code int) string {
	switch code {
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusTooManyRequests:
		return "rate_limited"
	}
	return "service_unavailable"
}

// number 只接受有限数字，拒绝字符串、布尔、null 与 NaN／Inf。
func number(raw json.RawMessage) (float64, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || math.Abs(value) > 1e15 {
		return 0, false
	}
	return value, true
}

func amount(raw json.RawMessage) (string, bool) {
	value, ok := number(raw)
	if !ok {
		return "", false
	}
	return formatUSD(value), true
}

// formatUSD 用两位小数展示美元；负数（欠费）保留符号。
func formatUSD(value float64) string {
	text := strconv.FormatFloat(value, 'f', 2, 64)
	if len(text) > maxDisplayAmount {
		return text[:maxDisplayAmount]
	}
	return text
}
