CREATE TABLE gateway_failover_settings (
 id INTEGER PRIMARY KEY CHECK(id=1),
 enabled INTEGER NOT NULL CHECK(enabled IN (0,1))
);
INSERT INTO gateway_failover_settings VALUES(1,1);

CREATE TABLE request_attempts (
 request_id TEXT NOT NULL REFERENCES request_records(id) ON DELETE CASCADE,
 attempt_index INTEGER NOT NULL CHECK(attempt_index BETWEEN 1 AND 3),
 started_at TEXT NOT NULL,
 finished_at TEXT NOT NULL DEFAULT '',
 provider_id TEXT NOT NULL,
 provider_name TEXT NOT NULL,
 account_id TEXT NOT NULL,
 account_name TEXT NOT NULL,
 provider_region TEXT NOT NULL DEFAULT '',
 billing_mode TEXT NOT NULL DEFAULT '',
 model TEXT NOT NULL,
 state TEXT NOT NULL DEFAULT 'pending' CHECK(state IN ('pending','success','failed','cancelled')),
 status_code INTEGER NOT NULL DEFAULT 0,
 error_code TEXT NOT NULL DEFAULT '',
 latency_ms INTEGER NOT NULL DEFAULT 0,
 input_tokens INTEGER NOT NULL DEFAULT 0 CHECK(input_tokens>=0),
 output_tokens INTEGER NOT NULL DEFAULT 0 CHECK(output_tokens>=0),
 input_known INTEGER NOT NULL DEFAULT 0,
 output_known INTEGER NOT NULL DEFAULT 0,
 usage_complete INTEGER NOT NULL DEFAULT 0,
 PRIMARY KEY(request_id,attempt_index)
);
