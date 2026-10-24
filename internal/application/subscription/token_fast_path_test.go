package subscription

import (
	"context"
	"errors"
	"testing"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/infrastructure/codex"
)

func TestTokenFastPathDoesNotOpenOfficialComponentOrBlockOtherAccount(t *testing.T) {
	f := newFixture(t)
	view, client, _ := f.begin(t, "")
	client.authorize("slow@example.invalid")
	first := awaitState(t, f.manager, view.ID, "completed").Account
	view, client, _ = f.begin(t, "")
	client.authorize("fast@example.invalid")
	second := awaitState(t, f.manager, view.ID, "completed").Account
	f.manager.factory = func(context.Context, string) (codex.Client, error) {
		t.Error("健康凭据准备不应启动官方组件")
		return nil, codex.ErrUnavailable
	}
	f.manager.readToken = func(ctx context.Context, _ string, id, email string, consume func([]byte, string) error) error {
		if ctx.Err() != nil || id != second.AuthAccountID || email != second.AuthEmail {
			return errors.New("身份约束未传递给保护读取器")
		}
		key := []byte("test-only-token")
		defer clear(key)
		return consume(key, id)
	}
	if err := f.manager.operations.LockAccount(context.Background(), first.ID); err != nil {
		t.Fatal(err)
	}
	defer f.manager.operations.UnlockAccount(first.ID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	used := false
	if err := f.manager.UseTokenAt(ctx, second.ID, second.UpdatedAt, func([]byte, string) error { used = true; return nil }); err != nil || !used {
		t.Fatal("独立账号被全局额度／准备队列阻塞", err)
	}
}

func TestTokenFastPathRechecksVersionAfterReadingCredential(t *testing.T) {
	f := newFixture(t)
	view, client, _ := f.begin(t, "")
	client.authorize("member@example.invalid")
	item := awaitState(t, f.manager, view.ID, "completed").Account
	f.manager.readToken = func(ctx context.Context, _ string, id, _ string, consume func([]byte, string) error) error {
		_, err := f.accounts.UpdateSubscription(ctx, item.ID, account.SubscriptionMetadata{
			Email: item.AuthEmail, Plan: "changed", State: "authenticated", AccountID: item.AuthAccountID,
		})
		if err != nil {
			return err
		}
		return consume([]byte("test-only-token"), id)
	}
	used := false
	err := f.manager.UseTokenAt(context.Background(), item.ID, item.UpdatedAt, func([]byte, string) error { used = true; return nil })
	if ErrorCode(err) != "identity_changed" || used {
		t.Fatal("准备期间身份版本改变后仍发布旧凭据", err)
	}
}

func TestTokenOpensOfficialComponentOnlyWhenRefreshIsRequired(t *testing.T) {
	f := newFixture(t)
	view, client, _ := f.begin(t, "")
	client.authorize("member@example.invalid")
	item := awaitState(t, f.manager, view.ID, "completed").Account
	opens, reads := 0, 0
	f.manager.factory = func(context.Context, string) (codex.Client, error) {
		opens++
		return client, nil
	}
	f.manager.readToken = func(_ context.Context, _ string, id, _ string, consume func([]byte, string) error) error {
		reads++
		if reads == 1 {
			return codex.ErrRefreshRequired
		}
		return consume([]byte("test-refreshed-token"), id)
	}
	if err := f.manager.UseToken(context.Background(), item.ID, func([]byte, string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := f.manager.UseToken(context.Background(), item.ID, func([]byte, string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if opens != 1 || reads != 3 {
		t.Fatal("健康读取仍重复启动官方组件，或续期后没有重读校验")
	}
}

func TestTokenPreparationDeadlineIsReleasedBeforeModelConsumption(t *testing.T) {
	f := newFixture(t)
	view, client, _ := f.begin(t, "")
	client.authorize("member@example.invalid")
	item := awaitState(t, f.manager, view.ID, "completed").Account
	var preparation context.Context
	f.manager.readToken = func(ctx context.Context, _ string, id, _ string, consume func([]byte, string) error) error {
		preparation = ctx
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > operationTimeout {
			t.Error("普通 HTTP 准备缺少独立期限")
		}
		return consume([]byte("test-only-token"), id)
	}
	parent := context.Background()
	if err := f.manager.UseToken(parent, item.ID, func([]byte, string) error {
		if preparation.Err() != context.Canceled || parent.Err() != nil {
			t.Error("准备计时器没有释放，或错误取消了模型消费上下文")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
