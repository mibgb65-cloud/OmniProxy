-- 推理等级来自请求正文的协议字段：Messages 的 output_config.effort、
-- Responses 的 reasoning.effort、Chat 的 reasoning_effort。
-- 只保存网关实际携带／发送的档位；未指定与历史记录保持空串，界面显示「—」，
-- 不从上游默认值或模型名推测。
ALTER TABLE request_records ADD COLUMN reasoning_effort TEXT NOT NULL DEFAULT ''
    CHECK (reasoning_effort = '' OR length(reasoning_effort) BETWEEN 1 AND 32);
