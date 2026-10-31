-- 账号终身用量：按记账身份索引（subject）累计，明细保留上限淘汰、账号／提供商删除都不减少。
-- 与 usage_totals（全局）、usage_provider_models（提供商／模型）同一差值口径，在记账事务内更新。
CREATE TABLE usage_accounts (
    subject             TEXT PRIMARY KEY,
    provider_name       TEXT NOT NULL DEFAULT '',
    account_name        TEXT NOT NULL DEFAULT '',
    email               TEXT NOT NULL DEFAULT '',
    workspace_id        TEXT NOT NULL DEFAULT '',
    requests            INTEGER NOT NULL DEFAULT 0 CHECK(requests >= 0),
    completed_usage     INTEGER NOT NULL DEFAULT 0 CHECK(completed_usage >= 0),
    input_tokens        INTEGER NOT NULL DEFAULT 0 CHECK(input_tokens >= 0),
    output_tokens       INTEGER NOT NULL DEFAULT 0 CHECK(output_tokens >= 0),
    input_known         INTEGER NOT NULL DEFAULT 0 CHECK(input_known >= 0),
    output_known        INTEGER NOT NULL DEFAULT 0 CHECK(output_known >= 0),
    cached_input_tokens INTEGER NOT NULL DEFAULT 0 CHECK(cached_input_tokens >= 0),
    cached_known        INTEGER NOT NULL DEFAULT 0 CHECK(cached_known >= 0)
);

-- 从每日桶回填已有累计：Token 与请求数不会因保留上限丢失。
-- 桶里没有"明细是否已知"的标记，用完整用量请求数作已知数；有数值的桶至少算一次已知，不把真实数值标成未知。
INSERT INTO usage_accounts(subject,provider_name,account_name,email,workspace_id,requests,completed_usage,input_tokens,output_tokens,input_known,output_known)
SELECT subject,MAX(provider_name),MAX(account_name),MAX(email),MAX(workspace_id),
       SUM(requests),SUM(completed_usage),SUM(input_tokens),SUM(output_tokens),
       SUM(CASE WHEN completed_usage > 0 THEN completed_usage WHEN input_tokens > 0 THEN 1 ELSE 0 END),
       SUM(CASE WHEN completed_usage > 0 THEN completed_usage WHEN output_tokens > 0 THEN 1 ELSE 0 END)
FROM usage_buckets WHERE subject NOT IN ('','unassigned') GROUP BY subject;

-- 缓存明细只能从仍在保留期内的请求明细回填：更早的部分保持未知，界面显示下界（≥），不伪造完整值。
UPDATE usage_accounts SET
    cached_input_tokens = COALESCE((SELECT SUM(r.cached_input_tokens) FROM request_records r
        WHERE r.subject = usage_accounts.subject AND r.cached_input_tokens IS NOT NULL), 0),
    cached_known = COALESCE((SELECT COUNT(*) FROM request_records r
        WHERE r.subject = usage_accounts.subject AND r.cached_input_tokens IS NOT NULL), 0)
WHERE EXISTS (SELECT 1 FROM request_records r
        WHERE r.subject = usage_accounts.subject AND r.cached_input_tokens IS NOT NULL);
