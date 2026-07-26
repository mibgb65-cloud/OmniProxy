package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var updateDownloadTimeout = 30 * time.Minute

var updateDownloadAllowedHosts = []string{
	"github.com",
	"objects.githubusercontent.com",
	"release-assets.githubusercontent.com",
}

// validateUpdateDownloadURL restricts installer and checksum downloads to the
// release hosts. The downloaded file is later launched with elevation, so an
// arbitrary URL accepted here would be a code-execution path. Plain HTTP is
// tolerated only for loopback hosts, which no production allowlist contains.
func validateUpdateDownloadURL(kind string, raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("%s URL is invalid: %w", kind, err)
	}
	host := strings.ToLower(parsed.Hostname())
	allowed := false
	for _, candidate := range updateDownloadAllowedHosts {
		if host == candidate {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("%s URL host %q is not an allowed update host", kind, host)
	}
	if parsed.Scheme != "https" && !isLoopbackHost(host) {
		return fmt.Errorf("%s URL must use https", kind)
	}
	return nil
}

func newUpdateDownloader() *updateDownloader {
	status := loadUpdateStatus()
	cleanupUpdateDirectory(status.FilePath)
	saveUpdateStatus(status)
	appendUpdateLog("update downloader initialized with state=%s version=%s", status.State, status.Version)
	return &updateDownloader{status: status}
}

func (d *updateDownloader) Status() updateDownloadStatus {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.status
}

func (d *updateDownloader) Start(ctx context.Context, client *http.Client, req updateDownloadRequest) (updateDownloadStatus, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if strings.TrimSpace(req.DownloadURL) == "" {
		return updateDownloadStatus{}, fmt.Errorf("download URL is required")
	}
	if strings.TrimSpace(req.ChecksumURL) == "" {
		return updateDownloadStatus{}, fmt.Errorf("checksum URL is required")
	}
	if err := validateUpdateDownloadURL("download", req.DownloadURL); err != nil {
		return updateDownloadStatus{}, err
	}
	if err := validateUpdateDownloadURL("checksum", req.ChecksumURL); err != nil {
		return updateDownloadStatus{}, err
	}
	fileName, err := updateInstallerFileName(req)
	if err != nil {
		return updateDownloadStatus{}, err
	}
	started := time.Now().Format(time.RFC3339)
	status := updateDownloadStatus{
		State:       "downloading",
		Version:     strings.TrimSpace(req.Version),
		FileName:    fileName,
		DownloadURL: strings.TrimSpace(req.DownloadURL),
		ChecksumURL: strings.TrimSpace(req.ChecksumURL),
		TotalBytes:  req.ExpectedSize,
		StartedAt:   started,
		UpdatedAt:   started,
	}

	d.mu.Lock()
	if d.status.State == "downloading" {
		current := d.status
		d.mu.Unlock()
		return current, fmt.Errorf("update download is already in progress")
	}
	d.status = status
	d.mu.Unlock()
	saveUpdateStatus(status)
	appendUpdateLog("update download started version=%s file=%s", status.Version, status.FileName)

	downloadCtx := ctx
	if downloadCtx == nil {
		downloadCtx = context.Background()
	}
	downloadCtx, cancel := context.WithTimeout(downloadCtx, updateDownloadTimeout)
	go func() {
		defer cancel()
		d.download(downloadCtx, client, req, fileName)
	}()
	return status, nil
}

func (d *updateDownloader) Install() (updateDownloadStatus, error) {
	d.mu.Lock()
	status := d.status
	d.mu.Unlock()
	if status.State != "downloaded" || status.FilePath == "" {
		return status, fmt.Errorf("no downloaded update installer is ready")
	}
	if !status.Verified {
		return status, fmt.Errorf("downloaded update has not been verified")
	}
	if _, err := os.Stat(status.FilePath); err != nil {
		return status, fmt.Errorf("update installer is unavailable: %w", err)
	}

	if err := updateInstallerStart(status.FilePath, updateInstallerArgs()); err != nil {
		d.fail(fmt.Sprintf("start update installer: %v", err))
		return d.Status(), err
	}

	now := time.Now().Format(time.RFC3339)
	d.mu.Lock()
	d.status.State = "installing"
	d.status.Error = ""
	d.status.UpdatedAt = now
	status = d.status
	d.mu.Unlock()
	saveUpdateStatus(status)
	appendUpdateLog("update installer started version=%s file=%s", status.Version, status.FilePath)
	return status, nil
}

func updateInstallerArgs() []string {
	if goruntime.GOOS == "windows" {
		return []string{"/S", "/OMNIPROXY_AUTOUPDATE=1"}
	}
	return nil
}

func shouldQuitAfterUpdateInstall() bool {
	return goruntime.GOOS == "windows"
}

func (d *updateDownloader) download(ctx context.Context, client *http.Client, req updateDownloadRequest, fileName string) {
	if ctx == nil {
		ctx = context.Background()
	}
	dir := updateDirectory()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		d.fail(fmt.Sprintf("create update directory: %v", err))
		return
	}

	filePath := filepath.Join(dir, fileName)
	tmpPath := filePath + ".download"
	cleanupUpdateDirectory(filePath)
	if err := downloadFile(ctx, client, strings.TrimSpace(req.DownloadURL), tmpPath, req.ExpectedSize, d.setProgress); err != nil {
		_ = os.Remove(tmpPath)
		d.fail(err.Error())
		return
	}
	_ = os.Remove(filePath)
	if err := os.Rename(tmpPath, filePath); err != nil {
		_ = os.Remove(tmpPath)
		d.fail(fmt.Sprintf("finish update download: %v", err))
		return
	}

	if err := verifyDownloadedInstaller(ctx, client, filePath, strings.TrimSpace(req.ChecksumURL)); err != nil {
		_ = os.Remove(filePath)
		d.fail(err.Error())
		return
	}

	now := time.Now().Format(time.RFC3339)
	size := fileSize(filePath)
	d.mu.Lock()
	d.status.State = "downloaded"
	d.status.FilePath = filePath
	d.status.BytesReceived = size
	if d.status.TotalBytes <= 0 {
		d.status.TotalBytes = size
	}
	d.status.Percent = 100
	d.status.Verified = true
	d.status.Error = ""
	d.status.UpdatedAt = now
	d.status.CompletedAt = now
	status := d.status
	d.mu.Unlock()
	saveUpdateStatus(status)
	appendUpdateLog("update download completed version=%s file=%s bytes=%d", status.Version, status.FilePath, size)
	cleanupUpdateDirectory(status.FilePath)
}

