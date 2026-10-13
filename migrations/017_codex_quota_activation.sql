-- 只有用户显式开启并保存后，才允许自动消耗少量额度激活窗口。
ALTER TABLE codex_settings ADD COLUMN auto_quota_activation INTEGER NOT NULL DEFAULT 0 CHECK(auto_quota_activation IN (0, 1));

-- 每个账号／窗口只保留当前身份和最近周期；发送前先登记，未知结果不重发。
CREATE TABLE codex_quota_activation_windows (
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    window_kind TEXT NOT NULL CHECK(window_kind IN ('five_hour', 'weekly')),
    identity_version TEXT NOT NULL CHECK(length(identity_version) = 64),
    observed_reset TEXT NOT NULL DEFAULT '',
    attempted_cycle TEXT NOT NULL DEFAULT '',
    attempted_at TEXT NOT NULL DEFAULT '',
    PRIMARY KEY(account_id, window_kind)
);
