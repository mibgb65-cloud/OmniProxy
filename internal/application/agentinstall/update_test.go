package agentinstall

import (
	"context"
	"errors"
	"testing"
	"time"

	runtime "omniproxy/internal/infrastructure/agentinstall"
)

type updateBackend struct {
	*fakeBackend
	check func(context.Context, string, string) (runtime.UpdateCheck, error)
}

func (b *updateBackend) CheckUpdate(ctx context.Context, id, current string) (runtime.UpdateCheck, error) {
	return b.check(ctx, id, current)
}

func TestCheckUpdateUsesFreshInstalledVersionWithoutInstalling(t *testing.T) {
	_, base := fixture()
	base.detections["claude-desktop"] = runtime.Detection{State: "installed", Version: "2.9.0.0"}
	calls := 0
	b := &updateBackend{fakeBackend: base, check: func(ctx context.Context, id, current string) (runtime.UpdateCheck, error) {
		calls++
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 20*time.Second || id != "claude-desktop" || current != "2.9.0.0" {
			t.Fatal("wrong query context or installed version")
		}
		return runtime.UpdateCheck{State: "available", CurrentVersion: current, LatestVersion: "2.10.0"}, nil
	}}
	s := New(b)
	defer closeService(t, s)
	if calls != 0 || base.inspectCalls != 0 {
		t.Fatal("construction triggered update check")
	}
	result, err := s.CheckUpdate(context.Background(), "claude-desktop")
	if err != nil || result.State != "available" || calls != 1 || base.installCalls != 0 {
		t.Fatal("update check installed software or returned wrong state", result, err)
	}
	if itemFor(t, s.snapshot(), "claude-desktop").Detection.Version != "2.9.0.0" {
		t.Fatal("fresh installed version was not cached")
	}
	base.detections["claude-desktop"] = runtime.Detection{State: "not_installed"}
	if _, err := s.CheckUpdate(context.Background(), "claude-desktop"); !errors.Is(err, runtime.ErrDetectionFailed) || calls != 1 {
		t.Fatal("missing installation reached update source")
	}
	base.detections["claude-desktop"] = runtime.Detection{State: "unknown"}
	if _, err := s.CheckUpdate(context.Background(), "claude-desktop"); !errors.Is(err, runtime.ErrDetectionFailed) || calls != 1 {
		t.Fatal("unknown installation reached update source")
	}
}

func TestCheckUpdateRejectsInvalidUnsupportedBusyAndClosed(t *testing.T) {
	s, base := fixture()
	defer closeService(t, s)
	for _, id := range []string{"claude-desktop;bad", "other", "codex-cli", "claude-desktop"} {
		if _, err := s.CheckUpdate(context.Background(), id); err == nil || base.inspectCalls != 0 {
			t.Fatal("unsupported or invalid request inspected software")
		}
	}
	s.active = "claude-code"
	if _, err := s.CheckUpdate(context.Background(), "claude-desktop"); err == nil || err.Error() != "agentinstall: busy" {
		t.Fatal("update check allowed during installation")
	}
	s.active, s.closed = "", true
	if _, err := s.CheckUpdate(context.Background(), "claude-desktop"); err == nil || base.inspectCalls != 0 {
		t.Fatal("closed service accepted update check")
	}
}
