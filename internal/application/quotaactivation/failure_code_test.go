package quotaactivation

import (
	"context"
	"errors"
	"testing"

	"omniproxy/internal/application/account"
)

// TestQuotaReadFailureIsReportedWithStableCode 断言额度读取失败不再被写成
// 笼统的 activation_failed，状态里能直接看到 quota_unavailable。
func TestQuotaReadFailureIsReportedWithStableCode(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	f.quotas.mu.Lock()
	f.quotas.err = errors.New("private quota error")
	f.quotas.mu.Unlock()
	if err := f.service.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	status := f.service.Status()
	if status.Failed != 1 || status.ErrorCode != "quota_unavailable" {
		t.Fatalf("具体原因丢失: %+v", status)
	}
}

// TestPreparationFailureKeepsSenderCode 断言发送方给出的具体原因（例如模型目录不可用）
// 会出现在状态里，而不是被 prepare_failed 覆盖。
func TestPreparationFailureKeepsSenderCode(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	f.service.sender = sendFunc(func(context.Context, account.Item, SendPlan) error {
		return errors.Join(ErrNotSent, failure("models_unavailable"))
	})
	if err := f.service.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	status := f.service.Status()
	if status.Failed != 1 || status.ErrorCode != "models_unavailable" || status.Sent != 0 {
		t.Fatalf("发送方原因丢失: %+v", status)
	}
}

// TestUncodedPreparationFailureUsesGenericCode 断言拿不到具体码时仍然给出稳定分类。
func TestUncodedPreparationFailureUsesGenericCode(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	f.service.sender = sendFunc(func(context.Context, account.Item, SendPlan) error { return ErrNotSent })
	if err := f.service.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if status := f.service.Status(); status.ErrorCode != "prepare_failed" {
		t.Fatalf("通用码缺失: %+v", status)
	}
}

// TestCodeOfReadsJoinedErrors 断言组合错误按顺序给出第一个稳定码。
func TestCodeOfReadsJoinedErrors(t *testing.T) {
	for _, test := range []struct {
		err  error
		want string
	}{
		{nil, ""},
		{ErrNotSent, ""},
		{failure("models_unavailable"), "models_unavailable"},
		{errors.Join(ErrNotSent, failure("models_unavailable")), "models_unavailable"},
		{errors.Join(failure("identity_changed"), failure("record_failed")), "identity_changed"},
		{errors.Join(ErrNotSent, ErrAwaitingQuota), ""},
		{errors.Join(failure("quota_unavailable"), errors.New("private")), "quota_unavailable"},
	} {
		if got := codeOf(test.err); got != test.want {
			t.Fatalf("codeOf(%v) = %q, want %q", test.err, got, test.want)
		}
	}
}
