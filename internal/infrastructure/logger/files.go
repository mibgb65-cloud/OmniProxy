package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Files maintains the on-disk log files created by Setup. It is safe to call
// its methods on the zero value (console-only logging).
type Files struct {
	dir    string
	path   string
	active *rotatingFile
}

// Close closes the active log file. It is safe to call more than once.
func (f *Files) Close() error {
	if f == nil || f.active == nil {
		return nil
	}
	return f.active.Close()
}

// Clean removes rotated log backups and truncates the active log file. It
// returns the number of bytes freed.
func (f *Files) Clean() (int64, error) {
	if f == nil || f.active == nil {
		return 0, nil
	}

	var freed int64
	entries, err := os.ReadDir(f.dir)
	if err != nil {
		return 0, fmt.Errorf("logger: read log dir: %w", err)
	}
	prefix := filepath.Base(f.path) + "."
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		if _, err := strconv.Atoi(strings.TrimPrefix(entry.Name(), prefix)); err != nil {
			continue // only numbered backups (e.g. omniproxy.log.1)
		}
		info, err := entry.Info()
		if err != nil {
			return freed, fmt.Errorf("logger: stat backup: %w", err)
		}
		if err := os.Remove(filepath.Join(f.dir, entry.Name())); err != nil {
			return freed, fmt.Errorf("logger: remove backup: %w", err)
		}
		freed += info.Size()
	}

	truncated, err := f.active.Truncate()
	return freed + truncated, err
}
