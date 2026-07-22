package proxy

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"net/url"
	"omniproxy/internal/history"
	"omniproxy/internal/logs"
	"omniproxy/internal/token"
	"strings"
	"time"
)

func (s *Service) serveCodexWebSocket(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	route := routeWithClient(r, s.router.Route(r.URL, nil))

	excluded := map[string]bool{}
	attempts := s.attemptsForRoute(route)

	var selected token.Token
	var upstream *websocket.Conn
	var upstreamResp *http.Response
	var lastErr error
	var lastStatus int
	lastRoute := route
	finishActive := func() {}
	retryChain := make([]history.RetryAttempt, 0, attempts)

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
		selected = tokenAttempt.selected

		targetURL, err := s.router.TargetWebSocketURL(attemptRoute, selected)
		if err != nil {
			lastErr = err
			s.tokens.Release(selected.ID)
			retryChain = appendRetryAttempt(retryChain, attempt, attemptRoute, &selected, 0, time.Since(attemptStart).Milliseconds(), false, err.Error())
			break
		}

		header := websocketRequestHeader(r.Header)
		removeClientIdentificationHeaders(header)
		if err := applyRouteAuth(header, selected, attemptRoute); err != nil {
			lastErr = err
			s.tokens.Release(selected.ID)
			retryChain = appendRetryAttempt(retryChain, attempt, attemptRoute, &selected, 0, time.Since(attemptStart).Milliseconds(), false, err.Error())
			break
		}

		finishActive = s.beginActiveRequest(r, attemptRoute, selected)
		dialer := websocket.Dialer{
			HandshakeTimeout:  45 * time.Second,
			Proxy:             s.proxyForRoute(attemptRoute),
			Subprotocols:      websocket.Subprotocols(r),
			EnableCompression: true,
		}
		upstream, upstreamResp, err = dialer.DialContext(r.Context(), targetURL, header)
		if err == nil {
			break
		}
		finishActive()

		lastErr = err
		if upstreamResp != nil {
			lastStatus = upstreamResp.StatusCode
		}
		if s.shouldRetryUpstreamResponse(attemptRoute, selected, lastStatus, attempt, attempts) {
			switchMessage := upstreamWebSocketSwitchMessage(attemptRoute, selected, lastStatus)
			retryChain = s.retryUpstreamAttempt(r, attemptRoute, selected, lastStatus, responseHeaders(upstreamResp), attempt, attemptStart, retryChain, excluded, upstreamRespBody(upstreamResp), fmt.Sprintf("upstream websocket returned %d", lastStatus), switchMessage, switchMessage)
			continue
		}
		retryChain = appendRetryAttempt(retryChain, attempt, attemptRoute, &selected, lastStatus, time.Since(attemptStart).Milliseconds(), false, fmt.Sprintf("upstream websocket failed: %v", err))
		s.tokens.Release(selected.ID)
		break
	}

	if upstream == nil {
		finishActive()
		status := http.StatusBadGateway
		if errors.Is(lastErr, token.ErrNoActiveToken) {
			status = http.StatusServiceUnavailable
		}
		if lastStatus != 0 {
			status = lastStatus
		}
		closeBody(upstreamRespBody(upstreamResp))
		s.logs.Add(logs.Entry{
			Level:      logs.LevelError,
			Method:     r.Method,
			Path:       r.URL.RequestURI(),
			ClientKey:  lastRoute.ClientKey,
			ClientName: lastRoute.ClientName,
			Model:      lastRoute.Model,
			Status:     status,
			Duration:   time.Since(start).Milliseconds(),
			Message:    fmt.Sprintf("websocket proxy failed: %v", lastErr),
		})
		if len(retryChain) == 0 {
			retryChain = appendRetryAttempt(retryChain, 1, lastRoute, nil, status, time.Since(start).Milliseconds(), false, fmt.Sprintf("websocket proxy failed: %v", lastErr))
		}
		s.recordHistory(r, lastRoute, nil, status, time.Since(start).Milliseconds(), token.TokenConsumption{}, logs.LevelError, fmt.Sprintf("websocket proxy failed: %v", lastErr), retryChain...)
		http.Error(w, http.StatusText(status), status)
		return
	}
	defer upstream.Close()
	closeBody(upstreamRespBody(upstreamResp))

	responseHeader := http.Header{}
	if subprotocol := upstream.Subprotocol(); subprotocol != "" {
		responseHeader.Set("Sec-Websocket-Protocol", subprotocol)
	}
	upgrader := websocket.Upgrader{
		CheckOrigin:       isAllowedWebSocketOrigin,
		EnableCompression: true,
	}
	client, err := upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		finishActive()
		s.tokens.Release(selected.ID)
		s.logs.Add(logs.Entry{
			Level:      logs.LevelError,
			Method:     r.Method,
			Path:       r.URL.RequestURI(),
			ClientKey:  lastRoute.ClientKey,
			ClientName: lastRoute.ClientName,
			Model:      lastRoute.Model,
			Status:     http.StatusBadRequest,
			Duration:   time.Since(start).Milliseconds(),
			TokenName:  token.DisplayName(selected),
			Message:    fmt.Sprintf("websocket client upgrade failed: %v", err),
		})
		s.recordHistory(r, lastRoute, &selected, http.StatusBadRequest, time.Since(start).Milliseconds(), token.TokenConsumption{}, logs.LevelError, fmt.Sprintf("websocket client upgrade failed: %v", err))
		return
	}
	defer client.Close()

	_ = s.tokens.RecordUsage(selected.ID, -1)
	_ = s.tokens.RecordProxyRequest(selected.ID)
	consumption, responseModel, err := proxyWebSocketMessages(client, upstream, func(usage token.TokenConsumption) {
		_ = s.tokens.RecordProxyConsumption(selected.ID, usage)
	})
	finishActive()
	s.tokens.Release(selected.ID)
	if responseModel != "" {
		lastRoute.Model = responseModel
	}

	level := logs.LevelInfo
	message := proxyLogMessage(lastRoute.Model, consumption, "websocket proxied")
	if err != nil && !isNormalWebSocketClose(err) {
		level = logs.LevelWarn
		message = fmt.Sprintf("websocket closed with error: %v", err)
	}
	s.logs.Add(logs.Entry{
		Level:      level,
		Method:     r.Method,
		Path:       r.URL.RequestURI(),
		ClientKey:  lastRoute.ClientKey,
		ClientName: lastRoute.ClientName,
		Model:      lastRoute.Model,
		Status:     http.StatusSwitchingProtocols,
		Duration:   time.Since(start).Milliseconds(),
		TokenName:  token.DisplayName(selected),
		Message:    message,
	})
	if len(retryChain) == 0 || retryChain[len(retryChain)-1].Status != http.StatusSwitchingProtocols {
		retryChain = appendRetryAttempt(retryChain, len(retryChain)+1, lastRoute, &selected, http.StatusSwitchingProtocols, time.Since(start).Milliseconds(), false, message)
	}
	s.recordHistory(r, lastRoute, &selected, http.StatusSwitchingProtocols, time.Since(start).Milliseconds(), consumption, level, message, retryChain...)
}

