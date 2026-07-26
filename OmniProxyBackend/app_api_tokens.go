package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"omniproxy/internal/logs"
	"omniproxy/internal/token"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *DesktopApp) ControlAPI() string {
	a.server.mu.Lock()
	defer a.server.mu.Unlock()
	return "http://127.0.0.1:" + intToString(a.server.cfg.ControlPort) + "/api"
}

func (a *DesktopApp) Tokens() []tokenResponse {
	return tokenResponses(a.server.tokens.List())
}

func (a *DesktopApp) CreateToken(req token.UpsertRequest) (tokenResponse, error) {
	return a.server.createToken(a.callContext(), req)
}

func (a *DesktopApp) UpdateToken(id string, req token.UpsertRequest) (tokenResponse, error) {
	return a.server.updateToken(id, req)
}

func (a *DesktopApp) DeleteToken(id string) error {
	return a.server.deleteToken(id)
}

func (a *DesktopApp) SetTokenDisabled(id string, disabled bool) (tokenResponse, error) {
	return a.server.setTokenDisabled(id, disabled)
}

func (a *DesktopApp) UseOnlyToken(id string) ([]tokenResponse, error) {
	return a.server.useOnlyToken(id)
}

func (a *DesktopApp) CancelUseOnlyToken(id string) ([]tokenResponse, error) {
	return a.server.cancelUseOnlyToken(id)
}

func (a *DesktopApp) SetTokenSelected(id string, selected bool) ([]tokenResponse, error) {
	return a.server.setTokenSelected(id, selected)
}

func (a *DesktopApp) ValidateToken(id string) (validationResponse, error) {
	return a.server.validateToken(a.callContext(), id)
}

func (a *DesktopApp) RefreshTokenAuth(id string) (tokenResponse, error) {
	return a.server.refreshAuthTokenResponse(a.callContext(), id)
}

func (a *DesktopApp) ConsumeCodexResetCredit(id string) (codexResetCreditConsumeResponse, error) {
	return a.server.consumeCodexResetCredit(a.callContext(), id)
}

func (a *DesktopApp) StartCodexOAuthLogin(refresh bool) (codexOAuthLoginStartResponse, error) {
	return a.server.startCodexOAuthLogin(refresh)
}

func (a *DesktopApp) CodexOAuthLoginStatus(loginID string) (codexOAuthLoginStatusResponse, error) {
	return a.server.codexOAuthLoginStatus(loginID)
}

func (a *DesktopApp) CompleteCodexOAuthLogin(loginID string) (tokenResponse, error) {
	return a.server.completeCodexOAuthLogin(a.callContext(), loginID)
}

func (a *DesktopApp) StartClaudeOAuthLogin(refresh bool) (claudeOAuthLoginStartResponse, error) {
	return a.server.startClaudeOAuthLogin(refresh)
}

func (a *DesktopApp) ClaudeOAuthLoginStatus(loginID string) (claudeOAuthLoginStatusResponse, error) {
	return a.server.claudeOAuthLoginStatus(loginID)
}

func (a *DesktopApp) CompleteClaudeOAuthLogin(loginID string) (tokenResponse, error) {
	return a.server.completeClaudeOAuthLogin(a.callContext(), loginID)
}

func (a *DesktopApp) ImportAPIKeys(req apiKeyBatchImportRequest) (apiKeyBatchImportResult, error) {
	return a.server.importAPIKeys(req)
}

func (a *DesktopApp) ExportTokens() (tokenExportResult, error) {
	if a.ctx == nil {
		return tokenExportResult{}, errors.New("application is not ready")
	}
	if a.server.tokens == nil {
		return tokenExportResult{}, errors.New("token manager is not ready")
	}

	items := a.server.tokens.List()
	data, err := encodeTokenExport(items, time.Now())
	if err != nil {
		return tokenExportResult{}, err
	}

	filename := fmt.Sprintf("omniproxy-tokens-%s.json", time.Now().Format("20060102-150405"))
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出账号池备份",
		DefaultFilename: filename,
		Filters:         []runtime.FileFilter{{DisplayName: "JSON 文件 (*.json)", Pattern: "*.json"}},
	})
	if err != nil {
		return tokenExportResult{}, err
	}
	if strings.TrimSpace(path) == "" {
		return tokenExportResult{Count: len(items), Message: "已取消导出账号池"}, nil
	}
	if strings.ToLower(filepath.Ext(path)) != ".json" {
		path += ".json"
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return tokenExportResult{}, err
	}
	a.server.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: fmt.Sprintf("token backup exported: %d accounts", len(items))})
	return tokenExportResult{
		Path:    path,
		Count:   len(items),
		Message: fmt.Sprintf("账号池已导出：%d 个账号", len(items)),
	}, nil
}

func (a *DesktopApp) ExportCodexAuthFiles() (codexAuthExportResult, error) {
	if a.ctx == nil {
		return codexAuthExportResult{}, errors.New("application is not ready")
	}
	if a.server.tokens == nil {
		return codexAuthExportResult{}, errors.New("token manager is not ready")
	}

	files := codexAuthExportFiles(a.server.tokens.List(), time.Now().Format("20060102-150405"))
	if len(files) == 0 {
		return codexAuthExportResult{}, errors.New("没有可导出的 Codex auth.json 账号")
	}

	directory, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                "选择 Codex auth.json 导出目录",
		CanCreateDirectories: true,
	})
	if err != nil {
		return codexAuthExportResult{}, err
	}
	if strings.TrimSpace(directory) == "" {
		return codexAuthExportResult{Count: len(files), Message: "已取消导出 Codex auth.json"}, nil
	}

	written, err := writeCodexAuthExportFiles(directory, files)
	if err != nil {
		return codexAuthExportResult{}, err
	}
	a.server.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: fmt.Sprintf("codex auth files exported: %d files", len(written))})
	return codexAuthExportResult{
		Directory: directory,
		Files:     written,
		Count:     len(written),
		Message:   fmt.Sprintf("Codex auth.json 已导出：%d 个文件", len(written)),
	}, nil
}
