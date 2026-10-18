package update

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"omniproxy/internal/version"
)

// Asset 是 GitHub 返回的发布资产元数据，不接受前端覆盖。
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

// Package 配对当前平台安装器及其 SHA256 文件。
type Package struct {
	Version     string `json:"version"`
	Repo        string `json:"repo"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	ChecksumURL string `json:"checksumUrl"`
	Size        int64  `json:"size"`
}

// InstallerName 与发布工作流共用同一命名约定。
func InstallerName(tag, goos, arch string) (string, error) {
	if !version.Valid(tag) || tag != strings.TrimSpace(tag) || goos != "windows" || arch != "amd64" {
		return "", errors.New("update: unsupported_package")
	}
	return fmt.Sprintf("OmniProxy-Setup-v%s-windows-%s.exe", strings.TrimPrefix(tag, "v"), arch), nil
}

// SelectPackage 必须且只能有一个匹配安装器及校验文件。
func SelectPackage(release Release, repo, goos, arch string) (Package, error) {
	name, err := InstallerName(release.TagName, goos, arch)
	if err != nil || !validRepo(repo) {
		return Package{}, errors.New("update: package_unavailable")
	}
	var binary, checksum *Asset
	for i := range release.Assets {
		asset := &release.Assets[i]
		if asset.Name == name {
			if binary != nil {
				return Package{}, errors.New("update: ambiguous_package")
			}
			binary = asset
		}
		if asset.Name == name+".sha256" {
			if checksum != nil {
				return Package{}, errors.New("update: ambiguous_package")
			}
			checksum = asset
		}
	}
	if binary == nil || checksum == nil || binary.Size < 1 || binary.Size > MaxInstallerSize || checksum.Size < 1 || checksum.Size > 4096 {
		return Package{}, errors.New("update: package_unavailable")
	}
	return Package{Version: release.TagName, Repo: repo, Name: name, URL: binary.URL, ChecksumURL: checksum.URL, Size: binary.Size}, nil
}

// ValidAssetURL 仅接受对应仓库、版本和文件的 GitHub 发布地址。
func ValidAssetURL(raw string, p Package, checksum bool) bool {
	expected, err := InstallerName(p.Version, "windows", "amd64")
	if len(raw) > 4096 || err != nil || p.Name != expected || !validRepo(p.Repo) {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.Fragment != "" || u.RawQuery != "" {
		return false
	}
	name := p.Name
	if checksum {
		name += ".sha256"
	}
	return u.Path == "/"+p.Repo+"/releases/download/"+p.Version+"/"+name
}

// ValidRedirectURL 允许 GitHub 实际资产 CDN，拒绝降级和未知重定向。
func ValidRedirectURL(raw string) bool {
	if len(raw) > 16384 {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Fragment != "" || u.Port() != "" {
		return false
	}
	switch u.Host {
	case "github.com", "objects.githubusercontent.com", "release-assets.githubusercontent.com":
		return true
	}
	return false
}
