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
	original := `{"tokens":{"access_token":"test-only-secret","account_id":"test-workspace"}}`
	path := filepath.Join(root, "secrets", "codex_auth.age")
	writeEncryptedAuthFixture(t, root, original, 10)
	t.Cleanup(func() { officialAuthCache.clearHome(root) })
	decoded, err := readEncryptedAuth(root, []byte("test-only-passphrase"))
	if err != nil || string(decoded) != original {
		t.Fatal("official namespace mismatch", err)
	}
	clear(decoded)
	if _, err := readEncryptedAuth(root, []byte("wrong-passphrase")); err == nil {
		t.Fatal("wrong key accepted")
	}
	if err := os.WriteFile(path, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readEncryptedAuth(root, []byte("test-only-passphrase")); err == nil {
		t.Fatal("corrupt ciphertext accepted")
	}
}

func writeEncryptedAuthFixture(t testing.TB, home, auth string, factor int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(home, "secrets"), 0700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(map[string]any{"version": 1, "secrets": map[string]string{"global/CODEX_AUTH": auth, "global/OTHER": "unrelated"}})
	if err != nil {
		t.Fatal(err)
	}
	defer clear(data)
	recipient, err := age.NewScryptRecipient("test-only-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	recipient.SetWorkFactor(factor)
	var ciphertext bytes.Buffer
	writer, err := age.Encrypt(&ciphertext, recipient)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "secrets", "codex_auth.age"), ciphertext.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
}
