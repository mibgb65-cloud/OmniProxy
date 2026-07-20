package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"omniproxy/internal/config"
	"omniproxy/internal/history"
	"omniproxy/internal/logs"
	"omniproxy/internal/proxy"
	"strconv"
	"strings"
	"time"
)

func (a *appServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.mu.Lock()
		cfg := a.cfg
		a.mu.Unlock()
		writeJSON(w, http.StatusOK, cfg)
	case http.MethodPut:
		var cfg config.Config
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		saved, err := a.saveConfig(cfg)
		if err != nil {
			if isConfigValidationError(err) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, saved)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *appServer) saveConfig(cfg config.Config) (config.Config, error) {
	a.configMu.Lock()
	defer a.configMu.Unlock()

	cfg = config.Normalize(cfg)
	if err := a.validateConfigForSave(cfg); err != nil {
		return config.Config{}, err
	}

	a.mu.Lock()
	oldCfg := a.cfg
	shouldRestartProxy := a.proxyServer != nil && proxyConfigChanged(oldCfg, cfg)
	shouldRestartControl := a.control != nil && controlConfigChanged(oldCfg, cfg)
	oldProxyServer := a.proxyServer
	a.cfg = cfg
	a.mu.Unlock()

	if a.configStore != nil {
		if err := a.configStore.Save(cfg); err != nil {
			a.restoreConfigInMemory(oldCfg)
			return config.Config{}, err
		}
	}

	var preparedControlServer *http.Server
	var preparedControlListener net.Listener
	if shouldRestartProxy && shouldRestartControl {
		var err error
		preparedControlServer, preparedControlListener, err = a.newControlServer(cfg)
		if err != nil {
			return config.Config{}, a.rollbackConfigChange(oldCfg, false, err)
		}
	}

	if shouldRestartProxy {
		if err := a.restartProxy(); err != nil {
			if preparedControlListener != nil {
				_ = preparedControlListener.Close()
			}
			return config.Config{}, a.rollbackConfigChange(oldCfg, a.proxyServerChanged(oldProxyServer), err)
		}
	}
	if shouldRestartControl {
		if preparedControlListener != nil {
			a.activateControlServer(preparedControlServer, preparedControlListener, cfg.ControlPort)
		} else {
			if err := a.restartControl(); err != nil {
				return config.Config{}, a.rollbackConfigChange(oldCfg, a.proxyServerChanged(oldProxyServer), err)
			}
		}
	}
	if a.tokens != nil {
		a.tokens.SetThreshold(cfg.SwitchThreshold)
	}
	if a.taskAutomation != nil {
		a.taskAutomation.UpdateConfig(cfg)
	}
	if a.history != nil {
		if err := a.history.SetRetentionDays(cfg.HistoryRetentionDays); err != nil {
			if a.logs != nil {
				a.logs.Add(logs.Entry{
					Level:   logs.LevelWarn,
					Message: fmt.Sprintf("apply request history retention failed: %v", err),
				})
			}
		}
	}
	a.syncPremProxyAfterConfigChange(oldCfg, cfg)

	if a.logs != nil {
		a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: "configuration updated"})
	}
	return cfg, nil
}

func (a *appServer) restoreConfigInMemory(cfg config.Config) {
	a.mu.Lock()
	a.cfg = cfg
	a.mu.Unlock()
}

func (a *appServer) proxyServerChanged(old *http.Server) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.proxyServer != old
}

func (a *appServer) rollbackConfigChange(oldCfg config.Config, restoreProxy bool, cause error) error {
	a.restoreConfigInMemory(oldCfg)
	rollbackErrs := []error{cause}
	if restoreProxy {
		if err := a.restartProxy(); err != nil {
			rollbackErrs = append(rollbackErrs, fmt.Errorf("restore proxy after config failure: %w", err))
		}
	}
	if a.configStore != nil {
		if err := a.configStore.Save(oldCfg); err != nil {
			rollbackErrs = append(rollbackErrs, fmt.Errorf("restore persisted config after config failure: %w", err))
		}
	}
	return errors.Join(rollbackErrs...)
}

func (a *appServer) validateConfigForSave(cfg config.Config) error {
	if err := validateConfiguredPorts(cfg); err != nil {
		return err
	}
	if _, err := proxy.NewService(cfg, a.tokens, a.logs, a.history); err != nil {
		return err
	}
	if _, err := proxy.NewValidator(cfg); err != nil {
		return err
	}
	return nil
}

func (a *appServer) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, logResponses(a.logs.List()))
}

