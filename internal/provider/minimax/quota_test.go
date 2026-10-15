package minimax

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestQuotaPercentagesAndLegacyWindows(t *testing.T) {
	raw := []byte(`{"base_resp":{"status_code":0},"model_remains":[{"model_name":"general","current_interval_total_count":0,"current_interval_usage_count":0,"current_interval_remaining_percent":13.5,"current_weekly_remaining_percent":100,"end_time":1792108800000,"weekly_end_time":1792627200000,"weekly_boost_permille":1500},{"model_name":"MiniMax-M*","current_interval_total_count":1500,"current_interval_usage_count":1497},{"model_name":"video","current_interval_total_count":0,"current_interval_usage_count":0},{"model_name":"image-01","current_interval_status":3}]}`)
	q, err := parseQuota(raw, time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC))
	if err != nil || len(q.Buckets) != 4 {
		t.Fatal(q, err)
	}
	general := q.Buckets[0]
	if general.Interval.RemainingPercent == nil || *general.Interval.RemainingPercent != 13.5 || general.Weekly.BoostPermille == nil || *general.Weekly.BoostPermille != 1500 {
		t.Fatal("explicit percentage or multiplier lost")
	}
	if general.Interval.ResetsAt != time.UnixMilli(1792108800000).UTC().Format(time.RFC3339) {
		t.Fatal("reset unit changed")
	}
	if q.Buckets[1].Interval.RemainingPercent == nil || *q.Buckets[1].Interval.RemainingPercent != 99.8 {
		t.Fatal("legacy remaining count inverted")
	}
	if q.Buckets[2].Interval.Status != "unknown" || q.Buckets[2].Interval.RemainingPercent != nil {
		t.Fatal("0/0 became full quota")
	}
	if q.Buckets[3].Interval.Status != "unlimited" {
		t.Fatal("unlimited became unknown")
	}
}

func TestQuotaRejectsUntrustedOrIncompleteValues(t *testing.T) {
	for _, row := range []string{
		`{"model_name":"general","current_interval_remaining_percent":101}`,
		`{"model_name":"general","current_interval_remaining_percent":-1}`,
		`{"model_name":"general","current_interval_total_count":1,"current_interval_usage_count":2}`,
		`{"model_name":"general","current_weekly_total_count":-1}`,
		`{"model_name":"general","end_time":253402300800000}`,
		`{"model_name":"general","weekly_boost_permille":-1}`,
		`{"model_name":"bad\nname"}`,
		`null`,
	} {
		if _, err := parseQuota([]byte(`{"base_resp":{"status_code":0},"model_remains":[`+row+`]}`), time.Now()); err == nil {
			t.Fatal("invalid resource accepted", row)
		}
	}
	for _, raw := range []string{`{}`, `{"model_remains":[]}`, `{"base_resp":{"status_code":1004},"model_remains":[]}`, `{"base_resp":{"status_code":0},"model_remains":[{"model_name":"general"},{"model_name":"general"}]}`} {
		if _, err := parseQuota([]byte(raw), time.Now()); err == nil {
			t.Fatal("invalid page accepted")
		}
	}
}

func TestQuotaUsesOnlySelectedOfficialManagementHost(t *testing.T) {
	for _, region := range []string{"cn", "global"} {
		o := Options{Region: region, BillingMode: "subscription"}
		calls := 0
		c := New(roundTrip(func(r *http.Request) (*http.Response, error) {
			calls++
			expected := "https://www.minimax.cn/v1/token_plan/remains"
			if region == "global" {
				expected = "https://www.minimax.io/v1/token_plan/remains"
			}
			if r.URL.String() != expected || r.Header.Get("Authorization") != "Bearer sk-cp-placeholder" {
				t.Fatal("wrong management scope")
			}
			return reply(200, `{"base_resp":{"status_code":0},"model_remains":[]}`), nil
		}))
		if _, err := c.ReadQuota(context.Background(), o, []byte("sk-cp-placeholder")); err != nil || calls != 1 {
			t.Fatal("quota", err, calls)
		}
		o.BillingMode = "payg"
		if _, err := c.ReadQuota(context.Background(), o, []byte("placeholder")); err == nil || calls != 1 {
			t.Fatal("payg queried subscription resources")
		}
	}
	c := New(roundTrip(func(*http.Request) (*http.Response, error) {
		return reply(401, `{"message":"secret-key-not-for-ui"}`), nil
	}))
	if _, err := c.ReadQuota(context.Background(), Options{Region: "cn", BillingMode: "subscription"}, []byte("placeholder")); err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatal("private error leaked")
	}
}
