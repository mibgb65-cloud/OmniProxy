package agentinstall

import (
	"math"

	runtime "omniproxy/internal/infrastructure/agentinstall"
)

func (s *Service) updateProgress(id string, progress runtime.Progress) {
	if progress.Phase != "preparing" && progress.Phase != "downloading" && progress.Phase != "installing" {
		return
	}
	if progress.DownloadedBytes < 0 || progress.TotalBytes < 0 || progress.DownloadedBytes > progress.TotalBytes || progress.TotalBytes > 1<<50 {
		return
	}
	if progress.Percent != nil {
		value := *progress.Percent
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 100 {
			return
		}
		progress.Percent = &value
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.items[id]
	if s.active != id || item.InstallState != "installing" {
		return
	}
	item.Progress = &progress
	s.items[id] = item
}
