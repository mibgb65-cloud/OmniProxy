// Package routing 管理持久化模型规则和只读匹配预览。
package routing

import (
	"context"
	"database/sql"
	"errors"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	core "omniproxy/internal/core/routing"
)

const maxRules = 1000

var (
	// ErrInvalidInput 表示字段或引用校验失败。
	ErrInvalidInput = errors.New("routing: invalid input")
	// ErrDuplicateAlias 表示别名已被另一条规则使用。
	ErrDuplicateAlias = errors.New("routing: duplicate alias")
	// ErrNotFound 表示编辑／删除的规则不存在。
	ErrNotFound = errors.New("routing: not found")
	// ErrLimitReached 限制配置数量，防止无限增长的规则集合。
	ErrLimitReached = errors.New("routing: limit reached")
)

// Input 是规则的可编辑字段；空 AccountID 表示提供商账号范围。
type Input struct {
	Alias      string `json:"alias"`
	ProviderID string `json:"providerId"`
	Model      string `json:"model"`
	AccountID  string `json:"accountId"`
	Enabled    bool   `json:"enabled"`
}

// Item 附带配置诊断状态，Ready 不表示密钥认证或配额已验证。
type Item struct {
	Rule         core.Rule `json:"rule"`
	ProviderName string    `json:"providerName"`
	AccountName  string    `json:"accountName"`
	ErrorCode    string    `json:"errorCode"`
}

// Preview 包含配置解析结果，不读取密钥、调用模型或返回凭证引用。
type Preview struct {
	AccountUpdatedAt    string `json:"accountUpdatedAt,omitempty"`
	InputModel          string `json:"inputModel"`
	Matched             bool   `json:"matched"`
	RuleID              string `json:"ruleId"`
	Model               string `json:"model"`
	ProviderID          string `json:"providerId"`
	ProviderName        string `json:"providerName"`
	AccountID           string `json:"accountId"`
	AccountName         string `json:"accountName"`
	CandidateCount      int    `json:"candidateCount"`
	Routable            bool   `json:"routable"`
	ErrorCode           string `json:"errorCode"`
	PreferredProviderID string `json:"preferredProviderId,omitempty"`
	PreferenceApplied   bool   `json:"preferenceApplied,omitempty"`
	ProviderPriority    int    `json:"providerPriority,omitempty"`
}

// ProviderReader 和 AccountReader 只允许读取元数据，不暴露密钥读取能力。
type ProviderReader interface {
	List(context.Context) ([]provider.Configuration, error)
}
type AccountReader interface {
	List(context.Context) ([]account.Item, error)
}

// Service 复用已有资源服务，通过参数化 SQL 保存规则。
type Service struct {
	db        *sql.DB
	providers ProviderReader
	accounts  AccountReader
}

// New 创建规则服务，不执行迁移或读写凭证。
func New(db *sql.DB, providers ProviderReader, accounts AccountReader) *Service {
	return &Service{db: db, providers: providers, accounts: accounts}
}
