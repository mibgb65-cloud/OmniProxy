-- Gemini 限流按项目共享；绑定仅针对当前 Key 版本，不复用订阅工作区身份。
CREATE TABLE google_project_bindings (
 account_id TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
 project_id TEXT NOT NULL,
 credential_version TEXT NOT NULL,
 updated_at TEXT NOT NULL
);
CREATE TABLE google_resource_observations (
 account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
 model TEXT NOT NULL,
 credential_version TEXT NOT NULL,
 category TEXT NOT NULL,
 metric TEXT NOT NULL DEFAULT '',
 quota_limit INTEGER,
 retry_at TEXT NOT NULL DEFAULT '',
 observed_at TEXT NOT NULL,
 PRIMARY KEY(account_id,model)
);
