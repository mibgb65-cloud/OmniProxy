package updater

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"

	"omniproxy/internal/infrastructure/appinstall"
	"omniproxy/internal/infrastructure/update"
	"omniproxy/internal/version"
)

var packageFile = regexp.MustCompile(`^package-[0-9a-f]{32}\.exe$`)

func (s *Service) saveLocked() error {
	if !filepath.IsAbs(s.opts.Directory) || os.MkdirAll(s.opts.Directory, 0700) != nil {
		return errors.New("update: storage_failed")
	}
	data, err := json.Marshal(s.record)
	if err != nil {
		return errors.New("update: storage_failed")
	}
	path := filepath.Join(s.opts.Directory, "status.json")
	tmp := path + ".tmp"
	if os.WriteFile(tmp, data, 0600) != nil {
		return errors.New("update: storage_failed")
	}
	if os.Rename(tmp, path) != nil {
		return errors.New("update: storage_failed")
	}
	return nil
}

// AcknowledgeInstalled 记录用户已经看过「更新完成」提示；不删除已下载的安装包
// 或安装结果，写入失败时提示会在下次启动重新出现而不是假装已确认。
func (s *Service) AcknowledgeInstalled() (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	notice := s.state.Installed
	if notice == nil {
		return s.snapshotLocked(), nil
	}
	kept := *notice
	previous := s.record.NoticeSeen
	s.record.NoticeSeen = kept.Version
	s.state.Installed = nil
	if err := s.saveLocked(); err != nil {
		s.record.NoticeSeen = previous
		s.state.Installed = &kept
		s.state.Download.State = "installed"
		return s.snapshotLocked(), err
	}
	return s.snapshotLocked(), nil
}

func (s *Service) restore() {
	path := filepath.Join(s.opts.Directory, "status.json")
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return
	}
	if err != nil || !info.Mode().IsRegular() || info.Size() > 131072 {
		s.state.Download = Transfer{State: "failed", Error: "update: invalid_cache"}
		return
	}
	data, err := os.ReadFile(path)
	var r record
	if err != nil || json.Unmarshal(data, &r) != nil || !packageFile.MatchString(r.File) || !s.opts.ValidPackage(r.Package) {
		s.state.Download = Transfer{State: "failed", Error: "update: invalid_cache"}
		return
	}
	name, err := update.InstallerName(r.Package.Version, s.opts.OS, s.opts.Arch)
	if err != nil || name != r.Package.Name || r.Package.Size < 1 || r.Package.Size > update.MaxInstallerSize {
		return
	}
	s.record = r
	s.state.Download = r.Transfer
	if r.Transfer.State == "installing" {
		data, err := os.ReadFile(filepath.Join(s.opts.Directory, "install-result.json"))
		var receipt appinstall.Receipt
		if err == nil && len(data) < 1024 && json.Unmarshal(data, &receipt) == nil && receipt.Version == r.Package.Version && !receipt.Success {
			s.state.Download.State = "failed"
			s.state.Download.Error = "update: install_failed"
			return
		}
	}
	if version.Compare(s.opts.Version, r.Package.Version) >= 0 {
		s.state.Download = Transfer{State: "installed", Version: r.Package.Version, Percent: 100}
		// 只有安装确实生效且用户还没确认过时，才宣称「刚刚升级到这个版本」。
		// 发布标签带 v 前缀而运行版本不带，来源说明只在两者指向同一版本时给出。
		if s.record.NoticeSeen != s.opts.Version {
			from := ""
			if version.Compare(r.Package.Version, s.opts.Version) == 0 {
				from = r.Previous
			}
			s.state.Installed = &Installed{Version: s.opts.Version, From: from}
		}
		return
	}
	switch r.Transfer.State {
	case "ready", "installing":
		if update.VerifyFile(filepath.Join(s.opts.Directory, r.File), r.Digest, r.Package.Size) != nil {
			s.state.Download = Transfer{State: "failed", Error: "update: package_unavailable"}
			return
		}
		s.state.Download.State = "ready"
	case "downloading", "verifying":
		s.state.Download.State = "failed"
		s.state.Download.Error = "update: interrupted"
	}
}
