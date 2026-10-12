package gateway

import (
	"context"
	"database/sql"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/credential"
	"omniproxy/internal/platform/profile"
)

// TestLiveDeepSeek 仅显式启用时读取现有数据库和凭证，不创建账号或写用户数据库。
func TestLiveDeepSeek(t *testing.T) {
	if os.Getenv("OMNIPROXY_LIVE_DEEPSEEK") != "1" {
		t.Skip("真实提供商验证需显式启用")
	}
	if path := os.Getenv("OMNIPROXY_LIVE_ENV_FILE"); path != "" {
		if err := profile.LoadDotEnv(path); err != nil {
			t.Fatal("cannot load profile environment")
		}
	}
	kind := profile.Development
	if configured := os.Getenv("OMNIPROXY_PROFILE"); configured != "" {
		kind = profile.Kind(configured)
	}
	prof, err := profile.New(kind)
	if err != nil {
		t.Fatal("cannot resolve runtime profile")
	}
	if os.Getenv("OMNIPROXY_DATA_DIR") == "" {
		cfg, err := profile.LoadConfig(kind)
		if err != nil {
			t.Fatal("cannot read profile configuration")
		}
		if cfg.DataDir != "" {
			prof.DataDir = cfg.DataDir
		}
		// 读取失败的搬迁可回退旧目录，但真实测试不持久化或修复用户配置。
		if _, err := os.Stat(prof.DatabasePath()); err != nil && cfg.PreviousDataDir != "" {
			prof.DataDir = cfg.PreviousDataDir
		}
	}
	if _, err := os.Stat(prof.DatabasePath()); err != nil {
		t.Fatal("existing database required")
	}
	dbURL := url.URL{Scheme: "file", Opaque: filepath.ToSlash(prof.DatabasePath()), RawQuery: "mode=ro"}
	db, err := sql.Open("sqlite", dbURL.String())
	if err != nil {
		t.Fatal("cannot open existing read-only database")
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Fatalf("cannot access read-only database: %v", err)
	}
	store, err := credential.New(prof)
	if err != nil {
		t.Fatal("cannot initialize protected credential access")
	}
	service := New(proxy.Options{}, provider.NewService(db), account.NewService(db, store), nil)
	if _, err := service.Start(); err != nil {
		t.Fatal("cannot start isolated test listener")
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if _, err := service.Stop(ctx); err != nil {
			t.Error("cannot stop isolated test listener")
		}
	}()
	for _, stream := range []bool{false, true} {
		started := time.Now()
		status, body := localGatewayRequest(t, service, "deepseek-flash", stream)
		t.Logf("provider=DeepSeek model=deepseek-flash stream=%t status=%d latency_ms=%d", stream, status, time.Since(started).Milliseconds())
		if status != http.StatusOK {
			t.Fatalf("live request failed: code=%s status=%d", gatewayResponseError(t, body), status)
		}
		if !gatewayReplyHasContent(body, stream) {
			t.Fatal("live request returned no complete reply")
		}
	}
}
