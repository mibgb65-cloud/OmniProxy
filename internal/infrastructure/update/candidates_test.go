package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBetaCandidatesPaginateAndUseVersionOrder(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/repos/owner/name/releases" || r.URL.Query().Get("per_page") != "100" {
			t.Errorf("unexpected URL %s", r.URL)
		}
		if r.URL.Query().Get("page") == "1" {
			items := make([]Release, 100)
			for i := range items {
				items[i] = Release{TagName: "v1.3.2"}
			}
			items[0] = Release{TagName: "v99.0.0", Draft: true}
			items[1] = Release{TagName: "v2.0.0-beta.2", Prerelease: true}
			items[2] = Release{TagName: "v999.0.0-beta.01", Prerelease: true}
			json.NewEncoder(w).Encode(items)
		} else {
			json.NewEncoder(w).Encode([]Release{{TagName: "v2.0.0-beta.10", Prerelease: true, HTMLURL: "javascript:alert(1)"}, {TagName: "v2.0.0-beta.1", Prerelease: true}})
		}
	}))
	defer server.Close()
	checker := &Checker{Repo: "owner/name", BaseURL: server.URL, IncludePrereleases: true}
	result, err := checker.Latest(context.Background())
	if err != nil || result.TagName != "v2.0.0-beta.10" || requests != 2 {
		t.Fatalf("result=%+v requests=%d err=%v", result, requests, err)
	}
	if result.HTMLURL != "https://github.com/owner/name/releases/tag/v2.0.0-beta.10" {
		t.Fatal("untrusted URL escaped normalization")
	}
}

func TestCandidateStablePromotionAndFiltering(t *testing.T) {
	for _, tc := range []struct {
		release    Release
		beta, want bool
	}{
		{Release{TagName: "v2.0.0-beta.1", Prerelease: true}, false, false},
		{Release{TagName: "v2.0.0-beta.1"}, false, false},
		{Release{TagName: "v2.0.0"}, true, true},
		{Release{TagName: "v2.0.0", Draft: true}, true, false},
		{Release{TagName: "nightly"}, true, false},
	} {
		if candidate(tc.release, tc.beta) != tc.want {
			t.Errorf("candidate=%+v beta=%t", tc.release, tc.beta)
		}
	}
}

func TestRejectMalformedRepositoriesAndResponses(t *testing.T) {
	for _, repo := range []string{"", "owner/name/extra", "../repo", "owner/name?token=secret", "owner/with space"} {
		if validRepo(repo) {
			t.Errorf("accepted %q", repo)
		}
	}
	for _, payload := range []string{`{}`, `null`, `[{"tag_name":"garbage"}]`, `[{"tag_name":"v9.0.0","draft":true}]`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(payload)) }))
		_, err := (&Checker{Repo: "owner/name", BaseURL: server.URL, IncludePrereleases: true}).Latest(context.Background())
		server.Close()
		if err == nil {
			t.Errorf("accepted %s", payload)
		}
	}
}
