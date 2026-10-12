package agentinstall

import (
	"context"
	"math"
	"testing"
	"time"

	runtime "omniproxy/internal/infrastructure/agentinstall"
)

func TestInstallProgressIsPublishedValidatedAndClearedAfterCompletion(t *testing.T) {
	s, b := fixture()
	defer closeService(t, s)
	s.List(context.Background())
	if _, err := s.Start(context.Background(), "claude-code"); err != nil {
		t.Fatal(err)
	}
	<-b.started
	percent := 25.0
	b.updates <- runtime.Progress{Phase: "downloading", Percent: &percent, DownloadedBytes: 25 << 20, TotalBytes: 100 << 20}
	deadline := time.Now().Add(time.Second)
	for {
		item := itemFor(t, s.List(context.Background()), "claude-code")
		if item.Progress != nil && item.Progress.Percent != nil && *item.Progress.Percent == 25 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("progress not published")
		}
		time.Sleep(time.Millisecond)
	}
	invalid := math.NaN()
	b.updates <- runtime.Progress{Phase: "installing", Percent: &invalid}
	close(b.release)
	item := itemFor(t, waitFinished(t, s), "claude-code")
	if item.Progress != nil || item.StartedAt == "" || item.FinishedAt == "" {
		t.Fatal("terminal progress or timestamps incorrect")
	}
}
