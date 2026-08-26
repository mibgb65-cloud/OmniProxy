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

const codexActivationRequestBody = `{"model":"gpt-5.6-luna","instructions":"Reply with OK only.","input":[{"type":"message","role":"user","content":"OK"}],"reasoning":{"effort":"low","summary":"auto"},"max_output_tokens":8,"stream":true,"store":false}`

func CodexUsageNeedsActivation(usage token.UsageInfo) bool {
	return CodexUsageNeedsActivationAt(usage, time.Now())
}

func CodexUsageNeedsActivationAt(usage token.UsageInfo, now time.Time) bool {
	resetExpired := func(resetAt int64) bool {
		return resetAt <= now.Unix()
	}
	if strings.EqualFold(strings.TrimSpace(usage.PlanType), "free") {
		return resetExpired(usage.SecondaryResetAt)
	}
	return resetExpired(usage.PrimaryResetAt) || resetExpired(usage.SecondaryResetAt)
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
