-- Oxion AI 使用独立版本，避免与并行任务争用迁移号。
-- 仅追加预设，保留已有提供商启停、账号及凭证引用。
INSERT INTO providers (id, kind, name, base_url, enabled, created_at, updated_at, source)
VALUES ('builtin_oxionai', 'oxionai', 'Oxion AI', '', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'builtin');
