package subscription

import (
	"context"
	"testing"

	"omniproxy/internal/infrastructure/codex"
)

// fixedUsage 记录调用并返回固定结果，用来区分 HTTP 快路径与 stdio 回退。
type fixedUsage struct {
	result  codex.RateLimits
	err     error
	credits *codex.ResetCreditSummary
	credErr error
	calls   int
	creditN int
	token   []byte
	id      string
}

func (u *fixedUsage) Read(_ context.Context, token []byte, accountID string) (codex.RateLimits, error) {
	u.calls++
	u.token, u.id = append([]byte(nil), token...), accountID
	return u.result, u.err
}

func (u *fixedUsage) ReadResetCredits(_ context.Context, _ []byte, _ string) (*codex.ResetCreditSummary, error) {
	u.creditN++
	return u.credits, u.credErr
}

// resetCreditCards 构造一份带明细的官方重置卡汇总。
func resetCreditCards(count int64) *codex.ResetCreditSummary {
	granted, expires := int64(1761346800), int64(1800000000)
	return &codex.ResetCreditSummary{AvailableCount: &count, Credits: []codex.ResetCredit{
		{ID: "credit-one", Title: "额度重置卡", ResetType: "codexRateLimits", Status: "available", GrantedAt: &granted, ExpiresAt: &expires},
	}}
}

func quotaWindows(accountID string) codex.RateLimits {
	used := int32(20)
	five, week, resetsAt := int64(300), int64(10080), int64(1800000000)
	return codex.RateLimits{AccountID: accountID, Snapshot: &codex.RateLimitSnapshot{
		Plan:      "plus",
		Primary:   &codex.RateLimitWindow{UsedPercent: &used, DurationMins: &five, ResetsAt: &resetsAt},
		Secondary: &codex.RateLimitWindow{UsedPercent: &used, DurationMins: &week},
	}}
}

// quotaSubject 准备一个已授权账号，并用固定令牌替换受保护存储读取。
func quotaSubject(t *testing.T) (*fixture, string, *fixedUsage, *bool) {
	t.Helper()
	f := newFixture(t)
	view, client, _ := f.begin(t, "")
	client.authorize("quota@example.test")
	item := awaitState(t, f.manager, view.ID, "completed").Account
	usage := &fixedUsage{result: quotaWindows(item.AuthAccountID)}
	f.manager.SetUsageSource(usage)
	f.manager.readToken = func(_ context.Context, _, expectedID, _ string, consume func([]byte, string) error) error {
		return consume([]byte("access-token"), expectedID)
	}
	opened := false
	f.manager.factory = func(context.Context, string) (codex.Client, error) {
		opened = true
		return client, nil
	}
	return f, item.ID, usage, &opened
}

