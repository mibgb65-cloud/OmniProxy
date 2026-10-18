package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"omniproxy/internal/version"
)

const maxResponseBytes = 4 << 20

func validRepo(repo string) bool {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || len(repo) > 201 {
		return false
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return false
		}
		for _, c := range part {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-' || c == '.') {
				return false
			}
		}
	}
	return true
}

func releaseURL(repo, tag string) string {
	return "https://github.com/" + repo + "/releases/tag/" + url.PathEscape(tag)
}

func candidate(release Release, beta bool) bool {
	return !release.Draft && version.Valid(release.TagName) &&
		(beta || !release.Prerelease && !version.IsPrerelease(release.TagName))
}

func (c *Checker) latestCandidate(ctx context.Context) (Release, error) {
	repo := strings.TrimSpace(c.Repo)
	if !validRepo(repo) {
		return Release{}, fmt.Errorf("update: invalid repository")
	}
	base := c.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	copyClient := *client
	copyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	var best Release
	// 不依赖发布时间或第一页顺序，旧分支补发也不能遮住更高的版本。
	for page := 1; page <= 10; page++ {
		endpoint := fmt.Sprintf("%s/repos/%s/releases?per_page=100&page=%d", strings.TrimRight(base, "/"), repo, page)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return Release{}, fmt.Errorf("update: build request failed")
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("User-Agent", "OmniProxy")
		resp, err := copyClient.Do(req)
		if err != nil {
			return Release{}, fmt.Errorf("update: release request failed")
		}
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
		closeErr := resp.Body.Close()
		if readErr != nil || closeErr != nil || len(data) > maxResponseBytes || resp.StatusCode != http.StatusOK {
			return Release{}, fmt.Errorf("update: release response failed (%d)", resp.StatusCode)
		}
		var releases []Release
		if json.Unmarshal(data, &releases) != nil || len(releases) > 100 {
			return Release{}, fmt.Errorf("update: invalid release response")
		}
		for _, release := range releases {
			if candidate(release, true) && (best.TagName == "" || version.Compare(release.TagName, best.TagName) > 0) {
				best = release
			}
		}
		if len(releases) < 100 {
			if best.TagName == "" {
				return Release{}, fmt.Errorf("update: no eligible releases")
			}
			best.HTMLURL = releaseURL(repo, best.TagName)
			return best, nil
		}
	}
	return Release{}, fmt.Errorf("update: release list limit exceeded")
}
