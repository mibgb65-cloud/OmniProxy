package netproxy

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDetectIdentifiesMixedPortAsSOCKS5(t *testing.T) {
	proxy := newFakeProxy(t, "mixed")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	detection, err := Detect(ctx, proxy.Endpoint())
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !detection.SOCKS5 || !detection.HTTP {
		t.Fatalf("expected mixed detection, got %#v", detection)
	}
	if detection.Scheme != SchemeSOCKS5 {
		t.Fatalf("mixed port should prefer socks5, got %q", detection.Scheme)
	}
}

func TestDetectIdentifiesPlainHTTPAndPlainSOCKS5(t *testing.T) {
	httpProxy := newFakeProxy(t, "http")
	socksProxy := newFakeProxy(t, "socks5")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	httpDetection, err := Detect(ctx, httpProxy.Endpoint())
	if err != nil || httpDetection.Scheme != SchemeHTTP || httpDetection.SOCKS5 {
		t.Fatalf("http detection = %#v, err = %v", httpDetection, err)
	}
	socksDetection, err := Detect(ctx, socksProxy.Endpoint())
	if err != nil || socksDetection.Scheme != SchemeSOCKS5 || socksDetection.HTTP {
		t.Fatalf("socks detection = %#v, err = %v", socksDetection, err)
	}
}

func TestDetectReportsUnavailableAndMismatch(t *testing.T) {
	proxy := newFakeProxy(t, "mixed")
	endpoint := proxy.Endpoint()
	proxy.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := Detect(ctx, endpoint); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("closed port should be unavailable, got %v", err)
	}

	listener, err := newPlainListener()
	if err != nil {
		t.Fatalf("listener: %v", err)
	}
	defer func() { _ = listener.Close() }()
	if _, detectErr := Detect(ctx, listener.Endpoint()); !errors.Is(detectErr, ErrProtocolMismatch) {
		t.Fatalf("plain listener should mismatch, got %v", detectErr)
	}
}

func TestDiscoverFindsDefaultPorts(t *testing.T) {
	proxy := newFakeProxy(t, "mixed")
	endpoint := proxy.Endpoint()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	found := Discover(ctx, "127.0.0.1", []int{endpoint.Port})
	if len(found) != 1 || found[0].Port != endpoint.Port || found[0].Scheme != SchemeSOCKS5 {
		t.Fatalf("discover = %#v", found)
	}
}
