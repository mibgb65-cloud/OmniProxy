-- 历史快照没有指向账号／提供商的删除外键，删除资源不能移除用量。
CREATE TABLE request_records (
 id TEXT PRIMARY KEY,
 started_at TEXT NOT NULL,
 finished_at TEXT NOT NULL DEFAULT '',
 day TEXT NOT NULL,
 protocol TEXT NOT NULL,
 transport TEXT NOT NULL,
 input_model TEXT NOT NULL DEFAULT '',
 model TEXT NOT NULL DEFAULT '',
 stream INTEGER NOT NULL DEFAULT 0,
 provider_id TEXT NOT NULL DEFAULT '',
 provider_name TEXT NOT NULL DEFAULT '',
 account_id TEXT NOT NULL DEFAULT '',
 account_name TEXT NOT NULL DEFAULT '',
 email TEXT NOT NULL DEFAULT '',
 workspace_id TEXT NOT NULL DEFAULT '',
 subject TEXT NOT NULL DEFAULT '',
 state TEXT NOT NULL DEFAULT 'pending',
 status_code INTEGER NOT NULL DEFAULT 0,
 error_code TEXT NOT NULL DEFAULT '',
 latency_ms INTEGER NOT NULL DEFAULT 0,
 input_tokens INTEGER NOT NULL DEFAULT 0 CHECK(input_tokens>=0),
 output_tokens INTEGER NOT NULL DEFAULT 0 CHECK(output_tokens>=0),
 input_known INTEGER NOT NULL DEFAULT 0,
 output_known INTEGER NOT NULL DEFAULT 0,
 usage_complete INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX request_records_time ON request_records(started_at DESC,id DESC);
CREATE INDEX request_records_subject ON request_records(subject);
CREATE TABLE usage_buckets (
 day TEXT NOT NULL,
 subject TEXT NOT NULL,
 model TEXT NOT NULL,
 provider_name TEXT NOT NULL,
 account_name TEXT NOT NULL,
 email TEXT NOT NULL,
 workspace_id TEXT NOT NULL,
 requests INTEGER NOT NULL DEFAULT 0,
 completed_usage INTEGER NOT NULL DEFAULT 0,
 input_tokens INTEGER NOT NULL DEFAULT 0 CHECK(input_tokens>=0),
 output_tokens INTEGER NOT NULL DEFAULT 0 CHECK(output_tokens>=0),
 PRIMARY KEY(day,subject,model)
);
CREATE TABLE usage_totals (
 id INTEGER PRIMARY KEY CHECK(id=1),
 requests INTEGER NOT NULL DEFAULT 0,
 completed_usage INTEGER NOT NULL DEFAULT 0,
 input_tokens INTEGER NOT NULL DEFAULT 0 CHECK(input_tokens>=0),
 output_tokens INTEGER NOT NULL DEFAULT 0 CHECK(output_tokens>=0)
);
INSERT INTO usage_totals(id) VALUES(1);
CREATE TABLE request_retention (
 id INTEGER PRIMARY KEY CHECK(id=1),
 max_records INTEGER NOT NULL CHECK(max_records BETWEEN 100 AND 1000000)
);
INSERT INTO request_retention VALUES(1,10000);
