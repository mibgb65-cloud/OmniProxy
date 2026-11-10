//go:build windows

package wails

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/modelcatalog"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/core/netproxy"
	"omniproxy/internal/infrastructure/credential"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/platform/profile"
)

func TestCreateOpenAIAPIKeyAccountsRequestsModelSync(t *testing.T) {
	// 同包的应用初始化测试会安装进程级代理；本夹具不能借用其已关闭的数据库。
	previous := netproxy.Installed()
	netproxy.Install(nil)
	t.Cleanup(func() { netproxy.Install(previous) })
	for _, batch := range []bool{false, true} {
		name := "single"
		if batch {
			name = "batch"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			prof := &profile.Profile{Kind: profile.Test, DataDir: t.TempDir()}
			db, err := database.Open(database.Config{Path: filepath.Join(prof.DataDir, "accounts.db")})
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
				t.Fatal(err)
			}
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/models" || r.Header.Get("Authorization") != "Bearer test-only-key" {
					t.Error("新建账号没有使用 API Key 同步目录")
				}
				_, _ = w.Write([]byte(`{"data":[{"id":"gpt-test"}]}`))
			}))
			defer upstream.Close()
			if _, err := db.Exec(`UPDATE providers SET base_url=? WHERE id='builtin_openai'`, upstream.URL+"/v1"); err != nil {
				t.Fatal(err)
			}
			svc := &AppService{prof: prof, db: db}
			accounts, err := svc.accountService()
			if err != nil {
				t.Fatal(err)
			}
			source := modelcatalog.New(db, accounts, provider.NewService(db), nil)
			results := make(chan error, 4)
			auto := modelcatalog.StartOpenAIAutoSync(ctx, source, nil, func(err error) { results <- err })
			defer auto.Close()
			svc.openAISync.auto = auto
			await := func() {
				t.Helper()
				select {
				case err := <-results:
					if err != nil {
						t.Fatal(err)
					}
				case <-time.After(5 * time.Second):
					t.Fatal("保存账号后没有触发目录同步")
				}
			}
			// 先消费启动扫描，后续结果必须由本次新增账号的通知触发。
			await()
			if batch {
				_, err = svc.CreateAccounts(account.BatchCreateInput{ProviderID: "builtin_openai", APIKeys: []string{"test-only-key"}})
			} else {
				_, err = svc.CreateAccount(account.CreateInput{ProviderID: "builtin_openai", APIKey: "test-only-key"})
			}
			if err != nil {
				t.Fatal(err)
			}
			await()
			items, err := source.List(ctx)
			if err != nil || len(items) != 1 || items[0].Model != "gpt-test" {
				t.Fatalf("新建账号后目录未就绪: %v %v", items, err)
			}
		})
	}
}

func TestAccountBindingsUseProtectedStore(t *testing.T) {
	prof := &profile.Profile{Kind: profile.Test, DataDir: t.TempDir()}
	db, err := database.Open(database.Config{Path: filepath.Join(prof.DataDir, "accounts.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	svc := &AppService{prof: prof, db: db}
	const secret = "test-only-key"
	item, err := svc.CreateAccount(account.CreateInput{ProviderID: "builtin_google", Name: "test", APIKey: secret})
	if err != nil {
		t.Fatal(err)
	}
	items, err := svc.ListAccounts()
	if err != nil || len(items) != 1 || items[0] != item {
		t.Fatalf("ListAccounts = (%+v, %v)", items, err)
	}
	var ref string
	if err := db.QueryRow(`SELECT credential_ref FROM accounts WHERE id = ?`, item.ID).Scan(&ref); err != nil {
		t.Fatal(err)
	}
	store, err := credential.New(prof)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ref)
	if err != nil || string(got) != secret {
		t.Fatalf("stored key could not be read: %v", err)
	}
	clear(got)
	if err := svc.DeleteAccount(item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ref); !errors.Is(err, credential.ErrNotFound) {
		t.Fatalf("deleted credential still exists: %v", err)
	}
	keys := []string{"batch-one-test-key", "batch-two-test-key"}
	batch, err := svc.CreateAccounts(account.BatchCreateInput{ProviderID: "builtin_google", APIKeys: keys})
	if err != nil || len(batch) != 2 {
		t.Fatalf("CreateAccounts count = %d, error = %v", len(batch), err)
	}
	for i, item := range batch {
		if item.Name != keys[i][:8] {
			t.Fatal("batch binding did not automatically name the account")
		}
		if err := db.QueryRow(`SELECT credential_ref FROM accounts WHERE id = ?`, item.ID).Scan(&ref); err != nil {
			t.Fatal(err)
		}
		stored, err := store.Get(ref)
		if err != nil || string(stored) != keys[i] {
			t.Fatal("batch credential did not survive protected storage")
		}
		clear(stored)
		oldRef := ref
		updated, err := svc.UpdateAccount(account.UpdateInput{ID: item.ID, Name: "updated", APIKey: "replacement-test-protected-key"})
		if err != nil || updated.CleanupWarning || updated.Account.Name != "updated" {
			t.Fatal("account update binding failed")
		}
		keys[i] = "replacement-test-protected-key"
		if _, err := store.Get(oldRef); !errors.Is(err, credential.ErrNotFound) {
			t.Fatal("old protected key survived successful replacement")
		}
		preview, err := svc.AccountKeyPreview(item.ID)
		if err != nil || preview == keys[i] {
			t.Fatal("preview binding leaked or failed to read the protected key")
		}
		copied := ""
		svc.copyText = func(value string) bool { copied = value; return true }
		if err := svc.CopyAccountKey(item.ID); err != nil || copied != keys[i] {
			t.Fatal("copy binding did not use the protected key")
		}
		svc.copyText = func(string) bool { return false }
		if err := svc.CopyAccountKey(item.ID); err == nil {
			t.Fatal("clipboard failure was ignored")
		}
		if err := svc.DeleteAccount(item.ID); err != nil {
			t.Fatal(err)
		}
	}
}
