-- Google 额度仅用于恢复显示，授权版本不包含真实令牌。
CREATE TABLE google_subscription_quota_snapshots (
    account_id TEXT PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    identity_key TEXT NOT NULL CHECK(length(identity_key) = 64),
    account_version TEXT NOT NULL CHECK(length(account_version) BETWEEN 1 AND 1024),
    fetched_at TEXT NOT NULL,
    snapshot_json TEXT NOT NULL CHECK(length(snapshot_json) <= 131072 AND json_valid(snapshot_json))
);
