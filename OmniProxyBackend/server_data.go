package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"omniproxy/internal/config"
	"omniproxy/internal/history"
	"omniproxy/internal/logs"
	"omniproxy/internal/storage"
	"omniproxy/internal/taskautomation"
	"omniproxy/internal/token"
	"path/filepath"
	"strings"
)

func newControlToken() (string, error) {
	var data [32]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(data[:]), nil
}

func newAppServer() (*appServer, error) {
	controlToken, err := newControlToken()
	if err != nil {
		return nil, fmt.Errorf("generate control token: %w", err)
	}
	recorder := logs.NewRecorder(500)
	defaultCfg := config.Default()
	server := &appServer{
		cfg:            defaultCfg,
		logs:           recorder,
		premProxy:      newPremProxyManager(recorder),
		taskAutomation: taskautomation.NewManager(defaultCfg, recorder),
		controlToken:   controlToken,
		updates:        newUpdateDownloader(),
	}
	info, configured, err := config.ResolveDataDir()
	if err != nil {
		return nil, fmt.Errorf("resolve data directory: %w", err)
	}
	if configured {
		if err := server.loadData(info.DataDir); err != nil {
			return nil, err
		}
	} else {
		server.dataDir = info.DataDir
	}
	return server, nil
}

func (a *appServer) loadData(dataDir string) error {
	dataDir, err := config.PrepareDataDir(dataDir)
	if err != nil {
		return fmt.Errorf("prepare data directory: %w", err)
	}

	cfgStore := config.NewStore(filepath.Join(dataDir, "config.json"))
	cfg, err := cfgStore.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	tokenStore := token.NewSecureStore(storage.NewJSONStore[[]token.Token](filepath.Join(dataDir, "tokens.json")))
	tokenManager, err := token.NewManager(tokenStore, cfg.SwitchThreshold)
	if err != nil {
		return fmt.Errorf("load tokens: %w", err)
	}
	historyStore, err := history.NewSQLiteStore(filepath.Join(dataDir, "request_history.db"))
	if err != nil {
		return fmt.Errorf("open request history database: %w", err)
	}
	if err := migrateLegacyRequestHistory(historyStore, filepath.Join(dataDir, "request_history.json")); err != nil {
		_ = historyStore.Close()
		return fmt.Errorf("migrate request history: %w", err)
	}
	if err := historyStore.RelabelTokenNames(tokenDisplayNamesByID(tokenManager.List())); err != nil {
		_ = historyStore.Close()
		return fmt.Errorf("relabel request history tokens: %w", err)
	}
	historyRecorder, err := history.NewRecorder(historyStore, requestHistoryMax)
	if err != nil {
		_ = historyStore.Close()
		return fmt.Errorf("load request history: %w", err)
	}
	if err := historyRecorder.SetRetentionDays(cfg.HistoryRetentionDays); err != nil {
		_ = historyStore.Close()
		return fmt.Errorf("apply request history retention: %w", err)
	}

	a.mu.Lock()
	a.dataDir = dataDir
	a.cfg = cfg
	a.configStore = cfgStore
	a.tokens = tokenManager
	a.history = historyRecorder
	a.mu.Unlock()
	if a.taskAutomation != nil {
		a.taskAutomation.UpdateConfig(cfg)
	}
	return nil
}

func migrateLegacyRequestHistory(store history.Store, legacyPath string) error {
	populated, err := historyStoreHasEntries(store)
	if err != nil {
		return err
	}
	if populated {
		return nil
	}

	legacyStore := storage.NewJSONStore[[]history.Entry](legacyPath)
	entries, err := legacyStore.Load()
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}
	return store.Save(entries)
}

// historyStoreHasEntries probes for a single row. Loading the table just to ask
// whether it is empty means reading every row and its retry chain at startup.
func historyStoreHasEntries(store history.Store) (bool, error) {
	if query, ok := store.(history.QueryStore); ok {
		entries, err := query.List(history.Filter{}, 1)
		if err != nil {
			return false, err
		}
		return len(entries) > 0, nil
	}
	entries, err := store.Load()
	if err != nil {
		return false, err
	}
	return len(entries) > 0, nil
}

func tokenDisplayNamesByID(items []token.Token) map[string]string {
	out := map[string]string{}
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			continue
		}
		if name := strings.TrimSpace(token.DisplayName(item)); name != "" {
			out[item.ID] = name
		}
	}
	return out
}

func (a *appServer) isLoaded() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.configStore != nil && a.tokens != nil
}

func (a *appServer) updateManager() *updateDownloader {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.updates == nil {
		a.updates = newUpdateDownloader()
	}
	return a.updates
}

func (a *appServer) includePrereleaseUpdates() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg.CheckBetaUpdates
}

func (a *appServer) dataDirectoryInfo() config.DataDirectoryInfo {
	a.mu.Lock()
	dataDir := a.dataDir
	a.mu.Unlock()

	info, _, err := config.ResolveDataDir()
	if err == nil && info.DataDir != "" {
		if dataDir != "" {
			info.DataDir = dataDir
		}
		return info
	}
	return config.DataDirectoryInfo{
		DataDir:       dataDir,
		BootstrapPath: config.BootstrapPath(),
		Source:        "unknown",
	}
}

func (a *appServer) changeDataDirectory(dataDir string, migrate bool) (config.DataDirectoryChangeResult, error) {
	if info, configured, err := config.ResolveDataDir(); err == nil && configured && info.EnvOverride {
		return config.DataDirectoryChangeResult{DataDir: info.DataDir, BootstrapPath: info.BootstrapPath, EnvOverride: true}, errors.New("data directory is controlled by OMNIPROXY_DATA_DIR")
	}

	nextDir, err := config.PrepareDataDir(dataDir)
	if err != nil {
		return config.DataDirectoryChangeResult{}, err
	}

	a.mu.Lock()
	previousDir := a.dataDir
	a.mu.Unlock()

	var copied []string
	var skipped []string
	if migrate && previousDir != "" {
		copied, skipped, err = config.CopyDataFiles(previousDir, nextDir)
		if err != nil {
			return config.DataDirectoryChangeResult{}, err
		}
	}
	if err := config.SaveBootstrap(nextDir); err != nil {
		return config.DataDirectoryChangeResult{}, err
	}

	return config.DataDirectoryChangeResult{
		DataDir:         nextDir,
		PreviousDataDir: previousDir,
		BootstrapPath:   config.BootstrapPath(),
		MigratedFiles:   copied,
		SkippedFiles:    skipped,
		RestartRequired: previousDir != "" && !strings.EqualFold(filepath.Clean(previousDir), filepath.Clean(nextDir)),
	}, nil
}
