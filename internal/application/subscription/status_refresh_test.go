package subscription

import (
	"context"
	"errors"
	"testing"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/infrastructure/codex"
)

type statusProbeClient struct {
	*testClient
	readAccount func(context.Context, bool) (*codex.Account, error)
	readID      func(context.Context) (string, error)
}

func (c statusProbeClient) Account(ctx context.Context, refresh bool) (*codex.Account, error) {
	return c.readAccount(ctx, refresh)
}
func (c statusProbeClient) AccountID(ctx context.Context) (string, error) {
	return c.readID(ctx)
}

func authorizedStatusFixture(t *testing.T) (*fixture, *testClient, account.Item) {
	t.Helper()
	f := newFixture(t)
	view, client, _ := f.begin(t, "")
	client.authorize("example@example.test")
	result := awaitState(t, f.manager, view.ID, "completed")
	if result.Account == nil {
		t.Fatal("authorized account missing")
	}
	return f, client, *result.Account
}

func TestStatusRefreshReadsManagedIdentityWithoutForcedRenewal(t *testing.T) {
	f, client, item := authorizedStatusFixture(t)
	f.manager.factory = func(context.Context, string) (codex.Client, error) {
		return statusProbeClient{testClient: client, readAccount: func(ctx context.Context, force bool) (*codex.Account, error) {
			if force {
				t.Error("ordinary status refresh forced OAuth renewal")
			}
			if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > operationTimeout {
				t.Error("status refresh has no bounded operation budget")
			}
			return client.Account(ctx, false)
		}, readID: func(context.Context) (string, error) {
			t.Error("known workspace ID triggered an unnecessary quota request")
			return "", errors.New("unnecessary request")
		}}, nil
	}
	updated, err := f.manager.Refresh(context.Background(), item.ID)
	if err != nil || updated.AuthState != "authenticated" || updated.AuthAccountID != item.AuthAccountID {
		t.Fatal("ordinary refresh lost authorization", err)
	}
}

func TestStatusRefreshFailuresKeepSavedAuthorization(t *testing.T) {
	for _, tc := range []struct {
		name, code string
		cause      error
	}{
		{"temporary", "refresh_unavailable", errors.New("private-upstream-detail")},
		{"deadline", "refresh_timeout", context.DeadlineExceeded},
		{"cancelled", "operation_cancelled", context.Canceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, client, item := authorizedStatusFixture(t)
			f.manager.factory = func(context.Context, string) (codex.Client, error) {
				return statusProbeClient{testClient: client, readAccount: func(context.Context, bool) (*codex.Account, error) {
					return nil, tc.cause
				}}, nil
			}
			_, err := f.manager.Refresh(context.Background(), item.ID)
			current, readErr := f.accounts.Metadata(context.Background(), item.ID)
			if ErrorCode(err) != tc.code || readErr != nil || current.AuthState != item.AuthState || current.UpdatedAt != item.UpdatedAt || current.AuthAccountID != item.AuthAccountID {
				t.Fatal("temporary refresh failure changed saved authorization", ErrorCode(err), readErr)
			}
		})
	}
}

func TestStatusRefreshQueueDeadlineIsNotUserCancellation(t *testing.T) {
	s := &Service{}
	if err := s.operations.Lock(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer s.operations.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := s.Refresh(ctx, "unused")
	if ErrorCode(err) != "refresh_busy" {
		t.Fatal("queue deadline was reported as an authorization cancellation", err)
	}
}

func TestStatusRefreshStillReportsMissingOrChangedIdentity(t *testing.T) {
	for _, missing := range []bool{false, true} {
		f, client, item := authorizedStatusFixture(t)
		f.manager.factory = func(context.Context, string) (codex.Client, error) {
			return statusProbeClient{testClient: client, readAccount: func(context.Context, bool) (*codex.Account, error) {
				if missing {
					return nil, nil
				}
				return &codex.Account{Type: "chatgpt", Email: "other@example.test", Plan: "team"}, nil
			}}, nil
		}
		updated, err := f.manager.Refresh(context.Background(), item.ID)
		if missing && (err != nil || updated.AuthState != "reauth_required") || !missing && ErrorCode(err) != "identity_changed" {
			t.Fatal("real authorization problem was hidden", err)
		}
	}
}

func TestStatusRefreshInitializationFailureKeepsSavedState(t *testing.T) {
	for _, cause := range []error{codex.ErrUnavailable, errors.New("private-initialization-detail")} {
		f, _, item := authorizedStatusFixture(t)
		f.manager.factory = func(context.Context, string) (codex.Client, error) { return nil, cause }
		_, err := f.manager.Refresh(context.Background(), item.ID)
		want := "refresh_unavailable"
		if cause == codex.ErrUnavailable {
			want = "codex_unavailable"
		}
		current, readErr := f.accounts.Metadata(context.Background(), item.ID)
		if ErrorCode(err) != want || readErr != nil || current.UpdatedAt != item.UpdatedAt || current.AuthState != "authenticated" {
			t.Fatal("component failure changed authorization or leaked raw errors", ErrorCode(err), readErr)
		}
	}
}

func TestStatusRefreshDoesNotPersistAfterCallerCancellation(t *testing.T) {
	f, client, item := authorizedStatusFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f.manager.factory = func(context.Context, string) (codex.Client, error) {
		return statusProbeClient{testClient: client, readAccount: func(context.Context, bool) (*codex.Account, error) {
			cancel()
			return &codex.Account{Type: "chatgpt", Email: item.AuthEmail, Plan: "team"}, nil
		}}, nil
	}
	_, err := f.manager.Refresh(ctx, item.ID)
	current, readErr := f.accounts.Metadata(context.Background(), item.ID)
	if ErrorCode(err) != "operation_cancelled" || readErr != nil || current.UpdatedAt != item.UpdatedAt || current.AuthPlan != item.AuthPlan {
		t.Fatal("cancelled status refresh persisted a stale result", err, readErr)
	}
}
