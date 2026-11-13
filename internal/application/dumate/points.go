package dumate

import (
	"context"
	"time"

	"omniproxy/internal/application/account"
	providerdumate "omniproxy/internal/provider/dumate"
)

// Points 是积分查询结果；State 区分实时、最近快照与查询失败。
type Points struct {
	// State 为 live、stale 或 unavailable。
	State     string                 `json:"state"`
	ErrorCode string                 `json:"errorCode,omitempty"`
	FetchedAt string                 `json:"fetchedAt,omitempty"`
	Points    *providerdumate.Points `json:"points,omitempty"`
}

// Points 查询账号积分与签到状态；控制台暂时失败时回退到保留期内的成功快照。
func (s *Service) Points(ctx context.Context, accountID string) (Points, error) {
	if _, err := s.accountFor(ctx, accountID); err != nil {
		return Points{}, err
	}
	readCtx, cancel := context.WithTimeout(ctx, pointsTimeout)
	defer cancel()
	var overview providerdumate.Overview
	var bonus providerdumate.LoginBonus
	var bonusKnown bool
	var queryErr error
	err := s.accounts.UseDuMateCredentials(readCtx, accountID, func(_ account.Item, credentials providerdumate.Credentials) error {
		overview, queryErr = s.console.Overview(readCtx, credentials)
		if queryErr != nil {
			return nil
		}
		// 签到状态读取失败不影响积分总览；界面据此区分未知与未签到。
		if value, bonusErr := s.console.LoginBonus(readCtx, credentials); bonusErr == nil {
			bonus, bonusKnown = value, true
		}
		return nil
	})
	if err != nil {
		return Points{}, failure("credential_unavailable")
	}
	return s.points(accountID, overview, bonus, bonusKnown, queryErr), nil
}

// points 组合一次查询结果，失败时只回退到保留期内真实成功的快照。
func (s *Service) points(accountID string, overview providerdumate.Overview, bonus providerdumate.LoginBonus, bonusKnown bool, queryErr error) Points {
	if queryErr != nil {
		cached, at, ok := s.latest(accountID)
		if ok {
			return Points{State: "stale", ErrorCode: importCode(queryErr), FetchedAt: fetchedAt(at), Points: &cached}
		}
		return Points{State: "unavailable", ErrorCode: importCode(queryErr)}
	}
	value := providerdumate.BuildPoints(overview, bonus, bonusKnown)
	s.record(accountID, value)
	return Points{State: "live", FetchedAt: fetchedAt(s.now()), Points: &value}
}

// fetchedAt 统一使用 UTC 秒级 RFC3339，前端按本地时区显示。
func fetchedAt(value time.Time) string { return value.UTC().Format(time.RFC3339) }
