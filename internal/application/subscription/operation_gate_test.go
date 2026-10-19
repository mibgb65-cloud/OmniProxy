package subscription

import (
	"context"
	"errors"
	"testing"
	"time"

	"omniproxy/internal/infrastructure/codex"
)

func TestOperationGateCancellationDoesNotReleaseAnotherOwner(t *testing.T) {
	var gate operationGate
	if err := gate.Lock(context.Background()); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		err := gate.Lock(ctx)
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal("cancelled waiter acquired or released the occupied gate")
		}
	}
	gate.Unlock()
	if err := gate.Lock(context.Background()); err != nil {
		t.Fatal("gate was not reusable after cancellation")
	}
	gate.Unlock()
}

func TestQuotaQueueCancellationReturnsBeforeOwnerReleasesGate(t *testing.T) {
	for _, timeout := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "timeout"}[timeout], func(t *testing.T) {
			f := newFixture(t)
			opened := false
			f.manager.factory = func(context.Context, string) (codex.Client, error) {
				opened = true
				return nil, codex.ErrUnavailable
			}
			if err := f.manager.operations.Lock(context.Background()); err != nil {
				t.Fatal(err)
			}
			defer f.manager.operations.Unlock()
			ctx, cancel := context.WithCancel(context.Background())
			want := "operation_cancelled"
			if timeout {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
				want = "quota_busy"
			}
			defer cancel()
			done := make(chan error, 1)
			go func() {
				_, err := f.manager.ReadQuota(ctx, "unused")
				done <- err
			}()
			if !timeout {
				cancel()
			}
			select {
			case err := <-done:
				info := QuotaErrorDiagnostic(err)
				if opened || info.Stage != "queue" || info.Code != want || ErrorCode(err) != want {
					t.Fatal("queued query opened official component or misclassified contention as network failure")
				}
			case <-time.After(time.Second):
				t.Fatal("quota query ignored cancellation while waiting for gate")
			}
		})
	}
}

func TestQueuedSubscriptionOperationsCanBeCancelled(t *testing.T) {
	for name, operation := range map[string]func(*Service, context.Context) error{
		"begin":   func(s *Service, ctx context.Context) error { _, err := s.Begin(ctx, Input{}); return err },
		"refresh": func(s *Service, ctx context.Context) error { _, err := s.Refresh(ctx, "unused"); return err },
		"logout":  func(s *Service, ctx context.Context) error { _, err := s.Logout(ctx, "unused"); return err },
		"delete":  func(s *Service, ctx context.Context) error { return s.Delete(ctx, "unused") },
		"models":  func(s *Service, ctx context.Context) error { _, err := s.Models(ctx, "unused"); return err },
		"token": func(s *Service, ctx context.Context) error {
			return s.UseToken(ctx, "unused", func([]byte, string) error { t.Error("cancelled consumer ran"); return nil })
		},
		"reset": func(s *Service, ctx context.Context) error {
			_, err := s.ConsumeReset(ctx, ResetInput{AccountID: "unused", CreditID: "credit-a", OperationID: "11111111-2222-4333-8444-555555555555"})
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			s := &Service{}
			if err := s.operations.Lock(context.Background()); err != nil {
				t.Fatal(err)
			}
			defer s.operations.Unlock()
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- operation(s, ctx) }()
			select {
			case err := <-done:
				if ErrorCode(err) != "operation_cancelled" {
					t.Fatal("queued operation did not return a stable cancellation")
				}
			case <-time.After(time.Second):
				t.Fatal("queued operation did not stop")
			}
		})
	}
}
