package wails

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/application/updater"
	"omniproxy/internal/infrastructure/update"
	"omniproxy/internal/platform/profile"
)

func TestCheckForUpdateAutomaticDownloadPolicy(t *testing.T) {
	for _, tc := range []struct {
		name               string
		kind               profile.Kind
		enabled, installed bool
		want               string
	}{
		{"default off", profile.Production, false, true, "idle"},
		{"opt in", profile.Production, true, true, "ready"},
		{"development", profile.Development, true, true, "idle"},
		{"test", profile.Test, true, true, "idle"},
		{"portable", profile.Production, true, false, "idle"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, prof := newTestService(t)
			prof.Kind = tc.kind
			data := []byte("verified installer fixture")
			name, _ := update.InstallerName("v99.0.0", "windows", "amd64")
			digest := fmt.Sprintf("%x", sha256.Sum256(data))
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.HasSuffix(r.URL.Path, ".sha256"):
					fmt.Fprintf(w, "%s  %s", digest, name)
				case strings.HasPrefix(r.URL.Path, "/assets/"):
					w.Write(data)
				default:
					json.NewEncoder(w).Encode(update.Release{TagName: "v99.0.0", Assets: []update.Asset{
						{Name: name, URL: server.URL + "/assets/" + name, Size: int64(len(data))},
						{Name: name + ".sha256", URL: server.URL + "/assets/" + name + ".sha256", Size: 128},
					}})
				}
			}))
			defer server.Close()
			if err := profile.SaveConfig(tc.kind, profile.Config{UpdateRepo: "owner/name", UpdateChannel: "stable", AutoDownloadUpdates: tc.enabled}); err != nil {
				t.Fatal(err)
			}
			launched := false
			svc.updates = updater.New(updater.Options{
				Directory: t.TempDir(), Version: "2.0.0", OS: "windows", Arch: "amd64", CanInstall: tc.installed,
				Client: server.Client(), Checker: func(repo string, beta bool) *update.Checker { return &update.Checker{Repo: repo, BaseURL: server.URL} },
				ValidPackage: func(p update.Package) bool {
					return strings.HasPrefix(p.URL, server.URL+"/assets/") && strings.HasPrefix(p.ChecksumURL, server.URL+"/assets/")
				},
				Launch: func(string, string, string, string, int64) error { launched = true; return nil },
			})
			defer svc.updates.Close(context.Background())
			result := svc.CheckForUpdate()
			if result.State != "available" {
				t.Fatalf("check=%+v", result)
			}
			deadline := time.Now().Add(3 * time.Second)
			state := svc.updates.Snapshot()
			for tc.want == "ready" && state.Download.State != "ready" && time.Now().Before(deadline) {
				time.Sleep(time.Millisecond)
				state = svc.updates.Snapshot()
			}
			if state.Download.State != tc.want || launched {
				t.Fatalf("state=%+v installed=%v", state, launched)
			}
		})
	}
}

func TestUpdateChannelCannotChangeDuringCheck(t *testing.T) {
	svc, prof := newTestService(t)
	// 显式偏好必须保留，不依赖当前构建恰好是 Beta 还是正式版。
	if err := profile.SaveConfig(prof.Kind, profile.Config{UpdateChannel: "beta"}); err != nil {
		t.Fatal(err)
	}
	started, release := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		json.NewEncoder(w).Encode([]update.Release{{TagName: "v99.0.0"}})
	}))
	defer server.Close()
	defer func() { close(release) }()
	svc.updates = updater.New(updater.Options{Directory: t.TempDir(), Version: "2.0.0", OS: "windows", Arch: "amd64", Checker: func(repo string, beta bool) *update.Checker {
		return &update.Checker{Repo: repo, BaseURL: server.URL, IncludePrereleases: beta}
	}})
	go svc.CheckForUpdate()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("check did not start")
	}
	if _, err := svc.SetUpdateChannel("stable"); err == nil {
		t.Fatal("channel changed while a release check was in flight")
	}
	prefs, err := svc.GetUpdatePreferences()
	if err != nil || prefs.Channel != "beta" {
		t.Fatalf("preferences changed: %+v %v", prefs, err)
	}
}
