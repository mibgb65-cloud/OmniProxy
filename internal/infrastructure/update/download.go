package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// MaxInstallerSize 限制不可信资产的磁盘及流量消耗。
const MaxInstallerSize int64 = 1 << 30

// DownloadClient 为下载保留独立的重定向策略，可由隔离测试注入。
func DownloadClient(client *http.Client, allowed func(string) bool) *http.Client {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Minute}
	}
	copy := *client
	copy.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 || !allowed(req.URL.String()) {
			return errors.New("update: redirect_denied")
		}
		return nil
	}
	return &copy
}

// Download 精确核对字节数、写入权限和校验和；失败文件由调用方清理。
func Download(ctx context.Context, client *http.Client, p Package, path string, progress func(int64)) (digest string, err error) {
	expected, err := Checksum(ctx, client, p.ChecksumURL, p.Name)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.URL, nil)
	if err != nil {
		return "", errors.New("update: invalid_request")
	}
	req.Header.Set("User-Agent", "OmniProxy-updater")
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.New("update: download_failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 || resp.ContentLength > 0 && resp.ContentLength != p.Size {
		return "", errors.New("update: size_mismatch")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", errors.New("update: write_failed")
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = errors.New("update: write_failed")
		}
	}()
	hash := sha256.New()
	buffer := make([]byte, 64<<10)
	var received int64
	reader := io.LimitReader(resp.Body, p.Size+1)
	for {
		n, readErr := reader.Read(buffer)
		if n > 0 {
			received += int64(n)
			if received > p.Size {
				return "", errors.New("update: size_mismatch")
			}
			if _, err := file.Write(buffer[:n]); err != nil {
				return "", errors.New("update: write_failed")
			}
			hash.Write(buffer[:n])
			progress(received)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", errors.New("update: download_interrupted")
		}
	}
	if received != p.Size {
		return "", errors.New("update: size_mismatch")
	}
	digest = hex.EncodeToString(hash.Sum(nil))
	if digest != expected {
		return "", errors.New("update: checksum_mismatch")
	}
	if file.Sync() != nil {
		return "", errors.New("update: write_failed")
	}
	return digest, nil
}

// Checksum 只接受单个 SHA256，文件名必须与所选资产对应。
func Checksum(ctx context.Context, client *http.Client, raw, name string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return "", errors.New("update: invalid_request")
	}
	req.Header.Set("User-Agent", "OmniProxy-updater")
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.New("update: checksum_unavailable")
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4097))
	if err != nil || resp.StatusCode != 200 || len(data) > 4096 {
		return "", errors.New("update: checksum_unavailable")
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 || len(fields) > 2 || len(fields) == 2 && strings.TrimPrefix(fields[1], "*") != name {
		return "", errors.New("update: invalid_checksum")
	}
	decoded, err := hex.DecodeString(fields[0])
	if err != nil || len(decoded) != 32 {
		return "", errors.New("update: invalid_checksum")
	}
	return strings.ToLower(fields[0]), nil
}

// VerifyFile 复验持久缓存，拒绝符号链接、大小变化和篡改。
func VerifyFile(path, digest string, size int64) error {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() != size || size < 1 || size > MaxInstallerSize {
		return errors.New("update: package_unavailable")
	}
	file, err := os.Open(path)
	if err != nil {
		return errors.New("update: package_unavailable")
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return errors.New("update: package_changed")
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return errors.New("update: package_unavailable")
	}
	if hex.EncodeToString(hash.Sum(nil)) != digest {
		return errors.New("update: checksum_mismatch")
	}
	return nil
}
