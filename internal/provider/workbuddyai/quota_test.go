package workbuddyai

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strings"
	"testing"
	"time"
)

func findWindow(windows []CreditWindow, name string) (CreditWindow, bool) {
	for _, window := range windows {
		if window.Name == name {
			return window, true
		}
	}
	return CreditWindow{}, false
}

func closeEnough(got, want float64) bool { return math.Abs(got-want) < 1e-6 }

func TestReadCreditsBuildsThreeWindows(t *testing.T) {
	var path string
	var header http.Header
	fixture := `{"code":0,"msg":"ok","data":{"IsPaidUser":true,"SubscriptionPackageCode":"PLAN",
"Packages":[
{"PackageCode":"PLAN","Kind":"plan_base","CycleTotalCapacity":"1000","CycleUsedCapacity":"250","CycleEndTime":"2027-01-01 00:00:00"},
{"PackageCode":"BONUS","CycleTotalCapacity":500,"CycleUsedCapacity":"100","DeductionEndTime":1900000000000},
{"PackageCode":"WALLET","CycleRemainCapacity":"88"}]}}`
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		path, header = r.URL.Path, r.Header.Clone()
		io.WriteString(w, fixture)
	})
	credits, err := client.ReadCredits(context.Background(), testCredentials())
	if err != nil {
		t.Fatal(err)
	}
	if path != "/billing/meter/get-user-resource-summary" {
		t.Fatalf("path = %q", path)
	}
	if header.Get("Authorization") != "Bearer access-token-1" || header.Get("X-User-Id") != "uid-1" {
		t.Fatalf("quota headers = %v", header)
	}
	if credits.Plan != "Pro" || credits.FetchedAt.IsZero() {
		t.Fatalf("credits = %+v", credits)
	}
	if len(credits.Windows) != 4 {
		t.Fatalf("windows = %+v", credits.Windows)
	}
	plan, _ := findWindow(credits.Windows, "Credits")
	if !closeEnough(plan.Used, 25) || !closeEnough(plan.Amount, 250) || !closeEnough(plan.Limit, 1000) {
		t.Fatalf("plan window = %+v", plan)
	}
	if plan.Display != "250 / 1,000" || plan.Unit != "credits" || !plan.Aside {
		t.Fatalf("plan window = %+v", plan)
	}
	if plan.ResetsAt != "2026-12-31T16:00:00Z" {
		t.Fatalf("plan resetsAt = %q", plan.ResetsAt)
	}
	if plan.RunsOutAt != "2026-12-31T16:00:00Z" || !closeEnough(plan.RunsOut, 750) {
		t.Fatalf("plan runsOut = (%q, %v)", plan.RunsOutAt, plan.RunsOut)
	}
	extra, _ := findWindow(credits.Windows, "Extra credits")
	if !closeEnough(extra.Used, 20) || !closeEnough(extra.Amount, 100) || !closeEnough(extra.Limit, 500) || !extra.Aside {
		t.Fatalf("extra window = %+v", extra)
	}
	if extra.Display != "100 / 500" {
		t.Fatalf("extra display = %q", extra.Display)
	}
	wantExtraEnd := time.UnixMilli(1900000000000).UTC().Format(time.RFC3339)
	if extra.RunsOutAt != wantExtraEnd || !closeEnough(extra.RunsOut, 400) {
		t.Fatalf("extra runsOut = (%q, %v)", extra.RunsOutAt, extra.RunsOut)
	}
	total, _ := findWindow(credits.Windows, "Total credits")
	if !closeEnough(total.Amount, 350) || !closeEnough(total.Limit, 1500) || !closeEnough(total.Used, 100*350.0/1500.0) {
		t.Fatalf("total window = %+v", total)
	}
	if total.Display != "350 / 1,500" || total.Aside || total.ResetsAt != "2026-12-31T16:00:00Z" || total.RunsOutAt != "2026-12-31T16:00:00Z" {
		t.Fatalf("total window = %+v", total)
	}
	prepaid, _ := findWindow(credits.Windows, "Prepaid credits")
	if !prepaid.Prepaid || !closeEnough(prepaid.Amount, 88) || prepaid.Unit != "credits" || prepaid.Display != "88 credits" {
		t.Fatalf("prepaid window = %+v", prepaid)
	}
	if prepaid.Used != 0 || prepaid.Limit != 0 {
		t.Fatalf("prepaid window = %+v", prepaid)
	}
}

