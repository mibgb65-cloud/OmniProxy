//go:build updateqa

package wails

import (
	"net"
	"net/url"
	"os"

	"omniproxy/internal/application/updater"
	"omniproxy/internal/infrastructure/update"
	"omniproxy/internal/platform/profile"
)

// 仅隔离验收编译副本可使用回环发布服务，正式构建不包含此入口。
func configureUpdateQA(opts *updater.Options, prof *profile.Profile) {
	if prof.Kind != profile.Test {
		return
	}
	base := os.Getenv("OMNIPROXY_QA_UPDATE_API")
	u, err := url.Parse(base)
	if err != nil || u.Scheme != "http" || net.ParseIP(u.Hostname()) == nil || !net.ParseIP(u.Hostname()).IsLoopback() || u.User != nil || u.Path != "" || u.RawQuery != "" {
		return
	}
	opts.Checker = func(repo string, beta bool) *update.Checker {
		return &update.Checker{Repo: repo, IncludePrereleases: beta, BaseURL: base}
	}
	opts.ValidPackage = func(p update.Package) bool {
		return p.URL == base+"/assets/"+p.Name && p.ChecksumURL == p.URL+".sha256"
	}
	opts.ValidRedirect = func(raw string) bool {
		target, err := url.Parse(raw)
		return err == nil && target.Scheme == "http" && target.Host == u.Host && target.User == nil
	}
}
