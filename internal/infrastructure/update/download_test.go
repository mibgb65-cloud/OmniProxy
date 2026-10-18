package update

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadExactSizeAndChecksum(t *testing.T) {
	data := []byte(strings.Repeat("fake installer", 1000))
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	name := "OmniProxy-Setup-v2.0.0-beta.2-windows-amd64.exe"
	for _, tc := range []struct {
		name, checksum string
		size           int64
		want           bool
	}{
		{"valid", digest + "  " + name, int64(len(data)), true},
		{"wrong hash", strings.Repeat("0", 64), int64(len(data)), false},
		{"wrong filename", digest + "  evil.exe", int64(len(data)), false},
		{"multiple checksums", digest + "\n" + digest, int64(len(data)), false},
		{"too short", digest, int64(len(data) + 1), false},
		{"too large", digest, int64(len(data) - 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, ".sha256") {
					w.Write([]byte(tc.checksum))
					return
				}
				w.Write(data)
			}))
			defer server.Close()
			p := Package{Name: name, URL: server.URL + "/asset", ChecksumURL: server.URL + "/asset.sha256", Size: tc.size}
			path := filepath.Join(t.TempDir(), "download.exe")
			var progress int64
			got, err := Download(context.Background(), server.Client(), p, path, func(n int64) { progress = n })
			if (err == nil) != tc.want {
				t.Fatalf("error=%v", err)
			}
			if tc.want {
				if got != digest || progress != int64(len(data)) {
					t.Fatal("progress or digest incorrect")
				}
				if err := VerifyFile(path, digest, p.Size); err != nil {
					t.Fatal(err)
				}
				os.WriteFile(path, make([]byte, len(data)), 0600)
				if VerifyFile(path, digest, p.Size) == nil {
					t.Fatal("tampered file accepted")
				}
			}
		})
	}
}

func TestDownloadRedirectAndChecksumBounds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "https://example.com/unsafe", http.StatusFound)
			return
		}
		w.Write([]byte(strings.Repeat("a", 4097)))
	}))
	defer server.Close()
	client := DownloadClient(server.Client(), ValidRedirectURL)
	for _, path := range []string{"/redirect", "/large"} {
		if _, err := Checksum(context.Background(), client, server.URL+path, "test.exe"); err == nil {
			t.Fatalf("accepted %s", path)
		}
	}
	p := Package{Repo: "owner/name", Version: "v2.0.0", Name: "OmniProxy-Setup-v2.0.0-windows-amd64.exe"}
	valid := "https://github.com/owner/name/releases/download/v2.0.0/" + p.Name
	if !ValidAssetURL(valid, p, false) {
		t.Fatal("valid GitHub asset rejected")
	}
	for _, raw := range []string{strings.Replace(valid, "https:", "http:", 1), valid + "?token=secret", strings.Replace(valid, "owner/name", "other/repo", 1), "https://github.com.evil/"} {
		if ValidAssetURL(raw, p, false) {
			t.Fatal("unsafe asset accepted")
		}
	}
}
