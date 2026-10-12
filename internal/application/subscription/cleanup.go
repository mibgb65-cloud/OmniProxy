package subscription

import (
	"context"
	"os"
	"path/filepath"

	"omniproxy/internal/application/account"
)

func (s *Service) logoutProfile(ctx context.Context, profile string) error {
	client, err := s.open(ctx, profile)
	if err != nil {
		return err
	}
	defer client.Close()
	return client.Logout(ctx)
}

func (s *Service) removeProfile(profile string) error {
	path, err := s.profilePath(profile)
	if err != nil {
		return err
	}
	if os.RemoveAll(path) != nil {
		return failure("cleanup_failed")
	}
	return nil
}

func (s *Service) cleanCancelled(job *session) error {
	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()
	if job.client != nil {
		if job.loginID != "" {
			// 回调已完成时 cancel 可能拒绝；仍须 logout 清理实际凭证。
			_ = job.client.Cancel(ctx, job.loginID)
		}
		if job.client.Logout(ctx) != nil {
			return failure("cleanup_failed")
		}
		job.client.Close()
	}
	return s.removeProfile(job.profile)
}

func (s *Service) cleanRetired(ctx context.Context, id string) error {
	_, profiles, err := s.accounts.SubscriptionProfiles(ctx, id)
	if err != nil {
		return err
	}
	for _, profile := range profiles[1:] {
		if err := s.logoutProfile(ctx, profile); err != nil {
			return err
		}
		if err := s.removeProfile(profile); err != nil {
			return err
		}
	}
	_, err = s.accounts.ClearRetired(ctx, id)
	return err
}

// recoverPending 只清理本应用标记且没有账号引用的隔离配置。
// 异常退出或系统凭证库暂时不可用时保留标记，下次登录前重试，不丢失清理位置。
func (s *Service) recoverPending(ctx context.Context) error {
	s.mu.Lock()
	busy := s.closed || s.current != nil
	s.mu.Unlock()
	if busy {
		return failure("busy")
	}
	items, err := s.accounts.List(ctx)
	if err != nil {
		return failure("account_unavailable")
	}
	referenced := map[string]bool{}
	for _, item := range items {
		if item.AuthMethod != account.AuthSubscription {
			continue
		}
		_, profiles, err := s.accounts.SubscriptionProfiles(ctx, item.ID)
		if err != nil {
			return failure("account_unavailable")
		}
		for _, profile := range profiles {
			referenced[profile] = true
		}
	}
	entries, err := os.ReadDir(s.root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return failure("cleanup_failed")
	}
	for _, entry := range entries {
		if !entry.IsDir() || !validID(entry.Name()) {
			continue
		}
		path, err := s.profilePath(entry.Name())
		if err != nil {
			return err
		}
		marker := filepath.Join(path, "omniproxy-pending")
		if _, err := os.Stat(marker); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return failure("cleanup_failed")
		}
		if referenced[entry.Name()] {
			if os.Remove(marker) != nil {
				return failure("cleanup_failed")
			}
			continue
		}
		if err := s.logoutProfile(ctx, entry.Name()); err != nil {
			return failure("cleanup_failed")
		}
		if err := s.removeProfile(entry.Name()); err != nil {
			return err
		}
	}
	return nil
}