func downloadFile(ctx context.Context, client *http.Client, rawURL string, filePath string, expectedSize int64, progress func(received int64, total int64)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "OmniProxy/"+strings.TrimSpace(appVersion))
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download update: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download update: server returned %d", resp.StatusCode)
	}

	total := resp.ContentLength
	if total <= 0 {
		total = expectedSize
	}
	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create update installer: %w", err)
	}
	defer out.Close()

	var received int64
	buffer := make([]byte, 64*1024)
	for {
		n, readErr := resp.Body.Read(buffer)
		if n > 0 {
			if _, err := out.Write(buffer[:n]); err != nil {
				return fmt.Errorf("write update installer: %w", err)
			}
			received += int64(n)
			if progress != nil {
				progress(received, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read update installer: %w", readErr)
		}
	}
	return nil
}

func verifyDownloadedInstaller(ctx context.Context, client *http.Client, filePath string, checksumURL string) error {
	expected, err := downloadSHA256(ctx, client, checksumURL)
	if err != nil {
		return err
	}
	actual, err := fileSHA256(filePath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(expected, actual) {
		return fmt.Errorf("verify update installer: checksum mismatch")
	}
	return nil
}

func downloadSHA256(ctx context.Context, client *http.Client, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "OmniProxy/"+strings.TrimSpace(appVersion))
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download update checksum: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download update checksum: server returned %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", fmt.Errorf("read update checksum: %w", err)
	}
	for _, field := range strings.Fields(string(body)) {
		candidate := strings.TrimSpace(field)
		if len(candidate) != sha256.Size*2 {
			continue
		}
		if _, err := strconv.ParseUint(candidate[:16], 16, 64); err == nil && isHexString(candidate) {
			return strings.ToLower(candidate), nil
		}
	}
	return "", fmt.Errorf("download update checksum: no SHA256 hash found")
}

func fileSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open update installer: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash update installer: %w", err)
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func isHexString(value string) bool {
	for _, char := range value {
		if !unicode.IsDigit(char) && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return false
		}
	}
	return true
}

func updateInstallerFileName(req updateDownloadRequest) (string, error) {
	return updateInstallerFileNameForPlatform(req, goruntime.GOOS)
}

func updateInstallerFileNameForPlatform(req updateDownloadRequest, goos string) (string, error) {
	name := strings.TrimSpace(req.FileName)
	if name == "" {
		name = fileNameFromURL(req.DownloadURL)
	}
	if name == "" {
		name = defaultUpdateInstallerFileName(goos)
	}
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	if !validUpdateInstallerExtension(name, goos) {
		return "", fmt.Errorf("update installer must be a %s file", strings.Join(updateInstallerExtensions(goos), " or "))
	}
	if strings.ToLower(strings.TrimSpace(goos)) == "windows" && strings.ContainsAny(name, `<>:"|?*`) {
		return "", fmt.Errorf("update installer file name contains invalid characters")
	}
	return name, nil
}

func defaultUpdateInstallerFileName(goos string) string {
	switch strings.ToLower(strings.TrimSpace(goos)) {
	case "darwin":
		return "OmniProxy.dmg"
	default:
		return "OmniProxy-Setup.exe"
	}
}

func validUpdateInstallerExtension(name string, goos string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	for _, ext := range updateInstallerExtensions(goos) {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

func updateInstallerExtensions(goos string) []string {
	switch strings.ToLower(strings.TrimSpace(goos)) {
	case "darwin":
		return []string{".dmg"}
	case "windows":
		return []string{".exe"}
	default:
		return []string{".exe"}
	}
}

func fileNameFromURL(rawURL string) string {
	parts := strings.Split(strings.TrimSpace(rawURL), "/")
	if len(parts) == 0 {
		return ""
	}
	name := parts[len(parts)-1]
	if question := strings.IndexByte(name, '?'); question >= 0 {
		name = name[:question]
	}
	return strings.TrimSpace(name)
}

func (d *updateDownloader) setProgress(received int64, total int64) {
	now := time.Now().Format(time.RFC3339)
	d.mu.Lock()
	d.status.BytesReceived = received
	if total > 0 {
		d.status.TotalBytes = total
		d.status.Percent = int((received * 100) / total)
		if d.status.Percent > 99 {
			d.status.Percent = 99
		}
	}
	d.status.UpdatedAt = now
	d.mu.Unlock()
}

func (d *updateDownloader) fail(message string) {
	now := time.Now().Format(time.RFC3339)
	d.mu.Lock()
	d.status.State = "failed"
	d.status.Error = message
	d.status.UpdatedAt = now
	d.status.CompletedAt = now
	status := d.status
	d.mu.Unlock()
	saveUpdateStatus(status)
	appendUpdateLog("update failed state=%s version=%s error=%s", status.State, status.Version, message)
}

func fileSize(filePath string) int64 {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0
	}
	return info.Size()
}
