-- 未采集的历史及非流式首字耗时保留 NULL，不从总耗时推测。
ALTER TABLE request_records ADD COLUMN first_token_ms INTEGER
 CHECK(first_token_ms IS NULL OR first_token_ms BETWEEN 0 AND 86400000);
ALTER TABLE dashboard_buckets ADD COLUMN first_token_ms INTEGER NOT NULL DEFAULT 0 CHECK(first_token_ms >= 0);
ALTER TABLE dashboard_buckets ADD COLUMN first_token_samples INTEGER NOT NULL DEFAULT 0 CHECK(first_token_samples >= 0);
