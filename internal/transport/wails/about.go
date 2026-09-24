package wails

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"time"

	"omniproxy/internal/infrastructure/update"
	"omniproxy/internal/platform/profile"
	"omniproxy/internal/version"
)

// Update check states returned by AppService.CheckForUpdate.
const (
	updateStateUpToDate     = "up-to-date"
	updateStateAvailable    = "available"
	updateStateUnconfigured = "unconfigured"
	updateStateError        = "error"
	updateCheckTimeout      = 15 * time.Second
	appName                 = "OmniProxy"
	appDescription          = "Local AI API gateway and multi-account management tool"
	wailsModulePath         = "github.com/wailsapp/wails/v3"
)

// newUpdateChecker builds the release checker for a repository. It is a
// package variable so tests can point it at a local server.
var newUpdateChecker = func(repo string) *update.Checker {
	return update.NewChecker(repo)
}

// RuntimeInfo is a snapshot of the application runtime for the About page.
type RuntimeInfo struct {
	Version       string `json:"version"`
	AppName       string `json:"appName"`
	Description   string `json:"description"`
	AppID         string `json:"appId"`
	Profile       string `json:"profile"`
	DataDir       string `json:"dataDir"`
	LogDir        string `json:"logDir"`
	ProxyPort     int    `json:"proxyPort"`
	ControlPort   int    `json:"controlPort"`
	GoVersion     string `json:"goVersion"`
	WailsVersion  string `json:"wailsVersion"`
	Platform      string `json:"platform"`
	SQLiteVersion string `json:"sqliteVersion"`
	StartedAt     string `json:"startedAt"`
}

// RuntimeInfo returns the current application runtime information.
func (s *AppService) RuntimeInfo() RuntimeInfo {
	return RuntimeInfo{
		Version:       version.Version,
		AppName:       appName,
		Description:   appDescription,
		AppID:         s.prof.AppID,
		Profile:       string(s.prof.Kind),
		DataDir:       s.prof.DataDir,
		LogDir:        s.prof.LogDir,
		ProxyPort:     s.prof.ProxyPort,
		ControlPort:   s.prof.ControlPort,
		GoVersion:     runtime.Version(),
		WailsVersion:  wailsVersion(),
		Platform:      fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		SQLiteVersion: s.sqliteVersion(),
		StartedAt:     s.startedAt.UTC().Format(time.RFC3339),
	}
}

// UpdateStatus reports the result of an update check.
type UpdateStatus struct {
	State          string `json:"state"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	ReleaseURL     string `json:"releaseUrl"`
	PublishedAt    string `json:"publishedAt"`
	Message        string `json:"message"`
}

// CheckForUpdate compares the running version against the latest GitHub
// release of the configured repository. When no repository is configured it
// returns the "unconfigured" state instead of pretending to check.
func (s *AppService) CheckForUpdate() UpdateStatus {
	cfg, err := profile.LoadConfig(s.prof.Kind)
	if err != nil {
		return UpdateStatus{
			State:          updateStateError,
			CurrentVersion: version.Version,
			Message:        err.Error(),
		}
	}

	checker := newUpdateChecker(cfg.UpdateRepo)
	ctx, cancel := context.WithTimeout(context.Background(), updateCheckTimeout)
	defer cancel()

	release, err := checker.Latest(ctx)
	if err != nil {
		state := updateStateError
		if cfg.UpdateRepo == "" {
			state = updateStateUnconfigured
		}
		return UpdateStatus{
			State:          state,
			CurrentVersion: version.Version,
			Message:        err.Error(),
		}
	}

	status := UpdateStatus{
		State:          updateStateUpToDate,
		CurrentVersion: version.Version,
		LatestVersion:  release.TagName,
		ReleaseURL:     release.HTMLURL,
		PublishedAt:    release.PublishedAt.UTC().Format(time.RFC3339),
	}
	if version.Compare(release.TagName, version.Version) > 0 {
		status.State = updateStateAvailable
	}
	return status
}

// sqliteVersion returns the SQLite library version, or empty on failure.
func (s *AppService) sqliteVersion() string {
	if s.db == nil {
		return ""
	}
	var v string
	if err := s.db.QueryRowContext(context.Background(), `SELECT sqlite_version()`).Scan(&v); err != nil {
		s.log.Warn("failed to read sqlite version", "error", err)
		return ""
	}
	return v
}

// wailsVersion reports the Wails framework version from build info, so it
// never drifts from go.mod.
func wailsVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, dep := range info.Deps {
		if dep.Path == wailsModulePath {
			return dep.Version
		}
	}
	return "unknown"
}