func (a *appServer) handleOpenRouterModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	refresh := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("refresh")), "true") ||
		strings.TrimSpace(r.URL.Query().Get("refresh")) == "1"
	result, err := a.openRouterModels(r.Context(), refresh)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *appServer) handleOpenRouterChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	var req openRouterChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	result, err := a.openRouterChat(r.Context(), req)
	if err != nil {
		var requestErr openRouterRequestError
		if errors.As(err, &requestErr) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *appServer) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	a.mu.Lock()
	recorder := a.history
	a.mu.Unlock()
	if recorder == nil {
		writeJSON(w, http.StatusOK, []historyResponse{})
		return
	}

	limit := defaultHistoryLimit
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	filter := history.Filter{
		Provider: strings.TrimSpace(r.URL.Query().Get("provider")),
		Client:   strings.TrimSpace(r.URL.Query().Get("client")),
		Level:    strings.TrimSpace(r.URL.Query().Get("level")),
		Status:   strings.TrimSpace(r.URL.Query().Get("status")),
		Model:    strings.TrimSpace(r.URL.Query().Get("model")),
		TokenID:  strings.TrimSpace(r.URL.Query().Get("tokenId")),
		Token:    strings.TrimSpace(r.URL.Query().Get("token")),
		Search:   strings.TrimSpace(r.URL.Query().Get("search")),
		Limit:    limit,
	}
	writeJSON(w, http.StatusOK, historyResponses(recorder.List(filter)))
}

func (a *appServer) handleHistorySummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	a.mu.Lock()
	recorder := a.history
	a.mu.Unlock()
	if recorder == nil {
		writeJSON(w, http.StatusOK, history.Summary{})
		return
	}

	days := 14
	if raw := strings.TrimSpace(r.URL.Query().Get("days")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			days = parsed
		}
	}
	filter := history.Filter{
		Provider: strings.TrimSpace(r.URL.Query().Get("provider")),
		Client:   strings.TrimSpace(r.URL.Query().Get("client")),
		Level:    strings.TrimSpace(r.URL.Query().Get("level")),
		Status:   strings.TrimSpace(r.URL.Query().Get("status")),
		Model:    strings.TrimSpace(r.URL.Query().Get("model")),
		TokenID:  strings.TrimSpace(r.URL.Query().Get("tokenId")),
		Token:    strings.TrimSpace(r.URL.Query().Get("token")),
		Search:   strings.TrimSpace(r.URL.Query().Get("search")),
	}
	writeJSON(w, http.StatusOK, recorder.Summary(filter, days))
}

func (a *appServer) handleHistoryClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	a.mu.Lock()
	recorder := a.history
	a.mu.Unlock()
	if recorder == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := recorder.ClearRequestHistory(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: "request history cleared"})
	w.WriteHeader(http.StatusNoContent)
}

func (a *appServer) handleBillingUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	a.mu.Lock()
	recorder := a.history
	a.mu.Unlock()
	if recorder == nil {
		writeJSON(w, http.StatusOK, []history.DailyUsage{})
		return
	}
	date := strings.TrimSpace(r.URL.Query().Get("date"))
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	writeJSON(w, http.StatusOK, recorder.DailyUsage(date))
}

func (a *appServer) handleBillingSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	a.mu.Lock()
	recorder := a.history
	a.mu.Unlock()
	if recorder == nil {
		writeJSON(w, http.StatusOK, history.BillingSummary{})
		return
	}
	days := 30
	if raw := strings.TrimSpace(r.URL.Query().Get("days")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			days = parsed
		}
	}
	writeJSON(w, http.StatusOK, recorder.BillingSummary(days))
}

func (a *appServer) handleBillingDates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	a.mu.Lock()
	recorder := a.history
	a.mu.Unlock()
	if recorder == nil {
		writeJSON(w, http.StatusOK, []string{})
		return
	}
	limit := 30
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	writeJSON(w, http.StatusOK, recorder.DailyUsageDates(limit))
}

func (a *appServer) handleBillingClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	a.mu.Lock()
	recorder := a.history
	a.mu.Unlock()
	if recorder == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := recorder.ClearDailyUsage(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: "billing daily usage cleared"})
	w.WriteHeader(http.StatusNoContent)
}

func (a *appServer) handleHistorySummariesRebuild(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	a.mu.Lock()
	recorder := a.history
	a.mu.Unlock()
	if recorder == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := recorder.RebuildSummaries(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: "history summaries rebuilt"})
	w.WriteHeader(http.StatusNoContent)
}

func (a *appServer) handleProxyStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"running": a.proxyServer != nil,
		"port":    a.cfg.ProxyPort,
	})
}

func (a *appServer) handleActiveProxyRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, activeRequestResponses(a.activeProxyRequests()))
}

func (a *appServer) handleProxyStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := a.startProxy(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"running": true})
}

func (a *appServer) handleProxyStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := a.stopProxy(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"running": false})
}

func (a *appServer) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	info, err := checkForUpdates(r.Context(), http.DefaultClient, a.includePrereleaseUpdates())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (a *appServer) handleUpdateDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req updateDownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	status, err := a.updateManager().Start(context.Background(), http.DefaultClient, req)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, status)
}

func (a *appServer) handleUpdateDownloadStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, a.updateManager().Status())
}

func (a *appServer) handleUpdateDiagnostics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, currentUpdateDiagnostics(a.updateManager().Status()))
}

func (a *appServer) handleUpdateInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	status, err := a.updateManager().Install()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (a *appServer) handleAppInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, currentAppInfo())
}

func (a *appServer) handleDataDirectory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, a.dataDirectoryInfo())
}
