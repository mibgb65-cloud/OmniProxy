-- 001_initial.sql: initial OmniProxy schema.
--
-- The schema must only ever change through migrations: append a new
-- <version>_<name>.sql file, never edit an applied migration
-- (技术架构选择.md §15). The schema_version table itself is created and
-- maintained by the migration runner (internal/infrastructure/database).

-- providers stores provider configuration metadata. Credentials are NOT
-- stored here; accounts reference the OS credential store instead.
CREATE TABLE IF NOT EXISTS providers (
    id         TEXT PRIMARY KEY,
    kind       TEXT NOT NULL,                -- openai | anthropic | google | deepseek | kimi | mimo | openai-compatible
    name       TEXT NOT NULL,
    base_url   TEXT NOT NULL DEFAULT '',
    enabled    INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- accounts stores account metadata. credential_ref points at the OS
-- credential store (DPAPI on Windows); it never contains a plaintext secret.
CREATE TABLE IF NOT EXISTS accounts (
    id             TEXT PRIMARY KEY,
    provider_id    TEXT NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    credential_ref TEXT NOT NULL DEFAULT '',
    priority       INTEGER NOT NULL DEFAULT 0,
    enabled        INTEGER NOT NULL DEFAULT 1,
    created_at     TEXT NOT NULL,
    updated_at     TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_accounts_provider_id ON accounts(provider_id);
