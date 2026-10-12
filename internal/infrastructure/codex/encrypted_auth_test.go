package codex

import (
	"bytes"
	"encoding/json"
	"filippo.io/age"
	"os"
	"path/filepath"
	"testing"
)

func TestOfficialEncryptedAuthNamespaceAndFailures(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "secrets"), 0700)
	original := `{"tokens":{"access_token":"test-only-secret","account_id":"test-workspace"}}`
	data, _ := json.Marshal(map[string]any{"version": 1, "secrets": map[string]string{"global/CODEX_AUTH": original, "global/OTHER": "unrelated"}})
	recipient, _ := age.NewScryptRecipient("test-only-passphrase")
	recipient.SetWorkFactor(10)
	var ciphertext bytes.Buffer
	writer, err := age.Encrypt(&ciphertext, recipient)
	if err != nil {
		t.Fatal(err)
	}
	writer.Write(data)
	writer.Close()
	path := filepath.Join(root, "secrets", "codex_auth.age")
	os.WriteFile(path, ciphertext.Bytes(), 0600)
	decoded, err := readEncryptedAuth(root, []byte("test-only-passphrase"))
	if err != nil || string(decoded) != original {
		t.Fatal("official namespace mismatch", err)
	}
	clear(decoded)
	if _, err := readEncryptedAuth(root, []byte("wrong-passphrase")); err == nil {
		t.Fatal("wrong key accepted")
	}
	os.WriteFile(path, []byte("corrupt"), 0600)
	if _, err := readEncryptedAuth(root, []byte("test-only-passphrase")); err == nil {
		t.Fatal("corrupt ciphertext accepted")
	}
}
