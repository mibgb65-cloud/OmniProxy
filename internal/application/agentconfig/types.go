// Package agentconfig 受控配置本地客户端，保持原提供商标识与会话文件不变。
package agentconfig

import (
	"context"
	"database/sql"
	"errors"
	catalog "omniproxy/internal/core/modelcatalog"
	"sync"
)

type Store interface {
	Put([]byte) (string, error)
	Get(string) ([]byte, error)
	Delete(string) error
}
type Catalog interface {
	AvailableModels(context.Context) ([]string, error)
}
type Paths struct {
	Codex, Claude, ClaudeDesktop                             string
	ClaudeDesktopNormal, ClaudeDesktopThirdParty             string
	OpenCode, HarnessWeb, HarnessDesktop, HarnessCredentials string
	KiloCode, QwenCode, MiMoCode                             string
	KimiCode, Pi, FactoryDroid, Goose                        string
	KimiCodeLegacy                                           bool
}
type Item struct {
	Client     string `json:"client"`
	Path       string `json:"path"`
	Exists     bool   `json:"exists"`
	Configured bool   `json:"configured"`
	Provider   string `json:"provider"`
	CanRestore bool   `json:"canRestore"`
	Revision   string `json:"revision"`
	ErrorCode  string `json:"errorCode"`
}
type Input struct {
	Client              string            `json:"client"`
	Model               string            `json:"model,omitempty"`
	ModelIDs            []string          `json:"modelIds,omitempty"`
	Efforts             map[string]string `json:"efforts,omitempty"`
	Revision            string            `json:"revision"`
	CatalogRevision     string            `json:"catalogRevision,omitempty"`
	AcceptCurrentConfig bool              `json:"acceptCurrentConfig,omitempty"`
}

// Plan 只展示受管字段和公开模型信息，不包含原配置内容或接入密钥。
type Plan struct {
	Summary         string            `json:"summary"`
	Mode            string            `json:"mode"`
	DefaultModel    string            `json:"defaultModel"`
	Models          []catalog.Entry   `json:"models"`
	Efforts         map[string]string `json:"efforts,omitempty"`
	CatalogRevision string            `json:"catalogRevision"`
}
type Service struct {
	db           *sql.DB
	store        Store
	paths        Paths
	port         int
	catalog      Catalog
	mu           sync.Mutex
	codexVersion string
}

// SetCodexVersion 同步启动或安装后的本机核心版本，不自行安装或升级客户端。
func (s *Service) SetCodexVersion(version string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codexVersion = version
}

func New(db *sql.DB, store Store, paths Paths, port int, catalog Catalog) *Service {
	return &Service{db: db, store: store, paths: paths, port: port, catalog: catalog}
}
func failure(code string) error { return errors.New("agent: " + code) }
func (s *Service) path(client string) (string, error) {
	switch client {
	case "codex":
		return s.paths.Codex, nil
	case "claude":
		return s.paths.Claude, nil
	case "claude-desktop":
		return s.paths.ClaudeDesktop, nil
	case "opencode":
		return s.paths.OpenCode, nil
	case "kilocode":
		return s.paths.KiloCode, nil
	case "qwen-code":
		return s.paths.QwenCode, nil
	case "mimo-code":
		return s.paths.MiMoCode, nil
	case "kimi-code":
		return s.paths.KimiCode, nil
	case "pi":
		return s.paths.Pi, nil
	case "factory-droid":
		return s.paths.FactoryDroid, nil
	case "goose":
		return s.paths.Goose, nil
	case "harness-web":
		return s.paths.HarnessWeb, nil
	case "harness-desktop":
		return s.paths.HarnessDesktop, nil
	}
	return "", failure("invalid_client")
}
