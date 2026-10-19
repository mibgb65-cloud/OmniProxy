-- 升级后保持原有耗尽切换；平衡模式仅由用户主动选择。
ALTER TABLE codex_settings ADD COLUMN account_rotation_mode TEXT NOT NULL DEFAULT 'queue'
    CHECK(account_rotation_mode IN ('queue', 'balanced'));
