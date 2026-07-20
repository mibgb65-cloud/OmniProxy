package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"omniproxy/internal/config"
	"omniproxy/internal/token"
	"strings"
	"time"
)

type ValidationResult struct {
	OK          bool             `json:"ok"`
	Status      int              `json:"status"`
	Duration    int64            `json:"durationMs"`
	Remaining   *int             `json:"remaining,omitempty"`
	Usage       *token.UsageInfo `json:"usage,omitempty"`
	Message     string           `json:"message"`
	CheckedPath string           `json:"checkedPath"`
}

type Validator struct {
	cfg         config.Config
	client      *http.Client
	proxyClient *http.Client
}

var (
	zhipuCodingPlanUsageURL = "https://api.z.ai/api/monitor/usage/quota/limit"
	zhipuAPIBalanceURL      = "https://bigmodel.cn/api/biz/tokenAccounts/list/my"
)

func NewValidator(cfg config.Config) (*Validator, error) {
	cfg = config.Normalize(cfg)
	if err := ValidateValidationURLs(cfg); err != nil {
		return nil, err
	}
	outboundProxy, err := outboundProxyURL(cfg)
	if err != nil {
		return nil, err
	}

	var proxyClient *http.Client
	if outboundProxy != nil {
		proxyClient = newHTTPClient(12*time.Second, outboundProxy)
	}

	return &Validator{
		cfg:         cfg,
		client:      newHTTPClient(12*time.Second, nil),
		proxyClient: proxyClient,
	}, nil
}

func (v *Validator) Validate(ctx context.Context, selected token.Token) (ValidationResult, error) {
	start := time.Now()
	target, err := v.validationURL(selected)
	if err != nil {
		return ValidationResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return ValidationResult{}, err
	}
	req.Header.Set("Accept", "application/json")
	if err := applyAuth(req.Header, selected); err != nil {
		return ValidationResult{}, err
	}

	resp, err := v.clientForToken(selected).Do(req)
	if err != nil {
		return ValidationResult{
			OK:          false,
			Duration:    time.Since(start).Milliseconds(),
			Message:     err.Error(),
			CheckedPath: target,
		}, err
	}
	defer closeBody(resp.Body)
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))

	result := ValidationResult{
		OK:          resp.StatusCode >= 200 && resp.StatusCode < 300,
		Status:      resp.StatusCode,
		Duration:    time.Since(start).Milliseconds(),
		Message:     http.StatusText(resp.StatusCode),
		CheckedPath: target,
	}
	if remaining, ok := parseRemaining(resp.Header); ok {
		result.Remaining = &remaining
		result.Usage = &token.UsageInfo{
			Source:       token.NormalizeProvider(selected.Provider),
			APIRemaining: remaining,
			Message:      "API rate-limit header",
		}
	}
	if result.OK && selected.CredentialType == token.CredentialTypeCodexAuthJSON {
		usage, ok := parseCodexUsage(body)
		if ok {
			v.hydrateCodexResetCredits(ctx, selected, &usage)
			result.Usage = &usage
			if usage.SubscriptionQuotaAvailable {
				remaining := usage.EffectiveRemainingPercent()
				result.Remaining = &remaining
			}
		}
	}
	if result.OK && token.NormalizeProvider(selected.Provider) == token.ProviderOpenRouter {
		if usage, remaining, ok := parseOpenRouterKeyUsage(body); ok {
			result.Usage = &usage
			if remaining != nil {
				result.Remaining = remaining
			}
		}
	}
	if result.OK && token.NormalizeProvider(selected.Provider) == token.ProviderSub2API {
		if usage, remaining, ok := parseSub2APIUsage(body); ok {
			result.Usage = &usage
			if remaining != nil {
				result.Remaining = remaining
			}
		}
	}
	if result.OK && token.NormalizeProvider(selected.Provider) == token.ProviderNewAPI {
		if usage, remaining, ok := parseNewAPIUsage(body); ok {
			result.Usage = &usage
			if remaining != nil {
				result.Remaining = remaining
			}
		}
	}
	if result.OK && selected.CredentialType != token.CredentialTypeCodexAuthJSON {
		if usage, remaining, ok := v.queryProviderQuota(ctx, selected); ok {
			result.Usage = &usage
			if remaining != nil {
				result.Remaining = remaining
			}
		}
	}
	return result, nil
}

