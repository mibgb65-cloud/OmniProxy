package provider

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

const probeTimeout = 6 * time.Second

// ProbeResult reports endpoint reachability and time to HTTP response headers.
// It does not authenticate an account or make a model request.
type ProbeResult struct {
	Reachable  bool  `json:"reachable"`
	LatencyMs  int64 `json:"latencyMs"`
	StatusCode int   `json:"statusCode"`
}

// Probe checks an endpoint without sending credentials or following redirects.
func (s *Service) Probe(ctx context.Context, id string) (ProbeResult, error) {
	if err := validateID(id); err != nil {
		return ProbeResult{}, err
	}
	item, err := s.get(ctx, id)
	if err != nil {
		return ProbeResult{}, err
	}
	endpoint := item.BaseURL
	if item.Source == SourceBuiltIn {
		definition, ok := definitionForKind(item.Kind)
		if !ok {
			return ProbeResult{}, fmt.Errorf("provider: unknown built-in endpoint")
		}
		endpoint = definition.DefaultBaseURL
	}
	endpoint, err = validateBaseURL(endpoint)
	if err != nil || endpoint == "" {
		return ProbeResult{}, fmt.Errorf("%w: no valid test endpoint", ErrInvalidInput)
	}

	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, endpoint, nil)
	if err != nil {
		return ProbeResult{}, fmt.Errorf("provider: build probe: %w", err)
	}
	transport := &http.Transport{
		Proxy:                  http.ProxyFromEnvironment,
		DialContext:            (&net.Dialer{Timeout: 3 * time.Second}).DialContext,
		TLSHandshakeTimeout:    3 * time.Second,
		ResponseHeaderTimeout:  5 * time.Second,
		MaxResponseHeaderBytes: 16 << 10,
		DisableKeepAlives:      true,
		DisableCompression:     true,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport: transport,
		Timeout:   probeTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	start := time.Now()
	response, err := client.Do(req)
	result := ProbeResult{LatencyMs: time.Since(start).Milliseconds()}
	if err != nil {
		if ctx.Err() != nil && ctx.Err() != context.DeadlineExceeded {
			return result, ctx.Err()
		}
		return result, nil // failed connection is the measured outcome
	}
	result.Reachable = true
	result.StatusCode = response.StatusCode
	if err := response.Body.Close(); err != nil {
		return result, fmt.Errorf("provider: close probe response: %w", err)
	}
	return result, nil
}
