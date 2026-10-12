-- 持久化同一次手动使用的幂等标识，网络超时或应用重启后仍可核对，避免重复消耗。
CREATE TABLE subscription_reset_attempts (
    operation_id TEXT PRIMARY KEY CHECK(length(operation_id) = 36),
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    auth_account_id TEXT NOT NULL CHECK(length(auth_account_id) BETWEEN 1 AND 128),
    credit_id TEXT NOT NULL DEFAULT '' CHECK(length(credit_id) <= 256),
    state TEXT NOT NULL CHECK(state IN ('pending', 'completed')),
    outcome TEXT NOT NULL DEFAULT '' CHECK(outcome IN ('', 'reset', 'alreadyRedeemed', 'nothingToReset', 'noCredit')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX subscription_reset_one_pending ON subscription_reset_attempts(account_id) WHERE state = 'pending';
