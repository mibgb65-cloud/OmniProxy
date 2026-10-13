-- Atria 使用独立版本，避免与并行的 GPT 长上下文任务争用 018。
-- 仅追加预设，保留已有提供商启停、账号及凭证引用。
INSERT INTO providers (id, kind, name, base_url, enabled, created_at, updated_at, source)
VALUES ('builtin_atria', 'atria', 'Atria', '', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'builtin');
