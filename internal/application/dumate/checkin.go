package dumate

import (
	"context"
	"math"

	"omniproxy/internal/application/account"
	providerdumate "omniproxy/internal/provider/dumate"
)

// Checkin 是一次签到的结果；Outcome 与界面文案一一对应。
type Checkin struct {
	// Outcome 为 claimed、done、pending 或 failed。
	Outcome string `json:"outcome"`
	// ErrorCode 只在 failed 或 pending 时出现。
	ErrorCode string  `json:"errorCode,omitempty"`
	Credit    float64 `json:"credit,omitempty"`
	// TotalTimes 是累计签到天数；未知时为 0。
	TotalTimes int    `json:"totalTimes,omitempty"`
	CheckedAt  string `json:"checkedAt"`
}

// Checkin 先读今日状态，再领取并核对到账；不凭 POST 成功就显示已领取。
func (s *Service) Checkin(ctx context.Context, accountID string) (Checkin, error) {
	if _, err := s.accountFor(ctx, accountID); err != nil {
		return Checkin{}, err
	}
	checkCtx, cancel := context.WithTimeout(ctx, checkinTimeout)
	defer cancel()
	result := Checkin{CheckedAt: fetchedAt(s.now())}
	credentialErr := s.accounts.UseDuMateCredentials(checkCtx, accountID, func(_ account.Item, credentials providerdumate.Credentials) error {
		s.claim(checkCtx, &result, credentials)
		return nil
	})
	if credentialErr != nil {
		return Checkin{}, failure("credential_unavailable")
	}
	if result.Outcome == "claimed" || result.Outcome == "done" {
		// 积分与签到天数已变化，下一次查询必须是实时结果。
		s.forget(accountID)
	}
	return result, nil
}

// claim 在凭证回调内完成状态读取、领取与到账核对。
func (s *Service) claim(ctx context.Context, result *Checkin, credentials providerdumate.Credentials) {
	before, beforeErr := s.console.LoginBonus(ctx, credentials)
	if beforeErr != nil && providerdumate.SignInExpired(beforeErr) {
		result.Outcome, result.ErrorCode = "failed", "sign_in_expired"
		return
	}
	beforeKnown := beforeErr == nil
	if beforeKnown && before.HasIssued {
		result.Outcome, result.TotalTimes = "done", before.TotalTimes
		return
	}
	if err := s.console.ClaimLoginBonus(ctx, credentials); err != nil {
		result.Outcome, result.ErrorCode = "failed", checkinCode(err)
		return
	}
	after, afterErr := s.console.LoginBonus(ctx, credentials)
	if afterErr != nil {
		result.Outcome, result.ErrorCode = "pending", "verify_failed"
		return
	}
	if !after.HasIssued {
		// 提交成功但账本还没到账：不宣称已领取，界面稍后重试。
		result.Outcome = "pending"
		return
	}
	result.Outcome, result.TotalTimes = "claimed", after.TotalTimes
	if beforeKnown {
		result.Credit = roundCredit(after.TotalPoints - before.TotalPoints)
	}
}

func checkinCode(err error) string {
	if providerdumate.SignInExpired(err) {
		return "sign_in_expired"
	}
	if code := providerdumate.ErrorCode(err); code != "" {
		return code
	}
	return "claim_failed"
}

func roundCredit(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 0
	}
	return math.Round(value*100) / 100
}
