package account

import (
	"context"
	"errors"
	"testing"
	"time"

	"omniproxy/internal/provider/stepfun"
)

const futureVersion = "9000-01-01T00:00:00Z"

func pinFutureVersion(t *testing.T, s *Service, id string) Item {
	t.Helper()
	if _, err := s.db.Exec("UPDATE accounts SET updated_at=? WHERE id=?", futureVersion, id); err != nil {
		t.Fatal(err)
	}
	item, err := s.Metadata(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return item
}

func requireAdvancedVersion(t *testing.T, previous, current string) {
	t.Helper()
	before, err := time.Parse(time.RFC3339Nano, previous)
	if err != nil {
		t.Fatal(err)
	}
	after, err := time.Parse(time.RFC3339Nano, current)
	if err != nil || !after.After(before) {
		t.Fatalf("account version did not advance: before=%s after=%s error=%v", previous, current, err)
	}
}

func TestAPIMutationsAdvancePersistedVersionAfterClockRollback(t *testing.T) {
	for _, change := range []string{"name", "billing", "key", "enabled"} {
		t.Run(change, func(t *testing.T) {
			s, _ := testService(t)
			s.store = &batchStore{values: map[string][]byte{}}
			ctx := context.Background()
			item, err := s.Create(ctx, CreateInput{ProviderID: "builtin_stepfun", Name: "test", APIKey: "placeholder-step-key", StepFun: &stepfun.Options{Region: "cn", BillingMode: "payg"}})
			if err != nil {
				t.Fatal(err)
			}
			item = pinFutureVersion(t, s, item.ID)
			// 重新构造服务，证明下界来自持久化账号，不依赖进程内计数器。
			s = NewService(s.db, s.store)
			var current Item
			if change == "enabled" {
				current, err = s.SetEnabled(ctx, item.ID, false)
			} else {
				input := UpdateInput{ID: item.ID, Name: "updated"}
				if change == "billing" {
					next := *item.StepFun
					next.BillingMode = "step_plan"
					input.StepFun = &next
				}
				if change == "key" {
					input.APIKey = "placeholder-replacement"
				}
				var result UpdateResult
				result, err = s.Update(ctx, input)
				current = result.Account
			}
			if err != nil {
				t.Fatal(err)
			}
			requireAdvancedVersion(t, item.UpdatedAt, current.UpdatedAt)
			called := false
			err = s.UseAPIKeyAt(ctx, item.ID, item.UpdatedAt, func([]byte) error { called = true; return nil })
			if !errors.Is(err, ErrConflict) || called {
				t.Fatal("old account snapshot was accepted")
			}
		})
	}
}

func TestRapidUpdatesKeepAdvancingAcrossServiceInstances(t *testing.T) {
	s, _ := testService(t)
	item, err := s.Create(context.Background(), CreateInput{ProviderID: "builtin_openai", Name: "test", APIKey: "placeholder-key"})
	if err != nil {
		t.Fatal(err)
	}
	item = pinFutureVersion(t, s, item.ID)
	for i := 0; i < 12; i++ {
		s = NewService(s.db, s.store)
		result, err := s.Update(context.Background(), UpdateInput{ID: item.ID, Name: "updated"})
		if err != nil {
			t.Fatal(err)
		}
		requireAdvancedVersion(t, item.UpdatedAt, result.Account.UpdatedAt)
		item = result.Account
	}
}

func TestVersionOverflowDoesNotWriteReplacementCredential(t *testing.T) {
	s, _ := testService(t)
	store := &batchStore{values: map[string][]byte{}}
	s.store = store
	item, err := s.Create(context.Background(), CreateInput{ProviderID: "builtin_openai", Name: "test", APIKey: "placeholder-key"})
	if err != nil {
		t.Fatal(err)
	}
	const maxVersion = "9999-12-31T23:59:59.999999999Z"
	if _, err := s.db.Exec("UPDATE accounts SET updated_at=? WHERE id=?", maxVersion, item.ID); err != nil {
		t.Fatal(err)
	}
	puts := store.puts
	_, err = s.Update(context.Background(), UpdateInput{ID: item.ID, Name: "test", APIKey: "placeholder-replacement"})
	if !errors.Is(err, ErrInvalidInput) || store.puts != puts || len(store.values) != 1 {
		t.Fatal("invalid timestamp wrote or replaced credential", err)
	}
	current, err := s.Metadata(context.Background(), item.ID)
	if err != nil || current.UpdatedAt != maxVersion {
		t.Fatal("failed timestamp mutation changed account", err)
	}
}
