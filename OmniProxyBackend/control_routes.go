package main

import (
	"encoding/json"
	"net/http"
	"omniproxy/internal/token"
	"strings"
)

func (a *appServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/control-token", a.handleControlToken)
	mux.HandleFunc("/api/tokens", a.handleTokens)
	mux.HandleFunc("/api/tokens/import-api-keys", a.handleImportAPIKeys)
	mux.HandleFunc("/api/tokens/", a.handleTokenByID)
	mux.HandleFunc("/api/config", a.handleConfig)
	mux.HandleFunc("/api/config/export", a.handleConfigExport)
	mux.HandleFunc("/api/config/import", a.handleConfigImport)
	mux.HandleFunc("/api/config/snapshots", a.handleConfigSnapshots)
	mux.HandleFunc("/api/config/snapshots/", a.handleConfigSnapshotByID)
	mux.HandleFunc("/api/logs", a.handleLogs)
	mux.HandleFunc("/api/gateway/diagnose", a.handleGatewayRouteDiagnostics)
	mux.HandleFunc("/api/models/sync", a.handleProviderModels)
	mux.HandleFunc("/api/diagnostics/bundle", a.handleDiagnosticsBundle)
	mux.HandleFunc("/api/history", a.handleHistory)
	mux.HandleFunc("/api/history/summary", a.handleHistorySummary)
	mux.HandleFunc("/api/history/clear", a.handleHistoryClear)
	mux.HandleFunc("/api/history/rebuild-summaries", a.handleHistorySummariesRebuild)
	mux.HandleFunc("/api/billing/summary", a.handleBillingSummary)
	mux.HandleFunc("/api/billing/usage", a.handleBillingUsage)
	mux.HandleFunc("/api/billing/dates", a.handleBillingDates)
	mux.HandleFunc("/api/billing/clear", a.handleBillingClear)
	mux.HandleFunc("/api/proxy/status", a.handleProxyStatus)
	mux.HandleFunc("/api/proxy/active-requests", a.handleActiveProxyRequests)
	mux.HandleFunc("/api/proxy/start", a.handleProxyStart)
	mux.HandleFunc("/api/proxy/stop", a.handleProxyStop)
	mux.HandleFunc("/api/app/info", a.handleAppInfo)
	mux.HandleFunc("/api/update/check", a.handleUpdateCheck)
	mux.HandleFunc("/api/update/download", a.handleUpdateDownload)
	mux.HandleFunc("/api/update/download/status", a.handleUpdateDownloadStatus)
	mux.HandleFunc("/api/update/diagnostics", a.handleUpdateDiagnostics)
	mux.HandleFunc("/api/update/install", a.handleUpdateInstall)
	mux.HandleFunc("/api/data-directory", a.handleDataDirectory)
	mux.HandleFunc("/api/clients/preview", a.handleClientConfigPreviews)
	mux.HandleFunc("/api/codex/configure", a.handleCodexConfigure)
	mux.HandleFunc("/api/codex/restore", a.handleCodexRestore)
	mux.HandleFunc("/api/codex/login/start", a.handleCodexOAuthLoginStart)
	mux.HandleFunc("/api/codex/login/status", a.handleCodexOAuthLoginStatus)
	mux.HandleFunc("/api/codex/login/complete", a.handleCodexOAuthLoginComplete)
	mux.HandleFunc("/api/claude/models/configure", a.handleClaudeModelsConfigure)
	mux.HandleFunc("/api/claude/restore", a.handleClaudeRestore)
	mux.HandleFunc("/api/claude/desktop/models/configure", a.handleClaudeDesktopModelsConfigure)
	mux.HandleFunc("/api/claude/desktop/restore", a.handleClaudeDesktopRestore)
	mux.HandleFunc("/api/deepseek-tui/configure", a.handleDeepSeekTUIConfigure)
	mux.HandleFunc("/api/deepseek-tui/restore", a.handleDeepSeekTUIRestore)
	mux.HandleFunc("/api/gemini/configure", a.handleGeminiConfigure)
	mux.HandleFunc("/api/gemini/restore", a.handleGeminiRestore)
	mux.HandleFunc("/api/openrouter/models", a.handleOpenRouterModels)
	mux.HandleFunc("/api/openrouter/chat", a.handleOpenRouterChat)
	mux.HandleFunc("/api/opencode/configure", a.handleOpenCodeConfigure)
	mux.HandleFunc("/api/opencode/restore", a.handleOpenCodeRestore)
	mux.HandleFunc("/api/pi/configure", a.handlePiConfigure)
	mux.HandleFunc("/api/pi/restore", a.handlePiRestore)
	return withControlTokenAuth(a.controlToken, mux)
}

