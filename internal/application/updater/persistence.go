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
