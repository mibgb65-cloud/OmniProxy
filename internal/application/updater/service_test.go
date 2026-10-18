package updater

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/infrastructure/update"
)

func fixture(t *testing.T, slow bool) (*Service, Options, *int) {
	t.Helper()
	data := []byte(strings.Repeat("test-package", 4096))
	name, _ := update.InstallerName("v2.0.0-beta.2", "windows", "amd64")
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".sha256") {
			fmt.Fprintf(w, "%s  %s", digest, name)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			if slow {
				w.Header().Set("Content-Length", fmt.Sprint(len(data)))
				w.WriteHeader(200)
				w.(http.Flusher).Flush()
				<-r.Context().Done()
				return
			}
			w.Write(data)
			return
		}
		json.NewEncoder(w).Encode([]update.Release{{TagName: "v2.0.0-beta.2", Prerelease: true, Body: "notes", Assets: []update.Asset{{Name: name, URL: server.URL + "/assets/" + name, Size: int64(len(data))}, {Name: name + ".sha256", URL: server.URL + "/assets/" + name + ".sha256", Size: 128}}}})
	}))
	t.Cleanup(server.Close)
	launches := 0
	opts := Options{Directory: t.TempDir(), Version: "2.0.0-beta.1", OS: "windows", Arch: "amd64", CanInstall: true, Client: server.Client(), Checker: func(repo string, beta bool) *update.Checker {
		return &update.Checker{Repo: repo, BaseURL: server.URL, IncludePrereleases: beta}
	}, ValidPackage: func(p update.Package) bool {
		return strings.HasPrefix(p.URL, server.URL+"/assets/") && strings.HasPrefix(p.ChecksumURL, server.URL+"/assets/")
	}, Launch: func(string, string, string, string, int64) error { launches++; return nil }}
	s := New(opts)
	s.Configure("owner/name", true)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		s.Close(ctx)
	})
	return s, opts, &launches
}

func waitTransfer(t *testing.T, s *Service, want string) Snapshot {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		state := s.Snapshot()
		if state.Download.State == want {
			return state
		}
		if state.Download.State == "failed" && want != "failed" {
			t.Fatalf("failed: %+v", state)
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("state=%+v, wanted %s", s.Snapshot(), want)
	return Snapshot{}
}

func TestDiscoverDownloadResumeAndLaunch(t *testing.T) {
	s, opts, launches := fixture(t, false)
	state, err := s.Check(context.Background())
	if err != nil || state.Release == nil || !state.Release.PackageAvailable {
		t.Fatalf("check=%+v %v", state, err)
	}
	if _, err := s.Start("v99.0.0"); err == nil {
		t.Fatal("unconfirmed version accepted")
	}
	if _, err := s.Start(state.Release.Version); err != nil {
		t.Fatal(err)
	}
	ready := waitTransfer(t, s, "ready")
	if ready.Download.Percent != 100 || ready.Download.Received != ready.Download.Total {
		t.Fatal("progress incomplete")
	}
	encoded, _ := json.Marshal(ready)
	if strings.Contains(string(encoded), "package-") || strings.Contains(string(encoded), "checksumUrl") {
		t.Fatal("internal paths leaked")
	}
	restored := New(opts)
	if err := restored.Configure("owner/name", true); err != nil {
		t.Fatal(err)
	}
	if restored.Snapshot().Download.State != "ready" {
		t.Fatal("verified download was not restored")
	}
	if _, err := restored.Install(state.Release.Version); err != nil || *launches != 1 {
		t.Fatalf("install=%v launches=%d", err, *launches)
	}
}

func TestCancellationAndBusyGuard(t *testing.T) {
	s, _, launches := fixture(t, true)
	state, err := s.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Start(state.Release.Version); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Start(state.Release.Version); err == nil {
		t.Fatal("parallel download accepted")
	}
	if err = s.Configure("owner/name", false); err == nil {
		t.Fatal("channel switched during download")
	}
	if _, err = s.Cancel(); err != nil {
		t.Fatal(err)
	}
	waitTransfer(t, s, "cancelled")
	if *launches != 0 {
		t.Fatal("cancelled download launched")
	}
}

func TestTamperingChannelAndElevationFailure(t *testing.T) {
	s, _, launches := fixture(t, false)
	state, err := s.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	s.Start(state.Release.Version)
	waitTransfer(t, s, "ready")
	if err := s.Configure("owner/name", false); err != nil {
		t.Fatal(err)
	}
	if s.Snapshot().Release != nil {
		t.Fatal("Beta download appeared in stable channel")
	}
	s.Configure("owner/name", true)
	s.opts.Launch = func(string, string, string, string, int64) error { return errors.New("update: elevation_cancelled") }
	if _, err := s.Install(state.Release.Version); err == nil || s.Snapshot().Download.State != "ready" {
		t.Fatal("cancelled elevation did not retain ready state")
	}
	path := filepath.Join(s.opts.Directory, s.record.File)
	os.WriteFile(path, make([]byte, s.record.Package.Size), 0600)
	if _, err := s.Install(state.Release.Version); err == nil || *launches != 0 {
		t.Fatal("tampered package was launched")
	}
}

func TestPersistedPathCannotEscapeCache(t *testing.T) {
	s, opts, _ := fixture(t, false)
	state, _ := s.Check(context.Background())
	s.Start(state.Release.Version)
	waitTransfer(t, s, "ready")
	s.record.File = "../outside.exe"
	data, _ := json.Marshal(s.record)
	os.WriteFile(filepath.Join(opts.Directory, "status.json"), data, 0600)
	other := New(opts)
	if other.Snapshot().Download.State != "failed" {
		t.Fatal("unsafe persisted path accepted")
	}
}

func TestCachedHashIsNotTrustedDuringInstall(t *testing.T) {
	s, _, launches := fixture(t, false)
	state, _ := s.Check(context.Background())
	s.Start(state.Release.Version)
	waitTransfer(t, s, "ready")
	path := filepath.Join(s.opts.Directory, s.record.File)
	replacement := make([]byte, s.record.Package.Size)
	if err := os.WriteFile(path, replacement, 0600); err != nil {
		t.Fatal(err)
	}
	s.record.Digest = fmt.Sprintf("%x", sha256.Sum256(replacement))
	if _, err := s.Install(state.Release.Version); err == nil || *launches != 0 {
		t.Fatal("local cache hash was incorrectly used as trust root")
	}
}
