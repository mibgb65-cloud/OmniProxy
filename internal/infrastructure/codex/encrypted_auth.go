package codex

import (
	"encoding/json"
	"filippo.io/age"
	"io"
	"os"
	"path/filepath"
)

// readEncryptedAuth 在内存解密官方 CodexAuth 命名空间，不写明文或读取其他秘密。
func readEncryptedAuth(home string, passphrase []byte) ([]byte, error) {
	path := filepath.Join(home, "secrets", "codex_auth.age")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return nil, ErrUnavailable
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil || real != path {
		return nil, ErrProtocol
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer file.Close()
	identity, err := age.NewScryptIdentity(string(passphrase))
	if err != nil {
		return nil, ErrProtocol
	}
	// 官方当前 Windows 存储使用 work factor 20；固定上限防止异常密文无限放大 KDF。
	identity.SetMaxWorkFactor(20)
	reader, err := age.Decrypt(io.LimitReader(file, (1<<20)+1), identity)
	if err != nil {
		return nil, ErrProtocol
	}
	plaintext, err := io.ReadAll(io.LimitReader(reader, (1<<20)+1))
	defer clear(plaintext)
	if err != nil || len(plaintext) > 1<<20 {
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
	if !ok || len(value) > 128<<10 {
		return nil, ErrProtocol
	}
	return []byte(value), nil
}
