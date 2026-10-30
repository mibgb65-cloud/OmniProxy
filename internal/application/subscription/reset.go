package subscription

import (
	"context"
	"strings"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/infrastructure/codex"
)

func resetView(value account.ResetAttempt) ResetOperation {
	return ResetOperation{OperationID: value.OperationID, CreditID: value.CreditID, State: value.State, Outcome: value.Outcome}
}

// ResetStatus 即使额度接口暂时不可用，也能恢复待核对操作或已完成回执。
func (s *Service) ResetStatus(ctx context.Context, id string) (ResetOperation, error) {
	item, err := s.accounts.Metadata(ctx, id)
	if err != nil || item.AuthMethod != account.AuthSubscription {
		return ResetOperation{}, failure("invalid_account")
	}
	attempt, err := s.accounts.LatestSubscriptionReset(ctx, id)
	if err != nil {
		return ResetOperation{}, failure("reset_record_failed")
	}
	return resetView(attempt), nil
}

// ConsumeReset 仅响应用户确认；请求结果不确定时保留同一幂等标识供显式核对。
func (s *Service) ConsumeReset(ctx context.Context, input ResetInput) (ResetResult, error) {
	if !validCreditID(input.CreditID) || !validOperationID(input.OperationID) {
		return ResetResult{}, failure("invalid_input")
	}
	if err := s.operations.LockAccount(ctx, input.AccountID); err != nil {
		return ResetResult{}, failure("operation_cancelled")
	}
	defer s.operations.UnlockAccount(input.AccountID)
	if ctx.Err() != nil {
		return ResetResult{}, failure("operation_cancelled")
	}
	if s.accountBusy(input.AccountID) {
		return ResetResult{}, failure("busy")
	}
	item, profile, err := s.accounts.SubscriptionProfile(ctx, input.AccountID)
	if err != nil {
		return ResetResult{}, failure("invalid_account")
	}
	if item.AuthState == "logged_out" || item.AuthState == "reauth_required" {
		return ResetResult{}, failure("quota_not_authenticated")
	}
	pending, err := s.accounts.PendingSubscriptionReset(ctx, input.AccountID)
	if err != nil {
		return ResetResult{}, failure("reset_record_failed")
	}
	// 双击、重开窗口或丢失响应不能绕过已有的待确认操作。
	if pending.OperationID != "" && pending.OperationID != input.OperationID {
		return ResetResult{Operation: resetView(pending)}, nil
	}
	var attempt account.ResetAttempt
	if input.OperationID != "" {
		attempt, err = s.accounts.SubscriptionResetAttempt(ctx, input.AccountID, input.OperationID)
		if err != nil {
			return ResetResult{}, failure("invalid_input")
		}
	}
	readCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	client, err := s.open(readCtx, profile)
	if err != nil {
		cancel()
		return ResetResult{}, err
	}
	defer client.Close()
	if err := verifyMember(readCtx, client, item); err != nil {
		cancel()
		return ResetResult{}, err
	}
	raw, err := client.RateLimits(readCtx)
	cancel()
	if err != nil {
		return ResetResult{}, failure("quota_unavailable")
	}
	if raw.AccountID == "" || item.AuthAccountID != "" && item.AuthAccountID != raw.AccountID || attempt.AuthAccountID != "" && attempt.AuthAccountID != raw.AccountID {
		return ResetResult{}, failure("identity_changed")
	}
	if attempt.OperationID == "" {
		if err := checkResetCredit(raw.ResetCredits, input.CreditID); err != nil {
			return ResetResult{}, err
		}
		attempt = account.ResetAttempt{OperationID: input.OperationID, AccountID: input.AccountID, AuthAccountID: raw.AccountID, CreditID: input.CreditID, State: "pending"}
		if s.accounts.SaveSubscriptionReset(ctx, attempt) != nil {
			return ResetResult{}, failure("reset_record_failed")
		}
	}
	result := ResetResult{Operation: resetView(attempt)}
	if attempt.State != "completed" {
		consumeCtx, stop := context.WithTimeout(ctx, 10*time.Second)
		outcome, consumeErr := client.ConsumeReset(consumeCtx, attempt.OperationID, attempt.CreditID)
		stop()
		if consumeErr != nil {
			return result, nil
		}
		result.Operation.Outcome = outcome
		// 即使 IPC 上下文超时，也独立保存已经确认的官方结果。
		saveCtx, finish := context.WithTimeout(context.Background(), 5*time.Second)
		saveErr := s.accounts.CompleteSubscriptionReset(saveCtx, input.AccountID, attempt.OperationID, outcome)
		finish()
		if saveErr == nil {
			result.Operation.State = "completed"
		}
	}
	refreshCtx, finishRefresh := context.WithTimeout(ctx, 10*time.Second)
	defer finishRefresh()
	fresh, refreshErr := client.RateLimits(refreshCtx)
	if refreshErr == nil && fresh.AccountID == raw.AccountID {
		quota, normalizeErr := normalizeQuota(fresh, item.AuthPlan)
		if normalizeErr == nil {
			result.Quota = &quota
			s.rememberQuota(item.ID, item.UpdatedAt, quota)
		} else {
			result.RefreshFailed = true
		}
	} else {
		result.RefreshFailed = true
	}
	return result, nil
}

func checkResetCredit(summary *codex.ResetCreditSummary, id string) error {
	if summary == nil || summary.AvailableCount == nil {
		return failure("reset_unavailable")
	}
	if *summary.AvailableCount <= 0 {
		return failure("reset_no_credit")
	}
	if id == "" {
		return nil
	}
	now := time.Now().Unix()
	for _, card := range summary.Credits {
		if card.ID != id {
			continue
		}
		if !usableResetCredit(card, now) {
			return failure("reset_not_usable")
		}
		return nil
	}
	return failure("reset_not_usable")
}

// usableResetCredit 判定单张卡能否兑换：官方状态与有效期是权威信号。
func usableResetCredit(card codex.ResetCredit, now int64) bool {
	if card.Status != "available" || !usableResetType(card.ResetType) {
		return false
	}
	return card.ExpiresAt == nil || *card.ExpiresAt > now
}

// usableResetType 归一官方 reset_type 的大小写与下划线差异；
// 界面徽标按 status 显示，这里若按字面量严格比较，就会出现「写着可使用、按钮不可用」。
func usableResetType(value string) bool {
	var normalized strings.Builder
	for _, char := range strings.ToLower(value) {
		if char == '_' || char == '-' || char == ' ' || char == '\t' {
			continue
		}
		normalized.WriteRune(char)
	}
	switch normalized.String() {
	case "", "codexratelimits", "ratelimits", "ratelimit":
		return true
	}
	return false
}
