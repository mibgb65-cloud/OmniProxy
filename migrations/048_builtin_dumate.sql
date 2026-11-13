-- 百度搭子 DuMate 个人版使用登录态 Cookie，不是千帆 API Key 或 Token Plan。
-- 只追加预设，保留已有提供商启停、账号及凭证引用。
INSERT INTO providers (id, kind, name, base_url, enabled, created_at, updated_at, source)
VALUES ('builtin_dumate', 'dumate', '百度搭子 DuMate', '', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'builtin');