func (v *Validator) validationURL(selected token.Token) (string, error) {
	if selected.CredentialType == token.CredentialTypeCodexAuthJSON {
		return v.cfg.CodexUsageEndpoint, nil
	}

	baseURL := v.baseURL(selected)
	if baseURL == "" {
		return "", fmt.Errorf("%s upstream base url is not configured", selected.Provider)
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	path := "/v1/models"
	switch token.NormalizeProvider(selected.Provider) {
	case token.ProviderAnthropic:
		path = "/v1/models"
	case token.ProviderGemini:
		path = "/v1beta/models"
	case token.ProviderOpenRouter:
		path = "/key"
	case token.ProviderTokenRouter:
		if basePathHasVersionSuffix(base.Path) {
			path = "/routing-rules"
		} else {
			path = "/v1/routing-rules"
		}
	case token.ProviderSub2API:
		if basePathHasVersionSuffix(base.Path) {
			path = "/usage"
		} else {
			path = "/v1/usage"
		}
	case token.ProviderNewAPI:
		out := *base
		out.Path = singleJoiningSlash(basePathWithoutVersionSuffix(base.Path), "/api/usage/token/")
		out.RawQuery = ""
		return out.String(), nil
	case token.ProviderAnyRouter:
		if basePathHasVersionSuffix(base.Path) {
			path = "/models"
		}
	case token.ProviderPrem:
		path = "/openai/v1/models"
	case token.ProviderZo:
		path = "/models/available"
	case token.ProviderXiaomi:
		path = "/models"
	case token.ProviderZhipu, token.ProviderMiniMax, token.ProviderCustom:
		path = "/models"
	}

	out := *base
	out.Path = singleJoiningSlash(base.Path, path)
	out.RawQuery = ""
	return out.String(), nil
}

func (v *Validator) baseURL(selected token.Token) string {
	return validationBaseURL(v.cfg, selected)
}

func (v *Validator) clientForToken(selected token.Token) *http.Client {
	if v.proxyClient != nil && outboundProxyMatchesToken(selected, v.cfg) {
		return v.proxyClient
	}
	return v.client
}

type codexRateLimitWindow struct {
	UsedPercent        float64 `json:"used_percent"`
	ResetAt            int64   `json:"reset_at"`
	WindowMinutes      int64   `json:"window_minutes"`
	LimitWindowSeconds int64   `json:"limit_window_seconds"`
}

func (window *codexRateLimitWindow) durationMinutes() int64 {
	if window == nil {
		return 0
	}
	if window.WindowMinutes > 0 {
		return window.WindowMinutes
	}
	if window.LimitWindowSeconds > 0 {
		return (window.LimitWindowSeconds + 59) / 60
	}
	return 0
}

func parseCodexUsage(body []byte) (token.UsageInfo, bool) {
	var payload struct {
		PlanType  string `json:"plan_type"`
		RateLimit struct {
			LimitReached    bool                  `json:"limit_reached"`
			PrimaryWindow   *codexRateLimitWindow `json:"primary_window"`
			SecondaryWindow *codexRateLimitWindow `json:"secondary_window"`
		} `json:"rate_limit"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return token.UsageInfo{}, false
	}

	usage := token.UsageInfo{
		Source:       "codex",
		PlanType:     payload.PlanType,
		LimitReached: payload.RateLimit.LimitReached,
		Message:      "Codex usage endpoint",
	}
	usage.CodexResetCreditsAvailable = codexResetCreditsAvailableFromUsage(body)

	freePlan := strings.EqualFold(strings.TrimSpace(payload.PlanType), "free")
	primaryWindow := payload.RateLimit.PrimaryWindow
	secondaryWindow := payload.RateLimit.SecondaryWindow
	primaryWindowMinutes := primaryWindow.durationMinutes()
	secondaryWindowMinutes := secondaryWindow.durationMinutes()
	switch {
	case primaryWindow != nil && secondaryWindow != nil && primaryWindowMinutes > 0 && secondaryWindowMinutes > 0:
		if primaryWindowMinutes <= secondaryWindowMinutes {
			assignCodexUsageWindow(&usage, "primary", primaryWindow)
			assignCodexUsageWindow(&usage, "secondary", secondaryWindow)
		} else {
			assignCodexUsageWindow(&usage, "secondary", primaryWindow)
			assignCodexUsageWindow(&usage, "primary", secondaryWindow)
		}
	default:
		if primaryWindow != nil {
			kind := codexQuotaKindFromWindowMinutes(primaryWindowMinutes, "primary")
			if freePlan && primaryWindowMinutes <= 0 {
				kind = "secondary"
			}
			assignCodexUsageWindow(&usage, kind, primaryWindow)
		}
		if secondaryWindow != nil {
			assignCodexUsageWindow(&usage, codexQuotaKindFromWindowMinutes(secondaryWindowMinutes, "secondary"), secondaryWindow)
		}
	}

	return usage, usage.PlanType != "" || usage.SubscriptionQuotaAvailable
}

func codexQuotaKindFromWindowMinutes(windowMinutes int64, fallback string) string {
	if windowMinutes <= 0 {
		return fallback
	}
	if windowMinutes <= 360 {
		return "primary"
	}
	return "secondary"
}

func assignCodexUsageWindow(usage *token.UsageInfo, kind string, window *codexRateLimitWindow) {
	if usage == nil || window == nil {
		return
	}
	used := percent(window.UsedPercent)
	remaining := 100 - used
	if remaining < 0 {
		remaining = 0
	}
	switch kind {
	case "secondary":
		usage.SecondaryUsedPercent = used
		usage.SecondaryRemainingPercent = remaining
		usage.SecondaryResetAt = window.ResetAt
	default:
		usage.PrimaryUsedPercent = used
		usage.PrimaryRemainingPercent = remaining
		usage.PrimaryResetAt = window.ResetAt
	}
	usage.SubscriptionQuotaAvailable = true
}
