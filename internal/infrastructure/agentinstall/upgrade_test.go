package agentinstall

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestUpdateRunsOnlyExactStorePackageWithSafeFlags(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows package manager")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "winget.exe"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", root)
	calls, progress := 0, 0
	b := &Backend{runInstall: func(_ context.Context, path string, args []string, report func(Progress)) error {
		calls++
		want := []string{"upgrade", "--id", "9P6K58THS811", "--exact", "--source", "msstore", "--include-unknown", "--silent", "--disable-interactivity", "--accept-package-agreements", "--accept-source-agreements"}
		if filepath.Base(path) != "winget.exe" || !reflect.DeepEqual(args, want) {
			t.Fatal("unexpected or unsafe update command")
		}
		report(Progress{Phase: "downloading"})
		return nil
	}}
	if err := b.Update(context.Background(), "claude-desktop", func(Progress) { progress++ }); err != nil || calls != 1 || progress != 1 {
		t.Fatal("update did not forward progress", err)
	}
	for _, id := range []string{"claude-desktop;bad", "claude-code", "codex-desktop", "--all"} {
		if err := b.Update(context.Background(), id, nil); err == nil || calls != 1 {
			t.Fatal("unsupported target updated")
		}
	}
	b.runInstall = func(context.Context, string, []string, func(Progress)) error { return errors.New("private output") }
	if err := b.Update(context.Background(), "claude-desktop", nil); !errors.Is(err, ErrUpdateInstallFailed) {
		t.Fatal("raw installer error escaped", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := b.Update(ctx, "claude-desktop", nil); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation lost", err)
	}
	t.Setenv("PATH", t.TempDir())
	if err := b.Update(context.Background(), "claude-desktop", nil); !errors.Is(err, ErrUnavailable) {
		t.Fatal("missing installer accepted", err)
	}
}

func TestDesktopVersionVerificationRejectsUnknownAndOlder(t *testing.T) {
	for _, tc := range []struct {
		current, target string
		want            bool
	}{
		{"2.10.0.0", "2.10.0", true}, {"2.11.0.0", "2.10.0", true}, {"2.9.0.0", "2.10.0", false}, {"Unknown", "2.10.0", false}, {"2.10.0.0", "", false},
	} {
		if DesktopVersionAtLeast(tc.current, tc.target) != tc.want {
			t.Fatal("incorrect verified version", tc.current, tc.target)
		}
	}
}
