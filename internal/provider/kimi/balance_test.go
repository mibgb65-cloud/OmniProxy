package kimi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

const validBalance = `{"code":0,"status":true,"scode":"0x0","data":{"available_balance":0.000000000000000001,"voucher_balance":0.000000000000000001,"cash_balance":-3.00001}}`

func TestBalanceUsesRegionalEndpointAndPreservesDecimals(t *testing.T) {
	for _, tc := range []struct{ region, host, currency string }{{"cn", "api.moonshot.cn", "CNY"}, {"global", "api.moonshot.ai", "USD"}} {
		t.Run(tc.region, func(t *testing.T) {
			var request *http.Request
			client := New(roundTrip(func(r *http.Request) (*http.Response, error) {
				request = r
				if r.Method != "GET" || r.URL.String() != "https://"+tc.host+"/v1/users/me/balance" || r.Header.Get("Authorization") != "Bearer placeholder-kimi-key" {
					t.Fatal("wrong balance endpoint or authentication")
				}
				if _, ok := r.Context().Deadline(); !ok {
					t.Fatal("balance query has no deadline")
				}
				return reply(200, validBalance), nil
			}))
			value, err := client.ReadBalance(context.Background(), Options{Region: tc.region}, []byte("placeholder-kimi-key"))
			if err != nil || value.Currency != tc.currency || value.Available != "0.000000000000000001" || value.Voucher != value.Available || value.Cash != "-3.00001" {
				t.Fatalf("balance=%+v err=%v", value, err)
			}
			if request.Header.Get("Authorization") != "" {
				t.Fatal("authorization was not removed after the request")
			}
		})
	}
}

func TestBalanceRejectsInvalidResponses(t *testing.T) {
	for name, body := range map[string]string{
		"missing envelope":  `{}`,
		"missing code":      strings.Replace(validBalance, `"code":0,`, "", 1),
		"business error":    strings.Replace(validBalance, `"code":0`, `"code":123`, 1),
		"failed status":     strings.Replace(validBalance, `"status":true`, `"status":false`, 1),
		"null status":       strings.Replace(validBalance, `"status":true`, `"status":null`, 1),
		"missing data":      `{"code":0,"status":true,"scode":"0x0"}`,
		"missing field":     strings.Replace(validBalance, `,"cash_balance":-3.00001`, "", 1),
		"null amount":       strings.Replace(validBalance, `-3.00001`, `null`, 1),
		"quoted amount":     strings.Replace(validBalance, `-3.00001`, `"3.00"`, 1),
		"negative voucher":  strings.Replace(validBalance, `"voucher_balance":0.000000000000000001`, `"voucher_balance":-1`, 1),
		"excess precision":  strings.Replace(validBalance, `-3.00001`, `0.0000000000000000001`, 1),
		"excess magnitude":  strings.Replace(validBalance, `-3.00001`, strings.Repeat("9", 25), 1),
		"wrong type":        strings.Replace(validBalance, `-3.00001`, `true`, 1),
		"trailing document": validBalance + `{}`,
		"oversized":         strings.Repeat(" ", maxBalanceBytes) + validBalance,
	} {
		t.Run(name, func(t *testing.T) {
			client := New(roundTrip(func(*http.Request) (*http.Response, error) { return reply(200, body), nil }))
			if _, err := client.ReadBalance(context.Background(), Options{Region: "cn"}, []byte("placeholder-key")); err == nil || err.Error() != "balance: invalid_response" {
				t.Fatalf("invalid response accepted: %v", err)
			}
		})
	}
}

func TestBalanceSanitizesHTTPFailuresAndNeverFollowsRedirects(t *testing.T) {
	for _, tc := range []struct {
		status int
		code   string
	}{{401, "unauthorized"}, {403, "forbidden"}, {429, "rate_limited"}, {500, "service_unavailable"}, {302, "service_unavailable"}} {
		t.Run(tc.code+http.StatusText(tc.status), func(t *testing.T) {
			calls := 0
			client := New(roundTrip(func(*http.Request) (*http.Response, error) {
				calls++
				r := reply(tc.status, `private-upstream-message`)
				r.Header.Set("Location", "https://untrusted.invalid/balance")
				return r, nil
			}))
			if _, err := client.ReadBalance(context.Background(), Options{Region: "cn"}, []byte("placeholder-key")); err == nil || err.Error() != "balance: "+tc.code || calls != 1 {
				t.Fatalf("unexpected failure or cross-region retry: %v, calls=%d", err, calls)
			}
		})
	}
	client := New(roundTrip(func(*http.Request) (*http.Response, error) { return nil, errors.New("private-network-error") }))
	if _, err := client.ReadBalance(context.Background(), Options{Region: "cn"}, []byte("placeholder-key")); err == nil || err.Error() != "balance: network_failed" {
		t.Fatalf("raw network error exposed: %v", err)
	}
}

type failedBalanceBody struct{}

func (failedBalanceBody) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (failedBalanceBody) Close() error             { return nil }

func TestBalanceValidatesInputsAndReadFailures(t *testing.T) {
	calls := 0
	client := New(roundTrip(func(*http.Request) (*http.Response, error) {
		calls++
		r := reply(200, "")
		r.Body = failedBalanceBody{}
		return r, nil
	}))
	for _, key := range [][]byte{nil, []byte("bad\r\nkey"), []byte(strings.Repeat("k", 8193))} {
		if _, err := client.ReadBalance(context.Background(), Options{Region: "cn"}, key); err == nil {
			t.Fatal("invalid key accepted")
		}
	}
	if _, err := client.ReadBalance(context.Background(), Options{Region: "unknown"}, []byte("placeholder-key")); err == nil || calls != 0 {
		t.Fatal("invalid configuration sent a credential")
	}
	if _, err := client.ReadBalance(context.Background(), Options{Region: "cn"}, []byte("placeholder-key")); err == nil || err.Error() != "balance: invalid_response" {
		t.Fatal("body read failure accepted")
	}
}
