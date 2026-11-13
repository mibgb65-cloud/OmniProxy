package dumate

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeDesktopFixture 生成与桌面端一致的 AES-256-GCM 登录文件。
func writeDesktopFixture(t *testing.T, dir string, cookies []Cookie, key []byte, profiles []desktopProfile, active string) {
	t.Helper()
	plain, err := json.Marshal(cookies)
	if err != nil {
		t.Fatal(err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	nonce := make([]byte, gcmIVBytes)
	if _, err := rand.Read(nonce); err != nil {
		t.Fatal(err)
	}
	sealed := aead.Seal(nil, nonce, plain, nil)
	ciphertext, tag := sealed[:len(sealed)-gcmTagBytes], sealed[len(sealed)-gcmTagBytes:]
	blob := append(append(append([]byte(nil), nonce...), tag...), ciphertext...)
	auth := map[string]any{
		"cookies":         base64.StdEncoding.EncodeToString(blob),
		"accountProfiles": profiles,
		"activeProfileId": active,
	}
	data, err := json.Marshal(auth)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, authFileName), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, keyFileName), key, 0o600); err != nil {
		t.Fatal(err)
	}
}

func testKey() []byte {
	key := make([]byte, aesKeyBytes)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return key
}

func TestReadDesktopDecryptsActiveProfileIdentity(t *testing.T) {
	dir := t.TempDir()
	profiles := []desktopProfile{{ProfileID: "p1", BCEAccountID: "111", DisplayName: "First"}, {ProfileID: "p2", BCEAccountID: "222", DisplayName: "Second"}}
	writeDesktopFixture(t, dir, []Cookie{{Name: SessionCookie, Value: "s", Domain: ".baidu.com"}, {Name: "bce-user-info", Value: "csrf", Domain: ".baidu.com"}}, testKey(), profiles, "p2")
	credentials, err := ReadDesktop(dir)
	if err != nil {
		t.Fatal(err)
	}
	if credentials.Source != SourceDesktop || credentials.AccountID != "222" || credentials.Name != "Second" {
		t.Fatalf("identity = %+v", credentials)
	}
	if len(credentials.Cookies) != 2 || credentials.Cookies[1].Name != "bce-user-info" {
		t.Fatalf("cookies = %+v", credentials.Cookies)
	}
}

func TestReadDesktopRejectsUnreadableLoginFiles(t *testing.T) {
	dir := t.TempDir()
	if _, err := ReadDesktop(dir); !errors.Is(err, ErrDesktopLoginMissing) {
		t.Fatalf("missing auth file = %v", err)
	}
	writeDesktopFixture(t, dir, []Cookie{{Name: SessionCookie, Value: "s"}}, testKey(), nil, "")
	if err := os.Remove(filepath.Join(dir, keyFileName)); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadDesktop(dir); !errors.Is(err, ErrDesktopLoginMissing) {
		t.Fatalf("missing key file = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, keyFileName), []byte("short"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadDesktop(dir); !errors.Is(err, ErrDesktopLoginInvalid) {
		t.Fatalf("short key = %v", err)
	}
}

func TestReadDesktopRejectsWrongKeyAndMissingSessionCookie(t *testing.T) {
	dir := t.TempDir()
	writeDesktopFixture(t, dir, []Cookie{{Name: SessionCookie, Value: "s"}}, testKey(), nil, "")
	wrong := testKey()
	wrong[0] = 0xff
	if err := os.WriteFile(filepath.Join(dir, keyFileName), wrong, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadDesktop(dir); !errors.Is(err, ErrDesktopLoginInvalid) {
		t.Fatalf("wrong key = %v", err)
	}
	writeDesktopFixture(t, dir, []Cookie{{Name: "visible", Value: "v"}}, testKey(), nil, "")
	if _, err := ReadDesktop(dir); !errors.Is(err, ErrDesktopLoginInvalid) {
		t.Fatal("login file without session cookie was accepted")
	}
}

func TestDesktopDirPrefersExplicitOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", `C:\Users\test\AppData\Roaming`)
	t.Setenv("DUMATE_USER_DATA_DIR", "")
	t.Setenv("OMNIPROXY_DUMATE_USER_DATA_DIR", dir)
	resolved, err := DesktopDir()
	if err != nil || resolved != filepath.Clean(dir) {
		t.Fatalf("override = (%q, %v)", resolved, err)
	}
	t.Setenv("OMNIPROXY_DUMATE_USER_DATA_DIR", "relative/path")
	if _, err := DesktopDir(); err == nil {
		t.Fatal("relative override accepted")
	}
	t.Setenv("OMNIPROXY_DUMATE_USER_DATA_DIR", "")
	t.Setenv("DUMATE_USER_DATA_DIR", "")
	resolved, err = DesktopDir()
	if err != nil || resolved != filepath.Join(`C:\Users\test\AppData\Roaming`, DesktopAppDir) {
		t.Fatalf("appdata fallback = (%q, %v)", resolved, err)
	}
}
