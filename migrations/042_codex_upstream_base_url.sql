-- Codex 透明转发恢复 v1 的可配置上游基址；默认仍是官方订阅端点。
ALTER TABLE codex_settings ADD COLUMN upstream_base_url TEXT NOT NULL DEFAULT 'https://chatgpt.com/backend-api/codex';
