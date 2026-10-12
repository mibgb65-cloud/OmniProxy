-- 独立保留运行总览；15 分钟粒度覆盖现代时区的整点、半小时和 45 分钟边界。
CREATE TABLE dashboard_buckets (
 bucket INTEGER PRIMARY KEY CHECK(bucket % 900 = 0),
 requests INTEGER NOT NULL DEFAULT 0 CHECK(requests >= 0),
 success INTEGER NOT NULL DEFAULT 0 CHECK(success >= 0),
 failed INTEGER NOT NULL DEFAULT 0 CHECK(failed >= 0),
 cancelled INTEGER NOT NULL DEFAULT 0 CHECK(cancelled >= 0),
 latency_ms INTEGER NOT NULL DEFAULT 0 CHECK(latency_ms >= 0),
 input_tokens INTEGER NOT NULL DEFAULT 0 CHECK(input_tokens >= 0),
 output_tokens INTEGER NOT NULL DEFAULT 0 CHECK(output_tokens >= 0),
 complete_usage INTEGER NOT NULL DEFAULT 0 CHECK(complete_usage >= 0)
);
INSERT INTO dashboard_buckets(bucket,requests,success,failed,cancelled,latency_ms,input_tokens,output_tokens,complete_usage)
SELECT (CAST(strftime('%s',started_at) AS INTEGER)/900)*900,COUNT(*),
 SUM(state='success'),SUM(state='failed'),SUM(state='cancelled'),
 SUM(CASE WHEN state IN ('success','failed') THEN MAX(latency_ms,0) ELSE 0 END),
 SUM(input_tokens),SUM(output_tokens),SUM(usage_complete)
FROM request_records GROUP BY (CAST(strftime('%s',started_at) AS INTEGER)/900)*900;
CREATE TABLE dashboard_meta (
 id INTEGER PRIMARY KEY CHECK(id=1),
 captured_from INTEGER NOT NULL,
 history_incomplete INTEGER NOT NULL CHECK(history_incomplete IN (0,1))
);
-- 已被淘汰的结果与耗时无法还原；保守标示升级前的统计可能不完整。
INSERT INTO dashboard_meta VALUES(1,unixepoch(),
 (SELECT requests FROM usage_totals WHERE id=1) > (SELECT COUNT(*) FROM request_records));
