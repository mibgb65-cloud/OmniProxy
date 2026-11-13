package gateway

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/database"
)

// googlePool 只提供账号快照，不接触凭据或上游。
type googlePool struct{ items []account.Item }

func (g googlePool) List(context.Context) ([]account.Item, error) { return g.items, nil }
func (g googlePool) UseAPIKey(context.Context, string, func([]byte) error) error {
	return proxy.Failure("unused", 500)
}

func googlePoolItems(authMethod string) []account.Item {
	return []account.Item{
		{ID: "a", ProviderID: "builtin_google", Enabled: true, AuthMethod: authMethod, AuthState: "authenticated", UpdatedAt: "v1"},
		{ID: "b", ProviderID: "builtin_google", Enabled: true, AuthMethod: authMethod, AuthState: "authenticated", UpdatedAt: "v1"},
	}
}

func googleRotationTarget() target {
	return target{kind: "google", authMethod: account.AuthSubscription,
		snapshot: proxy.TargetSnapshot{ProviderID: "builtin_google", AuthMethod: account.AuthSubscription}}
}

func googleRotationRequest() proxy.Request {
	return proxy.Request{Model: "gemini-4-hydrogen", Candidates: []proxy.AccountCandidate{
		{AccountID: "a", UpdatedAt: "v1"}, {AccountID: "b", UpdatedAt: "v1"},
	}}
}

func TestGoogleRotationQueueAndBalancedOrder(t *testing.T) {
	s := &Service{accounts: googlePool{items: googlePoolItems(account.AuthSubscription)}, failover: newFailoverState()}
	mode := "queue"
	s.SetGoogleAccountRotationMode(func(context.Context) (string, error) { return mode, nil })

	ordered, err := s.rotateAccounts(context.Background(), googleRotationTarget(), googleRotationRequest())
	if err != nil || len(ordered) != 2 || ordered[0].AccountID != "a" || ordered[1].AccountID != "b" {
		t.Fatalf("queue order = %+v, %v", ordered, err)
	}

	// 均衡模式在同池账号间轮流分配，连续请求不会全部压在第一个账号。
	mode = "balanced"
	var picks []string
	for range 4 {
		ordered, err := s.rotateAccounts(context.Background(), googleRotationTarget(), googleRotationRequest())
		if err != nil || len(ordered) != 2 {
			t.Fatalf("balanced rotation failed: %+v %v", ordered, err)
		}
		picks = append(picks, ordered[0].AccountID)
	}
	if strings.Join(picks, ",") != "a,b,a,b" {
		t.Fatalf("balanced picks = %v", picks)
	}

	// 固定账号与单候选不参与重排。
	fixed := googleRotationRequest()
	fixed.FixedAccount = true
	if ordered, err := s.rotateAccounts(context.Background(), googleRotationTarget(), fixed); err != nil || ordered[0].AccountID != "a" {
		t.Fatalf("fixed account rotated: %+v %v", ordered, err)
	}

	// 设置读取失败时明确失败，不静默退回队列。
	s.SetGoogleAccountRotationMode(func(context.Context) (string, error) { return "", proxy.Failure("configuration_unavailable", 503) })
	if _, err := s.rotateAccounts(context.Background(), googleRotationTarget(), googleRotationRequest()); err == nil {
		t.Fatal("settings failure ignored")
	}
}

// API Key 账号不属于订阅池，不能被当成轮换候选。
func TestGoogleRotationRejectsNonSubscriptionAccounts(t *testing.T) {
	s := &Service{accounts: googlePool{items: googlePoolItems(account.AuthAPIKey)}, failover: newFailoverState()}
	s.SetGoogleAccountRotationMode(func(context.Context) (string, error) { return "balanced", nil })
	if _, err := s.rotateAccounts(context.Background(), googleRotationTarget(), googleRotationRequest()); err == nil {
		t.Fatal("api key account accepted as subscription candidate")
	}
	// 没有安装读取器时（例如测试或精简装配）保持原顺序。
	plain := &Service{accounts: googlePool{items: googlePoolItems(account.AuthSubscription)}, failover: newFailoverState()}
	if ordered, err := plain.rotateAccounts(context.Background(), googleRotationTarget(), googleRotationRequest()); err != nil || ordered[0].AccountID != "a" {
		t.Fatalf("unwired rotation reordered: %+v %v", ordered, err)
	}
}

func TestGoogleSettingsPersistValidatedModes(t *testing.T) {
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "google-rotation.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	service := NewGoogleSettingsService(db)
	if settings, err := service.Read(ctx); err != nil || settings.AccountRotationMode != "queue" {
		t.Fatalf("default = %+v, %v", settings, err)
	}
	for _, mode := range []string{"balanced", "queue", "balanced"} {
		settings, err := service.SetAccountRotationMode(ctx, mode)
		if err != nil || settings.AccountRotationMode != mode {
			t.Fatalf("set %q = %+v, %v", mode, settings, err)
		}
		if saved, err := NewGoogleSettingsService(db).AccountRotationMode(ctx); err != nil || saved != mode {
			t.Fatalf("reload %q = %q, %v", mode, saved, err)
		}
	}
	for _, mode := range []string{"", "BALANCED", "balanced\n", strings.Repeat("x", 1024)} {
		if _, err := service.SetAccountRotationMode(ctx, mode); err == nil {
			t.Fatalf("invalid mode %q accepted", mode)
		}
	}
	if _, err := db.Exec(`DELETE FROM google_settings`); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Read(ctx); err == nil {
		t.Fatal("missing settings row accepted")
	}
}
