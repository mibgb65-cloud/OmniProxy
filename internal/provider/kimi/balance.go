package kimi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"time"
)

const maxBalanceBytes = 64 << 10

var balanceDecimal = regexp.MustCompile(`^-?(0|[1-9][0-9]{0,23})(\.[0-9]{1,18})?$`)

// Balance 保留官方十进制文本；现金可为欠费负数，不能当作累计充值。
type Balance struct {
	Currency  string `json:"currency"`
	Available string `json:"available"`
	Voucher   string `json:"voucher"`
	Cash      string `json:"cash"`
}

func balanceFailure(code string) error { return errors.New("balance: " + code) }

// ReadBalance 仅查询账号所选地区的固定官方接口，不尝试另一地区的 Key。
func (c *Client) ReadBalance(ctx context.Context, o Options, key []byte) (result Balance, err error) {
	base, err := o.BaseURL()
	if err != nil {
		return result, balanceFailure("configuration_unavailable")
	}
	if !ValidAPIKey(key) {
		return result, balanceFailure("credential_unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/users/me/balance", nil)
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
			result, err = Balance{}, balanceFailure("network_failed")
		}
	}()
	if response.StatusCode != http.StatusOK {
		code := "service_unavailable"
		switch response.StatusCode {
		case http.StatusUnauthorized:
			code = "unauthorized"
		case http.StatusForbidden:
			code = "forbidden"
		case http.StatusTooManyRequests:
			code = "rate_limited"
		}
		return result, balanceFailure(code)
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, maxBalanceBytes+1))
	defer clear(body)
	if readErr != nil || len(body) > maxBalanceBytes {
		return result, balanceFailure("invalid_response")
	}
	var raw struct {
		Code   *int   `json:"code"`
		Status *bool  `json:"status"`
		Scode  string `json:"scode"`
		Data   struct {
			Available json.RawMessage `json:"available_balance"`
			Voucher   json.RawMessage `json:"voucher_balance"`
			Cash      json.RawMessage `json:"cash_balance"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &raw) != nil || raw.Code == nil || *raw.Code != 0 || raw.Status == nil || !*raw.Status || raw.Scode != "0x0" {
		return result, balanceFailure("invalid_response")
	}
	for _, amount := range []json.RawMessage{raw.Data.Available, raw.Data.Voucher, raw.Data.Cash} {
		if !balanceDecimal.Match(amount) {
			return result, balanceFailure("invalid_response")
		}
	}
	if raw.Data.Voucher[0] == '-' {
		return result, balanceFailure("invalid_response")
	}
	currency := "CNY"
	if o.Region == "global" {
		currency = "USD"
	}
	return Balance{Currency: currency, Available: string(raw.Data.Available), Voucher: string(raw.Data.Voucher), Cash: string(raw.Data.Cash)}, nil
}
