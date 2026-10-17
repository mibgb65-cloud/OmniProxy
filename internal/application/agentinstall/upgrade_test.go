package agentinstall

import (
	"context"
	"errors"
	"testing"

	runtime "omniproxy/internal/infrastructure/agentinstall"
)

type upgradeBackend struct {
	*updateBackend
	version     string
	updateErr   error
	updateCalls int
}

func (b *upgradeBackend) Update(ctx context.Context, id string, report func(runtime.Progress)) error {
	b.mu.Lock()
	b.updateCalls++
	b.mu.Unlock()
	b.started <- id
	report(runtime.Progress{Phase: "downloading"})
	select {
	case <-b.release:
	case <-ctx.Done():
		return ctx.Err()
	}
	if b.version != "" {
		b.mu.Lock()
		b.detections[id] = runtime.Detection{State: "installed", Version: b.version}
		b.mu.Unlock()
	}
	return b.updateErr
}

func upgradeFixture(t *testing.T) (*Service, *upgradeBackend) {
	t.Helper()
	_, base := fixture()
	base.detections["claude-desktop"] = runtime.Detection{State: "installed", Version: "2.9.0.0"}
	b := &upgradeBackend{updateBackend: &updateBackend{fakeBackend: base, check: func(_ context.Context, _ string, current string) (runtime.UpdateCheck, error) {
		state := "available"
		if runtime.DesktopVersionAtLeast(current, "2.10.0") {
			state = "current"
		}
		return runtime.UpdateCheck{State: state, CurrentVersion: current, LatestVersion: "2.10.0"}, nil
	}}, version: "2.10.0.0"}
	s := New(b)
	s.List(context.Background())
	t.Cleanup(func() { closeService(t, s) })
	return s, b
}

func TestUpdateRequiresExplicitStartAndPreservesInstallationExclusion(t *testing.T) {
	s, b := upgradeFixture(t)
	if _, err := s.CheckUpdate(context.Background(), "claude-desktop"); err != nil || b.updateCalls != 0 {
		t.Fatal("checking performed update", err)
	}
	result, err := s.StartUpdate(context.Background(), "claude-desktop")
	if err != nil || !result.Busy || itemFor(t, result, "claude-desktop").Operation != "update" {
		t.Fatal("update did not start", err)
	}
	if id := <-b.started; id != "claude-desktop" {
		t.Fatal("wrong update target")
	}
	if _, err := s.StartUpdate(context.Background(), "claude-desktop"); err == nil {
		t.Fatal("duplicate update allowed")
	}
	if _, err := s.Start(context.Background(), "codex-cli"); err == nil {
		t.Fatal("installation allowed during update")
	}
	close(b.release)
	item := itemFor(t, waitFinished(t, s), "claude-desktop")
	if item.InstallState != "completed" || item.ErrorCode != "" || item.Detection.Version != "2.10.0.0" || item.UpdateVersion != "2.10.0" || b.installCalls != 0 {
		t.Fatal("update not verified", item)
	}
}

func TestUpdateDoesNotTreatOldInstalledVersionOrExitSuccessAsUpdated(t *testing.T) {
	for _, tc := range []struct {
		name, version, code string
		err                 error
	}{
		{"unchanged", "2.9.0.0", "update_unavailable", nil},
		{"partial-version", "2.9.1.0", "update_unavailable", nil},
		{"failed", "2.9.0.0", "update_failed", errors.New("private")},
		{"store-unavailable", "2.9.0.0", "update_unavailable", runtime.ErrUpdateUnavailable},
		{"updated-despite-error", "2.10.0.0", "", errors.New("private")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, b := upgradeFixture(t)
			b.version, b.updateErr = tc.version, tc.err
			close(b.release)
			if _, err := s.StartUpdate(context.Background(), "claude-desktop"); err != nil {
				t.Fatal(err)
			}
			item := itemFor(t, waitFinished(t, s), "claude-desktop")
			if item.ErrorCode != tc.code {
				t.Fatal("wrong result", item)
			}
			refreshed := itemFor(t, s.Refresh(context.Background()), "claude-desktop")
			if refreshed.ErrorCode != tc.code {
				t.Fatal("refresh masked failed update", refreshed)
			}
		})
	}
}

func TestUpdateRejectsInvalidUnknownCurrentAndInterrupted(t *testing.T) {
	s, b := upgradeFixture(t)
	for _, id := range []string{"claude-code", "claude-desktop;bad"} {
		if _, err := s.StartUpdate(context.Background(), id); err == nil {
			t.Fatal("invalid target accepted")
		}
	}
	b.check = func(context.Context, string, string) (runtime.UpdateCheck, error) {
		return runtime.UpdateCheck{State: "unknown"}, nil
	}
	if _, err := s.StartUpdate(context.Background(), "claude-desktop"); err == nil || b.updateCalls != 0 {
		t.Fatal("unknown release updated")
	}
	b.detections["claude-desktop"] = runtime.Detection{State: "installed", Version: "2.10.0.0"}
	b.check = func(context.Context, string, string) (runtime.UpdateCheck, error) {
		return runtime.UpdateCheck{State: "current"}, nil
	}
	if _, err := s.StartUpdate(context.Background(), "claude-desktop"); err != nil || b.updateCalls != 0 {
		t.Fatal("current version updated", err)
	}
	b.detections["claude-desktop"] = runtime.Detection{State: "installed", Version: "2.9.0.0"}
	b.check = func(context.Context, string, string) (runtime.UpdateCheck, error) {
		return runtime.UpdateCheck{State: "available", CurrentVersion: "2.9.0.0", LatestVersion: "2.10.0"}, nil
	}
	if _, err := s.StartUpdate(context.Background(), "claude-desktop"); err != nil {
		t.Fatal(err)
	}
	<-b.started
	if err := s.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	item := itemFor(t, s.snapshot(), "claude-desktop")
	if item.ErrorCode != "update_interrupted" {
		t.Fatal("interruption not reported", item)
	}
}
