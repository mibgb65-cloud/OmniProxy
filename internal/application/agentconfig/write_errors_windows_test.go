//go:build windows

package agentconfig

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type interruptedConfigFile struct {
	*os.File
	failed, permanent bool
}

func (f *interruptedConfigFile) WriteAt(body []byte, offset int64) (int, error) {
	if !f.failed || f.permanent {
		f.failed = true
		n, err := f.File.WriteAt(body[:len(body)/2], offset)
		return n, errors.Join(err, errors.New("synthetic write interruption"))
	}
	return f.File.WriteAt(body, offset)
}

func TestEncryptedWriteFailureRestoresOriginalBytes(t *testing.T) {
	for _, permanent := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "config.json")
		before := []byte(`{"theme":"dark","settings":"retained"}`)
		if err := os.WriteFile(path, before, 0600); err != nil {
			t.Fatal(err)
		}
		f, err := os.OpenFile(path, os.O_RDWR, 0600)
		if err != nil {
			t.Fatal(err)
		}
		writer := &interruptedConfigFile{File: f, permanent: permanent}
		writeErr := rewriteWithRollback(writer, []byte(`{"theme":"light"}`), before)
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		if permanent {
			if writeErr == nil || !strings.HasPrefix(writeErr.Error(), "agent: rollback_failed") {
				t.Fatal("failed rollback was not surfaced", writeErr)
			}
			continue
		}
		if writeErr == nil || writeErr.Error() != "agent: write_failed" {
			t.Fatal("interruption accepted", writeErr)
		}
		current, err := os.ReadFile(path)
		if err != nil || string(current) != string(before) {
			t.Fatal("partial write damaged original settings")
		}
	}
}
