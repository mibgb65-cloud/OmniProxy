-- 仅保存启动模式和配置 ID 等非敏感字段；客户端密钥和原配置仍由 DPAPI 备份。
CREATE TABLE agent_desktop_activation (
 client TEXT PRIMARY KEY CHECK (client = 'claude-desktop'),
 original_state TEXT NOT NULL,
 applied_hash TEXT NOT NULL,
 updated_at TEXT NOT NULL
);
