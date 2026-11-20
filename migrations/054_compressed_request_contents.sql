-- 054: 完整保存请求正文并压缩存储。
--
-- 旧表的 content 是明文（最大 256 KiB，可能已被截断）。新表用 data BLOB 保存
-- gzip 压缩后的完整正文，format 区分旧 text 行与新 gzip 行，preview 在采集时生成，
-- 记录列表无需解压。旧数据原样迁移，历史 truncated 标记保留。
CREATE TABLE request_contents_v2 (
    request_id     TEXT PRIMARY KEY,
    data           BLOB NOT NULL,
    format         TEXT NOT NULL DEFAULT 'gzip' CHECK (format IN ('gzip', 'text')),
    preview        TEXT NOT NULL DEFAULT '',
    original_bytes INTEGER NOT NULL CHECK (original_bytes BETWEEN 1 AND 33554432),
    stored_bytes   INTEGER NOT NULL CHECK (stored_bytes BETWEEN 1 AND 33554432),
    truncated      INTEGER NOT NULL DEFAULT 0 CHECK (truncated IN (0, 1)),
    captured_at    TEXT NOT NULL
);

INSERT INTO request_contents_v2(request_id, data, format, preview, original_bytes, stored_bytes, truncated, captured_at)
SELECT request_id, CAST(content AS BLOB), 'text', '', original_bytes, stored_bytes, truncated, captured_at
FROM request_contents;

DROP TABLE request_contents;
ALTER TABLE request_contents_v2 RENAME TO request_contents;
CREATE INDEX request_contents_captured ON request_contents(captured_at);
