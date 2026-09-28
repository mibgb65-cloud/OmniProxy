package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestBuiltInProbeEndpointsAreValid(t *testing.T) {
	for _, definition := range Catalog() {
		if _, err := validateBaseURL(definition.DefaultBaseURL); err != nil {
			t.Errorf("%s has an invalid endpoint: %v", definition.Kind, err)
		}
		if len(definition.Models) < 2 {
			t.Errorf("%s has no expandable model preview", definition.Kind)
		}
	}
}

func TestProbeTreatsAuthenticationResponseAsReachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead || r.Header.Get("Authorization") != "" {
			t.Errorf("probe sent unexpected method or credentials")
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	service := testService(t)
	item, err := service.Create(context.Background(), CreateInput{Name: "Local endpoint", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Probe(context.Background(), item.ID)
	if err != nil || !result.Reachable || result.StatusCode != http.StatusUnauthorized || result.LatencyMs < 0 {
		t.Fatalf("Probe = (%+v, %v)", result, err)
	}
}

func TestProbeDoesNotFollowRedirects(t *testing.T) {
	var followed atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		followed.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer source.Close()
	service := testService(t)
	item, err := service.Create(context.Background(), CreateInput{Name: "Redirect", BaseURL: source.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Probe(context.Background(), item.ID)
	if err != nil || !result.Reachable || result.StatusCode != http.StatusFound || followed.Load() != 0 {
		t.Fatalf("Probe redirect = (%+v, %v), followed=%d", result, err, followed.Load())
	}
}

func TestProbeReportsConnectionFailureWithoutCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	endpoint := server.URL
	server.Close()
	service := testService(t)
	item, err := service.Create(context.Background(), CreateInput{Name: "Unavailable", BaseURL: endpoint})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Probe(context.Background(), item.ID)
	if err != nil || result.Reachable || result.StatusCode != 0 {
		t.Fatalf("Probe closed endpoint = (%+v, %v)", result, err)
	}
}
