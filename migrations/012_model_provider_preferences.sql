-- 服务商偏好与手工映射分开保存；资源删除后保留偏好，供诊断和恢复自动选择。
CREATE TABLE model_provider_preferences (
    model TEXT PRIMARY KEY COLLATE BINARY CHECK(length(model) BETWEEN 1 AND 128),
    provider_id TEXT NOT NULL CHECK(length(provider_id) BETWEEN 1 AND 128),
    updated_at TEXT NOT NULL
);
