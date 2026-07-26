package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCheckForUpdatesReportsAvailableRelease(t *testing.T) {
	restore := overrideUpdateGlobals("v1.0.2", "")
	defer restore()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "OmniProxy/v1.0.2" {
			t.Fatalf("unexpected user agent: %q", r.Header.Get("User-Agent"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name": "v1.0.3",
			"name":     "OmniProxy v1.0.3",
			"html_url": "https://github.com/mibgb65-cloud/OmniProxy/releases/tag/v1.0.3",
			"body":     "release notes",
			"assets": []map[string]any{
				{
					"name":                 "OmniProxy-Setup-v1.0.3-windows-amd64.exe",
					"browser_download_url": "https://example.com/installer.exe",
					"size":                 123,
				},
				{
					"name":                 "OmniProxy-Setup-v1.0.3-windows-amd64.exe.sha256",
					"browser_download_url": "https://example.com/installer.exe.sha256",
				},
				{
					"name":                 "OmniProxy-v1.0.3-darwin-universal.dmg",
					"browser_download_url": "https://example.com/installer.dmg",
					"size":                 456,
				},
				{
					"name":                 "OmniProxy-v1.0.3-darwin-universal.dmg.sha256",
					"browser_download_url": "https://example.com/installer.dmg.sha256",
				},
			},
		})
	}))
	defer server.Close()
	latestReleaseURL = server.URL

	info, err := checkForUpdates(context.Background(), server.Client(), false)
	if err != nil {
		t.Fatal(err)
	}
	if !info.UpdateAvailable {
		t.Fatalf("expected update to be available, got %#v", info)
	}
	wantName, wantURL, wantChecksum, wantSize := expectedUpdateAsset("v1.0.3", "installer")
	if info.CurrentVersion != "v1.0.2" || info.LatestVersion != "v1.0.3" || info.DownloadURL != wantURL {
		t.Fatalf("unexpected update info: %#v", info)
	}
	if info.ChecksumURL != wantChecksum || info.DownloadFileName != wantName || info.DownloadSize != wantSize {
		t.Fatalf("unexpected update info: %#v", info)
	}
}

func TestCheckForUpdatesReportsPrereleaseWhenEnabled(t *testing.T) {
	restore := overrideUpdateGlobals("v1.0.2", "")
	defer restore()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Fatalf("unexpected releases path: %s", r.URL.String())
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"tag_name":   "v1.0.3",
				"name":       "OmniProxy v1.0.3",
				"html_url":   "https://github.com/mibgb65-cloud/OmniProxy/releases/tag/v1.0.3",
				"body":       "stable notes",
				"prerelease": false,
				"assets": []map[string]any{
					{
						"name":                 "OmniProxy-Setup-v1.0.3-windows-amd64.exe",
						"browser_download_url": "https://example.com/stable.exe",
					},
					{
						"name":                 "OmniProxy-Setup-v1.0.3-windows-amd64.exe.sha256",
						"browser_download_url": "https://example.com/stable.exe.sha256",
					},
					{
						"name":                 "OmniProxy-v1.0.3-darwin-universal.dmg",
						"browser_download_url": "https://example.com/stable.dmg",
					},
					{
						"name":                 "OmniProxy-v1.0.3-darwin-universal.dmg.sha256",
						"browser_download_url": "https://example.com/stable.dmg.sha256",
					},
				},
			},
			{
				"tag_name":   "v1.0.4-beta.1",
				"name":       "OmniProxy v1.0.4-beta.1",
				"html_url":   "https://github.com/mibgb65-cloud/OmniProxy/releases/tag/v1.0.4-beta.1",
				"body":       "beta notes",
				"prerelease": true,
				"assets": []map[string]any{
					{
						"name":                 "OmniProxy-Setup-v1.0.4-beta.1-windows-amd64.exe",
						"browser_download_url": "https://example.com/beta.exe",
						"size":                 456,
					},
					{
						"name":                 "OmniProxy-Setup-v1.0.4-beta.1-windows-amd64.exe.sha256",
						"browser_download_url": "https://example.com/beta.exe.sha256",
					},
					{
						"name":                 "OmniProxy-v1.0.4-beta.1-darwin-universal.dmg",
						"browser_download_url": "https://example.com/beta.dmg",
						"size":                 789,
					},
					{
						"name":                 "OmniProxy-v1.0.4-beta.1-darwin-universal.dmg.sha256",
						"browser_download_url": "https://example.com/beta.dmg.sha256",
					},
				},
			},
		})
	}))
	defer server.Close()
	releasesURL = server.URL

	info, err := checkForUpdates(context.Background(), server.Client(), true)
	if err != nil {
		t.Fatal(err)
	}
	if !info.UpdateAvailable || !info.Prerelease {
		t.Fatalf("expected prerelease update, got %#v", info)
	}
	_, wantURL, _, _ := expectedUpdateAsset("v1.0.4-beta.1", "beta")
	if info.LatestVersion != "v1.0.4-beta.1" || info.DownloadURL != wantURL {
		t.Fatalf("unexpected prerelease update info: %#v", info)
	}
}

