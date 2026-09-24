// Package update checks GitHub Releases for newer OmniProxy versions.
//
// It implements the read-only half of the update architecture from the
// technology baseline (GitHub Releases + Wails Updater): discovering the
// latest release. Downloading and installing updates is a separate feature.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// defaultBaseURL is the public GitHub API root.
const defaultBaseURL = "https://api.github.com"

// Release is the subset of the GitHub release payload we need.
type Release struct {
	TagName     string    `json:"tag_name"`
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
	Body        string    `json:"body"`
}

// Checker queries the latest release of a GitHub repository.
type Checker struct {
	// Repo is "owner/name", e.g. "mimanchi/OmniProxy".
	Repo string
	// BaseURL overrides the API root (tests inject a local server).
	BaseURL string
	// Client defaults to a 10-second-timeout HTTP client.
	Client *http.Client
}

// NewChecker creates a checker for owner/repo using the public GitHub API.
func NewChecker(repo string) *Checker {
	return &Checker{Repo: repo}
}

// Latest returns the latest published release. Prereleases and drafts are
// reported by the API's /releases/latest endpoint only when they are marked
// as the latest release; no extra filtering is applied here.
func (c *Checker) Latest(ctx context.Context) (Release, error) {
	repo := strings.TrimSpace(c.Repo)
	if repo == "" {
		return Release{}, fmt.Errorf("update: repository is not configured")
	}
	base := c.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	endpoint := fmt.Sprintf("%s/repos/%s/releases/latest", strings.TrimRight(base, "/"), url.PathEscape(repo))

	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Release{}, fmt.Errorf("update: build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "OmniProxy")

	resp, err := client.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("update: request %s: %w", endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return Release{}, fmt.Errorf("update: no releases found for %s", repo)
	}
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("update: github api returned %s", resp.Status)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return Release{}, fmt.Errorf("update: decode release: %w", err)
	}
	if release.TagName == "" {
		return Release{}, fmt.Errorf("update: release for %s has no tag", repo)
	}
	return release, nil
}
