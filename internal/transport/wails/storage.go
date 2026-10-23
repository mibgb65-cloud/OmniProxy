package wails

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"omniproxy/internal/application/updater"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/infrastructure/storage"
	"omniproxy/internal/platform/profile"
)

// Clean targets accepted by AppService.CleanStorage.
const (
	cleanTargetLogs     = "logs"
	cleanTargetDatabase = "database"
	cleanTargetUpdates  = "updates"
)

// StorageItem describes one storage location and its size.
type StorageItem struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

// StorageInfo is a snapshot of data storage usage for the settings page.
type StorageInfo struct {
	DataDir        string        `json:"dataDir"`
	LogDir         string        `json:"logDir"`
	DatabasePath   string        `json:"databasePath"`
	PendingDataDir string        `json:"pendingDataDir"`
	Items          []StorageItem `json:"items"`
	TotalBytes     int64         `json:"totalBytes"`
}

// StorageInfo returns the current data storage locations and their sizes.
func (s *AppService) StorageInfo() StorageInfo {
	dbPath := s.prof.DatabasePath()
	usage, err := storage.MeasureUsage(s.prof.DataDir, dbPath, s.prof.LogDir)
	if err != nil {
		s.log.Warn("failed to measure data storage usage", "error", err)
	}

	info := StorageInfo{
		DataDir:      s.prof.DataDir,
		LogDir:       s.prof.LogDir,
		DatabasePath: dbPath,
		Items: []StorageItem{
			{Name: cleanTargetDatabase, Path: dbPath, Bytes: usage.Database},
			{Name: cleanTargetLogs, Path: s.prof.LogDir, Bytes: usage.Logs},
			{Name: storage.DirCache, Path: filepath.Join(s.prof.DataDir, storage.DirCache), Bytes: usage.Cache},
			{Name: storage.DirUpdates, Path: filepath.Join(s.prof.DataDir, storage.DirUpdates), Bytes: usage.Updates},
			{Name: storage.DirCredentials, Path: filepath.Join(s.prof.DataDir, storage.DirCredentials), Bytes: usage.Credentials},
			{Name: "other", Path: s.prof.DataDir, Bytes: usage.Other},
		},
		TotalBytes: usage.Total(),
	}
	if pending, err := profile.LoadConfig(s.prof.Kind); err == nil &&
		pending.DataDir != "" && !samePath(pending.DataDir, s.prof.DataDir) {
		info.PendingDataDir = pending.DataDir
	}
	return info
}

// SetDataDirResult reports the outcome of a data-directory change.
type SetDataDirResult struct {
	PreviousDir      string `json:"previousDir"`
	NewDir           string `json:"newDir"`
	RequiresRestart  bool   `json:"requiresRestart"`
	ExistingDataSize int64  `json:"existingDataSize"`
}

// SetDataDir validates and persists a new data directory. The change takes
// effect on the next start; the data itself is moved at shutdown.
func (s *AppService) SetDataDir(path string) (SetDataDirResult, error) {
	existing, err := storage.DirSize(s.prof.DataDir)
	if err != nil {
		s.log.Warn("failed to measure existing data size", "error", err)
	}
	s.settingsMu.Lock()
	cfg, err := profile.SetDataDir(s.prof.Kind, s.prof.DataDir, path)
	s.settingsMu.Unlock()
	if err != nil {
		return SetDataDirResult{}, err
	}
	return SetDataDirResult{
		PreviousDir:      s.prof.DataDir,
		NewDir:           cfg.DataDir,
		RequiresRestart:  true,
		ExistingDataSize: existing,
	}, nil
}

// SelectDirectory opens a native folder picker and returns the chosen path.
// An empty string means the user cancelled. The title is provided by the
// caller so the dialog follows the UI language.
func (s *AppService) SelectDirectory(title string) (string, error) {
	app := application.Get()
	if app == nil {
		return "", errors.New("wails: application instance is not available")
	}
	path, err := app.Dialog.OpenFile().
		SetTitle(title).
		CanChooseDirectories(true).
		CanChooseFiles(false).
		CanCreateDirectories(true).
		SetDirectory(s.prof.DataDir).
		PromptForSingleSelection()
	if err != nil {
		// The native dialog reports cancellation as an error with no path.
		// Treat it as a cancel (the UI does nothing), but keep it in the log
		// so genuine dialog failures are not silently swallowed.
		s.log.Debug("folder selection ended without a result", "error", err)
		return "", nil
	}
	return path, nil
}

// CleanResult reports the space freed by a cleanup operation.
type CleanResult struct {
	Target     string `json:"target"`
	FreedBytes int64  `json:"freedBytes"`
}

// CleanStorage runs a cleanup operation: "logs" removes rotated log backups
// and truncates the active log; "database" vacuums the SQLite database;
// "updates" removes the helper copies and temporary files left by install
// attempts without touching the downloaded installer cache.
func (s *AppService) CleanStorage(target string) (CleanResult, error) {
	switch target {
	case cleanTargetLogs:
		freed, err := s.logFiles.Clean()
		if err != nil {
			return CleanResult{}, err
		}
		s.log.Info("cleaned log files", "freed_bytes", freed)
		return CleanResult{Target: target, FreedBytes: freed}, nil
	case cleanTargetUpdates:
		freed, err := updater.CleanCache(filepath.Join(s.prof.DataDir, storage.DirUpdates))
		if err != nil {
			return CleanResult{}, err
		}
		s.log.Info("cleaned update leftovers", "freed_bytes", freed)
		return CleanResult{Target: target, FreedBytes: freed}, nil
	case cleanTargetDatabase:
		before, err := storage.FileSize(s.prof.DatabasePath())
		if err != nil {
			return CleanResult{}, err
		}
		if err := database.Vacuum(context.Background(), s.db); err != nil {
			return CleanResult{}, err
		}
		after, err := storage.FileSize(s.prof.DatabasePath())
		if err != nil {
			return CleanResult{}, err
		}
		freed := before - after
		if freed < 0 {
			freed = 0
		}
		s.log.Info("vacuumed database", "freed_bytes", freed)
		return CleanResult{Target: target, FreedBytes: freed}, nil
	default:
		return CleanResult{}, fmt.Errorf("wails: unknown clean target %q", target)
	}
}

// relocateIfPending moves the data directory to the persisted location when
// it differs from the active one. Handles are closed first so files can be
// moved; a failure reverts the persisted setting so the next start keeps
// using the current location instead of an empty one.
func (s *AppService) relocateIfPending() {
	cfg, err := profile.LoadConfig(s.prof.Kind)
	if err != nil {
		s.log.Error("failed to load config for relocation", "error", err)
		return
	}
	target := cfg.DataDir
	if target == "" || samePath(target, s.prof.DataDir) {
		return
	}

	s.log.Info("relocating data directory", "from", s.prof.DataDir, "to", target)
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.log.Error("failed to close database before relocation", "error", err)
		}
	}
	if err := s.logFiles.Close(); err != nil {
		s.log.Error("failed to close log files before relocation", "error", err)
	}

	if err := storage.MoveDir(s.prof.DataDir, target); err != nil {
		s.log.Error("data directory relocation failed; reverting setting", "error", err)
		if rerr := profile.RevertDataDir(s.prof.Kind); rerr != nil {
			s.log.Error("failed to revert data directory setting", "error", rerr)
		}
		return
	}
	s.log.Info("data directory relocated", "to", target)
}
