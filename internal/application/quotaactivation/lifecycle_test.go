package quotaactivation

import (
	"context"
	"testing"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/gateway"
)

func TestLifecycleStartsOnceAndDisableCancelsCurrentBatch(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	started := make(chan struct{})
	f.service.sender = sendFunc(func(ctx context.Context, _ account.Item) error {
		f.sends.Add(1)
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	f.service.Start(context.Background())
	f.service.Start(context.Background())
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("enable did not start detection")
	}
	if _, err := gateway.NewCodexSettingsService(f.db).SetAutoQuotaActivation(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	f.service.SettingsChanged(false)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := f.service.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if f.sends.Load() != 1 {
		t.Fatal("duplicate worker started")
	}
	f.service.Wake(true)
	if f.service.Status().Running {
		t.Fatal("worker remains running after close")
	}
}
