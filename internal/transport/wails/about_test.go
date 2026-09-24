package wails

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"omniproxy/internal/infrastructure/update"
	"omniproxy/internal/platform/profile"
	"omniproxy/internal/version"
)

func TestRuntimeInfo(t *testing.T) {
	svc, prof := newTestService(t)

	info := svc.RuntimeInfo()
	if info.Version != version.Version {
		t.Errorf("Version = %q, want %q", info.Version, version.Version)
	}
	if info.AppName != "OmniProxy" || info.Description == "" {
		t.Errorf("app identity = %q / %q", info.AppName, info.Description)
	}
	if info.Profile != string(profile.Test) {
		t.Errorf("Profile = %q, want %q", info.Profile, profile.Test)
	}
	if info.DataDir != prof.DataDir || info.LogDir != prof.LogDir {
		t.Errorf("dirs = %q / %q", info.DataDir, info.LogDir)
	}
	if info.GoVersion == "" || info.GoVersion[:2] != "go" {
		t.Errorf("GoVersion = %q, want a go* version", info.GoVersion)
	}
	if info.WailsVersion == "" {
		t.Errorf("WailsVersion = %q, want non-empty", info.WailsVersion)
	}
	// Note: test binaries carry no module dependency info, so the value here
	// may legitimately be "unknown". A built binary reports the real version
	// from debug.ReadBuildInfo (verified against go.mod).
	if info.Platform == "" {
		t.Error("Platform should not be empty")
	}
	if info.SQLiteVersion == "" {
		t.Error("SQLiteVersion should not be empty")
	}
	started, err := time.Parse(time.RFC3339, info.StartedAt)
	if err != nil {
		t.Fatalf("StartedAt %q is not RFC3339: %v", info.StartedAt, err)
	}
	if time.Since(started) > time.Minute {
		t.Errorf("StartedAt = %v, want recent", started)
	}
}

func TestCheckForUpdateUnconfigured(t *testing.T) {
	svc, _ := newTestService(t)

	status := svc.CheckForUpdate()
	if status.State != updateStateUnconfigured {
		t.Fatalf("State = %q, want %q", status.State, updateStateUnconfigured)
	}
	if status.CurrentVersion != version.Version {
		t.Errorf("CurrentVersion = %q, want %q", status.CurrentVersion, version.Version)
	}
	if status.Message == "" {
		t.Error("unconfigured state must explain why")
	}
}

func TestCheckForUpdateAvailable(t *testing.T) {
	svc, _ := newTestService(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v99.0.0","html_url":"https://example.com/release","published_at":"2026-09-01T10:00:00Z"}`))
	}))
	defer server.Close()

	original := newUpdateChecker
	newUpdateChecker = func(repo string) *update.Checker {
		return &update.Checker{Repo: repo, BaseURL: server.URL}
	}
	defer func() { newUpdateChecker = original }()

	if err := profile.SaveConfig(profile.Test, profile.Config{UpdateRepo: "owner/name"}); err != nil {
		t.Fatal(err)
	}

	status := svc.CheckForUpdate()
	if status.State != updateStateAvailable {
		t.Fatalf("State = %q, want %q", status.State, updateStateAvailable)
	}
	if status.LatestVersion != "v99.0.0" {
		t.Errorf("LatestVersion = %q, want v99.0.0", status.LatestVersion)
	}
	if status.ReleaseURL != "https://example.com/release" {
		t.Errorf("ReleaseURL = %q", status.ReleaseURL)
	}
}

func TestCheckForUpdateUpToDate(t *testing.T) {
	svc, _ := newTestService(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v0.0.1","html_url":"https://example.com/release","published_at":"2026-09-01T10:00:00Z"}`))
	}))
	defer server.Close()

	original := newUpdateChecker
	newUpdateChecker = func(repo string) *update.Checker {
		return &update.Checker{Repo: repo, BaseURL: server.URL}
	}
	defer func() { newUpdateChecker = original }()

	if err := profile.SaveConfig(profile.Test, profile.Config{UpdateRepo: "owner/name"}); err != nil {
		t.Fatal(err)
	}

	status := svc.CheckForUpdate()
	if status.State != updateStateUpToDate {
		t.Fatalf("State = %q, want %q", status.State, updateStateUpToDate)
	}
}

func TestCheckForUpdateNetworkError(t *testing.T) {
	svc, _ := newTestService(t)

	original := newUpdateChecker
	newUpdateChecker = func(repo string) *update.Checker {
		return &update.Checker{Repo: repo, BaseURL: "http://127.0.0.1:1"}
	}
	defer func() { newUpdateChecker = original }()

	if err := profile.SaveConfig(profile.Test, profile.Config{UpdateRepo: "owner/name"}); err != nil {
		t.Fatal(err)
	}

	status := svc.CheckForUpdate()
	if status.State != updateStateError {
		t.Fatalf("State = %q, want %q", status.State, updateStateError)
	}
	if status.Message == "" {
		t.Error("error state must carry the failure reason")
	}
}
