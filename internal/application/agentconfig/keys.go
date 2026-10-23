package agentconfig

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"omniproxy/internal/core/agentendpoint"
)

// Key 每个客户端独立接入密钥，数据库仅保存凭据引用。
func (s *Service) Key(ctx context.Context, client string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.key(ctx, client)
}
func (s *Service) key(ctx context.Context, client string) ([]byte, error) {
	if _, err := s.path(client); err != nil {
		return nil, err
	}
	var ref string
	err := s.db.QueryRowContext(ctx, `SELECT credential_ref FROM agent_connections WHERE client=?`, client).Scan(&ref)
	if err == nil {
		key, err := s.store.Get(ref)
		if err != nil {
			return nil, failure("key_unavailable")
		}
		return key, nil
	}
	if err != sql.ErrNoRows {
		return nil, failure("key_unavailable")
	}
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return nil, failure("key_unavailable")
	}
	key := []byte("opk_" + hex.EncodeToString(value))
	clear(value)
	ref, err = s.store.Put(key)
	if err != nil {
		clear(key)
		return nil, failure("key_unavailable")
	}
	if _, err = s.db.ExecContext(ctx, `INSERT INTO agent_connections VALUES(?,?,1,?)`, client, ref, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		clear(key)
		cleanup := s.store.Delete(ref)
		return nil, errors.Join(failure("key_unavailable"), cleanup)
	}
	return key, nil
}

// Client 返回接入 Key 所属的受管客户端；只返回客户端标识，不返回或记录密钥。
// 无论命中位置如何都遍历全部引用，避免通过比较耗时推断 Key 归属。
func (s *Service) Client(ctx context.Context, key string) (string, bool) {
	if len(key) != 68 {
		return "", false
	}
	// 只接受当前受支持客户端的 Key，历史诊断凭据不能继续访问网关。
	rows, err := s.db.QueryContext(ctx, `SELECT client, credential_ref FROM agent_connections WHERE enabled=1 AND client IN ('codex','claude','claude-desktop','opencode','harness-web','harness-desktop','kilocode','qwen-code','mimo-code','kimi-code','pi','factory-droid','goose') LIMIT 13`)
	if err != nil {
		return "", false
	}
	defer rows.Close()
	type connection struct{ client, ref string }
	connections := []connection{}
	for rows.Next() {
		var item connection
		if rows.Scan(&item.client, &item.ref) != nil {
			return "", false
		}
		connections = append(connections, item)
	}
	if rows.Err() != nil {
		return "", false
	}
	matched := ""
	for _, item := range connections {
		secret, err := s.store.Get(item.ref)
		if err != nil {
			continue
		}
		if subtle.ConstantTimeCompare(secret, []byte(key)) == 1 {
			matched = item.client
		}
		clear(secret)
	}
	return matched, matched != ""
}

// Authorize 只比较短期解密的本地接入 Key，调用方的头部不会成为上游凭据。
func (s *Service) Authorize(ctx context.Context, key string) bool {
	_, ok := s.Client(ctx, key)
	return ok
}

// PrefixAccess 判定固定入口前缀能否在不带接入 Key 时放行。
//
// Codex 保留官方 auth.json 与其登录令牌，网关无法用接入 Key 校验，因此只有
// 该前缀对应的受管连接处于启用状态时按前缀归属放行；其他前缀仍必须带 Key。
func (s *Service) PrefixAccess(ctx context.Context, prefix string) (string, bool) {
	clients := agentendpoint.Clients(prefix)
	if len(clients) != 1 || clients[0] != "codex" {
		return "", false
	}
	var enabled bool
	if s.db.QueryRowContext(ctx, `SELECT enabled FROM agent_connections WHERE client='codex'`).Scan(&enabled) != nil || !enabled {
		return "", false
	}
	return "codex", true
}
func (s *Service) SetEnabled(ctx context.Context, client string, enabled bool) error {
	if _, err := s.path(client); err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE agent_connections SET enabled=?,updated_at=? WHERE client=?`, enabled, time.Now().UTC().Format(time.RFC3339Nano), client)
	if err != nil {
		return failure("key_unavailable")
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return failure("key_unavailable")
	}
	return nil
}
