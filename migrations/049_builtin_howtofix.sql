-- HowToFix 是 new-api 系的 OpenAI 兼容网关，只需 API Key，没有订阅登录。
-- 只追加预设，保留已有提供商启停、账号及凭证引用。
INSERT INTO providers (id, kind, name, base_url, enabled, created_at, updated_at, source)
VALUES ('builtin_howtofix', 'howtofix', 'HowToFix', '', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'builtin');
