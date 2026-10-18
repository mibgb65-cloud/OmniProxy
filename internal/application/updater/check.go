package updater

import (
	"context"
	"errors"
	"strings"
	"time"

	"omniproxy/internal/infrastructure/update"
	"omniproxy/internal/version"
)

// Check 按当前渠道发现版本并将安装资产冻结在服务端。
func (s *Service) Check(ctx context.Context) (Snapshot, error) {
	s.mu.Lock()
	if s.closed || s.busyLocked() || s.state.CheckState == "checking" {
		s.mu.Unlock()
		return s.Snapshot(), errors.New("update: busy")
	}
	repo, beta := s.repo, s.beta
	s.state.CheckState = "checking"
	s.state.CheckError = ""
	s.mu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	release, err := s.opts.Checker(repo, beta).Latest(ctx)
	s.mu.Lock()
	if err != nil {
		s.state.CheckState = "error"
		s.state.CheckError = "update: check_failed"
	} else {
		s.state.CheckState = "up-to-date"
		s.state.Release = nil
		s.selected = nil
		if s.record.Transfer.State == "ready" && (s.record.Package.Version != release.TagName || version.Compare(release.TagName, s.opts.Version) <= 0) {
			s.record.Transfer.State = "superseded"
			s.state.Download = Transfer{State: "idle"}
			if saveErr := s.saveLocked(); saveErr != nil {
				s.state.CheckError = saveErr.Error()
			}
		}
		if version.Compare(release.TagName, s.opts.Version) > 0 {
			s.state.CheckState = "available"
			notes := []rune(strings.TrimSpace(release.Body))
			if len(notes) > 12000 {
				notes = notes[:12000]
			}
			info := &ReleaseInfo{Version: release.TagName, Notes: string(notes), URL: release.HTMLURL}
			if !release.PublishedAt.IsZero() {
				info.PublishedAt = release.PublishedAt.UTC().Format(time.RFC3339)
			}
			p, packageErr := update.SelectPackage(release, repo, s.opts.OS, s.opts.Arch)
			if packageErr == nil && s.opts.ValidPackage(p) {
				info.PackageAvailable = true
				info.Size = p.Size
				s.selected = &p
			}
			s.state.Release = info
		}
	}
	s.mu.Unlock()
	return s.Snapshot(), err
}
