package netproxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func installFake(t *testing.T, resolver Resolver) {
	t.Helper()
	Install(resolver)
	t.Cleanup(func() { Install(nil) })
}

func newSnapshot(endpoint Endpoint, modes map[string]string) Snapshot {
	return Snapshot{
		Version:  "v1",
		Enabled:  true,
		Endpoint: endpoint,
		NoProxy:  []string{"localhost", "127.0.0.1", "::1"},
		Modes:    modes,
	}
}

func get(t *testing.T, client *http.Client, url string, scope Scope) (string, error) {
	t.Helper()
	request, err := http.NewRequestWithContext(WithProvider(context.Background(), scope.ProviderID, scope.ProviderKind), http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func TestTransportRoutesOnlySelectedProviderThroughProxy(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()
	proxy := newFakeProxy(t, "mixed")
	resolver := &fakeResolver{}
	resolver.Set(newSnapshot(proxy.Endpoint(), map[string]string{
		"prov_proxy":  ModeProxy,
		"prov_direct": ModeDirect,
	}))
	installFake(t, resolver)
	client := NewClient(5 * time.Second)

	body, err := get(t, client, upstream.URL, Scope{ProviderID: "prov_proxy", ProviderKind: "openai"})
	if err != nil || body != "ok" {
		t.Fatalf("proxied request: body=%q err=%v", body, err)
	}
	if proxy.Hits() == 0 {
		t.Fatal("expected the selected provider to go through the proxy")
	}
	before := proxy.Hits()

	body, err = get(t, client, upstream.URL, Scope{ProviderID: "prov_direct", ProviderKind: "deepseek"})
	if err != nil || body != "ok" {
		t.Fatalf("direct request: body=%q err=%v", body, err)
	}
	if proxy.Hits() != before {
		t.Fatal("direct provider must not use the proxy")
	}
}

func TestTransportDetectsMixedPortAndRecordsScheme(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()
	proxy := newFakeProxy(t, "mixed")
	resolver := &fakeResolver{}
	resolver.Set(newSnapshot(proxy.Endpoint(), map[string]string{"prov_proxy": ModeProxy}))
	installFake(t, resolver)

	body, err := get(t, NewClient(5*time.Second), upstream.URL, Scope{ProviderID: "prov_proxy"})
	if err != nil || body != "ok" {
		t.Fatalf("request: body=%q err=%v", body, err)
	}
	detections := resolver.Detections()
	if len(detections) != 1 || detections[0] != SchemeSOCKS5 {
		t.Fatalf("expected detection to be recorded as socks5, got %#v", detections)
	}
}

func TestTransportCorrectsWrongProtocol(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()
	// 端口只会说 SOCKS5，但配置里写死了 HTTP：必须自动纠偏。
	proxy := newFakeProxy(t, "socks5")
	resolver := &fakeResolver{}
	snapshot := newSnapshot(proxy.Endpoint(), map[string]string{"prov_proxy": ModeProxy})
	snapshot.Detected = SchemeHTTP
	resolver.Set(snapshot)
	installFake(t, resolver)

	body, err := get(t, NewClient(5*time.Second), upstream.URL, Scope{ProviderID: "prov_proxy"})
	if err != nil || body != "ok" {
		t.Fatalf("request after correction: body=%q err=%v", body, err)
	}
	detections := resolver.Detections()
	if len(detections) != 1 || detections[0] != SchemeSOCKS5 {
		t.Fatalf("expected corrected scheme to persist, got %#v", detections)
	}
}

func TestTransportReportsProxyUnavailableWithoutTouchingUpstream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()
	proxy := newFakeProxy(t, "mixed")
	endpoint := proxy.Endpoint()
	proxy.Close()

	resolver := &fakeResolver{}
	resolver.Set(newSnapshot(endpoint, map[string]string{"prov_proxy": ModeProxy}))
	installFake(t, resolver)

	_, err := get(t, NewClient(3*time.Second), upstream.URL, Scope{ProviderID: "prov_proxy"})
	if err == nil {
		t.Fatal("expected an error for a closed proxy port")
	}
	if code := CodeOf(err); code != "proxy_unavailable" {
		t.Fatalf("expected proxy_unavailable, got %q (%v)", code, err)
	}
}

func TestTransportRejectsUnconfiguredExplicitProxy(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()
	resolver := &fakeResolver{}
	resolver.Set(Snapshot{Version: "v1", Modes: map[string]string{"prov_proxy": ModeProxy}})
	installFake(t, resolver)

	_, err := get(t, NewClient(3*time.Second), upstream.URL, Scope{ProviderID: "prov_proxy"})
	if code := CodeOf(err); code != "proxy_not_configured" {
		t.Fatalf("expected proxy_not_configured, got %q (%v)", code, err)
	}
}

func TestWrapWithoutResolverKeepsBase(t *testing.T) {
	Install(nil)
	base := http.DefaultTransport
	if got := Wrap(base); got != base {
		t.Fatalf("Wrap without resolver must return base unchanged")
	}
}
