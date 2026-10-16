package agentconfig

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
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

// Authorize 只比较短期解密的本地接入 Key，调用方的头部不会成为上游凭据。
func (s *Service) Authorize(ctx context.Context, key string) bool {
	if len(key) != 68 {
		return false
	}
	// 只接受当前受支持客户端的 Key，历史诊断凭据不能继续访问网关。
	rows, err := s.db.QueryContext(ctx, `SELECT credential_ref FROM agent_connections WHERE enabled=1 AND client IN ('codex','claude','claude-desktop','opencode','harness-web','harness-desktop') LIMIT 6`)
	if err != nil {
		return false
	}
	defer rows.Close()
	refs := []string{}
	for rows.Next() {
		var ref string
		if rows.Scan(&ref) != nil {
			return false
		}
		refs = append(refs, ref)
	}
	if rows.Err() != nil {
		return false
	}
	valid := 0
	for _, ref := range refs {
		secret, err := s.store.Get(ref)
		if err != nil {
			continue
		}
		valid |= subtle.ConstantTimeCompare(secret, []byte(key))
		clear(secret)
	}
	return valid == 1
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
