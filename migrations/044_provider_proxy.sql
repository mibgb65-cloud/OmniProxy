-- 提供商出站代理：一份连接参数（单例）+ 每个提供商三态路由策略。
-- 密码不落库：proxy_settings 只保存 credential_ref（DPAPI 凭据文件）。
CREATE TABLE IF NOT EXISTS proxy_settings (
    id              INTEGER PRIMARY KEY CHECK(id = 1),
    enabled         INTEGER NOT NULL DEFAULT 0 CHECK(enabled IN (0, 1)),
    -- auto = mixed 端口，由探测决定实际协议（http | socks5）
    scheme          TEXT    NOT NULL DEFAULT 'auto' CHECK(scheme IN ('auto', 'http', 'socks5')),
    host            TEXT    NOT NULL DEFAULT '127.0.0.1',
    port            INTEGER NOT NULL DEFAULT 10808,
    username        TEXT    NOT NULL DEFAULT '',
    password_ref    TEXT    NOT NULL DEFAULT '',
    no_proxy        TEXT    NOT NULL DEFAULT 'localhost,127.0.0.1,::1',
    detected_scheme TEXT    NOT NULL DEFAULT '' CHECK(detected_scheme IN ('', 'http', 'socks5')),
    updated_at      TEXT    NOT NULL DEFAULT ''
);
INSERT OR IGNORE INTO proxy_settings(id, updated_at) VALUES(1, '');

-- inherit：跟随全局开关；direct：强制直连；proxy：必须走代理。
ALTER TABLE providers ADD COLUMN proxy_mode TEXT NOT NULL DEFAULT 'inherit';
