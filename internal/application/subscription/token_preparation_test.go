package subscription

import (
	"context"
	"errors"
	"testing"
)

func TestTokenPreparationFailureCancellationAndPanicReleaseGate(t *testing.T) {
	for _, outcome := range []string{"reader_failure", "cancelled", "panic"} {
		t.Run(outcome, func(t *testing.T) {
			f := newFixture(t)
			view, client, _ := f.begin(t, "")
			client.authorize("prepared@example.test")
			item := awaitState(t, f.manager, view.ID, "completed").Account
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var prepared []byte
			readerErr := errors.New("test-reader-error")
			f.manager.readToken = func(_ context.Context, _ string, expectedID, _ string, consume func([]byte, string) error) error {
				key := []byte("test-only-token")
				defer clear(key)
				err := consume(key, expectedID)
				if outcome == "reader_failure" {
					return readerErr
				}
				if outcome == "cancelled" {
					cancel()
				}
				return err
			}
			// 不确定的准备结果不能进入消费；消费回调 panic 时仍须清零副本并释放准备锁。
			used := false
			var err error
			func() {
				defer func() {
					recovered := recover()
					if outcome == "panic" && recovered != "test-panic" {
						t.Error("consumer panic was lost")
					}
					if outcome != "panic" && recovered != nil {
						t.Error("unexpected panic")
					}
				}()
				err = f.manager.UseToken(ctx, item.ID, func(key []byte, _ string) error {
					used, prepared = true, key
					panic("test-panic")
				})
			}()
			if outcome == "panic" && !used || outcome != "panic" && used {
				t.Fatal("unexpected token consumption")
			}
			if outcome == "reader_failure" && !errors.Is(err, readerErr) || outcome == "cancelled" && ErrorCode(err) != "operation_cancelled" {
				t.Fatal("preparation error was not preserved")
			}
			for _, value := range prepared {
				if value != 0 {
					t.Fatal("temporary token was not cleared after panic")
				}
			}
			quota, quotaErr := f.manager.ReadQuota(context.Background(), item.ID)
			if quotaErr != nil || quota.FetchedAt == "" {
				t.Fatal("preparation failure or panic left operation gate locked")
			}
		})
	}
}
