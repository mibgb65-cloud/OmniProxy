package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"omniproxy/internal/token"
	"strings"
	"time"
)

const codexActivationRequestBody = `{"model":"gpt-5.6-luna","instructions":"Reply with OK only.","input":[{"type":"message","role":"user","content":"OK"}],"reasoning":{"effort":"low","summary":"auto"},"stream":true,"store":false}`

const codexRollingWindowTolerance = 2 * time.Minute

func CodexUsageNeedsActivation(usage token.UsageInfo) bool {
	return CodexUsageNeedsActivationAt(usage, time.Now())
}

func CodexUsageNeedsActivationAt(usage token.UsageInfo, now time.Time) bool {
	primary, secondary := CodexUsageActivationPendingAt(usage, token.UsageInfo{}, now)
	return primary || secondary
}

func CodexUsageActivationPendingAt(usage, previous token.UsageInfo, now time.Time) (bool, bool) {
	freePlan := strings.EqualFold(strings.TrimSpace(usage.PlanType), "free")
	primaryPending := false
	if !freePlan {
		primaryPending = codexWindowPendingActivation(
			usage.PrimaryUsedPercent,
			usage.PrimaryUsedPercentExact,
			usage.PrimaryResetAt,
			previous.PrimaryResetAt,
			previous.PrimaryActivationPending,
			5*time.Hour,
			now,
		)
	}
	secondaryPending := codexWindowPendingActivation(
		usage.SecondaryUsedPercent,
		usage.SecondaryUsedPercentExact,
		usage.SecondaryResetAt,
		previous.SecondaryResetAt,
		previous.SecondaryActivationPending,
		7*24*time.Hour,
		now,
	)
	return primaryPending, secondaryPending
}

func CodexUsageHasCurrentWindowsAt(usage token.UsageInfo, now time.Time) bool {
	if usage.SecondaryResetAt <= now.Unix() {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(usage.PlanType), "free") || usage.PrimaryResetAt > now.Unix()
}

func codexWindowPendingActivation(used int, usedExact float64, resetAt, previousResetAt int64, previousPending bool, window time.Duration, now time.Time) bool {
	if resetAt <= now.Unix() {
		return true
	}
	if used > 0 || usedExact > 0 {
		return false
	}
	if previousPending {
		return true
	}
	if previousResetAt == resetAt {
		return false
	}
	expectedResetAt := now.Add(window).Unix()
	difference := time.Duration(resetAt-expectedResetAt) * time.Second
	if difference < 0 {
		difference = -difference
	}
	return difference <= codexRollingWindowTolerance
}

func (v *Validator) ActivateCodexUsage(ctx context.Context, selected token.Token) error {
	target, err := codexActivationEndpoint(v.cfg.CodexBaseURL)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewBufferString(codexActivationRequestBody))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenAI-Beta", "responses=experimental")
	req.Header.Set("Originator", "codex_cli_rs")
	req.Header.Set("User-Agent", "codex_cli_rs")
	if err := applyAuth(req.Header, selected); err != nil {
		return err
	}

	resp, err := v.clientForToken(selected).Do(req)
	if err != nil {
		return err
	}
	defer closeBody(resp.Body)
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if readErr != nil {
		return readErr
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message := fmt.Sprintf("Codex 自动激活请求返回 %d", resp.StatusCode)
		if payload, err := decodeObject(body); err == nil {
			if detail := codexResetCreditsErrorDetail(payload); detail != "" {
				message += "：" + detail
			}
		}
		return fmt.Errorf("%s", message)
	}
	return nil
}

func codexActivationEndpoint(baseURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("codex base url is not configured")
	}
	parsed.Path = singleJoiningSlash(parsed.Path, "/responses")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}
