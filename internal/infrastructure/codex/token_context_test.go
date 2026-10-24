package codex

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func freshAuthFixture(t *testing.T, email, access, refresh string) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"email": email, "tokens": map[string]string{
		"access_token": access, "refresh_token": refresh, "account_id": "shared-workspace",
	}})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestFreshAccessRejectsOtherMemberInSameWorkspace(t *testing.T) {
	raw := freshAuthFixture(t, "other@example.invalid", testJWT(time.Now().Add(time.Hour)), "test-refresh")
	access, _, err := freshAccess(context.Background(), "shared-workspace", "expected@example.invalid", raw, time.Now())
	if err == nil || len(access) != 0 {
		t.Fatal("同工作区不同成员取得了令牌")
	}
}

func TestOAuthCancellationStopsWithoutAccessFallback(t *testing.T) {
	started, stopped := make(chan struct{}), make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		close(started)
		<-r.Context().Done()
		close(stopped)
	}))
	defer upstream.Close()
	previous := oauthTokenURL
	oauthTokenURL = upstream.URL
	defer func() { oauthTokenURL = previous }()
	raw := freshAuthFixture(t, "member@example.invalid", testJWT(time.Now().Add(-time.Hour)), "test-refresh")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		access, _, err := RefreshAuthJSON(ctx, upstream.Client(), string(raw), time.Now())
		if len(access) != 0 {
			done <- errors.New("取消后回退到了失效令牌")
			return
		}
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("OAuth 请求未启动")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal("取消被折叠成凭据错误", err)
		}
	case <-time.After(time.Second):
		t.Fatal("续期忽略了请求取消")
	}
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("OAuth 网络请求仍在后台运行")
	}
}

func TestFreshAccessDefersRenewalToOfficialStore(t *testing.T) {
	raw := freshAuthFixture(t, "member@example.invalid", testJWT(time.Now().Add(-time.Hour)), "test-refresh-one")
	access, _, err := freshAccess(context.Background(), "shared-workspace", "member@example.invalid", raw, time.Now())
	if !errors.Is(err, ErrRefreshRequired) || len(access) != 0 {
		t.Fatal("隔离授权过期时未交回官方续期", err)
	}
	replaced := freshAuthFixture(t, "member@example.invalid", testJWT(time.Now().Add(time.Hour)), "test-refresh-two")
	access, id, err := freshAccess(context.Background(), "shared-workspace", "member@example.invalid", replaced, time.Now())
	if err != nil || len(access) == 0 || id != "shared-workspace" {
		t.Fatal("官方材料更新后仍复用了旧授权或无法读取", err)
	}
	clear(access)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	access, _, err = freshAccess(ctx, "shared-workspace", "member@example.invalid", replaced, time.Now())
	if !errors.Is(err, context.Canceled) || len(access) != 0 {
		t.Fatal("取消后仍借出授权", err)
	}
}
