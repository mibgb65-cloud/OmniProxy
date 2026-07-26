package history

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"strings"
	"time"
)

func (s *SQLiteStore) PruneBeforeDate(cutoffDate string) error {
	cutoffDate = strings.TrimSpace(cutoffDate)
	if cutoffDate == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Compared as a plain string so idx_request_history_time applies. RFC3339
	// sorts lexically, and a same-day timestamp is longer than the bare date,
	// so it still sorts after the cutoff and survives.
	if _, err := tx.Exec(`DELETE FROM request_history WHERE time < ?`, cutoffDate); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM billing_daily_usage WHERE date < ?`, cutoffDate); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM request_daily_summary WHERE date < ?`, cutoffDate); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) ClearDailyUsage() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM billing_daily_usage`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM billing_lifetime_summary`); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) ClearRequestHistory() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM request_history`)
	return err
}

func (s *SQLiteStore) RebuildSummaries() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM billing_daily_usage`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM billing_lifetime_summary`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM request_daily_summary`); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if err := s.rebuildDailyUsageIfEmptyLocked(); err != nil {
		return err
	}
	if err := s.rebuildBillingLifetimeIfEmptyLocked(); err != nil {
		return err
	}
	return s.rebuildRequestDailySummaryIfEmptyLocked()
}

func (s *SQLiteStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Close()
}

func (s *SQLiteStore) init() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		return err
	}
	// Best-effort: journal_mode=WAL needs shared-memory support, which some
	// network-mounted home directories lack. The store stays correct on the
	// default rollback journal, so a failure here must not break history.
	_, _ = s.db.Exec(`PRAGMA journal_mode = WAL`)
	_, _ = s.db.Exec(`PRAGMA synchronous = NORMAL`)
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS request_history (
  id INTEGER PRIMARY KEY,
  time TEXT NOT NULL,
  level TEXT NOT NULL,
  method TEXT,
  path TEXT,
  provider TEXT,
  protocol TEXT,
  client_key TEXT,
  client_name TEXT,
  model TEXT,
  status INTEGER,
  duration_ms INTEGER,
  token_id TEXT,
  token_name TEXT,
	input_tokens INTEGER,
	output_tokens INTEGER,
	total_tokens INTEGER,
	cooldown_triggered INTEGER,
	retry_chain TEXT,
  message TEXT NOT NULL
);
`)
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS billing_daily_usage (
  date TEXT NOT NULL,
  provider TEXT NOT NULL DEFAULT '',
  protocol TEXT NOT NULL DEFAULT '',
  client_key TEXT NOT NULL DEFAULT '',
  client_name TEXT NOT NULL DEFAULT '',
  token_key TEXT NOT NULL DEFAULT '',
  token_id TEXT NOT NULL DEFAULT '',
  token_name TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  request_count INTEGER NOT NULL DEFAULT 0,
  input_tokens INTEGER NOT NULL DEFAULT 0,
  output_tokens INTEGER NOT NULL DEFAULT 0,
  total_tokens INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (date, provider, protocol, client_key, model, token_key)
);
`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS billing_lifetime_summary (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  request_count INTEGER NOT NULL DEFAULT 0,
  input_tokens INTEGER NOT NULL DEFAULT 0,
  output_tokens INTEGER NOT NULL DEFAULT 0,
  total_tokens INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL
);
`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS request_daily_summary (
  date TEXT NOT NULL,
  provider TEXT NOT NULL DEFAULT '',
  protocol TEXT NOT NULL DEFAULT '',
  client_key TEXT NOT NULL DEFAULT '',
  client_name TEXT NOT NULL DEFAULT '',
  client_label TEXT NOT NULL DEFAULT '',
  level TEXT NOT NULL DEFAULT '',
  status INTEGER NOT NULL DEFAULT 0,
  model TEXT NOT NULL DEFAULT '',
  token_key TEXT NOT NULL DEFAULT '',
  token_id TEXT NOT NULL DEFAULT '',
  token_name TEXT NOT NULL DEFAULT '',
  request_count INTEGER NOT NULL DEFAULT 0,
  failed_count INTEGER NOT NULL DEFAULT 0,
  total_tokens INTEGER NOT NULL DEFAULT 0,
  duration_ms INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (date, provider, protocol, client_key, level, status, model, token_key)
);
`); err != nil {
		return err
	}
	if err := s.ensureColumn("client_key", "ALTER TABLE request_history ADD COLUMN client_key TEXT"); err != nil {
		return err
	}
	if err := s.ensureColumn("client_name", "ALTER TABLE request_history ADD COLUMN client_name TEXT"); err != nil {
		return err
	}
	if err := s.migrateBillingDailyUsageSchemaLocked(); err != nil {
		return err
	}
	if err := s.migrateRequestDailySummarySchemaLocked(); err != nil {
		return err
	}
	_, err = s.db.Exec(`
CREATE INDEX IF NOT EXISTS idx_request_history_provider_nocase ON request_history(provider COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_request_history_client_key_nocase ON request_history(client_key COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_request_history_level_nocase ON request_history(level COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_request_history_token_id_nocase ON request_history(token_id COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_request_history_token_id ON request_history(token_id);
DROP INDEX IF EXISTS idx_request_history_provider;
DROP INDEX IF EXISTS idx_request_history_client_key;
DROP INDEX IF EXISTS idx_request_history_level;
CREATE INDEX IF NOT EXISTS idx_request_history_status ON request_history(status);
CREATE INDEX IF NOT EXISTS idx_request_history_model ON request_history(model);
CREATE INDEX IF NOT EXISTS idx_request_history_token_name ON request_history(token_name);
CREATE INDEX IF NOT EXISTS idx_request_history_time ON request_history(time);
CREATE INDEX IF NOT EXISTS idx_billing_daily_usage_date ON billing_daily_usage(date);
CREATE INDEX IF NOT EXISTS idx_billing_daily_usage_model ON billing_daily_usage(model);
CREATE INDEX IF NOT EXISTS idx_billing_daily_usage_token_id ON billing_daily_usage(token_id);
CREATE INDEX IF NOT EXISTS idx_request_daily_summary_date ON request_daily_summary(date);
CREATE INDEX IF NOT EXISTS idx_request_daily_summary_provider ON request_daily_summary(provider);
CREATE INDEX IF NOT EXISTS idx_request_daily_summary_client_key ON request_daily_summary(client_key);
CREATE INDEX IF NOT EXISTS idx_request_daily_summary_model ON request_daily_summary(model);
CREATE INDEX IF NOT EXISTS idx_request_daily_summary_token_id ON request_daily_summary(token_id);
CREATE INDEX IF NOT EXISTS idx_request_daily_summary_token_name ON request_daily_summary(token_name);
`)
	if err != nil {
		return err
	}
	if err := s.rebuildDailyUsageIfEmptyLocked(); err != nil {
		return err
	}
	if err := s.rebuildBillingLifetimeIfEmptyLocked(); err != nil {
		return err
	}
	return s.rebuildRequestDailySummaryIfEmptyLocked()
}

