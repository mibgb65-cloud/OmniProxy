// Package updater 管理应用自身更新，前端只操作已发现的版本，不传执行路径。
package updater

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"omniproxy/internal/infrastructure/update"
	"omniproxy/internal/version"
)

// ReleaseInfo 是版本浮窗需要的公开信息。
type ReleaseInfo struct {
	Version          string `json:"version"`
	Notes            string `json:"notes"`
	URL              string `json:"url"`
	PublishedAt      string `json:"publishedAt"`
	PackageAvailable bool   `json:"packageAvailable"`
	Size             int64  `json:"size"`
}

// Transfer 不返回缓存路径、校验摘要或内部网络地址。
type Transfer struct {
	State    string `json:"state"`
	Version  string `json:"version"`
	Received int64  `json:"received"`
	Total    int64  `json:"total"`
	Percent  int    `json:"percent"`
	Error    string `json:"error"`
}

// Installed 表示「刚刚升级到这个运行版本」，只在用户确认前出现一次。
// From 为空时不宣称知道升级前的版本。
type Installed struct {
	Version string `json:"version"`
	From    string `json:"from"`
}

// Snapshot 是侧栏、浮窗和设置页共用的真实状态。
type Snapshot struct {
	CurrentVersion string       `json:"currentVersion"`
	CheckState     string       `json:"checkState"`
	CheckError     string       `json:"checkError"`
	Release        *ReleaseInfo `json:"release,omitempty"`
	Download       Transfer     `json:"download"`
	Installed      *Installed   `json:"installed,omitempty"`
	CanInstall     bool         `json:"canInstall"`
}

// Options 注入网络和系统执行边界，测试不能修改正式默认白名单。
type Options struct {
	Directory, Version, OS, Arch string
	Client                       *http.Client
	Checker                      func(repo string, beta bool) *update.Checker
	ValidPackage                 func(update.Package) bool
	ValidRedirect                func(string) bool
	Launch                       func(path, digest, receipt, version string, size int64) error
	CanInstall                   bool
}

type record struct {
	Package  update.Package `json:"package"`
	File     string         `json:"file"`
	Digest   string         `json:"digest"`
	Transfer Transfer       `json:"transfer"`
	// Previous 是发起安装时正在运行的版本，只用于「从 X 升级到 Y」的说明。
	Previous string `json:"previous,omitempty"`
	// NoticeSeen 是用户已经确认过「更新完成」的运行版本，空表示尚未提示。
	NoticeSeen string `json:"noticeSeen,omitempty"`
}

// Service 串行化检查、下载与安装，下载进度仅在内存聚合。
type Service struct {
	mu       sync.Mutex
	opts     Options
	state    Snapshot
	selected *update.Package
	record   record
	repo     string
	beta     bool
	cancel   context.CancelFunc
	closed   bool
	wg       sync.WaitGroup
}

// New 恢复当前 profile 的缓存，不会自动联网。
func New(opts Options) *Service {
	if opts.Checker == nil {
		opts.Checker = func(repo string, beta bool) *update.Checker {
			c := update.NewChecker(repo)
			c.IncludePrereleases = beta
			return c
		}
	}
	if opts.ValidPackage == nil {
		opts.ValidPackage = func(p update.Package) bool {
			return update.ValidAssetURL(p.URL, p, false) && update.ValidAssetURL(p.ChecksumURL, p, true)
		}
	}
	if opts.ValidRedirect == nil {
		opts.ValidRedirect = update.ValidRedirectURL
	}
	opts.Client = update.DownloadClient(opts.Client, opts.ValidRedirect)
	s := &Service{opts: opts, state: Snapshot{CurrentVersion: opts.Version, CheckState: "idle", Download: Transfer{State: "idle"}, CanInstall: opts.CanInstall}}
	s.restore()
	return s
}

// Snapshot 返回独立副本，避免 UI 或调用方修改服务内状态。
func (s *Service) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked()
}

// snapshotLocked 供已持锁的调用方使用；重复加锁会自锁。
func (s *Service) snapshotLocked() Snapshot {
	result := s.state
	if result.Release != nil {
		copy := *result.Release
		result.Release = &copy
	}
	if result.Installed != nil {
		copy := *result.Installed
		result.Installed = &copy
	}
	return result
}

// Busy 用于阻止检查、下载／安装期间切换更新渠道。
func (s *Service) Busy() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.busyLocked() || s.state.CheckState == "checking"
}
func (s *Service) busyLocked() bool {
	switch s.state.Download.State {
	case "downloading", "verifying", "installing":
		return true
	}
	return s.state.CheckState == "checking"
}

// Configure 防止上个渠道或仓库的候选被用于另一个更新选择。
func (s *Service) Configure(repo string, beta bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("update: closed")
	}
	if s.repo == repo && s.beta == beta {
		return nil
	}
	if s.busyLocked() || s.state.CheckState == "checking" {
		return errors.New("update: busy")
	}
	s.repo, s.beta = repo, beta
	s.selected = nil
	s.state.Release = nil
	s.state.CheckState = "idle"
	s.state.CheckError = ""
	// 已完成的升级提示属于运行版本，不随更新渠道切换消失。
	if s.record.Package.Repo == repo && s.record.Transfer.State == "ready" && (beta || !version.IsPrerelease(s.record.Package.Version)) {
		s.state.Download = s.record.Transfer
		p := s.record.Package
		s.selected = &p
		s.state.Release = &ReleaseInfo{Version: p.Version, PackageAvailable: true, Size: p.Size, URL: "https://github.com/" + p.Repo + "/releases/tag/" + p.Version}
		s.state.CheckState = "available"
	} else if s.state.Download.State == "ready" {
		s.state.Download = Transfer{State: "idle"}
	}
	return nil
}

// Close 取消网络任务；已经启动的独立安装辅助进程继续完成升级。
func (s *Service) Close(ctx context.Context) error {
	s.mu.Lock()
	s.closed = true
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
