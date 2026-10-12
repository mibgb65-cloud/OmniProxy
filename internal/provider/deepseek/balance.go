package deepseek

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
)

const balanceEndpoint = DefaultBaseURL + "/user/balance"
const maxBalanceBytes = 64 << 10

var balanceDecimal = regexp.MustCompile(`^-?[0-9]{1,24}(\.[0-9]{1,18})?$`)

// BalanceInfo 保留官方十进制文本，避免浮点舍入和多币种误加。
type BalanceInfo struct {
	Currency string `json:"currency"`
	Total    string `json:"total"`
	Granted  string `json:"granted"`
	ToppedUp string `json:"toppedUp"`
}

type Balance struct {
	Available bool          `json:"available"`
	Entries   []BalanceInfo `json:"entries"`
}

func balanceFailure(code string) error { return errors.New("balance: " + code) }

// Balance 只向固定官方地址发送凭据，不跟随提供商配置或重定向。
func (c *Client) Balance(ctx context.Context, key []byte) (result Balance, err error) {
	if len(key) == 0 || len(key) > 8192 {
		return result, balanceFailure("credential_unavailable")
	}
	for _, char := range key {
		if char < 33 || char > 126 {
			return result, balanceFailure("credential_unavailable")
		}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, balanceEndpoint, nil)
	if err != nil {
		return result, balanceFailure("network_failed")
	}
	request.Header.Set("Authorization", "Bearer "+string(key))
	request.Header.Set("Accept", "application/json")
	defer request.Header.Del("Authorization")
	response, err := c.http.Do(request)
	if err != nil {
		return result, balanceFailure("network_failed")
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && err == nil {
			result = Balance{}
			err = balanceFailure("network_failed")
		}
	}()
	if response.StatusCode != http.StatusOK {
		switch response.StatusCode {
		case 401:
			return result, balanceFailure("unauthorized")
		case 403:
			return result, balanceFailure("forbidden")
		case 429:
			return result, balanceFailure("rate_limited")
		default:
			return result, balanceFailure("service_unavailable")
		}
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, maxBalanceBytes+1))
	if readErr != nil || len(body) > maxBalanceBytes {
		return result, balanceFailure("invalid_response")
	}
	defer clear(body)
	var raw struct {
		Available *bool `json:"is_available"`
		Entries   *[]struct {
			Currency string `json:"currency"`
			Total    string `json:"total_balance"`
			Granted  string `json:"granted_balance"`
			ToppedUp string `json:"topped_up_balance"`
		} `json:"balance_infos"`
	}
	if json.Unmarshal(body, &raw) != nil || raw.Available == nil || raw.Entries == nil || len(*raw.Entries) > 2 {
		return result, balanceFailure("invalid_response")
	}
	result.Available, result.Entries = *raw.Available, []BalanceInfo{}
	seen := map[string]bool{}
	for _, entry := range *raw.Entries {
		if entry.Currency != "CNY" && entry.Currency != "USD" || seen[entry.Currency] || !balanceDecimal.MatchString(entry.Total) || !balanceDecimal.MatchString(entry.Granted) || !balanceDecimal.MatchString(entry.ToppedUp) {
			return Balance{}, balanceFailure("invalid_response")
		}
		seen[entry.Currency] = true
		result.Entries = append(result.Entries, BalanceInfo{Currency: entry.Currency, Total: entry.Total, Granted: entry.Granted, ToppedUp: entry.ToppedUp})
	}
	return result, nil
}
