-- 002_builtin_providers.sql: distinguish fixed presets from user-created endpoints.
-- Existing manually added rows keep their IDs and account references as custom.
ALTER TABLE providers ADD COLUMN source TEXT NOT NULL DEFAULT 'custom'
    CHECK (source IN ('builtin', 'custom'));

CREATE UNIQUE INDEX idx_providers_builtin_kind ON providers(kind)
    WHERE source = 'builtin';

INSERT INTO providers (id, kind, name, base_url, enabled, created_at, updated_at, source)
VALUES
    ('builtin_openai', 'openai', 'OpenAI', '', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'builtin'),
    ('builtin_anthropic', 'anthropic', 'Anthropic', '', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'builtin'),
    ('builtin_google', 'google', 'Google', '', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'builtin'),
    ('builtin_deepseek', 'deepseek', 'DeepSeek', '', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'builtin'),
    ('builtin_zhipu', 'zhipu', 'Zhipu AI', '', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'builtin'),
    ('builtin_kimi', 'kimi', 'Moonshot AI', '', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'builtin'),
    ('builtin_minimax', 'minimax', 'MiniMax', '', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'builtin'),
    ('builtin_mimo', 'mimo', 'Xiaomi MiMo', '', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'builtin'),
    ('builtin_stepfun', 'stepfun', 'StepFun', '', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'builtin');
