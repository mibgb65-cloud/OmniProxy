// Package update checks GitHub Releases for newer OmniProxy versions.
//
// 发布发现与下载网络边界由本包实现，更新状态由 application/updater 管理。
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// defaultBaseURL is the public GitHub API root.
const defaultBaseURL = "https://api.github.com"

// DefaultRepo 是本项目正式发布仓库；本地无须额外配置即可检查更新。
const DefaultRepo = "mibgb65-cloud/OmniProxy"

// Release is the subset of the GitHub release payload we need.
type Release struct {
	Assets      []Asset   `json:"assets"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	TagName     string    `json:"tag_name"`
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
	Body        string    `json:"body"`
}

// Checker queries the latest release of a GitHub repository.
type Checker struct {
	IncludePrereleases bool
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

// Latest 按所选渠道查询；Beta 使用列表而不是只含正式版的 latest 端点。
func (c *Checker) Latest(ctx context.Context) (Release, error) {
	if c.IncludePrereleases {
		return c.latestCandidate(ctx)
	}
	repo := strings.TrimSpace(c.Repo)
	if !validRepo(repo) {
		return Release{}, fmt.Errorf("update: invalid repository")
	}
	base := c.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	// repo 已逐段限制为 ASCII 名称；保留 owner/repo 的路由分隔符。
	endpoint := fmt.Sprintf("%s/repos/%s/releases/latest", strings.TrimRight(base, "/"), repo)

	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	copyClient := *client
	copyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Release{}, fmt.Errorf("update: build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "OmniProxy")

	resp, err := copyClient.Do(req)
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

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil || len(data) > maxResponseBytes {
		return Release{}, fmt.Errorf("update: invalid release response")
	}
	var release Release
	if err := json.Unmarshal(data, &release); err != nil {
		return Release{}, fmt.Errorf("update: invalid release response")
	}
	if !candidate(release, false) {
		return Release{}, fmt.Errorf("update: invalid stable release")
	}
	release.HTMLURL = releaseURL(repo, release.TagName)
	return release, nil
}
