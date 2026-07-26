package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"omniproxy/internal/claudedesktop"
	"omniproxy/internal/config"
	"omniproxy/internal/history"
	"omniproxy/internal/logs"
	"omniproxy/internal/token"
	"sort"
	"sync"
	"time"
)

type Service struct {
	cfg            config.Config
	tokens         *token.Manager
	logs           *logs.Recorder
	history        *history.Recorder
	router         Router
	retry          RetryPolicy
	client         *http.Client
	proxyClient    *http.Client
	proxyURL       *url.URL
	tokenRefresher TokenRefresher
	zoModelsMu     sync.Mutex
	zoModelsCache  map[string]zoModelCacheEntry
	activeMu       sync.RWMutex
	activeSeq      int64
	activeRequests map[int64]ActiveRequest
	activity       ActivityObserver
	wsMu           sync.Mutex
	wsSessions     map[io.Closer]struct{}
	wsClosed       bool
}

const maxProxyRequestBodyBytes = 32 * 1024 * 1024
const maxProxyResponseBodyBytes = 32 * 1024 * 1024

var errRequestBodyTooLarge = errors.New("request body too large")
var errResponseBodyTooLarge = errors.New("upstream response body too large")

type TokenRefresher func(context.Context, token.Token, bool) (token.Token, bool, error)

type ActiveRequest struct {
	ID         int64     `json:"id"`
	StartedAt  time.Time `json:"startedAt"`
	ClientKey  string    `json:"clientKey,omitempty"`
	ClientName string    `json:"clientName,omitempty"`
	Method     string    `json:"method,omitempty"`
	Path       string    `json:"path,omitempty"`
	Provider   string    `json:"provider,omitempty"`
	Protocol   string    `json:"protocol,omitempty"`
	Model      string    `json:"model,omitempty"`
	TokenID    string    `json:"tokenId,omitempty"`
	TokenName  string    `json:"tokenName,omitempty"`
}

type ActivityObserver interface {
	ActiveRequestStarted(ActiveRequest)
	ActiveRequestFinished(ActiveRequest)
}

type tokenAttemptOutcome struct {
	selected   token.Token
	retryChain []history.RetryAttempt
	ready      bool
	stop       bool
	err        error
}

func NewService(cfg config.Config, tokens *token.Manager, recorder *logs.Recorder, historyRecorders ...*history.Recorder) (*Service, error) {
	cfg = config.Normalize(cfg)
	if err := ValidateProxyBaseURLs(cfg); err != nil {
		return nil, err
	}
	outboundProxy, err := outboundProxyURL(cfg)
	if err != nil {
		return nil, err
	}

	var requestHistory *history.Recorder
	if len(historyRecorders) > 0 {
		requestHistory = historyRecorders[0]
	}

	var proxyClient *http.Client
	if outboundProxy != nil {
		proxyClient = newHTTPClient(0, outboundProxy)
	}

	return &Service{
		cfg:         cfg,
		tokens:      tokens,
		logs:        recorder,
		history:     requestHistory,
		router:      NewRouter(cfg),
		retry:       NewRetryPolicy(cfg),
		client:      newHTTPClient(0, nil),
		proxyClient: proxyClient,
		proxyURL:    outboundProxy,
	}, nil
}

func (s *Service) SetTokenRefresher(refresher TokenRefresher) {
	s.tokenRefresher = refresher
}

func (s *Service) SetActivityObserver(observer ActivityObserver) {
	s.activity = observer
}

func (s *Service) ActiveRequests() []ActiveRequest {
	s.activeMu.RLock()
	defer s.activeMu.RUnlock()

	out := make([]ActiveRequest, 0, len(s.activeRequests))
	for _, item := range s.activeRequests {
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].StartedAt.Before(out[j].StartedAt)
	})
	return out
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.serveSpecialProxyRequest(w, r) {
		return
	}
	start := time.Now()
	bodyBytes, ok := s.readHTTPProxyBody(w, r, start)
	if !ok {
		return
	}
	bodyBytes, ok = s.rewriteClaudeDesktopMessageBody(w, r, bodyBytes, start)
	if !ok {
		return
	}
	route := routeWithClient(r, s.router.Route(r.URL, bodyBytes))
	s.proxyHTTPWithRetries(w, r, route, bodyBytes, start)
}

func (s *Service) serveClaudeDesktopModels(w http.ResponseWriter, r *http.Request) {
	route := routeWithClient(r, routeInfo{Provider: token.ProviderAnthropic, Protocol: "anthropic", Path: r.URL.Path})
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
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
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	routes, err := claudedesktop.LoadRoutes()
	if err != nil {
		s.logs.Add(logs.Entry{
			Level:      logs.LevelError,
			Method:     r.Method,
			Path:       r.URL.RequestURI(),
			ClientKey:  route.ClientKey,
			ClientName: route.ClientName,
			Status:     http.StatusServiceUnavailable,
			Message:    err.Error(),
		})
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	if err := json.NewEncoder(w).Encode(claudedesktop.ModelListResponse(routes)); err != nil {
		s.logs.Add(logs.Entry{
			Level:      logs.LevelError,
			Method:     r.Method,
			Path:       r.URL.RequestURI(),
			ClientKey:  route.ClientKey,
			ClientName: route.ClientName,
			Status:     http.StatusOK,
			Message:    fmt.Sprintf("write Claude Desktop models response: %v", err),
		})
	}
}

func (s *Service) forward(ctx context.Context, original *http.Request, route routeInfo, body []byte, selected token.Token) (*http.Response, error) {
	if isCodexChatCompletionsRoute(route, selected) {
		return s.forwardCodexChatCompletions(ctx, original, route, body, selected)
	}
	if token.NormalizeProvider(route.Provider) == token.ProviderZo {
		return s.forwardZo(ctx, original, route, body, selected)
	}

	targetURL, err := s.router.TargetURL(route, selected)
	if err != nil {
		return nil, err
	}
	if updatedBody, changed := normalizeSub2APIRequestBody(original.URL.Path, route, body); changed {
		body = updatedBody
	}
	if updatedBody, changed := normalizeRequestBodyModel(body, route.Model); changed {
		body = updatedBody
	}

	req, err := http.NewRequestWithContext(ctx, original.Method, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	copyHeader(req.Header, original.Header)
	removeHopHeaders(req.Header)
	removeClientIdentificationHeaders(req.Header)
	if err := applyRouteAuth(req.Header, selected, route); err != nil {
		return nil, err
	}
	req.Host = req.URL.Host

	return s.clientForRoute(route).Do(req)
}

func (s *Service) clientForRoute(route routeInfo) *http.Client {
	if s.proxyClient != nil && outboundProxyMatchesRoute(route, s.cfg) {
		return s.proxyClient
	}
	return s.client
}

func (s *Service) proxyForRoute(route routeInfo) func(*http.Request) (*url.URL, error) {
	if s.proxyURL != nil && outboundProxyMatchesRoute(route, s.cfg) {
		return http.ProxyURL(s.proxyURL)
	}
	return http.ProxyFromEnvironment
}
