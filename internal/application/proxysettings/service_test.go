package proxysettings

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"omniproxy/internal/core/netproxy"
	"omniproxy/internal/infrastructure/database"
)

// fakeSecrets 在内存里模拟 DPAPI 凭据库，只记录密文引用。
type fakeSecrets struct {
	items map[string]string
}

func newFakeSecrets() *fakeSecrets { return &fakeSecrets{items: map[string]string{}} }

func (f *fakeSecrets) Put(secret []byte) (string, error) {
	ref := "cred-v1-fake" + string(rune('a'+len(f.items)))
	f.items[ref] = string(secret)
	return ref, nil
}

func (f *fakeSecrets) Get(ref string) ([]byte, error) {
	value, ok := f.items[ref]
	if !ok {
		return nil, errors.New("not found")
	}
	return []byte(value), nil
}

func (f *fakeSecrets) Delete(ref string) error {
	delete(f.items, ref)
	return nil
}

func newTestService(t *testing.T) (*Service, *fakeSecrets) {
	t.Helper()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "proxy.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, name := range []string{"001_initial.sql", "002_builtin_providers.sql", "044_provider_proxy.sql"} {
		content, readErr := os.ReadFile(filepath.Join("../../../migrations", name))
		if readErr != nil {
			t.Fatalf("read %s: %v", name, readErr)
		}
		if _, execErr := db.Exec(string(content)); execErr != nil {
			t.Fatalf("apply %s: %v", name, execErr)
		}
	}
	secrets := newFakeSecrets()
	return NewService(db, secrets), secrets
}

