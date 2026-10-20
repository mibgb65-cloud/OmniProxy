-- 缓存计数缺失保持 NULL，不能把历史未知缓存误当成零。
ALTER TABLE request_records ADD COLUMN cached_input_tokens INTEGER
 CHECK(cached_input_tokens IS NULL OR (cached_input_tokens>=0 AND cached_input_tokens<=input_tokens));

-- 刷新卡／提前重置后的消费从新基线开始，不能回退累计 Token。
CREATE TABLE subscription_quota_estimate_baselines (
 account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
 window_type TEXT NOT NULL CHECK(window_type IN ('five_hour','weekly')),
 reset_at TEXT NOT NULL,
 start_at TEXT NOT NULL,
 PRIMARY KEY(account_id,window_type)
);
