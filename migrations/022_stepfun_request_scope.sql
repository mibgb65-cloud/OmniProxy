-- 保存当次通道，账号随后改平台／计费方式也不能改写历史归属。
ALTER TABLE request_records ADD COLUMN provider_region TEXT NOT NULL DEFAULT '';
ALTER TABLE request_records ADD COLUMN billing_mode TEXT NOT NULL DEFAULT '';
