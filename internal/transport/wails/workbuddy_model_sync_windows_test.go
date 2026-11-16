//go:build windows

package wails

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"omniproxy/internal/application/workbuddy"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/platform/profile"
	"omniproxy/internal/provider/workbuddyai"
)

// 手动同步（ModelSync）必须和后台自动同步用同一套已接线目录服务。
// 这里固定住「Wails 侧漏接 WorkBuddy 订阅」这个回归：漏接时同步只会返回 unavailable。
func TestWorkBuddyManualModelSyncUsesWiredSubscription(t *testing.T) {
	ctx := context.Background()
	prof := &profile.Profile{Kind: profile.Test, DataDir: t.TempDir()}
	db, err := database.Open(database.Config{Path: filepath.Join(prof.DataDir, "workbuddy.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	service := &AppService{prof: prof, db: db}
	defer service.shutdownWorkBuddySubscription()
	accounts, err := service.accountService()
	if err != nil {
		t.Fatal(err)
	}
	cred := workbuddyai.Credentials{
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		ExpiresAt:    time.Now().Add(time.Hour),
		Domain:       "www.workbuddy.ai",
		TokenType:    "Bearer",
		UID:          "uid-1",
		AccountName:  "tester@example.test",
	}
	item, err := accounts.SaveWorkBuddyAccount(ctx, "builtin_workbuddy_ai", "", "", cred)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/config" {
			t.Errorf("unexpected upstream path %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		io.WriteString(w, `{"code":0,"data":{"models":[{"id":"model-a","name":"A"},{"id":"model-b","name":"B"}]}}`)
	}))
	defer server.Close()
	client, err := workbuddyai.NewClientAt(server.URL, server.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	service.workbuddyAuth.service = workbuddy.New(accounts, client)

	items, err := service.ModelSync(item.ID)
	if err != nil {
		t.Fatalf("manual sync failed: %v", err)
	}
	if len(items) != 2 || items[0].Model != "model-a" || items[1].Model != "model-b" {
		t.Fatalf("synced models = %+v", items)
	}
	var stored int
	if err := db.QueryRow(`SELECT COUNT(*) FROM account_models WHERE account_id=?`, item.ID).Scan(&stored); err != nil || stored != 2 {
		t.Fatalf("stored models = %d (%v)", stored, err)
	}
	// 目录必须绑定当前凭据版本，否则模型存在但不可路由。
	var version, bound string
	if err := db.QueryRow(`SELECT version FROM account_catalog_versions WHERE id=?`, item.ID).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT DISTINCT account_version FROM account_models WHERE account_id=?`, item.ID).Scan(&bound); err != nil {
		t.Fatal(err)
	}
	if version == "" || bound != version {
		t.Fatalf("catalog version = %q, bound = %q", version, bound)
	}
}