func (a *appServer) handleControlToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !isTrustedControlTokenOrigin(r.Header.Get("Origin")) {
		w.Header().Set("Cache-Control", "no-store")
		writeError(w, http.StatusForbidden, "control token is only available to the desktop app")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]string{
		"header": controlTokenHeader,
		"token":  a.controlToken,
	})
}

func (a *appServer) handleTokens(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, tokenResponses(a.tokens.List()))
	case http.MethodPost:
		var req token.UpsertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		item, err := a.createToken(r.Context(), req)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, item)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *appServer) handleImportAPIKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req apiKeyBatchImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	result, err := a.importAPIKeys(req)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	status := http.StatusOK
	if result.CreatedCount > 0 {
		status = http.StatusCreated
	}
	writeJSON(w, status, result)
}

func (a *appServer) handleTokenByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/tokens/"), "/")
	parts := strings.Split(rest, "/")
	id := parts[0]
	if id == "" {
		writeError(w, http.StatusNotFound, "token not found")
		return
	}

	if len(parts) == 2 && parts[1] == "validate" {
		a.handleTokenValidate(w, r, id)
		return
	}
	if len(parts) == 2 && parts[1] == "refresh" {
		a.handleTokenRefresh(w, r, id)
		return
	}
	if len(parts) == 2 && parts[1] == "reset-credit" {
		a.handleCodexResetCredit(w, r, id)
		return
	}
	if len(parts) == 2 && parts[1] == "disabled" {
		a.handleTokenDisabled(w, r, id)
		return
	}
	if len(parts) == 2 && parts[1] == "exclusive" {
		a.handleTokenExclusive(w, r, id)
		return
	}
	if len(parts) == 2 && parts[1] == "selected" {
		a.handleTokenSelected(w, r, id)
		return
	}
	if len(parts) > 1 {
		writeError(w, http.StatusNotFound, "token not found")
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req token.UpsertRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		item, err := a.updateToken(id, req)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := a.deleteToken(id); err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

type tokenDisabledRequest struct {
	Disabled bool `json:"disabled"`
}

type tokenSelectedRequest struct {
	Selected bool `json:"selected"`
}

func (a *appServer) handleTokenDisabled(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req tokenDisabledRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := a.setTokenDisabled(id, req.Disabled)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *appServer) handleTokenExclusive(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPut && r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var (
		items []tokenResponse
		err   error
	)
	if r.Method == http.MethodDelete {
		items, err = a.cancelUseOnlyToken(id)
	} else {
		items, err = a.useOnlyToken(id)
	}
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *appServer) handleTokenSelected(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req tokenSelectedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	items, err := a.setTokenSelected(id, req.Selected)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *appServer) handleTokenValidate(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	result, err := a.validateToken(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, result)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *appServer) handleTokenRefresh(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	item, err := a.refreshAuthTokenResponse(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *appServer) handleCodexResetCredit(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	result, err := a.consumeCodexResetCredit(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *appServer) handleCodexOAuthLoginStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	refresh := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("refresh")), "true")
	result, err := a.startCodexOAuthLogin(refresh)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *appServer) handleCodexOAuthLoginStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	result, err := a.codexOAuthLoginStatus(r.URL.Query().Get("loginId"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *appServer) handleCodexOAuthLoginComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req codexOAuthLoginCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	result, err := a.completeCodexOAuthLogin(r.Context(), req.LoginID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
