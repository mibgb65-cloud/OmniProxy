-- 只追加新服务商，保留已有提供商、账号和用户启停选择。
INSERT INTO providers (id, kind, name, base_url, enabled, created_at, updated_at, source)
VALUES ('builtin_river', 'river', 'River', '', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'builtin');
