package agentinstall

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOnlyFixedPackagesAndSafeFlagsAreInstalled(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows package manager")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "winget.exe"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", root)
	for _, product := range Products() {
		called := false
		b := &Backend{run: func(_ context.Context, path string, args ...string) ([]byte, error) {
			called = true
			if filepath.Base(path) != "winget.exe" || len(args) < 7 || args[0] != "install" || args[2] != product.PackageID {
				t.Fatal("unexpected installer command")
			}
			joined := strings.Join(args, " ")
			if !strings.Contains(joined, "--no-upgrade") || !strings.Contains(joined, "--exact") {
				t.Fatal("existing version not protected")
			}
			for _, bad := range []string{"--force", "--override", "--custom", "--ignore-security-hash", "--allow-reboot"} {
				if strings.Contains(joined, bad) {
					t.Fatal("unsafe installer argument")
				}
			}
			return nil, nil
		}}
		if err := b.Install(context.Background(), product.ID, nil); err != nil || !called {
			t.Fatal("fixed installer did not run", err)
		}
		called = false
		if err := b.Install(context.Background(), product.ID+";bad", nil); err == nil || called {
			t.Fatal("injected target reached command runner")
		}
	}
}

func TestOutputConsumptionIsBoundedAndVersionsAreValidated(t *testing.T) {
	buffer := &limitBuffer{}
	input := []byte(strings.Repeat("x", 128<<10))
	if n, err := buffer.Write(input); err != nil || n != len(input) || len(buffer.data) != 64<<10 {
		t.Fatal("output buffer not bounded")
	}
	for _, value := range []string{"1.2.3", "0.162.1", "26.1007.2314.0", "2.1.0-beta.1"} {
		if !validVersion(value) {
			t.Fatal("valid version rejected")
		}
	}
	for _, value := range []string{"", "private raw output", "1.2.3\nsecret", strings.Repeat("1", 129)} {
		if validVersion(value) {
			t.Fatal("untrusted output treated as version")
		}
	}
}
