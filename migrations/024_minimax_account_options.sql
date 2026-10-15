-- 两类 Key 共用 URL；旧账号仅能确定地区，计费类型必须由用户确认。
CREATE TABLE minimax_account_options (
    account_id TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    region TEXT NOT NULL CHECK(region IN ('','cn','global')),
    billing_mode TEXT NOT NULL CHECK(billing_mode IN ('','payg','subscription')),
    config_version INTEGER NOT NULL DEFAULT 1 CHECK(config_version>0)
);

INSERT INTO minimax_account_options(account_id,region,billing_mode)
SELECT a.id,CASE
 WHEN rtrim(p.base_url,'/') IN ('','https://api.minimax.io','https://api.minimax.io/v1') THEN 'global'
 WHEN rtrim(p.base_url,'/') IN ('https://api.minimax.cn','https://api.minimax.cn/v1','https://api.minimaxi.com','https://api.minimaxi.com/v1') THEN 'cn'
 ELSE '' END,''
FROM accounts a JOIN providers p ON p.id=a.provider_id
WHERE p.kind='minimax' AND a.auth_method='api_key';

DROP VIEW account_catalog_versions;
CREATE VIEW account_catalog_versions AS
SELECT a.id,a.credential_ref || ':' || a.auth_account_id ||
       CASE WHEN s.account_id IS NULL THEN '' ELSE ':step:' || s.config_version END ||
       CASE WHEN k.account_id IS NULL THEN '' ELSE ':kimi:' || k.config_version END ||
       CASE WHEN mm.account_id IS NULL THEN '' ELSE ':minimax:' || mm.config_version END AS version
FROM accounts a LEFT JOIN stepfun_account_options s ON s.account_id=a.id
LEFT JOIN kimi_account_options k ON k.account_id=a.id
LEFT JOIN minimax_account_options mm ON mm.account_id=a.id;
