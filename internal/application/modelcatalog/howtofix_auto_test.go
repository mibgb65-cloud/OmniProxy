package modelcatalog

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/infrastructure/database"
)

// howToFixAutoFixture 准备两个 HowToFix API Key 账号（一个停用）、一个 DeepSeek 账号和一个
// 未同步过的 HowToFix 账号，用来验证后台自动同步只处理启用的 API Key 账号。
func howToFixAutoFixture(t *testing.T) *Service {
	t.Helper()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "howtofix-auto.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,auth_method)
		VALUES('live','builtin_howtofix','HowToFix','protected-ref',1,'now','now','api_key'),
		      ('disabled','builtin_howtofix','Disabled','protected-ref',0,'now','now','api_key'),
		      ('other','builtin_deepseek','DeepSeek','protected-ref',1,'now','now','api_key'),
		      ('subscription','builtin_howtofix','Subscription','protected-ref',1,'now','now','subscription_login')`); err != nil {
		t.Fatal(err)
	}
	return New(db, account.NewService(db, googleStore{}), provider.NewService(db), nil)
}

func TestAPIKeyAutoSyncFillsOnlyStaleEnabledAccounts(t *testing.T) {
	s := howToFixAutoFixture(t)
	calls := make(chan string, 8)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		calls <- r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"object":"list","data":[{"id":"glm-5.3-flash","object":"model","owned_by":"howtofix"}]}`)
	}))
	defer upstream.Close()
	if _, err := s.db.Exec(`UPDATE providers SET base_url=? WHERE id='builtin_howtofix'`, upstream.URL+"/v1"); err != nil {
		t.Fatal(err)
	}

	refreshed := make(chan error, 4)
	var refreshCount int
	auto := StartAPIKeyAutoSync(context.Background(), s, "howtofix", func(context.Context) error {
		refreshCount++
		return nil
	}, func(err error) { refreshed <- err })
	t.Cleanup(auto.Close)

	select {
	case err := <-refreshed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("HowToFix 后台同步未完成")
	}
	// 只应该同步启用的 API Key 账号：停用账号、其他服务商和订阅账号都不访问上游。
	if len(calls) != 1 {
		t.Fatalf("upstream calls = %d, want 1", len(calls))
	}
	if got := <-calls; got != "Bearer not-a-real-google-key" {
		t.Fatalf("Authorization = %q", got)
	}
	if refreshCount != 1 {
		t.Fatalf("refresh count = %d, want 1", refreshCount)
	}
	items, err := s.List(context.Background())
	if err != nil || len(items) != 1 || items[0].AccountID != "live" || items[0].Model != "glm-5.3-flash" {
		t.Fatalf("catalog = %+v, %v", items, err)
	}

	// 同一运行内重复请求不会重复访问上游（staleOnly + 账号／版本去重）。
	auto.Request()
	select {
	case <-refreshed:
	case <-time.After(5 * time.Second):
		t.Fatal("第二次扫描未完成")
	}
	if len(calls) != 0 {
		t.Fatalf("unexpected repeat sync: %d", len(calls))
	}
}

func TestAPIKeyAutoSyncSkipsDisabledProvider(t *testing.T) {
	s := howToFixAutoFixture(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("disable provider must not reach upstream")
	}))
	defer upstream.Close()
	if _, err := s.db.Exec(`UPDATE providers SET base_url=?, enabled=0 WHERE id='builtin_howtofix'`, upstream.URL+"/v1"); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	auto := StartAPIKeyAutoSync(context.Background(), s, "howtofix", nil, func(error) { close(done) })
	t.Cleanup(auto.Close)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("后台同步未完成")
	}
	items, err := s.List(context.Background())
	if err != nil || len(items) != 0 {
		t.Fatalf("catalog = %+v, %v", items, err)
	}
}
