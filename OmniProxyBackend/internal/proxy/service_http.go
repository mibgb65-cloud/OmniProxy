package proxy

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"omniproxy/internal/claudedesktop"
	"omniproxy/internal/config"
	"omniproxy/internal/history"
	"omniproxy/internal/logs"
	"omniproxy/internal/token"
)

func (s *Service) serveSpecialProxyRequest(w http.ResponseWriter, r *http.Request) bool {
	if isCodexResponsesWebSocket(r) {
		s.serveCodexWebSocketRequest(w, r)
		return true
	}
	if r.URL != nil && claudedesktop.IsModelsPath(r.URL.Path) {
		s.serveClaudeDesktopModels(w, r)
		return true
	}
	if isAnthropicRouterProbe(r) {
		writeProxyProbe(w, r, `{"ok":true,"service":"omniproxy anthropic router"}`)
		return true
	}
	if isOpenCodeRouterProbe(r) {
		writeProxyProbe(w, r, `{"ok":true,"service":"omniproxy opencode router"}`)
		return true
	}
	if isPiRouterProbe(r) {
		writeProxyProbe(w, r, `{"ok":true,"service":"omniproxy pi router"}`)
		return true
	}
	if isCodexResponsesProbe(r) {
		writeProxyProbe(w, r, `{"ok":true,"service":"omniproxy codex proxy"}`)
		return true
	}
	return false
}

func (s *Service) serveCodexWebSocketRequest(w http.ResponseWriter, r *http.Request) {
	route := routeWithClient(r, s.router.Route(r.URL, nil))
	if s.cfg.WebSocketMode == config.WebSocketModeDisabled {
		s.logs.Add(logs.Entry{
			Level:      logs.LevelWarn,
			Method:     r.Method,
			Path:       r.URL.RequestURI(),
			ClientKey:  route.ClientKey,
			ClientName: route.ClientName,
			Status:     http.StatusForbidden,
			Message:    "websocket proxy disabled",
		})
		s.recordHistory(r, route, nil, http.StatusForbidden, 0, token.TokenConsumption{}, logs.LevelWarn, "websocket proxy disabled")
		http.Error(w, "websocket proxy disabled", http.StatusForbidden)
		return
	}
	if !isAllowedWebSocketOrigin(r) {
		s.logs.Add(logs.Entry{
			Level:      logs.LevelWarn,
			Method:     r.Method,
			Path:       r.URL.RequestURI(),
			ClientKey:  route.ClientKey,
			ClientName: route.ClientName,
			Status:     http.StatusForbidden,
			Message:    "websocket origin not allowed",
		})
		s.recordHistory(r, route, nil, http.StatusForbidden, 0, token.TokenConsumption{}, logs.LevelWarn, "websocket origin not allowed")
		http.Error(w, "websocket origin not allowed", http.StatusForbidden)
		return
	}
	s.serveCodexWebSocket(w, r)
}

func writeProxyProbe(w http.ResponseWriter, r *http.Request, body string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write([]byte(body))
	}
}

func (s *Service) readHTTPProxyBody(w http.ResponseWriter, r *http.Request, start time.Time) ([]byte, bool) {
	bodyBytes, decodedBody, err := readProxyRequestBody(r.Body, r.Header.Get("Content-Encoding"))
	if err != nil {
		route := routeWithClient(r, routeInfo{Provider: token.ProviderOpenAI, Path: r.URL.Path, RawQuery: r.URL.RawQuery})
		if errors.Is(err, errRequestBodyTooLarge) {
			s.recordHistory(r, route, nil, http.StatusRequestEntityTooLarge, time.Since(start).Milliseconds(), token.TokenConsumption{}, logs.LevelWarn, errRequestBodyTooLarge.Error())
			http.Error(w, errRequestBodyTooLarge.Error(), http.StatusRequestEntityTooLarge)
			return nil, false
		}
		s.recordHistory(r, route, nil, http.StatusBadRequest, time.Since(start).Milliseconds(), token.TokenConsumption{}, logs.LevelWarn, "failed to read request body")
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return nil, false
	}
	if decodedBody {
		r.Header.Del("Content-Encoding")
		r.Header.Del("Content-Length")
	}
	return bodyBytes, true
}

