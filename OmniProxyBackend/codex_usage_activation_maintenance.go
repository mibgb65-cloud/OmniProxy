package main

import (
	"context"
	"fmt"
	"net/http"
	"omniproxy/internal/history"
	"omniproxy/internal/logs"
	"omniproxy/internal/token"
)

type codexUsageActivationAttempt struct {
	Attempted              bool
	Activated              bool
	BeforePrimaryResetAt   int64
	BeforeSecondaryResetAt int64
	AfterPrimaryResetAt    int64
	AfterSecondaryResetAt  int64
	Duration               int64
}

func (a *appServer) recordCodexUsageActivationHistory(selected token.Token, activation codexUsageActivationAttempt, err error, source string) {
	a.mu.Lock()
	recorder := a.history
	a.mu.Unlock()
	if recorder == nil {
		return
	}
	if latest, latestErr := a.tokens.Get(selected.ID); latestErr == nil {
		selected = latest
	}
	message := "codex usage activation skipped; windows already active"
	if err != nil && !activation.Attempted {
		message = "codex usage activation check failed"
	} else if activation.Attempted {
		if err != nil || !activation.Activated {
			message = "codex usage activation failed"
		} else {
			message = "codex usage activation completed"
		}
	}
	message += fmt.Sprintf(" · primary_reset=%s · secondary_reset=%s", activationResetTransition(activation.BeforePrimaryResetAt, activation.AfterPrimaryResetAt), activationResetTransition(activation.BeforeSecondaryResetAt, activation.AfterSecondaryResetAt))
	if err != nil {
		message += " · " + err.Error()
	}
	level := logs.LevelInfo
	status := http.StatusOK
	if err != nil || (activation.Attempted && !activation.Activated) {
		level = logs.LevelWarn
		status = http.StatusBadGateway
	}
	path := "/maintenance/codex-usage-activation"
	if source == "manual" {
		path = "/maintenance/manual-codex-usage-activation"
	}
	recorder.Add(history.Entry{
		Level:     string(level),
		Method:    "POST",
		Path:      path,
		Provider:  token.NormalizeProvider(selected.Provider),
		Protocol:  "quota-activation",
		Model:     selected.CredentialType,
		Status:    status,
		Duration:  activation.Duration,
		TokenID:   selected.ID,
		TokenName: token.DisplayName(selected),
		Message:   message,
	})
	if a.logs != nil {
		a.logs.Add(logs.Entry{Level: level, TokenName: a.tokenDisplayName(selected), Message: message})
	}
}

func (a *appServer) recordCodexUsageActivationStarted(selected token.Token, activation codexUsageActivationAttempt, source string) {
	a.mu.Lock()
	recorder := a.history
	a.mu.Unlock()
	message := fmt.Sprintf("codex usage activation started · primary_reset=%s · secondary_reset=%s", activationResetTransition(activation.BeforePrimaryResetAt, 0), activationResetTransition(activation.BeforeSecondaryResetAt, 0))
	path := "/maintenance/codex-usage-activation"
	if source == "manual" {
		path = "/maintenance/manual-codex-usage-activation"
	}
	if recorder != nil {
		recorder.Add(history.Entry{
			Level:     string(logs.LevelInfo),
			Method:    "POST",
			Path:      path,
			Provider:  token.NormalizeProvider(selected.Provider),
			Protocol:  "quota-activation",
			Model:     selected.CredentialType,
			Status:    http.StatusAccepted,
			Duration:  activation.Duration,
			TokenID:   selected.ID,
			TokenName: token.DisplayName(selected),
			Message:   message,
		})
	}
	if a.logs != nil {
		a.logs.Add(logs.Entry{Level: logs.LevelInfo, TokenName: a.tokenDisplayName(selected), Message: message})
	}
}

func activationResetTransition(before, after int64) string {
	if after == 0 {
		after = before
	}
	if before == 0 {
		return fmt.Sprintf("-→%d", after)
	}
	if before == after {
		return fmt.Sprintf("%d", before)
	}
	return fmt.Sprintf("%d→%d", before, after)
}

func (a *appServer) scanCodexUsageActivationOnEnable(ctx context.Context) {
	items := a.tokens.List()
	total := 0
	failed := 0
	for _, item := range items {
		if item.Disabled || !isCodexToken(item) {
			continue
		}
		total++
		checkCtx, cancel := context.WithTimeout(ctx, healthRequestTimeout)
		result, err := a.validateAndRecordToken(checkCtx, item)
		cancel()
		a.recordTokenMaintenanceHistory(historyEventCodexEnableScan, item, result, err)
		if err != nil || !result.OK {
			failed++
		}
	}
	if total == 0 || a.logs == nil {
		return
	}
	level := logs.LevelInfo
	message := fmt.Sprintf("codex activation scan after enabling setting completed: %d accounts", total)
	if failed > 0 {
		level = logs.LevelWarn
		message = fmt.Sprintf("codex activation scan after enabling setting completed: %d accounts, %d failed", total, failed)
	}
	a.logs.Add(logs.Entry{Level: level, Message: message})
}