func TestReadCreditsPrefersCreditGroups(t *testing.T) {
	fixture := `{"code":0,"data":{"IsPaidUser":true,"SubscriptionPackageCode":"PLAN",
"CreditGroups":[{"Kind":"plan_base","CycleTotalCapacity":"1000","CycleUsedCapacity":"250"},
{"Kind":"plan_bonus","CycleTotalCapacity":"500","CycleUsedCapacity":"100"}],
"Packages":[{"PackageCode":"PLAN","CycleTotalCapacity":"1000","CycleUsedCapacity":"250","CycleEndTime":"2027-01-01 00:00:00"},
{"PackageCode":"BONUS","CycleTotalCapacity":"500","CycleUsedCapacity":"100"},
{"PackageCode":"WALLET","CycleRemainCapacity":12}]}}`
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, fixture)
	})
	credits, err := client.ReadCredits(context.Background(), testCredentials())
	if err != nil {
		t.Fatal(err)
	}
	plan, _ := findWindow(credits.Windows, "Credits")
	if !closeEnough(plan.Limit, 1000) {
		t.Fatalf("plan limit = %v, want 1000 (groups must not double count packages)", plan.Limit)
	}
	extra, _ := findWindow(credits.Windows, "Extra credits")
	if !closeEnough(extra.Limit, 500) {
		t.Fatalf("extra limit = %v", extra.Limit)
	}
	total, _ := findWindow(credits.Windows, "Total credits")
	if !closeEnough(total.Limit, 1500) || !closeEnough(total.Amount, 350) {
		t.Fatalf("total = %+v", total)
	}
	if total.ResetsAt != "2026-12-31T16:00:00Z" {
		t.Fatalf("total resetsAt = %q", total.ResetsAt)
	}
	prepaid, found := findWindow(credits.Windows, "Prepaid credits")
	if !found || !closeEnough(prepaid.Amount, 12) {
		t.Fatalf("prepaid = %+v", prepaid)
	}
}

func TestReadCreditsCollapsesSingleKind(t *testing.T) {
	fixture := `{"code":0,"data":{"Packages":[
{"PackageCode":"A","CycleTotalCapacity":"100","CycleUsedCapacity":"40"},
{"PackageCode":"B","CycleTotalCapacity":100,"CycleUsedCapacity":10}]}}`
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, fixture)
	})
	credits, err := client.ReadCredits(context.Background(), testCredentials())
	if err != nil {
		t.Fatal(err)
	}
	if len(credits.Windows) != 1 {
		t.Fatalf("windows = %+v", credits.Windows)
	}
	window := credits.Windows[0]
	if window.Name != "Credits" || !closeEnough(window.Used, 25) || !closeEnough(window.Amount, 50) || !closeEnough(window.Limit, 200) {
		t.Fatalf("window = %+v", window)
	}
	if window.Display != "50 / 200" || window.Aside || window.Prepaid {
		t.Fatalf("window = %+v", window)
	}
}

func TestReadCreditsPrepaidOnly(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"code":0,"data":{"Packages":[{"PackageCode":"W","CycleRemainCapacity":"7.5"}]}}`)
	})
	credits, err := client.ReadCredits(context.Background(), testCredentials())
	if err != nil {
		t.Fatal(err)
	}
	if len(credits.Windows) != 1 || credits.Windows[0].Name != "Prepaid credits" {
		t.Fatalf("windows = %+v", credits.Windows)
	}
	if !closeEnough(credits.Windows[0].Amount, 7.5) || credits.Windows[0].Display != "7.5 credits" {
		t.Fatalf("window = %+v", credits.Windows[0])
	}
}

func TestReadCreditsClampsOverdrawnPercent(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"code":0,"data":{"Packages":[{"CycleTotalCapacity":"1,000","CycleUsedCapacity":"1,500"}]}}`)
	})
	credits, err := client.ReadCredits(context.Background(), testCredentials())
	if err != nil {
		t.Fatal(err)
	}
	if got := credits.Windows[0].Used; got != 100 {
		t.Fatalf("used = %v, want 100", got)
	}
	if !closeEnough(credits.Windows[0].Limit, 1000) || !closeEnough(credits.Windows[0].Amount, 1500) {
		t.Fatalf("window = %+v", credits.Windows[0])
	}
}

