//go:build windows

package agentinstall

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBackgroundConsolePreservesExitCodeWithoutRawOutput(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	path := filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	err := runConsoleCommand(ctx, path, []string{"/d", "/c", "exit /b -1978335189"}, &progressWriter{})
	if !errors.Is(err, ErrInstallFailed) || commandExitCode(err) != 0x8A15002B || err.Error() != "agentinstall: install_failed" {
		t.Fatal("exit code lost or raw process detail exposed", err)
	}
}

func TestBackgroundConsoleStreamsProgressWithoutInstallingSoftware(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var updates []Progress
	writer := &progressWriter{report: func(value Progress) { updates = append(updates, value) }}
	path := filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	if err := runConsoleCommand(ctx, path, []string{"/d", "/c", "echo 25 MB / 100 MB"}, writer); err != nil {
		t.Fatal(err)
	}
	writer.flush()
	if len(updates) == 0 || updates[len(updates)-1].Percent == nil || *updates[len(updates)-1].Percent != 25 {
		t.Fatal("console progress not streamed")
	}
}

func TestBackgroundConsoleHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	path := filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	writer := &progressWriter{}
	started := time.Now()
	err := runConsoleCommand(ctx, path, []string{"/d", "/c", "ping -n 10 127.0.0.1 >nul"}, writer)
	if err == nil || ctx.Err() == nil || time.Since(started) > 3*time.Second {
		t.Fatal("console cancellation not bounded")
	}
}
