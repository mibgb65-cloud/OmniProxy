-- 请求正文记录默认关闭：开启后网关才把客户端正文写入 request_contents。
-- 正文与请求明细分表保存，列表查询不读取正文；明细被保留上限淘汰时同步清理。
ALTER TABLE request_retention ADD COLUMN record_content INTEGER NOT NULL DEFAULT 0
    CHECK (record_content IN (0, 1));

CREATE TABLE request_contents (
    request_id     TEXT PRIMARY KEY,
    content        TEXT NOT NULL CHECK (length(content) BETWEEN 1 AND 262144),
    original_bytes INTEGER NOT NULL CHECK (original_bytes BETWEEN 1 AND 33554432),
    stored_bytes   INTEGER NOT NULL CHECK (stored_bytes BETWEEN 1 AND 262144),
    truncated      INTEGER NOT NULL DEFAULT 0 CHECK (truncated IN (0, 1)),
    captured_at    TEXT NOT NULL
);
