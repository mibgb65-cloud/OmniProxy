package subscription

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"omniproxy/internal/infrastructure/codex"
)

type quotaProbeClient struct {
	codex.Client
	account func(context.Context, bool) (*codex.Account, error)
	limits  func(context.Context) (codex.RateLimits, error)
}

func (c quotaProbeClient) Account(ctx context.Context, refresh bool) (*codex.Account, error) {
	return c.account(ctx, refresh)
}
func (c quotaProbeClient) RateLimits(ctx context.Context) (codex.RateLimits, error) {
	return c.limits(ctx)
}

func TestQuotaRefreshesBeforeReadWithinOperationBudget(t *testing.T) {
	f := newFixture(t)
	view, client, _ := f.begin(t, "")
	client.authorize("quota@example.test")
	item := awaitState(t, f.manager, view.ID, "completed").Account
	client.mu.Lock()
	client.requireRefresh = true
	client.mu.Unlock()
	var calls []string
	f.manager.factory = func(ctx context.Context, _ string) (codex.Client, error) {
		deadline, ok := ctx.Deadline()
		if remaining := time.Until(deadline); !ok || remaining < 25*time.Second || remaining > operationTimeout {
			t.Fatal("quota startup and requests must share the bounded 30-second operation budget")
		}
		return quotaProbeClient{Client: client, account: func(ctx context.Context, refresh bool) (*codex.Account, error) {
			if !refresh {
				t.Fatal("quota did not request managed token refresh")
			}
			calls = append(calls, "refresh")
			return client.Account(ctx, refresh)
		}, limits: func(ctx context.Context) (codex.RateLimits, error) {
			calls = append(calls, "rate_limits")
			return client.RateLimits(ctx)
		}}, nil
	}
	if _, err := f.manager.ReadQuota(context.Background(), item.ID); err != nil {
		t.Fatal("refreshed quota query failed")
	}
	if !reflect.DeepEqual(calls, []string{"refresh", "rate_limits"}) {
		t.Fatal("quota read was not preceded by exactly one authorization refresh")
	}
}

func TestQuotaFailuresIdentifyStageWithoutDamagingAuthorization(t *testing.T) {
	for _, test := range []struct {
		name, stage, code, cause     string
		refreshErr, limitsErr        error
		missingIdentity, wrongMember bool
	}{
		{name: "refresh error", stage: "refresh", code: "quota_refresh_failed", cause: "operation_failed", refreshErr: errors.New("private-refresh-marker")},
		{name: "refresh timeout", stage: "refresh", code: "quota_timeout", cause: "quota_timeout", refreshErr: context.DeadlineExceeded},
		{name: "missing authorization", stage: "refresh", code: "quota_not_authenticated", cause: "quota_not_authenticated", missingIdentity: true},
		{name: "other Team member", stage: "identity", code: "identity_changed", cause: "identity_changed", wrongMember: true},
		{name: "quota request error", stage: "rate_limits", code: "quota_unavailable", cause: "request_failed", limitsErr: failure("request_failed")},
		{name: "missing quota snapshot", stage: "normalize", code: "quota_unavailable", cause: "quota_unavailable"},
		{name: "invalid quota protocol", stage: "rate_limits", code: "quota_unavailable", cause: "protocol_failed", limitsErr: codex.ErrProtocol},
		{name: "quota timeout", stage: "rate_limits", code: "quota_timeout", cause: "quota_timeout", limitsErr: context.DeadlineExceeded},
		{name: "cancelled query", stage: "rate_limits", code: "operation_cancelled", cause: "operation_cancelled", limitsErr: context.Canceled},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(t)
			view, client, _ := f.begin(t, "")
			client.authorize("quota@example.test")
			item := awaitState(t, f.manager, view.ID, "completed").Account
			reads := 0
			f.manager.factory = func(context.Context, string) (codex.Client, error) {
				return quotaProbeClient{Client: client, account: func(ctx context.Context, refresh bool) (*codex.Account, error) {
					if test.refreshErr != nil || test.missingIdentity {
						return nil, test.refreshErr
					}
					identity, err := client.Account(ctx, refresh)
					if test.wrongMember {
						identity.Email = "other@example.test"
					}
					return identity, err
				}, limits: func(context.Context) (codex.RateLimits, error) {
					reads++
					return codex.RateLimits{}, test.limitsErr
				}}, nil
			}
			_, err := f.manager.ReadQuota(context.Background(), item.ID)
			info := QuotaErrorDiagnostic(err)
			if err == nil || err.Error() != "subscription: "+test.code || info.Stage != test.stage || info.Code != test.code || info.CauseCode != test.cause {
				t.Fatal("quota failure lost its safe stage or classification")
			}
			canRead := test.refreshErr == nil && !test.missingIdentity && !test.wrongMember
			if (reads == 1) != canRead {
				t.Fatal("quota read continued after failed refresh or changed membership")
			}
			saved, saveErr := f.accounts.Metadata(context.Background(), item.ID)
			if saveErr != nil || saved.AuthState != item.AuthState || saved.UpdatedAt != item.UpdatedAt || client.logoutCount != 0 {
				t.Fatal("quota failure modified persisted authorization")
			}
		})
	}
}

func TestQuotaExpiredContextDoesNotOpenOfficialComponent(t *testing.T) {
	for _, timeout := range []bool{false, true} {
		f := newFixture(t)
		opened := false
		f.manager.factory = func(context.Context, string) (codex.Client, error) { opened = true; return nil, codex.ErrUnavailable }
		ctx, cancel := context.WithCancel(context.Background())
		want := "operation_cancelled"
		if timeout {
			cancel()
			ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			want = "quota_timeout"
		}
		cancel()
		_, err := f.manager.ReadQuota(ctx, "unused")
		info := QuotaErrorDiagnostic(err)
		if opened || info.Stage != "queue" || info.Code != want {
			t.Fatal("expired queued quota request opened the official component")
		}
	}
}
