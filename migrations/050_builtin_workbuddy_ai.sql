-- WorkBuddy AI 是腾讯 CodeBuddy 计划的国际版订阅；只接受浏览器登录，没有 API Key。
-- 只追加预设，保留已有提供商启停、账号及凭证引用。
INSERT INTO providers (id, kind, name, base_url, enabled, created_at, updated_at, source)
VALUES ('builtin_workbuddy_ai', 'workbuddy_ai', 'WorkBuddy AI', '', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'builtin');
