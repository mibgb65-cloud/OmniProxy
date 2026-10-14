-- 阶跃两平台与计费方式属于账号，不能由提供商地址统一覆盖所有 Key。
CREATE TABLE stepfun_account_options (
    account_id TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    region TEXT NOT NULL CHECK(region IN ('cn', 'global')),
    billing_mode TEXT NOT NULL CHECK(billing_mode IN ('payg', 'step_plan')),
    config_version INTEGER NOT NULL DEFAULT 1 CHECK(config_version > 0)
);

-- 仅迁移可确定含义的旧官方配置；自定义地址保留，要求用户明确选择。
INSERT INTO stepfun_account_options(account_id, region, billing_mode)
SELECT a.id,
       CASE WHEN rtrim(p.base_url, '/') LIKE 'https://api.stepfun.ai%' THEN 'global' ELSE 'cn' END,
       CASE WHEN rtrim(p.base_url, '/') IN ('https://api.stepfun.com/step_plan/v1', 'https://api.stepfun.ai/step_plan/v1') THEN 'step_plan' ELSE 'payg' END
FROM accounts a JOIN providers p ON p.id=a.provider_id
WHERE p.kind='stepfun' AND a.auth_method='api_key'
  AND rtrim(p.base_url, '/') IN ('', 'https://api.stepfun.com/v1', 'https://api.stepfun.ai/v1', 'https://api.stepfun.com/step_plan/v1', 'https://api.stepfun.ai/step_plan/v1');

-- 普通账号保持原版本；Step 的选项版本也参与目录读写过滤，避免切通道时混入旧目录。
CREATE VIEW account_catalog_versions AS
SELECT a.id, a.credential_ref || ':' || a.auth_account_id ||
       CASE WHEN s.account_id IS NULL THEN '' ELSE ':step:' || s.config_version END AS version
FROM accounts a LEFT JOIN stepfun_account_options s ON s.account_id=a.id;