func (s *Service) rewriteClaudeDesktopMessageBody(w http.ResponseWriter, r *http.Request, body []byte, start time.Time) ([]byte, bool) {
	if r.URL == nil || !claudedesktop.IsMessagesPath(r.URL.Path) {
		return body, true
	}
	route := routeWithClient(r, routeInfo{Provider: token.ProviderAnthropic, Protocol: "anthropic", Path: r.URL.Path})
	if err := claudedesktop.ValidateGatewayAuth(r.Header); err != nil {
		s.logs.Add(logs.Entry{
			Level:      logs.LevelWarn,
			Method:     r.Method,
			Path:       r.URL.RequestURI(),
			ClientKey:  route.ClientKey,
			ClientName: route.ClientName,
			Status:     http.StatusUnauthorized,
			Message:    err.Error(),
		})
		s.recordHistory(r, route, nil, http.StatusUnauthorized, time.Since(start).Milliseconds(), token.TokenConsumption{}, logs.LevelWarn, err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return nil, false
	}
	routes, err := claudedesktop.LoadRoutes()
	if err != nil {
		s.recordHistory(r, route, nil, http.StatusServiceUnavailable, time.Since(start).Milliseconds(), token.TokenConsumption{}, logs.LevelError, err.Error())
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return nil, false
	}
	updatedBody, _, err := claudedesktop.RewriteRequestBody(body, routes)
	if err != nil {
		s.recordHistory(r, route, nil, http.StatusBadRequest, time.Since(start).Milliseconds(), token.TokenConsumption{}, logs.LevelWarn, err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil, false
	}
	return updatedBody, true
}

func (s *Service) proxyHTTPWithRetries(w http.ResponseWriter, r *http.Request, route routeInfo, body []byte, start time.Time) {
	excluded := map[string]bool{}
	attempts := s.attemptsForRoute(route)
	retryChain := make([]history.RetryAttempt, 0, attempts)
	lastRoute := route
	var lastErr error
	var lastStatus int

	for attempt := 1; attempt <= attempts; attempt++ {
		attemptStart := time.Now()
		attemptRoute, tokenAttempt := s.prepareCandidateTokenAttempt(r.Context(), r, route, excluded, retryChain, attempt, attemptStart)
		lastRoute = attemptRoute
		retryChain = tokenAttempt.retryChain
		if !tokenAttempt.ready {
			lastErr = tokenAttempt.err
			if tokenAttempt.stop {
				break
			}
			continue
		}
		selected := tokenAttempt.selected

		finishActive, _ := s.beginActiveRequest(r, attemptRoute, selected)
		resp, err := s.forward(r.Context(), r, attemptRoute, body, selected)
		if err != nil {
			finishActive()
			lastErr = err
			_ = s.tokens.RecordProxyUsage(selected.ID, token.TokenConsumption{})
			s.tokens.Release(selected.ID)
			excluded[selected.ID] = true
			s.logs.Add(logs.Entry{
				Level:      logs.LevelWarn,
				Method:     r.Method,
				Path:       r.URL.RequestURI(),
				ClientKey:  attemptRoute.ClientKey,
				ClientName: attemptRoute.ClientName,
				Model:      attemptRoute.Model,
				TokenName:  token.DisplayName(selected),
				Message:    proxyLogMessage(attemptRoute.Model, token.TokenConsumption{}, "upstream request failed, trying next token"),
			})
			retryChain = appendRetryAttempt(retryChain, attempt, attemptRoute, &selected, http.StatusBadGateway, time.Since(attemptStart).Milliseconds(), false, fmt.Sprintf("upstream request failed: %v", err))
			continue
		}

		lastStatus = resp.StatusCode
		if remaining, ok := parseRemaining(resp.Header); ok {
			_ = s.tokens.RecordUsage(selected.ID, remaining)
		} else {
			_ = s.tokens.RecordUsage(selected.ID, -1)
		}

		if s.shouldRetryUpstreamResponse(attemptRoute, selected, resp.StatusCode, attempt, attempts) {
			finishActive()
			switchMessage := upstreamSwitchMessage(attemptRoute, selected, resp.StatusCode)
			retryChain = s.retryUpstreamAttempt(r, attemptRoute, selected, resp.StatusCode, resp.Header, attempt, attemptStart, retryChain, excluded, resp.Body, fmt.Sprintf("upstream returned %d", resp.StatusCode), proxyLogMessage(attemptRoute.Model, token.TokenConsumption{}, switchMessage), switchMessage)
			continue
		}

		consumption, responseBody := s.writeResponse(w, resp)
		finishActive()
		attemptRoute.Model = completedRequestModel(r, attemptRoute.Model, resp.Header, responseBody)
		_ = s.tokens.RecordProxyUsage(selected.ID, consumption)
		cooldownUntil := s.cooldownUntilForUpstreamResponse(attemptRoute, selected, resp.StatusCode, resp.Header)
		cooldownTriggered := cooldownUntil != nil
		if cooldownTriggered {
			_ = s.tokens.MarkExhaustedUntil(selected.ID, fmt.Sprintf("upstream returned %d", resp.StatusCode), cooldownUntil)
		}
		s.tokens.Release(selected.ID)
		historyMessage := proxyHistoryMessage(resp.StatusCode, attemptRoute.Model, consumption, "request proxied", responseBody)
		retryChain = appendRetryAttempt(retryChain, attempt, attemptRoute, &selected, resp.StatusCode, time.Since(attemptStart).Milliseconds(), cooldownTriggered, historyMessage)
		s.logs.Add(logs.Entry{
			Level:      levelForStatus(resp.StatusCode),
			Method:     r.Method,
			Path:       r.URL.RequestURI(),
			ClientKey:  attemptRoute.ClientKey,
			ClientName: attemptRoute.ClientName,
			Model:      attemptRoute.Model,
			Status:     resp.StatusCode,
			Duration:   time.Since(start).Milliseconds(),
			TokenName:  token.DisplayName(selected),
			Message:    proxyLogMessage(attemptRoute.Model, consumption, "request proxied"),
		})
		s.recordHistory(r, attemptRoute, &selected, resp.StatusCode, time.Since(start).Milliseconds(), consumption, levelForStatus(resp.StatusCode), historyMessage, retryChain...)
		return
	}

	s.writeHTTPProxyFailure(w, r, lastRoute, retryChain, lastErr, lastStatus, start)
}

func (s *Service) writeHTTPProxyFailure(w http.ResponseWriter, r *http.Request, lastRoute routeInfo, retryChain []history.RetryAttempt, lastErr error, lastStatus int, start time.Time) {
	status := http.StatusBadGateway
	if errors.Is(lastErr, token.ErrNoActiveToken) {
		status = http.StatusServiceUnavailable
	}
	if lastStatus != 0 {
		status = lastStatus
	}

	s.logs.Add(logs.Entry{
		Level:      logs.LevelError,
		Method:     r.Method,
		Path:       r.URL.RequestURI(),
		ClientKey:  lastRoute.ClientKey,
		ClientName: lastRoute.ClientName,
		Model:      lastRoute.Model,
		Status:     status,
		Duration:   time.Since(start).Milliseconds(),
		Message:    fmt.Sprintf("proxy failed: %v", lastErr),
	})
	if len(retryChain) == 0 {
		retryChain = appendRetryAttempt(retryChain, 1, lastRoute, nil, status, time.Since(start).Milliseconds(), false, fmt.Sprintf("proxy failed: %v", lastErr))
	}
	s.recordHistory(r, lastRoute, nil, status, time.Since(start).Milliseconds(), token.TokenConsumption{}, logs.LevelError, fmt.Sprintf("proxy failed: %v", lastErr), retryChain...)
	http.Error(w, http.StatusText(status), status)
}
