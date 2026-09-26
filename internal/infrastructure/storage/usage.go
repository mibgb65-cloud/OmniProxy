package storage

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Usage is a snapshot of regular-file sizes in the application data directory.
type Usage struct {
	Database int64
	Logs     int64
	Cache    int64
	Other    int64
}

// Total returns the sum of all measured categories.
func (u Usage) Total() int64 {
	return u.Database + u.Logs + u.Cache + u.Other
}

// MeasureUsage groups files in one walk, so every counted file belongs to
// exactly one category. A missing data directory has zero usage.
func MeasureUsage(dataDir, databasePath, logDir string) (Usage, error) {
	var usage Usage
	databaseName := filepath.Base(databasePath)
	logName := filepath.Base(logDir)
	err := filepath.WalkDir(dataDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if path == dataDir && !entry.IsDir() {
			return fmt.Errorf("%s is not a directory", dataDir)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(dataDir, path)
		if err != nil {
			return err
		}
		size := info.Size()
		switch {
		case inSubdir(rel, logName):
			usage.Logs += size
		case inSubdir(rel, "cache"):
			usage.Cache += size
		case isDatabaseFile(rel, databaseName):
			usage.Database += size
		default:
			usage.Other += size
		}
		return nil
	})
	if err != nil {
		return Usage{}, fmt.Errorf("storage: measure usage of %s: %w", dataDir, err)
	}
	return usage, nil
}

func inSubdir(rel, name string) bool {
	return strings.HasPrefix(rel, name+string(filepath.Separator))
}

func isDatabaseFile(rel, name string) bool {
	switch rel {
	case name, name + "-wal", name + "-shm", name + "-journal":
		return true
	default:
		return false
	}
}