func TestReadCreditsPlanLabel(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"code":0,"data":{"IsPaidUser":true,"Packages":[{"CycleTotalCapacity":"10","CycleUsedCapacity":"1"}]}}`)
	})
	cred := testCredentials()
	cred.Plan = ""
	credits, err := client.ReadCredits(context.Background(), cred)
	if err != nil || credits.Plan != "Pro" {
		t.Fatalf("paid plan label = (%q, %v)", credits.Plan, err)
	}
	free := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"code":0,"data":{"Packages":[{"CycleTotalCapacity":"10","CycleUsedCapacity":"1"}]}}`)
	})
	credits, err = free.ReadCredits(context.Background(), cred)
	if err != nil || credits.Plan != "Free plan" {
		t.Fatalf("free plan label = (%q, %v)", credits.Plan, err)
	}
}

func TestReadCreditsRejectsUnparseableBody(t *testing.T) {
	cases := map[string]string{
		"no numbers":     `{"code":0,"data":{"Packages":[{"PackageCode":"X","CycleTotalCapacity":"abc"}],"Foo":1}}`,
		"not json":       `nope`,
		"array root":     `[{"CycleTotalCapacity":1}]`,
		"negative total": `{"code":0,"data":{"Packages":[{"CycleTotalCapacity":-5}]}}`,
	}
	for name, body := range cases {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			io.WriteString(w, body)
		})
		if _, err := client.ReadCredits(context.Background(), testCredentials()); ErrorCode(err) != CodeInvalidResponse {
			t.Errorf("%s => %v", name, err)
		}
	}
	oversized := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"packages":`+strings.Repeat(" ", maxCreditsBytes)+`}`)
	})
	if _, err := oversized.ReadCredits(context.Background(), testCredentials()); err == nil {
		t.Error("oversized quota body accepted")
	}
}

func TestReadCreditsMapsFailures(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   string
	}{
		{status: http.StatusUnauthorized, want: CodeUnauthorized},
		{status: http.StatusTooManyRequests, want: CodeRateLimited},
		{status: http.StatusInternalServerError, want: CodeUpstreamUnavailable},
		{status: http.StatusOK, body: `{"code":11128}`, want: CodeChannelRejected},
		{status: http.StatusOK, body: `{"code":10085}`, want: CodeUserAgentRefused},
		{status: http.StatusBadRequest, body: `{"code":11159}`, want: CodeSignatureInvalid},
	}
	for _, test := range cases {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(test.status)
			io.WriteString(w, test.body)
		})
		if _, err := client.ReadCredits(context.Background(), testCredentials()); ErrorCode(err) != test.want {
			t.Fatalf("status %d body %s => %v, want %q", test.status, test.body, err, test.want)
		}
	}
	if _, err := New(nil).ReadCredits(context.Background(), Credentials{}); ErrorCode(err) != CodeCredentialUnavailable {
		t.Fatalf("invalid credential => %v", err)
	}
}

func TestQuotaHelpers(t *testing.T) {
	if got := compactAmount(1234567); got != "1,234,567" {
		t.Fatalf("compactAmount = %q", got)
	}
	if got := compactAmount(1234.5); got != "1,234.5" {
		t.Fatalf("compactAmount decimal = %q", got)
	}
	if got := compactAmount(0); got != "0" {
		t.Fatalf("compactAmount zero = %q", got)
	}
	if normalizeKey("Cycle_Total-Capacity") != "cycletotalcapacity" {
		t.Fatalf("normalizeKey = %q", normalizeKey("Cycle_Total-Capacity"))
	}
	if at, ok := quotaTime("2026-12-31 16:00:00"); !ok || at.Format(time.RFC3339) != "2026-12-31T08:00:00Z" {
		t.Fatalf("beijing time = (%v, %v)", at, ok)
	}
	for _, value := range []any{"", "abc", json.Number("x"), true, nil, float64(-1)} {
		if _, ok := creditNumber(value); ok {
			t.Errorf("creditNumber(%v) accepted", value)
		}
	}
}

// 计费接口与对话接口形态不同：POST + 空对象正文 + 短 UA，不带 Agent 头。
// 用 GET 会被上游网关 404，这是 beta.16 上 Credits 读不出来而界面只报通用失败的原因。
func TestReadCreditsUsesBillingRequestShape(t *testing.T) {
	var method, body string
	var header http.Header
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		method, header = r.Method, r.Header.Clone()
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		io.WriteString(w, `{"code":0,"data":{"SubscriptionPackageName":"Free","IsPaidUser":false,"TotalCredits":"100","CreditGroups":[{"Kind":"plan_base","CycleTotalCapacity":"100","CycleUsedCapacity":"0"}],"Packages":[{"CycleTotalCapacity":"100","CycleRemainCapacity":"100"}]}}`)
	})
	if _, err := client.ReadCredits(context.Background(), testCredentials()); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPost {
		t.Fatalf("method = %q, want POST", method)
	}
	if strings.TrimSpace(body) != "{}" {
		t.Fatalf("body = %q, want {}", body)
	}
	if header.Get("User-Agent") != "WorkBuddy/"+AppVersion {
		t.Fatalf("billing UA = %q", header.Get("User-Agent"))
	}
	if header.Get("Content-Type") != "application/json" || header.Get("X-Product") != "SaaS" || header.Get("X-IDE-Type") != "WorkBuddy" {
		t.Fatalf("billing headers = %v", header)
	}
	// 对话专属头不能出现在计费请求上。
	for _, name := range []string{"X-Agent-Intent", "X-Agent-Type", "X-Stainless-Lang", "X-Requested-With"} {
		if header.Get(name) != "" {
			t.Fatalf("chat header %s leaked into billing request", name)
		}
	}
}

// 真实账号（Free 计划）的资源总览只有 plan_base 有总量：应给一个合计窗口与上游套餐名。
func TestReadCreditsFreePlanPayload(t *testing.T) {
	fixture := `{"code":0,"msg":"OK","data":{"Packages":[{"PackageCode":"TCACA_code_035","CycleTotalCapacity":"100","CycleRemainCapacity":"100","CycleUsedCapacity":"0","CapacityUnit":"credits","TotalCount":1}],
"TotalCredits":"100","SubscriptionPackageCode":"","SubscriptionPackageName":"Free","IsPaidUser":false,
"CreditGroups":[{"Kind":"plan_base","Name":"Plan Base Credits","CycleTotalCapacity":"100","CycleRemainCapacity":"100","CycleUsedCapacity":"0","CapacityUnit":"credits"},
{"Kind":"plan_bonus","Name":"Plan Bonus Credits","CycleTotalCapacity":"0","CycleUsedCapacity":"0"},
{"Kind":"addon","Name":"Top-up Credits","CycleTotalCapacity":"0","CycleUsedCapacity":"0"},
{"Kind":"benefit","Name":"Platform Reward Credits","CycleTotalCapacity":"0","CycleUsedCapacity":"0"}]}}`
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, fixture) })
	cred := testCredentials()
	cred.Plan = ""
	credits, err := client.ReadCredits(context.Background(), cred)
	if err != nil {
		t.Fatal(err)
	}
	if credits.Plan != "Free" {
		t.Fatalf("plan = %q, want Free (SubscriptionPackageName)", credits.Plan)
	}
	if len(credits.Windows) != 1 {
		t.Fatalf("windows = %+v", credits.Windows)
	}
	window := credits.Windows[0]
	if window.Name != "Credits" || !closeEnough(window.Used, 0) || !closeEnough(window.Limit, 100) || window.Aside {
		t.Fatalf("window = %+v", window)
	}
}