func websocketRequestHeader(src http.Header) http.Header {
	dst := http.Header{}
	for key, values := range src {
		if isWebSocketRequestHeader(key) {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
	return dst
}

func isAllowedWebSocketOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" && scheme != "wails" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "wails.localhost"
}

func isWebSocketRequestHeader(key string) bool {
	switch http.CanonicalHeaderKey(key) {
	case "Connection",
		"Host",
		"Origin",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Proxy-Connection",
		"Sec-Websocket-Accept",
		"Sec-Websocket-Extensions",
		"Sec-Websocket-Key",
		"Sec-Websocket-Protocol",
		"Sec-Websocket-Version",
		"Te",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade":
		return true
	default:
		return false
	}
}

func proxyWebSocketMessages(client *websocket.Conn, upstream *websocket.Conn, onUsage func(token.TokenConsumption)) (token.TokenConsumption, string, error) {
	resultCh := make(chan websocketCopyResult, 2)
	go func() {
		result := copyWebSocketMessages(upstream, client, false, nil)
		result.fromUpstream = false
		resultCh <- result
	}()
	go func() {
		result := copyWebSocketMessages(client, upstream, true, onUsage)
		result.fromUpstream = true
		resultCh <- result
	}()

	first := <-resultCh
	closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	deadline := time.Now().Add(time.Second)
	_ = client.WriteControl(websocket.CloseMessage, closeMessage, deadline)
	_ = upstream.WriteControl(websocket.CloseMessage, closeMessage, deadline)
	_ = client.Close()
	_ = upstream.Close()
	second := <-resultCh
	responseModel := ""
	requestModel := ""
	for _, result := range []websocketCopyResult{first, second} {
		if result.fromUpstream && result.model != "" {
			responseModel = result.model
		}
		if !result.fromUpstream && result.model != "" {
			requestModel = result.model
		}
	}
	if responseModel == "" {
		responseModel = requestModel
	}
	return addTokenConsumption(first.consumption, second.consumption), responseModel, first.err
}

type websocketCopyResult struct {
	consumption  token.TokenConsumption
	model        string
	fromUpstream bool
	err          error
}

func copyWebSocketMessages(dst *websocket.Conn, src *websocket.Conn, captureUsage bool, onUsage func(token.TokenConsumption)) websocketCopyResult {
	var total token.TokenConsumption
	model := ""
	for {
		messageType, reader, err := src.NextReader()
		if err != nil {
			return websocketCopyResult{consumption: total, model: model, err: err}
		}
		writer, err := dst.NextWriter(messageType)
		if err != nil {
			return websocketCopyResult{consumption: total, model: model, err: err}
		}
		target := io.Writer(writer)
		capture := &usageCapture{}
		if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
			target = io.MultiWriter(writer, capture)
		}
		_, copyErr := io.Copy(target, reader)
		closeErr := writer.Close()
		if copyErr != nil {
			return websocketCopyResult{consumption: total, model: model, err: copyErr}
		}
		if closeErr != nil {
			return websocketCopyResult{consumption: total, model: model, err: closeErr}
		}
		if capture.buf.Len() > 0 {
			header := http.Header{"Content-Type": []string{"application/json"}}
			if parsedModel := parseResponseModel(header, capture.Bytes()); parsedModel != "" {
				model = parsedModel
			}
			if captureUsage {
				usage := parseTokenConsumption(header, capture.Bytes())
				if tokenConsumptionAvailable(usage) {
					total = addTokenConsumption(total, usage)
					if onUsage != nil {
						onUsage(usage)
					}
				}
			}
		}
	}
}

func tokenConsumptionAvailable(usage token.TokenConsumption) bool {
	return usage.InputTokens > 0 || usage.OutputTokens > 0 || usage.TotalTokens > 0 ||
		usage.CacheCreationTokens > 0 || usage.CacheReadTokens > 0
}

func addTokenConsumption(left token.TokenConsumption, right token.TokenConsumption) token.TokenConsumption {
	return token.TokenConsumption{
		InputTokens:         left.InputTokens + right.InputTokens,
		OutputTokens:        left.OutputTokens + right.OutputTokens,
		TotalTokens:         left.TotalTokens + right.TotalTokens,
		CacheCreationTokens: left.CacheCreationTokens + right.CacheCreationTokens,
		CacheReadTokens:     left.CacheReadTokens + right.CacheReadTokens,
	}
}

func isNormalWebSocketClose(err error) bool {
	return websocket.IsCloseError(err,
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseNoStatusReceived)
}
