package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"omniproxy/internal/logs"
	"omniproxy/internal/proxy"
	"omniproxy/internal/token"
)

type codexResetCreditConsumeResponse struct {
	Consumed     bool          `json:"consumed"`
	Refreshed    bool          `json:"refreshed"`
	RefreshError string        `json:"refreshError,omitempty"`
	Message      string        `json:"message"`
	Token        tokenResponse `json:"token"`
}

func (a *appServer) consumeCodexResetCredit(ctx context.Context, id string) (codexResetCreditConsumeResponse, error) {
	a.codexResetCreditMu.Lock()
	defer a.codexResetCreditMu.Unlock()

	selected, err := a.tokens.Get(id)
	if err != nil {
		return codexResetCreditConsumeResponse{}, err
	}
	if token.NormalizeProvider(selected.Provider) != token.ProviderOpenAI || selected.CredentialType != token.CredentialTypeCodexAuthJSON {
		return codexResetCreditConsumeResponse{}, errors.New("只有 Codex auth.json 账号可以使用额度刷新卡")
	}

	selected, _, err = a.refreshAuthTokenIfNeeded(ctx, selected, false)
	if err != nil {
		return codexResetCreditConsumeResponse{}, err
	}
	a.mu.Lock()
	cfg := a.cfg
	a.mu.Unlock()
	validator, err := proxy.NewValidator(cfg)
	if err != nil {
		return codexResetCreditConsumeResponse{}, err
	}

	redeemRequestID := uuid.NewString()
	err = validator.ConsumeCodexResetCredit(ctx, selected, redeemRequestID)
	if proxy.CodexResetCreditsAuthFailed(err) {
		refreshed, changed, refreshErr := a.refreshAuthTokenIfNeeded(ctx, selected, true)
		if refreshErr != nil {
			return codexResetCreditConsumeResponse{}, refreshErr
		}
		if changed {
			selected = refreshed
			err = validator.ConsumeCodexResetCredit(ctx, selected, redeemRequestID)
		}
	}
	if err != nil {
		return codexResetCreditConsumeResponse{}, err
	}

	estimateResetAt := time.Now()
	estimateBaseline := selected.Stats
	if current, getErr := a.tokens.Get(id); getErr == nil {
		estimateBaseline = current.Stats
	}
	_ = a.tokens.InvalidateCodexResetCredits(id)
	selected.Usage.CodexResetCreditsCheckedAt = 0
	selected.Usage.CodexResetCreditsError = ""
	validation, refreshErr := a.validateAndRecordToken(ctx, selected)
	latest, getErr := a.tokens.Get(id)
	if getErr != nil {
		latest = selected
	}
	if refreshErr == nil && validation.OK && codexWeeklyQuotaWindowReset(selected.Usage, latest.Usage, estimateResetAt) {
		if baselineErr := a.tokens.RecordCodexWeeklyEstimateReset(id, estimateResetAt, estimateBaseline); baselineErr == nil {
			if updated, updatedErr := a.tokens.Get(id); updatedErr == nil {
				latest = updated
			}
		}
	}
	response := codexResetCreditConsumeResponse{
		Consumed:  true,
		Refreshed: refreshErr == nil && validation.OK,
		Message:   "额度刷新卡已使用，5 小时额度已重置",
		Token:     tokenResponseFor(latest),
	}
	if refreshErr != nil {
		response.RefreshError = refreshErr.Error()
		response.Message = "额度刷新卡已使用，但最新额度同步失败"
	} else if !validation.OK {
		response.RefreshError = fmt.Sprintf("额度接口返回 %d %s", validation.Status, http.StatusText(validation.Status))
		response.Message = "额度刷新卡已使用，但最新额度同步失败"
	}
	if a.logs != nil {
		a.logs.Add(logs.Entry{Level: logs.LevelInfo, TokenName: a.tokenDisplayName(latest), Message: "Codex quota reset credit consumed"})
	}
	return response, nil
}

func codexWeeklyQuotaWindowReset(before token.UsageInfo, after token.UsageInfo, consumedAt time.Time) bool {
	if before.SecondaryResetAt <= 0 || after.SecondaryResetAt <= 0 {
		return false
	}
	if before.SecondaryResetAt <= consumedAt.Unix() {
		return false
	}
	if before.SecondaryResetAt != after.SecondaryResetAt {
		return true
	}
	return codexWeeklyUsedPercent(before)-codexWeeklyUsedPercent(after) > 0.01
}

func codexWeeklyUsedPercent(usage token.UsageInfo) float64 {
	if usage.SecondaryUsedPercentExact > 0 {
		return usage.SecondaryUsedPercentExact
	}
	return float64(usage.SecondaryUsedPercent)
}
