package legacyimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/infrastructure/legacy"
)

type memoryStore struct {
	values        map[string][]byte
	calls, failAt int
}

func (s *memoryStore) Put(value []byte) (string, error) {
	s.calls++
	if s.calls == s.failAt {
		return "", errors.New("storage failure")
	}
	ref := fmt.Sprintf("fake-ref-%d", s.calls)
	s.values[ref] = append([]byte(nil), value...)
	return ref, nil
}
func (s *memoryStore) Get(ref string) ([]byte, error) {
	value, ok := s.values[ref]
	if !ok {
		return nil, errors.New("missing credential")
	}
	return append([]byte(nil), value...), nil
}
func (s *memoryStore) Delete(ref string) error { delete(s.values, ref); return nil }

func newImportTest(t *testing.T) (*Service, *memoryStore, string) {
	t.Helper()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "target.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	store := &memoryStore{values: make(map[string][]byte)}
	service := New(db, store)
	service.decode = func(value string) ([]byte, error) {
		if value == "unreadable" {
			return nil, errors.New("secret error must not escape")
		}
		return []byte(value), nil
	}
	return service, store, t.TempDir()
}

func writeTokens(t *testing.T, dir string, tokens []legacy.Token) []byte {
	t.Helper()
	data, err := json.Marshal(tokens)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tokens.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestPreviewImportPreservesDisabledSkipsUnsupportedAndIsIdempotent(t *testing.T) {
	s, store, dir := newImportTest(t)
	ctx := context.Background()
	original := writeTokens(t, dir, []legacy.Token{
		{ID: "1", Provider: "deepseek", Name: "保留名称", Value: "sk-fake-deepseek-only-test", Disabled: true},
		{ID: "2", Provider: "openai", CredentialType: "codex_auth_json", Value: "fake-subscription-secret"},
		{ID: "3", Provider: "minimax", Value: "fake-ambiguous-billing"},
		{ID: "4", Provider: "featherless", Value: "fake-unsupported-key"},
		{ID: "5", Provider: "deepseek", Value: "sk-fake-deepseek-only-test"},
		{ID: "6", Provider: "deepseek", Value: "unreadable"},
		{ID: "7", Provider: "xiaomi", CredentialType: "mimo_token_plan", Region: "sgp", Value: "tp-fake-only-test"},
		{ID: "1", Provider: "deepseek", Value: "sk-fake-different-key-same-old-id"},
	})
	preview, err := s.Preview(ctx, dir)
	if err != nil || preview.Ready != 2 {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	for i, want := range []string{"ready", "login_required", "billing_confirmation", "unsupported", "duplicate", "credential_unavailable", "ready", "duplicate"} {
		if preview.Entries[i].State != want {
			t.Errorf("row %d=%s want %s", i, preview.Entries[i].State, want)
		}
	}
	encoded, _ := json.Marshal(preview)
	if strings.Contains(string(encoded), "sk-fake") || strings.Contains(string(encoded), "tp-fake") || strings.Contains(string(encoded), "fake-subscription-secret") || len(store.values) != 0 {
		t.Fatal("preview leaked or saved credentials")
	}
	result, err := s.Import(ctx, dir, preview.Snapshot)
	if err != nil || result.Imported != 2 || result.Failed != 0 {
		t.Fatalf("import=%+v %v", result, err)
	}
	var name string
	var enabled bool
	if err := s.db.QueryRow("SELECT name,enabled FROM accounts WHERE provider_id='builtin_deepseek'").Scan(&name, &enabled); err != nil || name != "保留名称" || enabled {
		t.Fatalf("metadata lost: %s %t %v", name, enabled, err)
	}
	again, err := s.Preview(ctx, dir)
	if err != nil || again.Ready != 0 {
		t.Fatalf("not idempotent: %+v %v", again, err)
	}
	if _, err := s.Import(ctx, dir, preview.Snapshot); err != nil || len(store.values) != 2 {
		t.Fatal("retry duplicated credentials")
	}
	data, _ := os.ReadFile(filepath.Join(dir, "tokens.json"))
	if string(data) != string(original) {
		t.Fatal("source changed")
	}
}

func TestSnapshotChangesPreventWritesAndPartialFailureCanRetry(t *testing.T) {
	s, store, dir := newImportTest(t)
	ctx := context.Background()
	writeTokens(t, dir, []legacy.Token{{ID: "1", Provider: "deepseek", Value: "sk-fake-one-only-test"}})
	preview, err := s.Preview(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	writeTokens(t, dir, []legacy.Token{{ID: "1", Provider: "deepseek", Value: "sk-fake-one-only-test"}, {ID: "2", Provider: "deepseek", Value: "sk-fake-two-only-test"}})
	if _, err := s.Import(ctx, dir, preview.Snapshot); err == nil || len(store.values) != 0 {
		t.Fatal("changed source was imported")
	}
	preview, err = s.Preview(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	store.failAt = 2
	result, err := s.Import(ctx, dir, preview.Snapshot)
	if err != nil || result.Imported != 1 || result.Failed != 1 || len(store.values) != 1 {
		t.Fatalf("partial result=%+v %v", result, err)
	}
	store.failAt = 0
	result, err = s.Import(ctx, dir, preview.Snapshot)
	if err != nil || result.Imported != 1 || result.Failed != 0 || len(store.values) != 2 {
		t.Fatalf("retry=%+v %v", result, err)
	}
}

func TestFailedMetadataTransactionCleansNewCredential(t *testing.T) {
	s, store, dir := newImportTest(t)
	_, err := s.db.Exec(`CREATE TRIGGER reject_legacy BEFORE INSERT ON accounts BEGIN SELECT RAISE(ABORT,'rejected'); END`)
	if err != nil {
		t.Fatal(err)
	}
	writeTokens(t, dir, []legacy.Token{{ID: "1", Provider: "deepseek", Value: "sk-fake-only-test"}})
	preview, err := s.Preview(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Import(context.Background(), dir, preview.Snapshot)
	if err != nil || result.Failed != 1 || result.Imported != 0 || len(store.values) != 0 {
		t.Fatalf("orphan credential: %+v %v", result, err)
	}
}
