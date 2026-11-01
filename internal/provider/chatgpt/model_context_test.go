package chatgpt

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestModelContextWindowsUsesAuthenticatedFixedEndpoint(t *testing.T) {
	var sent *http.Request
	client := New(transportFunc(func(r *http.Request) (*http.Response, error) {
		sent = r
		if r.Method != http.MethodGet || r.URL.String() != "https://chatgpt.com/backend-api/codex/models?client_version=0.166.0-alpha.5.1" ||
			r.Header.Get("Authorization") != "Bearer qa-only" || r.Header.Get("ChatGPT-Account-ID") != "qa-account" {
			t.Error("model discovery endpoint or scoped identity changed")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(
			`{"models":[{"slug":"gpt-test","context_window":500000,"max_context_window":872000},{"slug":"unknown","max_context_window":1000000}]}`))}, nil
	}))
	windows, err := client.ModelContextWindows(context.Background(), []byte("qa-only"), "qa-account", "0.166.0-alpha.5.1")
	if err != nil || len(windows) != 1 || windows["gpt-test"] != 500000 {
		t.Fatal("default capacity lost or maximum capacity assumed", err)
	}
	if sent.Header.Get("Authorization") != "" || sent.Header.Get("ChatGPT-Account-ID") != "" {
		t.Fatal("credential headers retained")
	}
}

func TestModelContextCatalogRejectsInvalidAndAmbiguousCapacities(t *testing.T) {
	for _, body := range []string{
		`{`, `{}`, `{"models":null}`, `{"models":[]}`,
		`{"models":[{"slug":"bad id","context_window":1000}]}`,
		`{"models":[{"slug":"qa","context_window":0}]}`,
		`{"models":[{"slug":"qa","context_window":-1}]}`,
		`{"models":[{"slug":"qa","context_window":10000001}]}`,
		`{"models":[{"slug":"qa","context_window":1.5}]}`,
		`{"models":[{"slug":"qa","context_window":"500000"}]}`,
		`{"models":[{"slug":"qa","context_window":500000},{"slug":"qa","context_window":1000000}]}`,
		`{"models":[` + strings.Repeat(`{"slug":"qa"},`, 1000) + `{"slug":"last"}]}`,
	} {
		if _, err := parseModelContextWindows([]byte(body)); err == nil {
			t.Fatal("invalid model capacity accepted")
		}
	}
}

func TestModelContextWindowsBoundsFailuresAndNeverLeaksResponse(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
		err    error
	}{
		{"auth", 401, "private-upstream-value", nil},
		{"redirect", 302, "private-upstream-value", nil},
		{"oversize", 200, strings.Repeat("x", maxModelCatalogBytes+1), nil},
		{"transport", 0, "", errors.New("private-upstream-value")},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := New(transportFunc(func(*http.Request) (*http.Response, error) {
				if test.err != nil {
					return nil, test.err
				}
				return &http.Response{StatusCode: test.status, Body: io.NopCloser(strings.NewReader(test.body))}, nil
			}))
			_, err := client.ModelContextWindows(context.Background(), []byte("qa-only"), "qa-account", "0.163.0")
			if err == nil || strings.Contains(err.Error(), "private") {
				t.Fatal("failure missing or upstream details leaked", err)
			}
		})
	}
}

func TestModelContextWindowsRejectsInvalidRequestBeforeTransport(t *testing.T) {
	client := New(transportFunc(func(*http.Request) (*http.Response, error) {
		t.Error("invalid input reached transport")
		return nil, nil
	}))
	for _, test := range []struct{ key, account, version string }{
		{"", "qa", "0.163.0"}, {"qa\ninjected", "qa", "0.163.0"},
		{"qa", "", "0.163.0"}, {"qa", "qa\r\ninjected", "0.163.0"},
		{"qa", "qa", ""}, {"qa", "qa", "0.163.0&override=1"},
	} {
		if _, err := client.ModelContextWindows(context.Background(), []byte(test.key), test.account, test.version); err == nil {
			t.Fatal("invalid discovery request accepted")
		}
	}
}
