package agentinstall

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	runtime "omniproxy/internal/infrastructure/agentinstall"
)

type fakeBackend struct {
	mu                         sync.Mutex
	detections                 map[string]runtime.Detection
	available                  bool
	started                    chan string
	release                    chan struct{}
	err                        error
	installCalls, inspectCalls int
	updates                    chan runtime.Progress
}

func (b *fakeBackend) Available() bool { return b.available }
func (b *fakeBackend) Inspect(_ context.Context, id string) runtime.Detection {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.inspectCalls++
	if item, ok := b.detections[id]; ok {
		return item
	}
	return runtime.Detection{State: "not_installed"}
}
func (b *fakeBackend) Install(ctx context.Context, id string, report func(runtime.Progress)) error {
	b.mu.Lock()
	b.installCalls++
	b.mu.Unlock()
	b.started <- id
loop:
	for {
		select {
		case update := <-b.updates:
			if report != nil {
				report(update)
			}
		case <-b.release:
			break loop
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if b.err != nil {
		return b.err
	}
	b.mu.Lock()
	b.detections[id] = runtime.Detection{State: "installed", Version: "1.2.3"}
	b.mu.Unlock()
	return nil
}
func fixture() (*Service, *fakeBackend) {
	b := &fakeBackend{detections: map[string]runtime.Detection{}, available: true, started: make(chan string, 4), release: make(chan struct{}), updates: make(chan runtime.Progress)}
	return New(b), b
}
func itemFor(t *testing.T, snapshot Snapshot, id string) Item {
	t.Helper()
	for _, item := range snapshot.Items {
		if item.ID == id {
			return item
		}
	}
	t.Fatal("missing installation item")
	return Item{}
}
func waitFinished(t *testing.T, s *Service) Snapshot {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		result := s.List(context.Background())
		if !result.Busy {
			return result
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("installation did not finish")
	return Snapshot{}
}
func closeService(t *testing.T, s *Service) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.Close(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestInstallIsSerializedAndPollsDoNotRerunDetection(t *testing.T) {
	s, b := fixture()
	defer closeService(t, s)
	s.List(context.Background())
	result, err := s.Start(context.Background(), "codex-cli")
	if err != nil || !result.Busy {
		t.Fatalf("start: %+v %v", result, err)
	}
	if <-b.started != "codex-cli" {
		t.Fatal("wrong installer target")
	}
	b.mu.Lock()
	checks := b.inspectCalls
	b.mu.Unlock()
	for range 10 {
		if !s.List(context.Background()).Busy {
			t.Fatal("lost running state")
		}
	}
	b.mu.Lock()
	after := b.inspectCalls
	b.mu.Unlock()
	if after != checks {
		t.Fatal("polling restarted detector")
	}
	if _, err := s.Start(context.Background(), "claude-code"); err == nil {
		t.Fatal("concurrent installation accepted")
	}
	close(b.release)
	completed := itemFor(t, waitFinished(t, s), "codex-cli")
	if completed.InstallState != "completed" || completed.Detection.State != "installed" || completed.Detection.Version != "1.2.3" {
		t.Fatalf("result: %+v", completed)
	}
}

func TestInvalidIDsAndExistingInstallationsNeverExecuteInstaller(t *testing.T) {
	s, b := fixture()
	defer closeService(t, s)
	for _, id := range []string{"", "codex-cli --override bad", "../codex", "CODEx-cli", strings.Repeat("x", 1024)} {
		if _, err := s.Start(context.Background(), id); err == nil {
			t.Fatal("invalid target accepted")
		}
	}
	b.detections["codex-cli"] = runtime.Detection{State: "installed", Version: "0.100.0"}
	result, err := s.Start(context.Background(), "codex-cli")
	if err != nil || result.Busy || itemFor(t, result, "codex-cli").Detection.Version != "0.100.0" || b.installCalls != 0 {
		t.Fatal("existing version changed")
	}
}

func TestDetectionAndMissingInstallerErrorsDoNotStartInstallation(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		s, b := fixture()
		b.available = false
		if unknown {
			b.detections["claude-code"] = runtime.Detection{State: "unknown", ErrorCode: "detection_failed"}
		}
		if _, err := s.Start(context.Background(), "claude-code"); err == nil {
			t.Fatal("unsafe installation accepted")
		}
		if b.installCalls != 0 || s.List(context.Background()).Busy {
			t.Fatal("installer ran or busy state leaked")
		}
		closeService(t, s)
	}
}

func TestFailureDoesNotLeakInstallerOutputAndCanRetry(t *testing.T) {
	s, b := fixture()
	defer closeService(t, s)
	b.err = errors.New("private token and command output")
	if _, err := s.Start(context.Background(), "claude-code"); err != nil {
		t.Fatal(err)
	}
	<-b.started
	close(b.release)
	result := waitFinished(t, s)
	if itemFor(t, result, "claude-code").ErrorCode != "install_failed" {
		t.Fatal("failure hidden")
	}
	encoded, err := json.Marshal(result)
	if err != nil || strings.Contains(string(encoded), "private") {
		t.Fatal("installer output leaked")
	}
	b.err = nil
	if _, err := s.Start(context.Background(), "claude-code"); err != nil {
		t.Fatal(err)
	}
	if itemFor(t, waitFinished(t, s), "claude-code").Detection.State != "installed" {
		t.Fatal("retry did not install")
	}
}

func TestTimeoutAndShutdownLeaveResultUncertain(t *testing.T) {
	s, b := fixture()
	s.installTimeout = 10 * time.Millisecond
	if _, err := s.Start(context.Background(), "claude-code"); err != nil {
		t.Fatal(err)
	}
	<-b.started
	if itemFor(t, waitFinished(t, s), "claude-code").ErrorCode != "install_interrupted" {
		t.Fatal("timeout misreported")
	}
	closeService(t, s)
	if _, err := s.Start(context.Background(), "claude-code"); err == nil {
		t.Fatal("closed service accepted installation")
	}
}
