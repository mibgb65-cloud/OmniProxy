package howtofix

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func balanceServer(t *testing.T, subscription string, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk_test_key" {
			t.Errorf("Authorization = %q", got)
		}
		if r.URL.Path != "/v1/dashboard/billing/subscription" {
			t.Errorf("unexpected path %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if status != 0 {
			w.WriteHeader(status)
		}
		io.WriteString(w, subscription)
	}))
}

func TestReadBalanceShowsRemaining(t *testing.T) {
	// 真实账号返回的字段形状：三者同值，hard_limit_usd 就是剩余美元余额。
	upstream := balanceServer(t, `{"object":"billing_subscription","has_payment_method":true,"soft_limit_usd":49.999998,"hard_limit_usd":49.999998,"system_hard_limit_usd":49.999998,"access_until":0}`, 0)
	defer upstream.Close()

	value, err := New(nil).ReadBalance(context.Background(), upstream.URL+"/v1", []byte("sk_test_key"))
	if err != nil {
		t.Fatal(err)
	}
	if value.Currency != "USD" || value.Remaining != "50.00" {
		t.Fatalf("balance = %+v", value)
	}
}

func TestReadBalanceRejectsUnsafeResponses(t *testing.T) {
	cases := map[string]string{
		"缺少余额字段":  `{"object":"billing_subscription"}`,
		"余额不是数字":  `{"hard_limit_usd":"12.34"}`,
		"不是 JSON": `not-json`,
		"空响应":     ``,
	}
	for name, subscription := range cases {
		t.Run(name, func(t *testing.T) {
			upstream := balanceServer(t, subscription, 0)
			defer upstream.Close()
			if _, err := New(nil).ReadBalance(context.Background(), upstream.URL+"/v1", []byte("sk_test_key")); err == nil ||
				!strings.HasPrefix(err.Error(), "balance: ") {
				t.Fatalf("unsafe response must fail with a balance code: %v", err)
			}
		})
	}
}

func TestReadBalanceMapsUpstreamStatusCodes(t *testing.T) {
	cases := map[int]string{
		http.StatusUnauthorized:    "balance: unauthorized",
		http.StatusForbidden:       "balance: forbidden",
		http.StatusTooManyRequests: "balance: rate_limited",
		http.StatusBadGateway:      "balance: service_unavailable",
	}
	for status, want := range cases {
		upstream := balanceServer(t, `{"hard_limit_usd":1}`, status)
		if _, err := New(nil).ReadBalance(context.Background(), upstream.URL+"/v1", []byte("sk_test_key")); err == nil || err.Error() != want {
			t.Fatalf("status %d: err = %v, want %q", status, err, want)
		}
		upstream.Close()
	}
}

func TestReadBalanceRejectsMissingKey(t *testing.T) {
	if _, err := New(nil).ReadBalance(context.Background(), DefaultBaseURL, nil); err == nil || err.Error() != "balance: credential_unavailable" {
		t.Fatalf("err = %v", err)
	}
}
