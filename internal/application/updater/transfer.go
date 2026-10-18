package updater

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"time"

	"omniproxy/internal/infrastructure/update"
	"omniproxy/internal/version"
)

// Start 仅使用后台已经确认的目标；旧浮窗不能下载另一个新版本。
func (s *Service) Start(expected string) (Snapshot, error) {
	s.mu.Lock()
	if s.closed || s.busyLocked() || s.selected == nil || s.selected.Version != expected || version.Compare(expected, s.opts.Version) <= 0 {
		s.mu.Unlock()
		return s.Snapshot(), errors.New("update: target_changed")
	}
	p := *s.selected
	if !s.opts.ValidPackage(p) {
		s.mu.Unlock()
		return s.Snapshot(), errors.New("update: invalid_package")
	}
	if packageFile.MatchString(s.record.File) {
		if err := os.Remove(filepath.Join(s.opts.Directory, s.record.File)); err != nil && !os.IsNotExist(err) {
			s.mu.Unlock()
			return s.Snapshot(), errors.New("update: cleanup_failed")
		}
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		s.mu.Unlock()
		return s.Snapshot(), errors.New("update: storage_failed")
	}
	file := "package-" + hex.EncodeToString(nonce[:]) + ".exe"
	s.record = record{Package: p, File: file, Transfer: Transfer{State: "downloading", Version: p.Version, Total: p.Size}}
	s.state.Download = s.record.Transfer
	if err := s.saveLocked(); err != nil {
		s.state.Download.State = "failed"
		s.state.Download.Error = err.Error()
		s.mu.Unlock()
		return s.Snapshot(), err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	s.cancel = cancel
	s.wg.Add(1)
	go s.download(ctx, p, file)
	s.mu.Unlock()
	return s.Snapshot(), nil
}

func (s *Service) download(ctx context.Context, p update.Package, file string) {
	defer s.wg.Done()
	path := filepath.Join(s.opts.Directory, file)
	digest, err := update.Download(ctx, s.opts.Client, p, path, func(n int64) {
		s.mu.Lock()
		if s.record.File == file && s.state.Download.State == "downloading" {
			s.state.Download.Received = n
			s.state.Download.Percent = int(n * 100 / p.Size)
		}
		s.mu.Unlock()
	})
	s.mu.Lock()
	defer s.mu.Unlock()
	wasCancelled := ctx.Err() != nil
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	if err != nil || wasCancelled {
		cleanupErr := os.Remove(path)
		s.state.Download.State = "failed"
		s.state.Download.Error = "update: download_failed"
		if err != nil {
			s.state.Download.Error = err.Error()
		}
		if wasCancelled {
			s.state.Download.State = "cancelled"
			s.state.Download.Error = ""
		}
		if cleanupErr != nil && !os.IsNotExist(cleanupErr) {
			s.state.Download.State = "failed"
			s.state.Download.Error = "update: cleanup_failed"
		}
	} else {
		s.state.Download.State = "ready"
		s.state.Download.Percent = 100
		s.record.Digest = digest
	}
	s.record.Transfer = s.state.Download
	if err := s.saveLocked(); err != nil {
		s.state.Download.State = "failed"
		s.state.Download.Error = err.Error()
	}
}

// Cancel 不伪装完成；后台负责关闭响应、清理本次文件并保存终态。
func (s *Service) Cancel() (Snapshot, error) {
	s.mu.Lock()
	if s.state.Download.State != "downloading" || s.cancel == nil {
		s.mu.Unlock()
		return s.Snapshot(), errors.New("update: not_downloading")
	}
	s.cancel()
	s.mu.Unlock()
	return s.Snapshot(), nil
}

// Install 复验文件后启动独立辅助程序；UAC 被取消时应用保持运行。
func (s *Service) Install(expected string) (Snapshot, error) {
	s.mu.Lock()
	if !s.opts.CanInstall || s.opts.Launch == nil || s.state.Download.State != "ready" || s.record.Package.Version != expected || s.record.Package.Repo != s.repo || !s.opts.ValidPackage(s.record.Package) {
		s.mu.Unlock()
		return s.Snapshot(), errors.New("update: install_unavailable")
	}
	s.state.Download.State = "verifying"
	r := s.record
	s.mu.Unlock()
	err := update.VerifyFile(filepath.Join(s.opts.Directory, r.File), r.Digest, r.Package.Size)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		// 缓存状态不是信任根；安装前重新向已校验的发布地址确认摘要。
		var remote string
		remote, err = update.Checksum(ctx, s.opts.Client, r.Package.ChecksumURL, r.Package.Name)
		cancel()
		if err == nil && remote != r.Digest {
			err = errors.New("update: checksum_mismatch")
		}
	}
	s.mu.Lock()
	if err != nil {
		s.state.Download.State = "failed"
		s.state.Download.Error = err.Error()
		s.record.Transfer = s.state.Download
		saveErr := s.saveLocked()
		s.mu.Unlock()
		return s.Snapshot(), errors.Join(err, saveErr)
	}
	s.state.Download.State = "installing"
	s.record.Transfer = s.state.Download
	err = s.saveLocked()
	s.mu.Unlock()
	if err == nil {
		err = s.opts.Launch(filepath.Join(s.opts.Directory, r.File), r.Digest, filepath.Join(s.opts.Directory, "install-result.json"), r.Package.Version, r.Package.Size)
	}
	if err != nil {
		s.mu.Lock()
		s.state.Download.State = "ready"
		s.state.Download.Error = "update: install_start_failed"
		s.record.Transfer = s.state.Download
		saveErr := s.saveLocked()
		s.mu.Unlock()
		return s.Snapshot(), errors.Join(err, saveErr)
	}
	return s.Snapshot(), nil
}
