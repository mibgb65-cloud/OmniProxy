-- 接入客户端标识来自受管 Agent 各自的接入 Key，不是原始头部或凭据。
-- 历史记录无法回溯归属，保持空串表示未知。
ALTER TABLE request_records ADD COLUMN client_id TEXT NOT NULL DEFAULT ''
    CHECK (client_id = '' OR length(client_id) BETWEEN 1 AND 32);
