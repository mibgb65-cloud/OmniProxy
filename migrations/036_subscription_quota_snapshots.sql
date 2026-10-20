-- 仅保存公开额度显示信息，不保存 Token、密钥或凭据内容。
CREATE TABLE subscription_quota_snapshots (
    account_id TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    identity_key TEXT NOT NULL CHECK(length(identity_key) = 64),
    profile_key TEXT NOT NULL CHECK(length(profile_key) = 64),
    workspace_known INTEGER NOT NULL CHECK(workspace_known IN (0, 1)),
    fetched_at TEXT NOT NULL,
    snapshot_json TEXT NOT NULL CHECK(length(snapshot_json) <= 16384 AND json_valid(snapshot_json))
);
