// Package dumate 负责百度搭子 DuMate 个人版的登录态导入、积分查询与签到编排。
package dumate

import (
	"context"
	"errors"
	"sync"
	"time"

	"omniproxy/internal/application/account"
	providerdumate "omniproxy/internal/provider/dumate"
)

const (
	// importTimeout 覆盖读取本机登录文件与一次控制台身份核对。
	importTimeout = 20 * time.Second
	// pointsTimeout 覆盖积分总览与签到状态两次只读请求。
	pointsTimeout = 30 * time.Second
	// checkinTimeout 是单次签到的总预算，包含领取前后两次状态核对。
	checkinTimeout = 30 * time.Second
	// snapshotRetention 是「最近一次成功」可用于展示的时长。
	snapshotRetention = time.Hour
	// maxSnapshots 限制内存快照条数。
	maxSnapshots = 64
)

// Console 是控制台只读／签到接口；测试可注入假实现。
type Console interface {
	UserInfo(context.Context, providerdumate.Credentials) (providerdumate.Identity, error)
	Overview(context.Context, providerdumate.Credentials) (providerdumate.Overview, error)
	LoginBonus(context.Context, providerdumate.Credentials) (providerdumate.LoginBonus, error)
	ClaimLoginBonus(context.Context, providerdumate.Credentials) error
}

// AccountAccess 是账号服务中本功能需要的受控子集。
type AccountAccess interface {
	Metadata(context.Context, string) (account.Item, error)
	UseDuMateCredentials(context.Context, string, func(account.Item, providerdumate.Credentials) error) error
	SaveDuMateAccount(context.Context, string, string, string, providerdumate.Credentials) (account.Item, error)
	FindDuMateAccount(context.Context, string, string) (account.Item, bool, error)
	CheckDuMateProvider(context.Context, string) error
}

// Service 复用一个有界内存快照与可注入的控制台客户端。
type Service struct {
	console     Console
	accounts    AccountAccess
	now         func() time.Time
	desktop     func() (string, error)
	readDesktop func(string) (providerdumate.Credentials, error)
	snapshots   map[string]snapshot
	snapMu      sync.Mutex
}

type snapshot struct {
	points    providerdumate.Points
	fetchedAt time.Time
}

// New 构造服务；console 为 nil 时使用默认 HTTPS 控制台客户端。
func New(accounts AccountAccess, console Console) *Service {
	if console == nil {
		console = providerdumate.NewConsole(nil)
	}
	return &Service{accounts: accounts, console: console, snapshots: map[string]snapshot{}, now: time.Now, desktop: providerdumate.DesktopDir, readDesktop: providerdumate.ReadDesktop}
}

// failure 生成固定错误码，前端按码显示本地化文案。
func failure(code string) error { return errors.New("dumate: " + code) }

// record 保存一次成功读取，供控制台暂时失败时显示最近快照。
func (s *Service) record(accountID string, points providerdumate.Points) {
	s.snapMu.Lock()
	defer s.snapMu.Unlock()
	if len(s.snapshots) >= maxSnapshots {
		var oldest string
		var oldestAt time.Time
		for key, entry := range s.snapshots {
			if oldest == "" || entry.fetchedAt.Before(oldestAt) {
				oldest, oldestAt = key, entry.fetchedAt
			}
		}
		delete(s.snapshots, oldest)
	}
	s.snapshots[accountID] = snapshot{points: points, fetchedAt: s.now()}
}

// latest 返回保留期内的最近成功快照。
func (s *Service) latest(accountID string) (providerdumate.Points, time.Time, bool) {
	s.snapMu.Lock()
	defer s.snapMu.Unlock()
	entry, ok := s.snapshots[accountID]
	if !ok || entry.fetchedAt.IsZero() || s.now().Sub(entry.fetchedAt) > snapshotRetention {
		return providerdumate.Points{}, time.Time{}, false
	}
	return entry.points, entry.fetchedAt, true
}

func (s *Service) forget(accountID string) {
	s.snapMu.Lock()
	defer s.snapMu.Unlock()
	delete(s.snapshots, accountID)
}

// accountFor 读取账号并确认它属于 DuMate 提供商。
func (s *Service) accountFor(ctx context.Context, id string) (account.Item, error) {
	item, err := s.accounts.Metadata(ctx, id)
	if err != nil {
		return account.Item{}, failure("invalid_account")
	}
	if item.AuthMethod != account.AuthDuMateCookie || s.accounts.CheckDuMateProvider(ctx, item.ProviderID) != nil {
		return account.Item{}, failure("not_dumate_account")
	}
	return item, nil
}
