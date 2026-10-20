//go:build windows

package wails

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"omniproxy/internal/application/googlesubscription"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/platform/profile"
	"omniproxy/internal/provider/antigravity"
)

func TestGoogleModelsAutomaticallySyncOnStartupRefreshAndAuthorization(t *testing.T) {
	ctx := context.Background()
	prof := &profile.Profile{Kind: profile.Test, DataDir: t.TempDir()}
	db, err := database.Open(database.Config{Path: filepath.Join(prof.DataDir, "auto.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	service := &AppService{prof: prof, db: db}
	defer service.shutdownGoogleSubscription()
	accounts, err := service.accountService()
	if err != nil {
		t.Fatal(err)
	}
	cred := antigravity.Credentials{
		OAuthConfig: antigravity.OAuthConfig{ClientID: "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com", ClientSecret: "GOCSPX-" + strings.Repeat("x", 28)},
		AccessToken: "test-access", RefreshToken: "test-refresh", ExpiresAt: time.Now().Add(time.Hour),
		Subject: "1001", Email: "test@example.test", Project: "test-project",
	}
	item, err := accounts.SaveGoogleSubscription(ctx, "builtin_google", "", "", cred)
	if err != nil {
		t.Fatal(err)
	}
	var model atomic.Value
	model.Store("gemini-first")
	manager := googlesubscription.New(accounts, antigravity.New(googleQuotaTransport(func(r *http.Request) (*http.Response, error) {
		body := ""
		switch r.URL.Path {
		case "/v1internal:fetchAvailableModels":
			id := model.Load().(string)
			body = fmt.Sprintf(`{"models":{%q:{"displayName":"Gemini"}},"agentModelSorts":[{"groups":[{"modelIds":[%q]}]}]}`, id, id)
		case "/token":
			body = `{"access_token":"test-access","refresh_token":"test-refresh","expires_in":3600}`
		case "/oauth2/v2/userinfo":
			body = `{"id":"1001","email":"test@example.test","verified_email":true}`
		case "/v1internal:loadCodeAssist":
			body = `{"cloudaicompanionProject":"test-project"}`
		default:
			return nil, fmt.Errorf("unexpected test endpoint")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})))
	manager.OnChange(service.requestGoogleModelSync)
	service.googleAuth.service = manager
	updated := make(chan struct{}, 8)
	service.googleAuth.emitModels = func() { updated <- struct{}{} }
	awaitModel := func(expected string) {
		t.Helper()
		select {
		case <-updated:
		case <-time.After(5 * time.Second):
			t.Fatal("自动目录更新未完成")
		}
		var actual string
		if err := db.QueryRow(`SELECT model FROM account_models WHERE account_id=?`, item.ID).Scan(&actual); err != nil || actual != expected {
			t.Fatal("自动同步未保存实际模型", err, actual)
		}
	}
	service.startGoogleModelSync()
	service.startGoogleModelSync()
	awaitModel("gemini-first")
	model.Store("gemini-after-refresh")
	if models, err := service.GoogleSubscriptionModels(item.ID); err != nil || len(models) != 1 {
		t.Fatal("读取模型失败", err)
	}
	awaitModel("gemini-after-refresh")
	model.Store("gemini-after-auth")
	if _, err := service.SubscriptionRefresh(item.ID); err != nil {
		t.Fatal("刷新授权失败", err)
	}
	awaitModel("gemini-after-auth")
	if service.subscription.service != nil || service.gateway != nil {
		t.Fatal("同步启动了无关的 Codex 授权或网关")
	}
	service.shutdownGoogleSubscription()
	service.requestGoogleModelSync()
	if len(updated) != 0 {
		t.Fatal("重复启动或关闭后发布了额外事件")
	}
}
