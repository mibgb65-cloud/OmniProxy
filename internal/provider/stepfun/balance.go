package stepfun

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
)

// Balance 是普通账户查询结果；累计金额不能当作剩余分项，币种未在接口中给出。
type Balance struct {
	Currency     string `json:"currency,omitempty"`
	Type         string `json:"type"`
	Amount       string `json:"amount"`
	TotalCash    string `json:"totalCash"`
	TotalVoucher string `json:"totalVoucher"`
}

func failure(code string) error { return errors.New("stepfun: " + code) }

var decimalPattern = regexp.MustCompile(`^-?(0|[1-9][0-9]{0,14})(\.[0-9]{1,8})?$`)

// ReadBalance 查询所选区域按量余额；Step Plan 配额不调用此接口。
func (c *Client) ReadBalance(ctx context.Context, o Options, key []byte) (Balance, error) {
	if o.BillingMode != "payg" {
		return Balance{}, failure("unsupported")
	}
	body, err := c.read(ctx, o, key, "/accounts")
	if err != nil {
		return Balance{}, err
	}
	defer clear(body)
	var value struct {
		Object  string          `json:"object"`
		Type    string          `json:"type"`
		Amount  json.RawMessage `json:"balance"`
		Cash    json.RawMessage `json:"total_cash_balance"`
		Voucher json.RawMessage `json:"total_voucher_balance"`
	}
	if json.Unmarshal(body, &value) != nil || value.Object != "account" || (value.Type != "prepaid" && value.Type != "postpaid") {
		return Balance{}, failure("invalid_response")
	}
	for _, amount := range []json.RawMessage{value.Amount, value.Cash, value.Voucher} {
		if !decimalPattern.Match(amount) {
			return Balance{}, failure("invalid_response")
		}
	}
	if value.Cash[0] == '-' || value.Voucher[0] == '-' {
		return Balance{}, failure("invalid_response")
	}
	result := Balance{Type: value.Type, Amount: string(value.Amount), TotalCash: string(value.Cash), TotalVoucher: string(value.Voucher)}
	// 用户已通过国内控制台确认金额单位为人民币；海外单位仍未确认。
	if o.Region == "cn" {
		result.Currency = "CNY"
	}
	return result, nil
}
