-- 003_model_routes.sql: 精确匹配模型映射。
-- 保留目标标识而不级联删除：目标删除后规则显示异常，不改变现有账号／提供商删除流程。
CREATE TABLE model_routes (
    id TEXT PRIMARY KEY,
    alias TEXT NOT NULL UNIQUE COLLATE BINARY CHECK(length(alias) BETWEEN 1 AND 128),
    provider_id TEXT NOT NULL CHECK(length(provider_id) BETWEEN 1 AND 128),
    model TEXT NOT NULL CHECK(length(model) BETWEEN 1 AND 128),
    account_id TEXT NOT NULL DEFAULT '' CHECK(length(account_id) <= 128),
    enabled INTEGER NOT NULL DEFAULT 1 CHECK(enabled IN (0,1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
