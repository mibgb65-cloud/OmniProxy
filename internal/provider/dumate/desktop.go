package dumate

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	// DesktopAppDir 是搭子桌面端的默认用户数据目录名。
	DesktopAppDir = "qianfan-desktop-app"
	authFileName  = "auth.json"
	keyFileName   = ".cookie-key"
	// aesKeyBytes 是 AES-256 密钥长度，桌面端不会使用其他长度。
	aesKeyBytes = 32
	// gcmIVBytes 与 gcmTagBytes 是参考实现确认过的密文布局。
	gcmIVBytes  = 12
	gcmTagBytes = 16

	maxAuthFileBytes = 1 << 20
	maxKeyFileBytes  = 64
	maxDecodedBytes  = 1 << 20
)

var (
	// ErrDesktopLoginMissing 表示本机没有可用的搭子登录文件。
	ErrDesktopLoginMissing = errors.New("dumate: desktop login missing")
	// ErrDesktopLoginInvalid 表示登录文件存在但无法安全解析。
	ErrDesktopLoginInvalid = errors.New("dumate: desktop login invalid")
)

// DesktopDir 解析桌面端用户数据目录：显式环境变量优先，其次 %APPDATA%。
func DesktopDir() (string, error) {
	for _, key := range []string{"OMNIPROXY_DUMATE_USER_DATA_DIR", "DUMATE_USER_DATA_DIR"} {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			continue
		}
		if !filepath.IsAbs(value) {
			return "", ErrDesktopLoginInvalid
		}
		return filepath.Clean(value), nil
	}
	appData := strings.TrimSpace(os.Getenv("APPDATA"))
	if appData == "" || !filepath.IsAbs(appData) {
		return "", ErrDesktopLoginMissing
	}
	return filepath.Join(appData, DesktopAppDir), nil
}

type desktopProfile struct {
	ProfileID    string `json:"profileId"`
	BCEAccountID string `json:"bceAccountId"`
	DisplayName  string `json:"displayName"`
}

type desktopAuthFile struct {
	Cookies         json.RawMessage  `json:"cookies"`
	AccountProfiles []desktopProfile `json:"accountProfiles"`
	ActiveProfileID string           `json:"activeProfileId"`
}

type desktopCookie struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Domain string `json:"domain"`
	Path   string `json:"path"`
}

// ReadDesktop 只读解析 dir 下的登录文件；失败时返回固定错误，不泄露路径细节。
func ReadDesktop(dir string) (Credentials, error) {
	auth, err := readAuthFile(filepath.Join(dir, authFileName))
	if err != nil {
		return Credentials{}, err
	}
	cookies, err := decryptCookies(filepath.Join(dir, keyFileName), auth.Cookies)
	if err != nil {
		return Credentials{}, err
	}
	profile := activeProfile(auth)
	return Credentials{Cookies: cookies, Source: SourceDesktop, AccountID: profile.BCEAccountID, Name: profile.DisplayName}, nil
}

// activeProfile 优先取 activeProfileId 指向的档案；缺失时退回第一个档案。
func activeProfile(auth desktopAuthFile) desktopProfile {
	for _, profile := range auth.AccountProfiles {
		if auth.ActiveProfileID != "" && profile.ProfileID == auth.ActiveProfileID {
			return profile
		}
	}
	if len(auth.AccountProfiles) > 0 {
		return auth.AccountProfiles[0]
	}
	return desktopProfile{}
}

func readAuthFile(path string) (desktopAuthFile, error) {
	data, err := readBoundedFile(path, maxAuthFileBytes)
	if errors.Is(err, os.ErrNotExist) {
		return desktopAuthFile{}, ErrDesktopLoginMissing
	}
	if err != nil {
		return desktopAuthFile{}, ErrDesktopLoginInvalid
	}
	var auth desktopAuthFile
	if json.Unmarshal(data, &auth) != nil || len(auth.Cookies) == 0 {
		return desktopAuthFile{}, ErrDesktopLoginInvalid
	}
	return auth, nil
}

// decryptCookies 按 IV(12) + Tag(16) + Ciphertext 布局解出 Cookie 数组。
func decryptCookies(keyPath string, encoded json.RawMessage) ([]Cookie, error) {
	var payload string
	if json.Unmarshal(encoded, &payload) != nil || payload == "" || len(payload) > maxDecodedBytes*2 {
		return nil, ErrDesktopLoginInvalid
	}
	key, err := readBoundedFile(keyPath, maxKeyFileBytes)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrDesktopLoginMissing
	}
	if err != nil || len(key) != aesKeyBytes {
		return nil, ErrDesktopLoginInvalid
	}
	defer clearBytes(key)
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil || len(raw) <= gcmIVBytes+gcmTagBytes || len(raw) > maxDecodedBytes {
		return nil, ErrDesktopLoginInvalid
	}
	plain, err := decryptGCM(key, raw)
	if err != nil {
		return nil, ErrDesktopLoginInvalid
	}
	defer clearBytes(plain)
	var decoded []desktopCookie
	if json.Unmarshal(plain, &decoded) != nil || len(decoded) == 0 || len(decoded) > maxCookies {
		return nil, ErrDesktopLoginInvalid
	}
	cookies := make([]Cookie, 0, len(decoded))
	for _, item := range decoded {
		cookie := Cookie(item)
		if !validCookie(cookie) {
			continue
		}
		cookies = append(cookies, cookie)
	}
	if Validate(cookies) != nil {
		return nil, ErrDesktopLoginInvalid
	}
	return cookies, nil
}

func decryptGCM(key, raw []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := raw[:gcmIVBytes]
	tag := raw[gcmIVBytes : gcmIVBytes+gcmTagBytes]
	ciphertext := raw[gcmIVBytes+gcmTagBytes:]
	return aead.Open(nil, nonce, append(append([]byte(nil), ciphertext...), tag...), nil)
}

func readBoundedFile(path string, limit int64) ([]byte, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > limit {
		return nil, ErrDesktopLoginInvalid
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(data)) > limit {
		clearBytes(data)
		return nil, ErrDesktopLoginInvalid
	}
	return data, nil
}

func clearBytes(data []byte) {
	for i := range data {
		data[i] = 0
	}
}
