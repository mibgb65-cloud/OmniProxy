//go:build windows

package credential

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"omniproxy/internal/infrastructure/storage"
	"omniproxy/internal/platform/profile"
)

func testStore(t *testing.T, kind profile.Kind, dataDir string) *Store {
	t.Helper()
	store, err := New(&profile.Profile{Kind: kind, DataDir: dataDir})
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestStoreRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name   string
		secret []byte
	}{
		{name: "API key", secret: []byte("synthetic-api-key-for-test")},
		{name: "large token bundle", secret: bytes.Repeat([]byte("synthetic-token-bundle-"), 200)},
		{name: "maximum-sized bundle", secret: bytes.Repeat([]byte{0xa5}, MaxSecretSize)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dataDir := t.TempDir()
			store := testStore(t, profile.Test, dataDir)
			ref, err := store.Put(tc.secret)
			if err != nil {
				t.Fatalf("Put: %v", err)
			}
			if !strings.HasPrefix(ref, refPrefix) {
				t.Fatalf("unexpected reference format: %q", ref)
			}

			path := filepath.Join(dataDir, "credentials", "test", ref+".bin")
			encrypted, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(encrypted, tc.secret) {
				t.Fatal("protected file contains plaintext")
			}
			if bytes.Equal(encrypted, tc.secret) {
				t.Fatal("protected file equals plaintext")
			}

			// A new store instance must read data persisted by the previous one.
			got, err := testStore(t, profile.Test, dataDir).Get(ref)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if !bytes.Equal(got, tc.secret) {
				t.Fatal("round trip changed the secret")
			}
			clear(got)
			if err := store.Delete(ref); err != nil {
				t.Fatalf("Delete: %v", err)
			}
			if _, err := store.Get(ref); !errors.Is(err, ErrNotFound) {
				t.Fatalf("Get after Delete = %v, want ErrNotFound", err)
			}
			if err := store.Delete(ref); !errors.Is(err, ErrNotFound) {
				t.Fatalf("second Delete = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestStoreProfileIsolation(t *testing.T) {
	dataDir := t.TempDir()
	development := testStore(t, profile.Development, dataDir)
	production := testStore(t, profile.Production, dataDir)
	ref, err := development.Put([]byte("synthetic-profile-secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := production.Get(ref); !errors.Is(err, ErrNotFound) {
		t.Fatalf("production Get = %v, want ErrNotFound", err)
	}
	if err := production.Delete(ref); !errors.Is(err, ErrNotFound) {
		t.Fatalf("production Delete = %v, want ErrNotFound", err)
	}
	if _, err := development.Get(ref); err != nil {
		t.Fatalf("development credential lost: %v", err)
	}
}

func TestStoreSurvivesDataDirectoryMove(t *testing.T) {
	source := filepath.Join(t.TempDir(), "current")
	destination := filepath.Join(t.TempDir(), "relocated")
	store := testStore(t, profile.Test, source)
	secret := []byte("synthetic-secret-after-move")
	ref, err := store.Put(secret)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.MoveDir(source, destination); err != nil {
		t.Fatal(err)
	}
	got, err := testStore(t, profile.Test, destination).Get(ref)
	if err != nil {
		t.Fatalf("Get after relocation: %v", err)
	}
	if !bytes.Equal(got, secret) {
		t.Fatal("relocated credential changed")
	}
	clear(got)
}

func TestStoreRejectsInvalidInputs(t *testing.T) {
	store := testStore(t, profile.Test, t.TempDir())
	var zero Store
	if _, err := zero.Put([]byte("value")); !errors.Is(err, ErrInvalidStore) {
		t.Fatalf("zero Store.Put = %v, want ErrInvalidStore", err)
	}
	if _, err := zero.Get(refPrefix + strings.Repeat("0", 32)); !errors.Is(err, ErrInvalidStore) {
		t.Fatalf("zero Store.Get = %v, want ErrInvalidStore", err)
	}
	for _, secret := range [][]byte{nil, {}, make([]byte, MaxSecretSize+1)} {
		if _, err := store.Put(secret); !errors.Is(err, ErrInvalidSecret) {
			t.Fatalf("Put invalid secret = %v, want ErrInvalidSecret", err)
		}
	}
	for _, ref := range []string{"", "../other", refPrefix + strings.Repeat("g", 32), refPrefix + strings.Repeat("A", 32)} {
		if _, err := store.Get(ref); !errors.Is(err, ErrInvalidRef) {
			t.Fatalf("Get invalid ref = %v, want ErrInvalidRef", err)
		}
		if err := store.Delete(ref); !errors.Is(err, ErrInvalidRef) {
			t.Fatalf("Delete invalid ref = %v, want ErrInvalidRef", err)
		}
	}
	if _, err := New(nil); err == nil {
		t.Fatal("New(nil) should fail")
	}
	if _, err := New(&profile.Profile{Kind: profile.Test, DataDir: "relative"}); err == nil {
		t.Fatal("New with relative data directory should fail")
	}
	if _, err := New(&profile.Profile{Kind: "invalid", DataDir: t.TempDir()}); err == nil {
		t.Fatal("New with invalid profile kind should fail")
	}
}

func TestStoreRejectsDamagedProtectedFile(t *testing.T) {
	dataDir := t.TempDir()
	store := testStore(t, profile.Test, dataDir)
	secret := []byte("synthetic-tamper-test-secret")
	ref, err := store.Put(secret)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dataDir, "credentials", "test", ref+".bin")
	encrypted, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	encrypted[len(encrypted)-1] ^= 0xff
	if err := os.WriteFile(path, encrypted, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ref)
	if err == nil || got != nil || strings.Contains(err.Error(), string(secret)) {
		t.Fatalf("Get tampered file = (%v, %v), want safe error", got, err)
	}
	if err := os.WriteFile(path, make([]byte, maxEncryptedSize+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ref); err == nil {
		t.Fatal("Get oversized file should fail")
	}
}
