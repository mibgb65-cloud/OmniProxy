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
