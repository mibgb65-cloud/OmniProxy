package stepfun

import (
	"context"
	"net/http"
	"testing"
)

func TestDiscoveryAndBalanceUseOrdinaryManagementEndpoint(t *testing.T) {
	client := New(roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "api.stepfun.ai" || r.Header.Get("Authorization") != "Bearer placeholder-step-key" {
			t.Fatal("wrong regional management endpoint")
		}
		if r.URL.Path == "/v1/models" {
			return reply(200, `{"data":[{"id":"step-5-preview"},{"id":"step-5-preview"},{"id":"stepaudio-3-tts"}]}`), nil
		}
		if r.URL.Path != "/v1/accounts" {
			t.Fatal("wrong management path")
		}
		return reply(200, `{"object":"account","type":"postpaid","balance":-12.3400,"total_cash_balance":100,"total_voucher_balance":5}`), nil
	}))
	o := Options{Region: "global", BillingMode: "payg"}
	ids, err := client.DiscoverModels(context.Background(), o, []byte("placeholder-step-key"))
	if err != nil || len(ids) != 1 || ids[0] != "step-5-preview" {
		t.Fatalf("models=%v %v", ids, err)
	}
	value, err := client.ReadBalance(context.Background(), o, []byte("placeholder-step-key"))
	if err != nil || value.Amount != "-12.3400" || value.TotalCash != "100" || value.Type != "postpaid" || value.Currency != "" {
		t.Fatalf("balance=%+v %v", value, err)
	}
	o.BillingMode = "step_plan"
	if _, err := client.ReadBalance(context.Background(), o, []byte("placeholder-step-key")); err == nil {
		t.Fatal("ordinary balance used for subscription")
	}
	if _, err := client.DiscoverModels(context.Background(), o, []byte("placeholder-step-key")); err == nil {
		t.Fatal("ordinary models used for subscription")
	}
}

func TestDomesticBalanceUsesConfirmedChineseYuan(t *testing.T) {
	client := New(roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://api.stepfun.com/v1/accounts" {
			t.Fatal("wrong domestic endpoint")
		}
		return reply(200, `{"object":"account","type":"prepaid","balance":123.4500,"total_cash_balance":1000,"total_voucher_balance":5}`), nil
	}))
	value, err := client.ReadBalance(context.Background(), Options{Region: "cn", BillingMode: "payg"}, []byte("placeholder-step-key"))
	if err != nil || value.Currency != "CNY" || value.Amount != "123.4500" {
		t.Fatalf("domestic currency or decimal changed: %+v %v", value, err)
	}
}

func TestMalformedManagementDataIsRejected(t *testing.T) {
	for _, body := range []string{`{}`, `{"object":"account","type":"prepaid","balance":1e100,"total_cash_balance":0,"total_voucher_balance":0}`, `{"object":"account","type":"unknown","balance":10,"total_cash_balance":0,"total_voucher_balance":0}`, `{"object":"account","type":"prepaid","balance":null,"total_cash_balance":0,"total_voucher_balance":0}`, `{"object":"account","type":"prepaid","balance":"10","total_cash_balance":0,"total_voucher_balance":0}`, `{"object":"account","type":"prepaid","balance":10,"total_cash_balance":-1,"total_voucher_balance":0}`} {
		client := New(roundTrip(func(*http.Request) (*http.Response, error) { return reply(200, body), nil }))
		if _, err := client.ReadBalance(context.Background(), Options{Region: "cn", BillingMode: "payg"}, []byte("placeholder-step-key")); err == nil {
			t.Fatal("malformed balance accepted")
		}
	}
}
