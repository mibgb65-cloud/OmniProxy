package minimax

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"
)

// Quota 仅包含官方公开资源数据，不混入本机用量或账号密钥。
type Quota struct {
	Buckets   []QuotaBucket `json:"buckets"`
	FetchedAt string        `json:"fetchedAt"`
}

// QuotaBucket 同一套餐的资源分组不能相加为账号总余额。
type QuotaBucket struct {
	Model    string      `json:"model"`
	Interval QuotaWindow `json:"interval"`
	Weekly   QuotaWindow `json:"weekly"`
}

// QuotaWindow 未知／无限额和百分比零是不同状态。
type QuotaWindow struct {
	RemainingPercent *float64 `json:"remainingPercent,omitempty"`
	ResetsAt         string   `json:"resetsAt,omitempty"`
	Status           string   `json:"status"`
	BoostPermille    *int64   `json:"boostPermille,omitempty"`
}

func quotaFailure(code string) error { return errors.New("minimax: " + code) }

// ReadQuota 使用固定官方管理域名；仅接受订阅账号，不尝试普通余额接口。
func (c *Client) ReadQuota(ctx context.Context, o Options, key []byte) (quota Quota, result error) {
	address, err := o.QuotaURL()
	if err != nil || !ValidKeyForOptions(o, key) {
		return Quota{}, quotaFailure("invalid_configuration")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return Quota{}, quotaFailure("network_failed")
	}
	req.Header.Set("Authorization", "Bearer "+string(key))
	defer req.Header.Del("Authorization")
	response, err := c.http.Do(req)
	if err != nil {
		return Quota{}, quotaFailure("network_failed")
	}
	defer func() {
		if err := response.Body.Close(); err != nil && result == nil {
			result = quotaFailure("network_failed")
		}
	}()
	if response.StatusCode != 200 {
		code := "query_failed"
		switch response.StatusCode {
		case 401:
			code = "unauthorized"
		case 403:
			code = "forbidden"
		case 429:
			code = "rate_limited"
		}
		return Quota{}, quotaFailure(code)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, 131073))
	defer clear(raw)
	if err != nil || len(raw) > 131072 {
		return Quota{}, quotaFailure("invalid_response")
	}
	return parseQuota(raw, time.Now().UTC())
}
