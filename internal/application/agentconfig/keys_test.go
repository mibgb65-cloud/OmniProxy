package agentconfig

import (
	"context"
	"strings"
	"testing"
)

func TestOnlySupportedClientKeysAuthorize(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	legacyKey := []byte("opk_" + strings.Repeat("d", 64))
	ref, err := s.store.Put(legacyKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO agent_connections VALUES('diagnostic',?,1,'2026-10-12T00:00:00Z')`, ref); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Key(ctx, "diagnostic"); err == nil {
		t.Fatal("removed diagnostic client accepted")
	}
	if s.Authorize(ctx, string(legacyKey)) {
		t.Fatal("legacy diagnostic key still authorizes")
	}
	clear(legacyKey)
	for _, client := range Clients() {
		key, err := s.Key(ctx, client)
		if err != nil {
			t.Fatal(err)
		}
		valid := s.Authorize(ctx, string(key))
		// 每把 Key 只归属自己的客户端，请求统计才能按 Agent 分开。
		identified, ok := s.Client(ctx, string(key))
		clear(key)
		if !valid || !ok || identified != client {
			t.Fatalf("supported client %s no longer authorizes or identifies", client)
		}
	}
	if client, ok := s.Client(ctx, strings.Repeat("x", 68)); ok || client != "" {
		t.Fatal("unknown key identified as a client")
	}
}

// Codex 保留官方 auth.json，无法用接入 Key 校验；只有该受管连接启用时才按前缀放行。
func TestPrefixAccessOnlyAdmitsEnabledCodexEntry(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	if client, ok := s.PrefixAccess(ctx, "/codex"); ok || client != "" {
		t.Fatal("codex entry admitted before the connection exists")
	}
	if _, err := s.Key(ctx, "codex"); err != nil {
		t.Fatal(err)
	}
	client, ok := s.PrefixAccess(ctx, "/codex")
	if !ok || client != "codex" {
		t.Fatalf("enabled codex entry rejected: %q %t", client, ok)
	}
	// 其他入口必须带接入 Key，不能沿用 Codex 的放行。
	if _, ok := s.PrefixAccess(ctx, "/claude"); ok {
		t.Fatal("claude entry admitted without an access key")
	}
	if err := s.SetEnabled(ctx, "codex", false); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.PrefixAccess(ctx, "/codex"); ok {
		t.Fatal("disabled codex connection still admitted")
	}
}
