-- 保留旧首选字段，已有偏好自动成为第一个优先服务商。
ALTER TABLE model_provider_preferences ADD COLUMN provider_order TEXT NOT NULL DEFAULT '' CHECK(length(provider_order) <= 14000);
UPDATE model_provider_preferences SET provider_order = json_array(provider_id);
