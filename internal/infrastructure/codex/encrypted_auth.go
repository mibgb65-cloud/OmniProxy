package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"filippo.io/age"
	"io"
	"os"
	"path/filepath"
)

const maxEncryptedAuthBytes = 1 << 20
const maxAuthValueBytes = 128 << 10
const maxAuthPassphraseBytes = 64 << 10

// readEncryptedAuth 在内存解密官方 CodexAuth 命名空间，不写明文或读取其他秘密。
func readEncryptedAuth(home string, passphrase []byte) ([]byte, error) {
	return readEncryptedAuthContext(context.Background(), home, passphrase)
}

func readEncryptedAuthContext(ctx context.Context, home string, passphrase []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(home) || len(passphrase) == 0 || len(passphrase) > maxAuthPassphraseBytes {
		return nil, ErrProtocol
	}
	path := filepath.Join(home, "secrets", "codex_auth.age")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > maxEncryptedAuthBytes {
		officialAuthCache.clearHome(home)
		return nil, ErrUnavailable
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil || real != path {
		officialAuthCache.clearHome(home)
		return nil, ErrProtocol
	}
	file, err := os.Open(path)
	if err != nil {
		officialAuthCache.clearHome(home)
		return nil, ErrUnavailable
	}
	defer file.Close()
	ciphertext, err := io.ReadAll(io.LimitReader(file, maxEncryptedAuthBytes+1))
	if err != nil || len(ciphertext) == 0 || len(ciphertext) > maxEncryptedAuthBytes {
		officialAuthCache.clearHome(home)
		return nil, ErrUnavailable
	}
	// 每次读取完整有界密文与系统 passphrase；mtime 不足以识别授权更新或替换。
	return officialAuthCache.readContext(ctx, path, ciphertext, passphrase)
}

func decryptAuthPayload(ciphertext, passphrase []byte) ([]byte, error) {
	identity, err := age.NewScryptIdentity(string(passphrase))
	if err != nil {
		return nil, ErrProtocol
	}
	// 官方当前 Windows 存储使用 work factor 20；固定上限防止异常密文无限放大 KDF。
	identity.SetMaxWorkFactor(maxAuthWorkFactor)
	tracked := &authScryptIdentity{ScryptIdentity: identity}
	defer tracked.reclaim()
	reader, err := age.Decrypt(bytes.NewReader(ciphertext), tracked)
	if err != nil {
		return nil, ErrProtocol
	}
	plaintext, err := io.ReadAll(io.LimitReader(reader, maxEncryptedAuthBytes+1))
	defer clear(plaintext)
	if err != nil || len(plaintext) > maxEncryptedAuthBytes {
		return nil, ErrProtocol
	}
	var contents struct {
		Version int               `json:"version"`
		Secrets map[string]string `json:"secrets"`
	}
	if json.Unmarshal(plaintext, &contents) != nil || contents.Version > 1 || contents.Version < 0 {
		return nil, ErrProtocol
	}
	value, ok := contents.Secrets["global/CODEX_AUTH"]
	if !ok || len(value) == 0 || len(value) > maxAuthValueBytes {
		return nil, ErrProtocol
	}
	return []byte(value), nil
}
