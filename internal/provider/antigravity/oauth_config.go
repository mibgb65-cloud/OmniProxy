package antigravity

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"regexp"
)

// 桌面客户端 ID 是公开标识；secret 从用户安装的官方组件读取，不进入仓库。
const desktopClientID = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com"

// 公开桌面配置的指纹用于区分 agy 内嵌的多个 Google 客户端，不包含 secret 本身。
const desktopConfigFingerprint = "1d2f041093fd95aa8995a038c711d50a7960da09a505381c09a745d6ad0ecc60"

var clientPattern = regexp.MustCompile(`GOCSPX-[A-Za-z0-9_-]{28}`)

// OAuthConfig 仅在后端内存和受保护凭证内流转。
type OAuthConfig struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

// InstalledOAuthConfig 有界读取官方 agy 的公开客户端配置，不运行程序或读取已有登录。
func InstalledOAuthConfig() (OAuthConfig, error) {
	path, err := exec.LookPath("agy")
	if err != nil {
		return OAuthConfig{}, Failure("agy_unavailable")
	}
	f, err := os.Open(path)
	if err != nil {
		return OAuthConfig{}, Failure("agy_unavailable")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1024 || info.Size() > 256<<20 {
		return OAuthConfig{}, Failure("client_incompatible")
	}
	return readOAuthConfig(f, desktopConfigFingerprint)
}

func readOAuthConfig(reader io.Reader, fingerprint string) (OAuthConfig, error) {
	buffer := make([]byte, 64*1024+256)
	defer clear(buffer)
	foundID, secret, tail := false, "", 0
	for total := 0; total <= 256<<20; {
		n, err := reader.Read(buffer[tail:])
		total += n
		chunk := buffer[:tail+n]
		foundID = foundID || bytes.Contains(chunk, []byte(desktopClientID))
		for _, candidate := range clientPattern.FindAll(chunk, -1) {
			hash := sha256.Sum256(candidate)
			if hex.EncodeToString(hash[:]) != fingerprint {
				continue
			}
			if secret != "" && secret != string(candidate) {
				return OAuthConfig{}, Failure("client_incompatible")
			}
			secret = string(candidate)
		}
		if err == io.EOF {
			if foundID && secret != "" {
				return OAuthConfig{desktopClientID, secret}, nil
			}
			break
		}
		if err != nil || n == 0 {
			break
		}
		tail = min(len(chunk), 256)
		copy(buffer, chunk[len(chunk)-tail:])
	}
	return OAuthConfig{}, Failure("client_incompatible")
}
