-- Kimi 两地区的 Key 不通用，保存账号级平台，避免全局地址混用凭证。
CREATE TABLE kimi_account_options (
    account_id TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    region TEXT NOT NULL CHECK(region IN ('cn','global')),
    config_version INTEGER NOT NULL DEFAULT 1 CHECK(config_version>0)
);

-- 旧空地址的原默认是 .ai；仅识别既有官方地址，其他配置保留为需确认。
INSERT INTO kimi_account_options(account_id,region)
SELECT a.id,CASE WHEN rtrim(p.base_url,'/') IN ('https://api.moonshot.cn','https://api.moonshot.cn/v1') THEN 'cn' ELSE 'global' END
FROM accounts a JOIN providers p ON p.id=a.provider_id
WHERE p.kind='kimi' AND a.auth_method='api_key'
AND rtrim(p.base_url,'/') IN ('','https://api.moonshot.cn','https://api.moonshot.cn/v1','https://api.moonshot.ai','https://api.moonshot.ai/v1');

DROP VIEW account_catalog_versions;
CREATE VIEW account_catalog_versions AS
SELECT a.id,a.credential_ref || ':' || a.auth_account_id ||
       CASE WHEN s.account_id IS NULL THEN '' ELSE ':step:' || s.config_version END ||
       CASE WHEN k.account_id IS NULL THEN '' ELSE ':kimi:' || k.config_version END AS version
FROM accounts a LEFT JOIN stepfun_account_options s ON s.account_id=a.id
LEFT JOIN kimi_account_options k ON k.account_id=a.id;
