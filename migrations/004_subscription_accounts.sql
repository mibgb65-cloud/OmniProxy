-- 兼容已有 API Key 账号，订阅账号单独保存认证类型与公开身份元数据。
ALTER TABLE accounts ADD COLUMN auth_method TEXT NOT NULL DEFAULT 'api_key';
ALTER TABLE accounts ADD COLUMN auth_email TEXT NOT NULL DEFAULT '';
ALTER TABLE accounts ADD COLUMN auth_plan TEXT NOT NULL DEFAULT '';
ALTER TABLE accounts ADD COLUMN auth_state TEXT NOT NULL DEFAULT '';
