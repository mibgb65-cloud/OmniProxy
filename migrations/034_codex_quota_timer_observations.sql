-- 观测时间用于区分滑动占位目标与固定周期；不改写既有发送登记。
ALTER TABLE codex_quota_activation_windows ADD COLUMN observed_at TEXT NOT NULL DEFAULT '';
ALTER TABLE codex_quota_activation_windows ADD COLUMN confirmed_reset TEXT NOT NULL DEFAULT '';
