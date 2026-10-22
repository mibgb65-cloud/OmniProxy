-- 每个模型可固定一个已确认的思考档位，网关在请求侧覆盖客户端选择；
-- 只在模型的推理档位枚举中出现过的取值才允许写入，未确认能力保持空。
CREATE TABLE IF NOT EXISTS model_effort_pins (
    model_id   TEXT PRIMARY KEY,
    effort     TEXT NOT NULL CHECK (length(effort) BETWEEN 1 AND 16),
    updated_at TEXT NOT NULL
);