func (s *SQLiteStore) ensureColumn(name string, statement string) error {
	rows, err := s.db.Query(`PRAGMA table_info(request_history)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			columnName string
			columnType string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &columnName, &columnType, &notNull, &defaultVal, &pk); err != nil {
			return err
		}
		if strings.EqualFold(columnName, name) {
			return rows.Err()
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = s.db.Exec(statement)
	return err
}

func (s *SQLiteStore) migrateBillingDailyUsageSchemaLocked() error {
	ok, err := s.tableHasColumn("billing_daily_usage", "token_key")
	if err != nil || ok {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`ALTER TABLE billing_daily_usage RENAME TO billing_daily_usage_legacy`); err != nil {
		return err
	}
	if _, err := tx.Exec(`
CREATE TABLE billing_daily_usage (
  date TEXT NOT NULL,
  provider TEXT NOT NULL DEFAULT '',
  protocol TEXT NOT NULL DEFAULT '',
  client_key TEXT NOT NULL DEFAULT '',
  client_name TEXT NOT NULL DEFAULT '',
  token_key TEXT NOT NULL DEFAULT '',
  token_id TEXT NOT NULL DEFAULT '',
  token_name TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  request_count INTEGER NOT NULL DEFAULT 0,
  input_tokens INTEGER NOT NULL DEFAULT 0,
  output_tokens INTEGER NOT NULL DEFAULT 0,
  total_tokens INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (date, provider, protocol, client_key, model, token_key)
)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`
INSERT INTO billing_daily_usage (
  date, provider, protocol, client_key, client_name, token_key, token_id, token_name, model,
  request_count, input_tokens, output_tokens, total_tokens, updated_at
)
SELECT
  date,
  provider,
  protocol,
  client_key,
  client_name,
  '' AS token_key,
  '' AS token_id,
  '' AS token_name,
  model,
  request_count,
  input_tokens,
  output_tokens,
  total_tokens,
  updated_at
FROM billing_daily_usage_legacy`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DROP TABLE billing_daily_usage_legacy`); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) migrateRequestDailySummarySchemaLocked() error {
	ok, err := s.tableHasColumn("request_daily_summary", "token_key")
	if err != nil || ok {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`ALTER TABLE request_daily_summary RENAME TO request_daily_summary_legacy`); err != nil {
		return err
	}
	if _, err := tx.Exec(`
CREATE TABLE request_daily_summary (
  date TEXT NOT NULL,
  provider TEXT NOT NULL DEFAULT '',
  protocol TEXT NOT NULL DEFAULT '',
  client_key TEXT NOT NULL DEFAULT '',
  client_name TEXT NOT NULL DEFAULT '',
  client_label TEXT NOT NULL DEFAULT '',
  level TEXT NOT NULL DEFAULT '',
  status INTEGER NOT NULL DEFAULT 0,
  model TEXT NOT NULL DEFAULT '',
  token_key TEXT NOT NULL DEFAULT '',
  token_id TEXT NOT NULL DEFAULT '',
  token_name TEXT NOT NULL DEFAULT '',
  request_count INTEGER NOT NULL DEFAULT 0,
  failed_count INTEGER NOT NULL DEFAULT 0,
  total_tokens INTEGER NOT NULL DEFAULT 0,
  duration_ms INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (date, provider, protocol, client_key, level, status, model, token_key)
)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`
INSERT INTO request_daily_summary (
  date, provider, protocol, client_key, client_name, client_label,
  level, status, model, token_key, token_id, token_name,
  request_count, failed_count, total_tokens, duration_ms, updated_at
)
SELECT
  date,
  provider,
  protocol,
  client_key,
  client_name,
  client_label,
  level,
  status,
  model,
  COALESCE(NULLIF(token_name, ''), '') AS token_key,
  '' AS token_id,
  token_name,
  request_count,
  failed_count,
  total_tokens,
  duration_ms,
  updated_at
FROM request_daily_summary_legacy`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DROP TABLE request_daily_summary_legacy`); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) tableHasColumn(tableName string, column string) (bool, error) {
	rows, err := s.db.Query(`PRAGMA table_info(` + tableName + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			columnName string
			columnType string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &columnName, &columnType, &notNull, &defaultVal, &pk); err != nil {
			return false, err
		}
		if strings.EqualFold(columnName, column) {
			return true, rows.Err()
		}
	}
	return false, rows.Err()
}

func (s *SQLiteStore) rebuildDailyUsageIfEmptyLocked() error {
	var historyCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM request_history`).Scan(&historyCount); err != nil {
		return err
	}
	if historyCount == 0 {
		return nil
	}
	var usageCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM billing_daily_usage`).Scan(&usageCount); err != nil {
		return err
	}
	if usageCount > 0 {
		return nil
	}
	_, err := s.db.Exec(`
INSERT INTO billing_daily_usage (
  date, provider, protocol, client_key, client_name, token_key, token_id, token_name, model,
  request_count, input_tokens, output_tokens, total_tokens, updated_at
)
SELECT
  substr(time, 1, 10) AS date,
  COALESCE(provider, '') AS provider,
  COALESCE(protocol, '') AS protocol,
  COALESCE(client_key, '') AS client_key,
  COALESCE(MAX(NULLIF(client_name, '')), '') AS client_name,
  CASE
    WHEN TRIM(COALESCE(token_id, '')) != '' THEN TRIM(token_id)
    ELSE TRIM(COALESCE(token_name, ''))
  END AS token_key,
  COALESCE(MAX(NULLIF(token_id, '')), '') AS token_id,
  COALESCE(MAX(NULLIF(token_name, '')), '') AS token_name,
  TRIM(model) AS model,
  COUNT(*) AS request_count,
  SUM(
    CASE
      WHEN COALESCE(input_tokens, 0) > 0 THEN input_tokens
      WHEN COALESCE(total_tokens, 0) > COALESCE(output_tokens, 0) THEN COALESCE(total_tokens, 0) - COALESCE(output_tokens, 0)
      ELSE 0
    END
  ) AS input_tokens,
  SUM(COALESCE(output_tokens, 0)) AS output_tokens,
  SUM(
    CASE
      WHEN COALESCE(total_tokens, 0) > 0 THEN total_tokens
      ELSE COALESCE(input_tokens, 0) + COALESCE(output_tokens, 0)
    END
  ) AS total_tokens,
  MAX(time) AS updated_at
FROM request_history
WHERE TRIM(COALESCE(model, '')) != ''
  AND (
    COALESCE(total_tokens, 0) > 0 OR
    COALESCE(input_tokens, 0) > 0 OR
    COALESCE(output_tokens, 0) > 0
  )
  AND UPPER(TRIM(COALESCE(method, ''))) != 'CHECK'
  AND LOWER(TRIM(COALESCE(path, ''))) NOT LIKE '/maintenance/%'
  AND LOWER(TRIM(COALESCE(protocol, ''))) NOT IN ('health-check', 'quota-refresh', 'token-validation')
GROUP BY
  date,
  provider,
  protocol,
  client_key,
  CASE
    WHEN TRIM(COALESCE(token_id, '')) != '' THEN TRIM(token_id)
    ELSE TRIM(COALESCE(token_name, ''))
  END,
  model`)
	return err
}

func (s *SQLiteStore) rebuildBillingLifetimeIfEmptyLocked() error {
	var lifetimeCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM billing_lifetime_summary`).Scan(&lifetimeCount); err != nil {
		return err
	}
	if lifetimeCount > 0 {
		return nil
	}
	var out BillingSummary
	if err := s.db.QueryRow(`
SELECT
  COALESCE(SUM(request_count), 0),
  COALESCE(SUM(input_tokens), 0),
  COALESCE(SUM(output_tokens), 0),
  COALESCE(SUM(total_tokens), 0)
FROM billing_daily_usage`).Scan(&out.RequestCount, &out.InputTokens, &out.OutputTokens, &out.TotalTokens); err != nil {
		return err
	}
	if out.RequestCount == 0 && out.TotalTokens == 0 {
		return nil
	}
	_, err := s.db.Exec(`
INSERT INTO billing_lifetime_summary (id, request_count, input_tokens, output_tokens, total_tokens, updated_at)
VALUES (1, ?, ?, ?, ?, ?)`,
		out.RequestCount,
		out.InputTokens,
		out.OutputTokens,
		out.TotalTokens,
		time.Now().Format(time.RFC3339Nano),
	)
	return err
}

