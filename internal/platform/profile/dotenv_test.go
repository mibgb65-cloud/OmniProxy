package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvSetsUnsetVariables(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "# comment\n\nA_KEY=first\nB_KEY=\"quoted value\"\nexport C_KEY='single'\nD_KEY=already-set\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("D_KEY", "preset")
	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv error: %v", err)
	}

	if got := os.Getenv("A_KEY"); got != "first" {
		t.Errorf("A_KEY = %q, want %q", got, "first")
	}
	if got := os.Getenv("B_KEY"); got != "quoted value" {
		t.Errorf("B_KEY = %q, want %q", got, "quoted value")
	}
	if got := os.Getenv("C_KEY"); got != "single" {
		t.Errorf("C_KEY = %q, want %q", got, "single")
	}
	if got := os.Getenv("D_KEY"); got != "preset" {
		t.Errorf("D_KEY = %q, want existing value preserved", got)
	}
}

func TestLoadDotEnvMissingFileIsNoop(t *testing.T) {
	if err := LoadDotEnv(filepath.Join(t.TempDir(), ".env")); err != nil {
		t.Fatalf("missing .env should not error, got: %v", err)
	}
}
