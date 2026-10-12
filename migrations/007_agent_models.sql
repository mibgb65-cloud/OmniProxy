-- 模型快照按账号隔离；客户端备份和接入密钥仅保存 DPAPI 引用。
CREATE TABLE account_models (
 account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
 model TEXT NOT NULL,
 name TEXT NOT NULL,
 fetched_at TEXT NOT NULL,
 account_version TEXT NOT NULL,
 PRIMARY KEY(account_id,model)
);
CREATE TABLE agent_connections (
 client TEXT PRIMARY KEY,
 credential_ref TEXT NOT NULL,
 enabled INTEGER NOT NULL DEFAULT 1,
 updated_at TEXT NOT NULL
);
CREATE TABLE agent_backups (
 client TEXT PRIMARY KEY,
 credential_ref TEXT NOT NULL,
 applied_hash TEXT NOT NULL,
 updated_at TEXT NOT NULL
);