func (s *SQLiteStore) rebuildRequestDailySummaryIfEmptyLocked() error {
	var historyCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM request_history`).Scan(&historyCount); err != nil {
		return err
	}
	if historyCount == 0 {
		return nil
	}
	var summaryCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM request_daily_summary`).Scan(&summaryCount); err != nil {
		return err
	}
	if summaryCount > 0 {
		return nil
	}
	_, err := s.db.Exec(`
INSERT INTO request_daily_summary (
  date, provider, protocol, client_key, client_name, client_label,
  level, status, model, token_key, token_id, token_name,
  request_count, failed_count, total_tokens, duration_ms, updated_at
)
SELECT
  substr(time, 1, 10) AS date,
  COALESCE(provider, '') AS provider,
  COALESCE(protocol, '') AS protocol,
  COALESCE(client_key, '') AS client_key,
  COALESCE(MAX(NULLIF(client_name, '')), '') AS client_name,
  COALESCE(MAX(NULLIF(` + clientRankLabelSQL + `, '')), '') AS client_label,
  COALESCE(level, '') AS level,
  COALESCE(status, 0) AS status,
  TRIM(COALESCE(model, '')) AS model,
  CASE
    WHEN TRIM(COALESCE(token_id, '')) != '' THEN TRIM(token_id)
    ELSE TRIM(COALESCE(token_name, ''))
  END AS token_key,
  COALESCE(MAX(NULLIF(token_id, '')), '') AS token_id,
  COALESCE(MAX(NULLIF(token_name, '')), '') AS token_name,
  COUNT(*) AS request_count,
  COALESCE(SUM(CASE WHEN ` + failedHistorySQL + ` THEN 1 ELSE 0 END), 0) AS failed_count,
  COALESCE(SUM(
    CASE
      WHEN COALESCE(total_tokens, 0) > 0 THEN total_tokens
      ELSE COALESCE(input_tokens, 0) + COALESCE(output_tokens, 0)
    END
  ), 0) AS total_tokens,
  COALESCE(SUM(COALESCE(duration_ms, 0)), 0) AS duration_ms,
  MAX(time) AS updated_at
FROM request_history
GROUP BY
  substr(time, 1, 10),
  COALESCE(provider, ''),
  COALESCE(protocol, ''),
  COALESCE(client_key, ''),
  COALESCE(level, ''),
  COALESCE(status, 0),
  TRIM(COALESCE(model, '')),
  CASE
    WHEN TRIM(COALESCE(token_id, '')) != '' THEN TRIM(token_id)
    ELSE TRIM(COALESCE(token_name, ''))
  END`)
	return err
}