func TestQuotaUsesOfficialHTTPInterfaceWithoutStartingComponent(t *testing.T) {
	f, id, usage, opened := quotaSubject(t)
	quota, err := f.manager.ReadQuota(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if *opened || usage.calls != 1 || string(usage.token) != "access-token" {
		t.Fatal("quota read did not go through the official HTTP interface")
	}
	if quota.Plan != "plus" || quota.FiveHour == nil || quota.FiveHour.RemainingPercent != 80 || quota.Weekly == nil {
		t.Fatal("official usage response was not normalized into display windows")
	}
	if quota.FiveHour.ResetsAt != "2027-01-15T08:00:00Z" {
		t.Fatalf("reset time was not preserved: %q", quota.FiveHour.ResetsAt)
	}
}

func TestQuotaFallsBackWhenUsageInterfaceIsTransient(t *testing.T) {
	for _, failure := range []error{codex.ErrUsageTransient, codex.ErrProtocol} {
		f, id, usage, opened := quotaSubject(t)
		usage.err = failure
		quota, err := f.manager.ReadQuota(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if !*opened || usage.calls != 1 || quota.FiveHour == nil {
			t.Fatal("transient usage failure must fall back to the existing official component")
		}
	}
}

func TestQuotaReportsRejectedAuthorizationWithoutFallback(t *testing.T) {
	f, id, usage, opened := quotaSubject(t)
	usage.err = codex.ErrUsageUnauthorized
	_, err := f.manager.ReadQuota(context.Background(), id)
	info := QuotaErrorDiagnostic(err)
	if err == nil || err.Error() != "subscription: quota_not_authenticated" || info.Stage != "usage" || info.Code != "quota_not_authenticated" {
		t.Fatal("rejected authorization must not be reported as a generic quota failure")
	}
	if *opened {
		t.Fatal("rejected authorization must not start the official component")
	}
}

func TestQuotaRequiresOfficialComponentWhenRefreshIsNeeded(t *testing.T) {
	f, id, usage, _ := quotaSubject(t)
	f.manager.readToken = func(context.Context, string, string, string, func([]byte, string) error) error {
		return codex.ErrRefreshRequired
	}
	if _, err := f.manager.ReadQuota(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if usage.calls != 0 {
		t.Fatal("expired material must be refreshed by the official component, not sent over HTTP")
	}
}

func TestQuotaStillReturnsWhenSnapshotCannotBeSaved(t *testing.T) {
	f, id, _, _ := quotaSubject(t)
	// 空存储模拟本地快照不可写：读取已经成功，不能因此把额度变成失败。
	f.manager.SetQuotaCache(&QuotaCache{})
	warned := ""
	f.manager.SetWarn(func(message string, _ ...any) { warned = message })
	quota, err := f.manager.ReadQuota(context.Background(), id)
	if err != nil || quota.Weekly == nil {
		t.Fatal("a failed display snapshot must not discard a successful quota read")
	}
	if warned != "subscription quota snapshot not saved" {
		t.Fatal("snapshot failure was not reported to the log callback")
	}
}

func TestDisplayQuotaCarriesOfficialResetCredits(t *testing.T) {
	f, id, usage, opened := quotaSubject(t)
	usage.credits = resetCreditCards(1)
	quota, err := f.manager.ReadDisplayQuota(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if *opened || usage.calls != 1 || usage.creditN != 1 {
		t.Fatal("display quota must read windows and reset credits once over HTTP")
	}
	summary := quota.ResetCredits
	if summary == nil || summary.AvailableCount != 1 || !summary.DetailsKnown || len(summary.Credits) != 1 {
		t.Fatal("official reset credits were not published with the display quota")
	}
	if summary.Credits[0].ID != "credit-one" || summary.Credits[0].GrantedAt != "2025-10-24T23:00:00Z" {
		t.Fatal("reset credit details were not normalized for the interface")
	}
}

func TestObservationQuotaSkipsResetCredits(t *testing.T) {
	f, id, usage, _ := quotaSubject(t)
	usage.credits = resetCreditCards(1)
	quota, err := f.manager.ReadQuota(context.Background(), id)
	if err != nil || quota.Weekly == nil {
		t.Fatal("quota observation must still read the official windows")
	}
	if usage.creditN != 0 || quota.ResetCredits != nil {
		t.Fatal("background observation must not request or expose reset credits")
	}
}

func TestDisplayQuotaKeepsWindowsWhenResetCreditsFail(t *testing.T) {
	for _, failure := range []error{codex.ErrUsageTransient, codex.ErrProtocol, codex.ErrUsageUnauthorized} {
		f, id, usage, _ := quotaSubject(t)
		usage.credErr = failure
		warned := ""
		f.manager.SetWarn(func(message string, _ ...any) { warned = message })
		quota, err := f.manager.ReadDisplayQuota(context.Background(), id)
		if err != nil || quota.FiveHour == nil || quota.Weekly == nil {
			t.Fatal("a reset credit failure must not discard a successful quota read")
		}
		if quota.ResetCredits != nil {
			t.Fatal("unreadable reset credits must stay unknown instead of zero")
		}
		if warned != "subscription reset credits not read" {
			t.Fatal("reset credit failure was not reported to the log callback")
		}
	}
}
