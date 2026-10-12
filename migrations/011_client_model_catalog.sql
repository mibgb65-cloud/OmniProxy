-- 未同步新元数据的旧快照保留为空，不猜测上下文和推理能力。
ALTER TABLE account_models ADD COLUMN metadata_json TEXT NOT NULL DEFAULT '{}';

-- 受管模型目录同样通过加密备份恢复，数据库仅保存凭据引用和文件版本。
CREATE TABLE agent_catalog_backups (
 client TEXT PRIMARY KEY,
 credential_ref TEXT NOT NULL,
 applied_hash TEXT NOT NULL,
 updated_at TEXT NOT NULL
);
