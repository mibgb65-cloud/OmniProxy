package deepseek

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type balanceTransport func(*http.Request) (*http.Response, error)

func (f balanceTransport) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }
func balanceResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

const validBalanceJSON = `{"is_available":true,"balance_infos":[{"currency":"CNY","total_balance":"110.000000000000000001","granted_balance":"10.00","topped_up_balance":"100.00"},{"currency":"USD","total_balance":"-0.03","granted_balance":"0.00","topped_up_balance":"0.00"}]}`

func TestBalanceContractAndDecimalPrecision(t *testing.T) {
	var held *http.Request
	client := New(balanceTransport(func(request *http.Request) (*http.Response, error) {
		held = request
		if request.Method != "GET" || request.URL.String() != "https://api.deepseek.com/user/balance" || request.Header.Get("Authorization") != "Bearer placeholder-balance-key" || request.Header.Get("Accept") != "application/json" {
			t.Fatal("invalid balance contract")
		}
		return balanceResponse(200, validBalanceJSON), nil
	}))
	info, err := client.Balance(context.Background(), []byte("placeholder-balance-key"))
	if err != nil || !info.Available || len(info.Entries) != 2 || info.Entries[0].Total != "110.000000000000000001" || info.Entries[1].Total != "-0.03" {
		t.Fatal("money precision or currency changed")
	}
	if held.Header.Get("Authorization") != "" {
		t.Fatal("authorization header retained after query")
	}
}

func TestBalanceRejectsMalformedResponseAndCredential(t *testing.T) {
	for _, body := range []string{`{}`, `{"is_available":true,"balance_infos":null}`, `{"is_available":"true","balance_infos":[]}`, strings.Replace(validBalanceJSON, `"CNY"`, `"EUR"`, 1), strings.Replace(validBalanceJSON, `"10.00"`, `10.00`, 1), strings.Replace(validBalanceJSON, `"10.00"`, `"NaN"`, 1), validBalanceJSON + `{}`, strings.Repeat("x", maxBalanceBytes+1)} {
		client := New(balanceTransport(func(*http.Request) (*http.Response, error) { return balanceResponse(200, body), nil }))
		if _, err := client.Balance(context.Background(), []byte("placeholder-key")); err == nil || err.Error() != "balance: invalid_response" {
			t.Fatal("invalid balance response accepted or raw data exposed")
		}
	}
	for _, key := range []string{"", "bad\nkey", strings.Repeat("x", 8193)} {
		client := New(balanceTransport(func(*http.Request) (*http.Response, error) { t.Fatal("invalid key reached network"); return nil, nil }))
		if _, err := client.Balance(context.Background(), []byte(key)); err == nil {
			t.Fatal("invalid key accepted")
		}
	}
}

func TestBalanceErrorsAndRedirectsAreSafe(t *testing.T) {
	for status, code := range map[int]string{401: "unauthorized", 403: "forbidden", 429: "rate_limited", 503: "service_unavailable", 302: "service_unavailable"} {
		calls := 0
		client := New(balanceTransport(func(*http.Request) (*http.Response, error) {
			calls++
			response := balanceResponse(status, `private-balance-data`)
			response.Header.Set("Location", "https://untrusted.invalid")
			return response, nil
		}))
		if _, err := client.Balance(context.Background(), []byte("placeholder-key")); err == nil || err.Error() != "balance: "+code || calls != 1 {
			t.Fatal("unsafe error or followed redirect")
		}
	}
	client := New(balanceTransport(func(*http.Request) (*http.Response, error) { return nil, errors.New("private-network-data") }))
	if _, err := client.Balance(context.Background(), []byte("placeholder-key")); err == nil || err.Error() != "balance: network_failed" {
		t.Fatal("raw network error exposed")
	}
}

func TestBalanceRequestHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	client := New(balanceTransport(func(request *http.Request) (*http.Response, error) {
		<-request.Context().Done()
		return nil, request.Context().Err()
	}))
	if _, err := client.Balance(ctx, []byte("placeholder-key")); err == nil || err.Error() != "balance: network_failed" {
		t.Fatal("balance request ignored cancellation or exposed raw timeout")
	}
}
