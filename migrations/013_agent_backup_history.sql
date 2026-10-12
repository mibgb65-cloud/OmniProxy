-- 重新读取并接入外部修改后的配置时，保留原加密备份引用，不能静默删除历史。
CREATE TABLE agent_backup_history (
    id INTEGER PRIMARY KEY,
    client TEXT NOT NULL,
    credential_ref TEXT NOT NULL UNIQUE,
    applied_hash TEXT NOT NULL,
    archived_at TEXT NOT NULL
);