func TestSaveValidatesAndRedacts(t *testing.T) {
	service, secrets := newTestService(t)
	ctx := context.Background()

	saved, err := service.Save(ctx, SaveInput{Enabled: true, Scheme: netproxy.SchemeAuto, Host: "127.0.0.1", Port: 10808, Username: "user", Password: "secret", NoProxy: "localhost,127.0.0.1"})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if !saved.Enabled || !saved.HasPassword || saved.Endpoint != "auto://127.0.0.1:10808" {
		t.Fatalf("settings = %#v", saved)
	}
	if len(secrets.items) != 1 {
		t.Fatalf("expected one stored secret, got %#v", secrets.items)
	}

	// 空密码表示保留原密码，重新保存不会丢凭证。
	updated, err := service.Save(ctx, SaveInput{Enabled: true, Scheme: netproxy.SchemeSOCKS5, Host: "127.0.0.1", Port: 10809, Username: "user", NoProxy: ""})
	if err != nil {
		t.Fatalf("resave: %v", err)
	}
	if !updated.HasPassword || updated.Scheme != netproxy.SchemeSOCKS5 {
		t.Fatalf("settings after resave = %#v", updated)
	}

	cleared, err := service.Save(ctx, SaveInput{Enabled: true, Scheme: netproxy.SchemeHTTP, Host: "127.0.0.1", Port: 10809, Username: "user", ClearPassword: true})
	if err != nil {
		t.Fatalf("clear: %v", err)
	}
	if cleared.HasPassword || len(secrets.items) != 0 {
		t.Fatalf("expected password cleared: %#v %#v", cleared, secrets.items)
	}

	if _, err := service.Save(ctx, SaveInput{Enabled: true, Scheme: "socks9", Host: "127.0.0.1", Port: 10808}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid scheme rejected, got %v", err)
	}
	if _, err := service.Save(ctx, SaveInput{Enabled: true, Scheme: netproxy.SchemeAuto, Host: "127.0.0.1", Port: 0}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid port rejected, got %v", err)
	}
	if _, err := service.Save(ctx, SaveInput{Enabled: true, Scheme: netproxy.SchemeAuto, Host: "127.0.0.1", Port: 10808, Password: "orphan"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected password without username rejected, got %v", err)
	}
}

func TestSnapshotCarriesProviderModesAndDetectedScheme(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()
	if _, err := service.Save(ctx, SaveInput{Enabled: true, Scheme: netproxy.SchemeAuto, Host: "127.0.0.1", Port: 10808, Username: "user", Password: "secret"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	db := service.db
	if _, err := db.Exec(`UPDATE providers SET proxy_mode='proxy' WHERE id='builtin_openai';
		INSERT INTO providers (id,kind,source,name,base_url,enabled,created_at,updated_at,proxy_mode) VALUES
		('prov_custom','openai-compatible','custom','Local','http://127.0.0.1:11434/v1',1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z','direct')`); err != nil {
		t.Fatalf("insert providers: %v", err)
	}
	if err := service.RememberDetectedScheme(ctx, netproxy.SchemeSOCKS5); err != nil {
		t.Fatalf("remember: %v", err)
	}
	snapshot, err := service.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if !snapshot.Enabled || snapshot.Endpoint.Password != "secret" || snapshot.Detected != netproxy.SchemeSOCKS5 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	if snapshot.Modes["builtin_openai"] != netproxy.ModeProxy || snapshot.Modes["prov_custom"] != netproxy.ModeDirect {
		t.Fatalf("modes = %#v", snapshot.Modes)
	}
	if snapshot.HostModes["api.openai.com"] != netproxy.ModeProxy {
		t.Fatalf("host modes = %#v", snapshot.HostModes)
	}
	if snapshot.HostModes["127.0.0.1:11434"] != netproxy.ModeDirect {
		t.Fatalf("host modes = %#v", snapshot.HostModes)
	}
	decision := netproxy.Decide(snapshot, netproxy.Scope{ProviderID: "builtin_openai"}, "chatgpt.com:443")
	if !decision.UseProxy {
		t.Fatalf("expected openai to use proxy, got %#v", decision)
	}
}

func TestSnapshotInvalidatedBySaveAndEndpointChangeClearsDetection(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()
	first, err := service.Save(ctx, SaveInput{Enabled: true, Scheme: netproxy.SchemeAuto, Host: "127.0.0.1", Port: 10808})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := service.RememberDetectedScheme(ctx, netproxy.SchemeSOCKS5); err != nil {
		t.Fatalf("remember: %v", err)
	}
	snapshot, err := service.Snapshot(ctx)
	if err != nil || snapshot.Detected != netproxy.SchemeSOCKS5 {
		t.Fatalf("snapshot = %#v err=%v", snapshot, err)
	}
	if _, err := service.Save(ctx, SaveInput{Enabled: true, Scheme: netproxy.SchemeAuto, Host: "127.0.0.1", Port: 10809}); err != nil {
		t.Fatalf("save: %v", err)
	}
	read, err := service.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if read.DetectedScheme != "" {
		t.Fatalf("expected detection cleared after endpoint change, got %q", read.DetectedScheme)
	}
	if read.UpdatedAt == first.UpdatedAt {
		t.Fatal("expected updated_at to change")
	}
}

func TestSnapshotWithoutMigrationFallsBackToDisabled(t *testing.T) {
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "empty.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()
	content, readErr := os.ReadFile(filepath.Join("../../../migrations", "001_initial.sql"))
	if readErr != nil {
		t.Fatalf("read: %v", readErr)
	}
	if _, execErr := db.Exec(string(content)); execErr != nil {
		t.Fatalf("apply: %v", execErr)
	}
	service := NewService(db, newFakeSecrets())
	snapshot, err := service.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snapshot.Enabled || snapshot.Configured() {
		t.Fatalf("expected disabled snapshot, got %#v", snapshot)
	}
}

func TestTestReportsUnavailableEndpoint(t *testing.T) {
	service, _ := newTestService(t)
	result, err := service.Test(context.Background(), SaveInput{Enabled: true, Scheme: netproxy.SchemeAuto, Host: "127.0.0.1", Port: 1})
	if err != nil {
		t.Fatalf("test: %v", err)
	}
	if result.Reachable || result.ErrorCode != "proxy_unavailable" {
		t.Fatalf("result = %#v", result)
	}
}