func TestLatestVersionedReleaseSkipsPrereleaseWhenDisabled(t *testing.T) {
	releases := []githubRelease{
		{TagName: "v1.0.4-beta.1", Prerelease: true},
		{TagName: "v1.0.3"},
	}
	release, ok := latestVersionedRelease(releases, false)
	if !ok || release.TagName != "v1.0.3" {
		t.Fatalf("expected latest stable release, got ok=%v release=%#v", ok, release)
	}
	release, ok = latestVersionedRelease(releases, true)
	if !ok || release.TagName != "v1.0.4-beta.1" {
		t.Fatalf("expected prerelease candidate when enabled, got ok=%v release=%#v", ok, release)
	}
}

func TestLatestVersionedReleaseSortsNumericPrereleaseIdentifiers(t *testing.T) {
	releases := []githubRelease{
		{TagName: "v1.2.1-beta.9", Prerelease: true},
		{TagName: "v1.2.1-beta.10", Prerelease: true},
		{TagName: "v1.2.1-beta.2", Prerelease: true},
	}
	release, ok := latestVersionedRelease(releases, true)
	if !ok || release.TagName != "v1.2.1-beta.10" {
		t.Fatalf("expected beta.10 candidate, got ok=%v release=%#v", ok, release)
	}
}

func TestUpdateDownloadAssetFromAssetsSelectsInstallerAndChecksum(t *testing.T) {
	asset := updateDownloadAssetFromAssetsForPlatform([]githubReleaseAsset{
		{Name: "source.zip", BrowserDownloadURL: "https://example.com/source.zip"},
		{Name: "OmniProxy-Setup-v1.2.3-windows-amd64.exe.sha256", BrowserDownloadURL: "https://example.com/setup.exe.sha256"},
		{Name: "OmniProxy-Setup-v1.2.3-windows-amd64.exe", BrowserDownloadURL: "https://example.com/setup.exe", Size: 42},
	}, "windows", "amd64")
	if asset.URL != "https://example.com/setup.exe" || asset.ChecksumURL != "https://example.com/setup.exe.sha256" || asset.FileName != "OmniProxy-Setup-v1.2.3-windows-amd64.exe" || asset.Size != 42 {
		t.Fatalf("unexpected asset selection: %#v", asset)
	}
}

func TestUpdateDownloadAssetFromAssetsSelectsDarwinUniversal(t *testing.T) {
	asset := updateDownloadAssetFromAssetsForPlatform([]githubReleaseAsset{
		{Name: "OmniProxy-Setup-v1.2.3-windows-amd64.exe", BrowserDownloadURL: "https://example.com/setup.exe"},
		{Name: "OmniProxy-v1.2.3-darwin-arm64.dmg", BrowserDownloadURL: "https://example.com/arm64.dmg"},
		{Name: "OmniProxy-v1.2.3-darwin-universal.dmg.sha256", BrowserDownloadURL: "https://example.com/universal.dmg.sha256"},
		{Name: "OmniProxy-v1.2.3-darwin-universal.dmg", BrowserDownloadURL: "https://example.com/universal.dmg", Size: 84},
	}, "darwin", "arm64")
	if asset.URL != "https://example.com/universal.dmg" || asset.ChecksumURL != "https://example.com/universal.dmg.sha256" || asset.FileName != "OmniProxy-v1.2.3-darwin-universal.dmg" || asset.Size != 84 {
		t.Fatalf("unexpected darwin asset selection: %#v", asset)
	}
}

