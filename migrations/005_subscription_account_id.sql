-- 官方订阅账号 ID 与本地账号记录 ID 分离，已有账号通过刷新身份补全。
ALTER TABLE accounts ADD COLUMN auth_account_id TEXT NOT NULL DEFAULT '' CHECK(length(auth_account_id) <= 128);
