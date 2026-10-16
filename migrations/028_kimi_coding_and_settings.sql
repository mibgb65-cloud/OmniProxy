-- 既有账号保持普通 API 计费，不从 Key 或地址猜测订阅权益。
ALTER TABLE kimi_account_options ADD COLUMN billing_mode TEXT NOT NULL DEFAULT 'payg'
    CHECK(billing_mode IN ('payg','coding'));

CREATE TABLE kimi_settings (
    id INTEGER PRIMARY KEY CHECK(id=1),
    preferred_billing_mode TEXT NOT NULL CHECK(preferred_billing_mode IN ('coding','payg')),
    preferred_region TEXT NOT NULL CHECK(preferred_region IN ('cn','global'))
);
INSERT INTO kimi_settings VALUES(1,'coding','cn');