func TestUpdateDownloadAssetFromAssetsDoesNotUseOtherPlatformChecksum(t *testing.T) {
	asset := updateDownloadAssetFromAssetsForPlatform([]githubReleaseAsset{
		{Name: "OmniProxy-Setup-v1.2.3-windows-amd64.exe", BrowserDownloadURL: "https://example.com/setup.exe"},
		{Name: "OmniProxy-Setup-v1.2.3-windows-amd64.exe.sha256", BrowserDownloadURL: "https://example.com/setup.exe.sha256"},
		{Name: "OmniProxy-v1.2.3-darwin-universal.dmg", BrowserDownloadURL: "https://example.com/universal.dmg"},
	}, "darwin", "arm64")
	if asset.URL != "https://example.com/universal.dmg" {
		t.Fatalf("expected darwin dmg to be selected, got %#v", asset)
	}
	if asset.ChecksumURL != "" {
		t.Fatalf("expected missing darwin checksum to stay empty, got %#v", asset)
	}
}

func TestUpdateDownloaderDownloadsAndVerifiesInstaller(t *testing.T) {
	useTestUpdateDirectory(t)

	content := []byte("fake installer bytes")
	sum := sha256.Sum256(content)
	fileName := testInstallerFileName()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/installer.exe":
			w.Header().Set("Content-Length", fmt.Sprint(len(content)))
			_, _ = w.Write(content)
		case "/installer.exe.sha256":
			_, _ = fmt.Fprintf(w, "%x  %s\n", sum, fileName)
		default:
			t.Fatalf("unexpected download path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	allowLoopbackUpdateHost(t)

	downloader := newUpdateDownloader()
	status, err := downloader.Start(context.Background(), server.Client(), updateDownloadRequest{
		Version:      "v9.9.9",
		DownloadURL:  server.URL + "/installer.exe",
		ChecksumURL:  server.URL + "/installer.exe.sha256",
		FileName:     fileName,
		ExpectedSize: int64(len(content)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "downloading" {
		t.Fatalf("expected downloading status, got %#v", status)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		status = downloader.Status()
		if status.State == "downloaded" || status.State == "failed" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	defer func() {
		if status.FilePath != "" {
			_ = os.Remove(status.FilePath)
		}
	}()
	if status.State != "downloaded" {
		t.Fatalf("expected downloaded status, got %#v", status)
	}
	if !status.Verified || status.Percent != 100 || status.BytesReceived != int64(len(content)) {
		t.Fatalf("unexpected download status: %#v", status)
	}
	saved, err := os.ReadFile(status.FilePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(saved) != string(content) {
		t.Fatalf("unexpected saved content: %q", saved)
	}
}

func TestUpdateDownloaderFailsHungDownloadAfterTimeout(t *testing.T) {
	useTestUpdateDirectory(t)

	oldTimeout := updateDownloadTimeout
	updateDownloadTimeout = 20 * time.Millisecond
	defer func() {
		updateDownloadTimeout = oldTimeout
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/installer.exe":
			time.Sleep(200 * time.Millisecond)
			_, _ = w.Write([]byte("late installer bytes"))
		case "/installer.exe.sha256":
			t.Fatal("checksum should not be requested after a timed-out installer download")
		default:
			t.Fatalf("unexpected download path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	allowLoopbackUpdateHost(t)

	downloader := newUpdateDownloader()
	_, err := downloader.Start(context.Background(), server.Client(), updateDownloadRequest{
		Version:     "v9.9.9",
		DownloadURL: server.URL + "/installer.exe",
		ChecksumURL: server.URL + "/installer.exe.sha256",
		FileName:    testInstallerFileName(),
	})
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(time.Second)
	status := downloader.Status()
	for time.Now().Before(deadline) {
		status = downloader.Status()
		if status.State == "failed" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if status.State != "failed" || status.Error == "" {
		t.Fatalf("expected timed-out download to fail, got %#v", status)
	}
}

func TestUpdateDownloaderInstallStartsSilentAutoUpdate(t *testing.T) {
	useTestUpdateDirectory(t)

	filePath := filepath.Join(t.TempDir(), "OmniProxy-Setup-test.exe")
	if err := os.WriteFile(filePath, []byte("fake installer"), 0o600); err != nil {
		t.Fatal(err)
	}

	oldStart := updateInstallerStart
	defer func() {
		updateInstallerStart = oldStart
	}()

	var gotPath string
	var gotArgs []string
	updateInstallerStart = func(filePath string, args []string) error {
		gotPath = filePath
		gotArgs = append([]string(nil), args...)
		return nil
	}

	downloader := newUpdateDownloader()
	downloader.status = updateDownloadStatus{
		State:    "downloaded",
		FileName: filepath.Base(filePath),
		FilePath: filePath,
		Verified: true,
	}

	status, err := downloader.Install()
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "installing" {
		t.Fatalf("expected installing status, got %#v", status)
	}
	if gotPath != filePath {
		t.Fatalf("expected installer path %q, got %q", filePath, gotPath)
	}
	if !reflect.DeepEqual(gotArgs, updateInstallerArgs()) {
		t.Fatalf("unexpected installer args: %#v", gotArgs)
	}
}

func TestUpdateDownloaderRestoresVerifiedDownloadStatus(t *testing.T) {
	dir := useTestUpdateDirectory(t)
	filePath := filepath.Join(dir, "OmniProxy-Setup-v9.9.9-windows-amd64.exe")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("fake installer"), 0o600); err != nil {
		t.Fatal(err)
	}

	saveUpdateStatus(updateDownloadStatus{
		State:    "downloaded",
		Version:  "v9.9.9",
		FileName: filepath.Base(filePath),
		FilePath: filePath,
		Verified: true,
		Percent:  100,
	})

	status := newUpdateDownloader().Status()
	if status.State != "downloaded" || status.FilePath != filePath || !status.Verified {
		t.Fatalf("expected verified download status to be restored, got %#v", status)
	}
}

func TestUpdateDownloaderMarksInterruptedDownloadFailed(t *testing.T) {
	useTestUpdateDirectory(t)

	saveUpdateStatus(updateDownloadStatus{
		State:   "downloading",
		Version: "v9.9.9",
	})

	status := newUpdateDownloader().Status()
	if status.State != "failed" || status.Error == "" {
		t.Fatalf("expected interrupted download to be marked failed, got %#v", status)
	}
}

func TestUpdateDownloaderRejectsPersistedInstallerOutsideUpdateDirectory(t *testing.T) {
	useTestUpdateDirectory(t)
	filePath := filepath.Join(t.TempDir(), "OmniProxy-Setup-outside.exe")
	if err := os.WriteFile(filePath, []byte("fake installer"), 0o600); err != nil {
		t.Fatal(err)
	}

	saveUpdateStatus(updateDownloadStatus{
		State:    "downloaded",
		Version:  "v9.9.9",
		FileName: filepath.Base(filePath),
		FilePath: filePath,
		Verified: true,
	})

	status := newUpdateDownloader().Status()
	if status.State != "failed" || status.FilePath != "" || status.Verified {
		t.Fatalf("expected outside installer path to be rejected, got %#v", status)
	}
}

func TestCurrentUpdateDiagnosticsReportsStatusAndLogTail(t *testing.T) {
	dir := useTestUpdateDirectory(t)
	status := updateDownloadStatus{State: "failed", Version: "v9.9.9", Error: "boom"}
	saveUpdateStatus(status)
	appendUpdateLog("first line")
	appendUpdateLog("second line")
	if err := os.WriteFile(filepath.Join(dir, "OmniProxy-Setup-test.exe"), []byte("installer"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "OmniProxy-Setup-test.exe.download"), []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}

	diagnostics := currentUpdateDiagnostics(status)
	if diagnostics.Directory != dir || !diagnostics.StatusExists || !diagnostics.LogExists {
		t.Fatalf("unexpected diagnostics paths: %#v", diagnostics)
	}
	if diagnostics.Status.State != "failed" || diagnostics.Status.Error != "boom" {
		t.Fatalf("unexpected diagnostics status: %#v", diagnostics.Status)
	}
	if !strings.Contains(diagnostics.LogTail, "second line") {
		t.Fatalf("expected log tail to include latest line, got %q", diagnostics.LogTail)
	}
	if diagnostics.InstallerCount != 1 || diagnostics.PartialCount != 1 {
		t.Fatalf("unexpected file counts: %#v", diagnostics)
	}
}

func TestHandleUpdateDiagnostics(t *testing.T) {
	useTestUpdateDirectory(t)
	server := &appServer{
		updates:      &updateDownloader{status: updateDownloadStatus{State: "idle"}},
		controlToken: "test-control-token",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/update/diagnostics", nil)
	req.Header.Set(controlTokenHeader, "test-control-token")
	res := httptest.NewRecorder()
	server.routes().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", res.Code, res.Body.String())
	}
	var payload updateDiagnostics
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Status.State != "idle" {
		t.Fatalf("unexpected diagnostics response: %#v", payload)
	}
}

func TestCleanupUpdateDirectoryRemovesOldInstallersAndPartials(t *testing.T) {
	dir := useTestUpdateDirectory(t)
	preservePath := filepath.Join(dir, "OmniProxy-Setup-preserve.exe")
	paths := []string{
		filepath.Join(dir, "OmniProxy-Setup-old.exe"),
		filepath.Join(dir, "OmniProxy-Setup-mid.exe"),
		filepath.Join(dir, "OmniProxy-Setup-new.exe"),
		preservePath,
		filepath.Join(dir, "OmniProxy-Setup-partial.exe.download"),
	}
	for index, path := range paths {
		if err := os.WriteFile(path, []byte("fake installer"), 0o600); err != nil {
			t.Fatal(err)
		}
		modTime := time.Now().Add(time.Duration(index) * time.Minute)
		if err := os.Chtimes(path, modTime, modTime); err != nil {
			t.Fatal(err)
		}
	}

	cleanupUpdateDirectory(preservePath)

	if _, err := os.Stat(paths[0]); !os.IsNotExist(err) {
		t.Fatalf("expected oldest installer to be removed, stat err=%v", err)
	}
	if _, err := os.Stat(paths[1]); err != nil {
		t.Fatalf("expected recent installer to be kept: %v", err)
	}
	if _, err := os.Stat(paths[2]); err != nil {
		t.Fatalf("expected recent installer to be kept: %v", err)
	}
	if _, err := os.Stat(preservePath); err != nil {
		t.Fatalf("expected active installer to be preserved: %v", err)
	}
	if _, err := os.Stat(paths[4]); !os.IsNotExist(err) {
		t.Fatalf("expected partial download to be removed, stat err=%v", err)
	}
}

func TestUpdateInstallerFileNameValidatesPlatformExtensions(t *testing.T) {
	if name, err := updateInstallerFileNameForPlatform(updateDownloadRequest{FileName: "OmniProxy-v1.2.3-darwin-universal.dmg"}, "darwin"); err != nil || name != "OmniProxy-v1.2.3-darwin-universal.dmg" {
		t.Fatalf("expected darwin dmg to be accepted, name=%q err=%v", name, err)
	}
	if _, err := updateInstallerFileNameForPlatform(updateDownloadRequest{FileName: "OmniProxy-v1.2.3-darwin-universal.dmg"}, "windows"); err == nil {
		t.Fatal("expected windows updater to reject dmg installer")
	}
}

func TestCheckForUpdatesSkipsDevelopmentVersion(t *testing.T) {
	restore := overrideUpdateGlobals("dev", "")
	defer restore()

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	latestReleaseURL = server.URL

	info, err := checkForUpdates(context.Background(), server.Client(), false)
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("development builds should not call the release API")
	}
	if info.UpdateAvailable || info.CurrentVersion != "dev" {
		t.Fatalf("unexpected development update info: %#v", info)
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		left  string
		right string
		want  int
	}{
		{left: "v1.0.3", right: "v1.0.2", want: 1},
		{left: "v1.0.10", right: "v1.0.2", want: 1},
		{left: "v1.0.2", right: "v1.0.2", want: 0},
		{left: "v1.0.2", right: "v1.0.3", want: -1},
		{left: "v1.0.9", right: "v1.0.9-beta.1", want: 1},
		{left: "v1.0.9-beta.2", right: "v1.0.9-beta.1", want: 1},
		{left: "v1.0.9-beta.10", right: "v1.0.9-beta.9", want: 1},
		{left: "v1.0.9-beta.2", right: "v1.0.9-beta.10", want: -1},
		{left: "v1.0.9-beta.1.1", right: "v1.0.9-beta.1", want: 1},
		{left: "v1.0.9-rc.1", right: "v1.0.9-beta.10", want: 1},
		{left: "dev", right: "v1.0.3", want: 0},
	}
	for _, tt := range tests {
		if got := compareVersions(tt.left, tt.right); got != tt.want {
			t.Fatalf("compareVersions(%q, %q) = %d, want %d", tt.left, tt.right, got, tt.want)
		}
	}
}

func overrideUpdateGlobals(version string, releaseURL string) func() {
	oldVersion := appVersion
	oldReleaseURL := latestReleaseURL
	oldReleasesURL := releasesURL
	appVersion = version
	if releaseURL != "" {
		latestReleaseURL = releaseURL
	}
	return func() {
		appVersion = oldVersion
		latestReleaseURL = oldReleaseURL
		releasesURL = oldReleasesURL
	}
}

func useTestUpdateDirectory(t *testing.T) string {
	t.Helper()
	oldTempDir := updateTempDir
	dir := t.TempDir()
	updateTempDir = func() string {
		return dir
	}
	t.Cleanup(func() {
		updateTempDir = oldTempDir
	})
	return dir
}

func TestValidateUpdateDownloadURLRejectsUntrustedSources(t *testing.T) {
	allowed := []string{
		"https://github.com/mibgb65-cloud/OmniProxy/releases/download/v1.2.7/OmniProxy-Setup.exe",
		"https://objects.githubusercontent.com/installer.exe",
		"https://GitHub.com/installer.exe",
	}
	for _, raw := range allowed {
		if err := validateUpdateDownloadURL("download", raw); err != nil {
			t.Fatalf("expected %q to be allowed: %v", raw, err)
		}
	}

	rejected := []string{
		"http://github.com/installer.exe",
		"https://evil.com/installer.exe",
		"https://github.com.evil.com/installer.exe",
		"file:///C:/Windows/System32/calc.exe",
		"",
	}
	for _, raw := range rejected {
		if err := validateUpdateDownloadURL("download", raw); err == nil {
			t.Fatalf("expected %q to be rejected", raw)
		}
	}
}

func allowLoopbackUpdateHost(t *testing.T) {
	t.Helper()
	oldHosts := updateDownloadAllowedHosts
	updateDownloadAllowedHosts = []string{"127.0.0.1"}
	t.Cleanup(func() {
		updateDownloadAllowedHosts = oldHosts
	})
}

func expectedUpdateAsset(version string, stem string) (string, string, string, int64) {
	if runtime.GOOS == "darwin" {
		name := fmt.Sprintf("OmniProxy-%s-darwin-universal.dmg", version)
		return name, fmt.Sprintf("https://example.com/%s.dmg", stem), fmt.Sprintf("https://example.com/%s.dmg.sha256", stem), 456
	}
	name := fmt.Sprintf("OmniProxy-Setup-%s-windows-amd64.exe", version)
	return name, fmt.Sprintf("https://example.com/%s.exe", stem), fmt.Sprintf("https://example.com/%s.exe.sha256", stem), 123
}

func testInstallerFileName() string {
	if runtime.GOOS == "darwin" {
		return fmt.Sprintf("OmniProxy-test-%d.dmg", time.Now().UnixNano())
	}
	return fmt.Sprintf("OmniProxy-Setup-test-%d.exe", time.Now().UnixNano())
}
