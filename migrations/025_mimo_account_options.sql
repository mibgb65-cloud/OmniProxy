-- MiMo 普通按量与三个 Token Plan 集群各自使用独立通道，不能自动回退消费。
CREATE TABLE mimo_account_options (
    account_id TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    region TEXT NOT NULL CHECK(region IN ('','cn','sgp','ams')),
    billing_mode TEXT NOT NULL CHECK(billing_mode IN ('','payg','token_plan')),
    config_version INTEGER NOT NULL DEFAULT 1 CHECK(config_version>0),
    CHECK(billing_mode='' OR (billing_mode='payg' AND region='') OR
          (billing_mode='token_plan' AND region IN ('cn','sgp','ams')))
);

-- 旧实现只允许官方按量／回环地址；未知地址保留为需确认，不猜 Key 或套餐。
INSERT INTO mimo_account_options(account_id,region,billing_mode)
SELECT a.id,'',CASE WHEN rtrim(p.base_url,'/') IN ('','https://api.xiaomimimo.com','https://api.xiaomimimo.com/v1')
 OR p.base_url GLOB 'http://127.0.0.1:*' OR p.base_url GLOB 'https://127.0.0.1:*'
 OR p.base_url GLOB 'http://localhost:*' OR p.base_url GLOB 'https://localhost:*'
 OR p.base_url GLOB 'http://[[]::1]:*' OR p.base_url GLOB 'https://[[]::1]:*'
 THEN 'payg' ELSE '' END
FROM accounts a JOIN providers p ON p.id=a.provider_id
WHERE p.kind='mimo' AND a.auth_method='api_key';

DROP VIEW account_catalog_versions;
CREATE VIEW account_catalog_versions AS
SELECT a.id,a.credential_ref || ':' || a.auth_account_id ||
       CASE WHEN s.account_id IS NULL THEN '' ELSE ':step:' || s.config_version END ||
       CASE WHEN k.account_id IS NULL THEN '' ELSE ':kimi:' || k.config_version END ||
       CASE WHEN mm.account_id IS NULL THEN '' ELSE ':minimax:' || mm.config_version END ||
       CASE WHEN mi.account_id IS NULL THEN '' ELSE ':mimo:' || mi.config_version END AS version
FROM accounts a LEFT JOIN stepfun_account_options s ON s.account_id=a.id
LEFT JOIN kimi_account_options k ON k.account_id=a.id
LEFT JOIN minimax_account_options mm ON mm.account_id=a.id
LEFT JOIN mimo_account_options mi ON mi.account_id=a.id;
