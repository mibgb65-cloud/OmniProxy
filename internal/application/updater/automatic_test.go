package updater

import (
	"context"
	"testing"
)

func TestAutomaticDownloadVerifiesWithoutInstallingOrRepeating(t *testing.T) {
	s, _, launches := fixture(t, false)
	if _, err := s.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.StartAutomatic(); err != nil {
		t.Fatal(err)
	}
	waitTransfer(t, s, "ready")
	if *launches != 0 {
		t.Fatal("automatic download must never install")
	}
	if _, err := s.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	state, err := s.StartAutomatic()
	if err != nil || state.Download.State != "ready" {
		t.Fatalf("cached package downloaded again: %+v %v", state, err)
	}
}

func TestAutomaticDownloadRespectsCancellationAndAllowsManualRetry(t *testing.T) {
	s, _, _ := fixture(t, true)
	if _, err := s.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.StartAutomatic(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Cancel(); err != nil {
		t.Fatal(err)
	}
	waitTransfer(t, s, "cancelled")
	if _, err := s.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	state, err := s.StartAutomatic()
	if err != nil || state.Download.State != "cancelled" {
		t.Fatalf("cancelled package restarted automatically: %+v %v", state, err)
	}
	if _, err := s.Start(state.Release.Version); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Cancel(); err != nil {
		t.Fatal(err)
	}
	waitTransfer(t, s, "cancelled")
}

func TestAutomaticDownloadEligibility(t *testing.T) {
	for _, tc := range []struct {
		name     string
		install  bool
		transfer string
		want     string
	}{
		{"portable", false, "idle", "idle"},
		{"failed", true, "failed", "failed"},
		{"cancelled", true, "cancelled", "cancelled"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := fixture(t, false)
			state, err := s.Check(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			s.mu.Lock()
			s.opts.CanInstall = tc.install
			s.state.Download = Transfer{State: tc.transfer, Version: state.Release.Version}
			s.mu.Unlock()
			state, err = s.StartAutomatic()
			if err != nil || state.Download.State != tc.want {
				t.Fatalf("unexpected automatic transfer: %+v %v", state, err)
			}
		})
	}
}

func TestAutomaticDownloadWithoutCheckedCandidateIsNoOp(t *testing.T) {
	s, _, _ := fixture(t, false)
	state, err := s.StartAutomatic()
	if err != nil || state.Download.State != "idle" {
		t.Fatalf("downloaded without a checked candidate: %+v %v", state, err)
	}
}
