-- WorkBuddy 国内版（腾讯 CodeBuddy 桌面版，copilot.tencent.com）是独立站点与账号体系，
-- 与国际版 workbuddy_ai 并列；只接受浏览器登录，没有 API Key。
-- 只追加预设，保留已有提供商启停、账号及凭证引用。
INSERT INTO providers (id, kind, name, base_url, enabled, created_at, updated_at, source)
VALUES ('builtin_workbuddy', 'workbuddy', 'WorkBuddy (China)', '', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'builtin');
