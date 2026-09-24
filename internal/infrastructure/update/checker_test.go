package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLatestReturnsRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/name/releases/latest" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != "application/vnd.github+json" {
			t.Errorf("Accept header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"tag_name": "v1.2.0",
			"html_url": "https://github.com/owner/name/releases/tag/v1.2.0",
			"published_at": "2026-09-01T10:00:00Z",
			"body": "notes"
		}`))
	}))
	defer server.Close()

	checker := &Checker{Repo: "owner/name", BaseURL: server.URL}
	release, err := checker.Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest error: %v", err)
	}
	if release.TagName != "v1.2.0" {
		t.Errorf("TagName = %q, want v1.2.0", release.TagName)
	}
	if release.HTMLURL == "" {
		t.Error("HTMLURL should not be empty")
	}
	want := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	if !release.PublishedAt.Equal(want) {
		t.Errorf("PublishedAt = %v, want %v", release.PublishedAt, want)
	}
}

func TestLatestRejectsEmptyRepo(t *testing.T) {
	checker := &Checker{Repo: "  "}
	if _, err := checker.Latest(context.Background()); err == nil {
		t.Fatal("expected error for empty repo")
	}
}

func TestLatestHandlesNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	checker := &Checker{Repo: "owner/name", BaseURL: server.URL}
	if _, err := checker.Latest(context.Background()); err == nil {
		t.Fatal("expected error for 404 (no releases)")
	}
}

func TestLatestHandlesServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	checker := &Checker{Repo: "owner/name", BaseURL: server.URL}
	if _, err := checker.Latest(context.Background()); err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestLatestHandlesMalformedPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name": ""}`))
	}))
	defer server.Close()

	checker := &Checker{Repo: "owner/name", BaseURL: server.URL}
	if _, err := checker.Latest(context.Background()); err == nil {
		t.Fatal("expected error for a release without a tag")
	}
}

func TestLatestHandlesUnreachableServer(t *testing.T) {
	checker := &Checker{Repo: "owner/name", BaseURL: "http://127.0.0.1:1", Client: &http.Client{Timeout: time.Second}}
	if _, err := checker.Latest(context.Background()); err == nil {
		t.Fatal("expected error for an unreachable server")
	}
}
