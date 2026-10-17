package agentinstall

import (
	"context"
	"errors"
	"testing"
	"time"

	runtime "omniproxy/internal/infrastructure/agentinstall"
)

func TestStartUpdateReusesFreshVerifiedReleaseWhenNetworkBecomesUnavailable(t *testing.T) {
	s, b := upgradeFixture(t)
	if _, err := s.CheckUpdate(context.Background(), "claude-desktop"); err != nil {
		t.Fatal(err)
	}
	queries := 0
	b.check = func(context.Context, string, string) (runtime.UpdateCheck, error) {
		queries++
		return runtime.UpdateCheck{}, runtime.ErrUpdateFailed
	}
	close(b.release)
	if _, err := s.StartUpdate(context.Background(), "claude-desktop"); err != nil {
		t.Fatal("fresh verified target should not need a second network query", err)
	}
	item := itemFor(t, waitFinished(t, s), "claude-desktop")
	if queries != 0 || item.InstallState != "completed" || item.Detection.Version != "2.10.0.0" {
		t.Fatal("cached target update failed", item, queries)
	}
}

func TestUpdateTargetRechecksExpiredChangedAndExplicitQueries(t *testing.T) {
	for _, mode := range []string{"expired", "future", "changed", "explicit"} {
		t.Run(mode, func(t *testing.T) {
			s, b := upgradeFixture(t)
			if _, err := s.CheckUpdate(context.Background(), "claude-desktop"); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "expired":
				s.lastUpdate.at = time.Now().Add(-updateCheckTTL)
			case "future":
				s.lastUpdate.at = time.Now().Add(time.Minute)
			case "changed":
				b.detections["claude-desktop"] = runtime.Detection{State: "installed", Version: "2.9.1.0"}
			}
			queries := 0
			b.check = func(context.Context, string, string) (runtime.UpdateCheck, error) {
				queries++
				return runtime.UpdateCheck{}, runtime.ErrUpdateFailed
			}
			var err error
			if mode == "explicit" {
				_, err = s.CheckUpdate(context.Background(), "claude-desktop")
			} else {
				_, err = s.StartUpdate(context.Background(), "claude-desktop")
			}
			if !errors.Is(err, runtime.ErrUpdateFailed) || queries != 1 || b.updateCalls != 0 || s.lastUpdate != nil {
				t.Fatal("stale or explicitly failed check reused", err, queries)
			}
		})
	}
}

func TestUpdateTargetDetectsRemovedOrAlreadyUpdatedClientWithoutNetwork(t *testing.T) {
	for _, mode := range []string{"removed", "current"} {
		t.Run(mode, func(t *testing.T) {
			s, b := upgradeFixture(t)
			if _, err := s.CheckUpdate(context.Background(), "claude-desktop"); err != nil {
				t.Fatal(err)
			}
			b.check = func(context.Context, string, string) (runtime.UpdateCheck, error) {
				t.Fatal("local installation change should be handled first")
				return runtime.UpdateCheck{}, nil
			}
			b.detections["claude-desktop"] = runtime.Detection{State: "installed", Version: "2.10.0.0"}
			if mode == "removed" {
				b.detections["claude-desktop"] = runtime.Detection{State: "not_installed"}
			}
			_, err := s.StartUpdate(context.Background(), "claude-desktop")
			if mode == "removed" && !errors.Is(err, runtime.ErrDetectionFailed) || mode == "current" && err != nil || b.updateCalls != 0 {
				t.Fatal("changed installation updated", err)
			}
		})
	}
}
