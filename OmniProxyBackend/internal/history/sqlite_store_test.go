package history

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestSQLiteStoreEnablesWALJournal(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "request_history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	var mode string
	if err := store.db.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(mode, "wal") {
		t.Fatalf("expected WAL journal mode, got %q", mode)
	}
}

func TestSQLiteStoreListFiltersAndPrunes(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "request_history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	recorder, err := NewRecorder(store, 2)
	if err != nil {
		t.Fatal(err)
	}
	recorder.pruneEvery = 1
	recorder.Add(Entry{
		Time:        time.Now().Add(-2 * time.Minute),
		Level:       "info",
		Method:      "POST",
		Path:        "/v1/chat/completions",
		Provider:    "openai",
		ClientKey:   "codex",
		ClientName:  "Codex",
		Model:       "gpt-5.5",
		Status:      200,
		TokenName:   "primary",
		TotalTokens: 42,
		Message:     "request proxied",
	})
	recorder.Add(Entry{
		Time:              time.Now().Add(-time.Minute),
		Level:             "warn",
		Method:            "POST",
		Path:              "/v1/chat/completions",
		Provider:          "openai",
		ClientKey:         "opencode",
		ClientName:        "OpenCode",
		Model:             "gpt-5.5",
		Status:            429,
		TokenName:         "backup",
		CooldownTriggered: true,
		RetryChain: []RetryAttempt{{
			Attempt:           1,
			Provider:          "openai",
			Status:            429,
			TokenName:         "backup",
			CooldownTriggered: true,
		}},
		Message: "upstream returned 429",
	})
	recorder.Add(Entry{
		Time:       time.Now(),
		Level:      "error",
		Method:     "POST",
		Path:       "/anthropic-router/v1/messages",
		Provider:   "deepseek",
		Protocol:   "anthropic",
		ClientKey:  "claude",
		ClientName: "Claude Code",
		Model:      "deepseek-v4",
		Status:     502,
		TokenName:  "deepseek-main",
		Message:    "proxy failed",
	})

	if all := recorder.List(Filter{}); len(all) != 2 {
		t.Fatalf("expected prune to keep latest 2 entries, got %d: %#v", len(all), all)
	}
	errors := recorder.List(Filter{Status: "error"})
	if len(errors) != 2 {
		t.Fatalf("expected two error entries, got %#v", errors)
	}
	deepseek := recorder.List(Filter{Provider: "deepseek", Search: "anthropic"})
	if len(deepseek) != 1 || deepseek[0].Protocol != "anthropic" {
		t.Fatalf("expected filtered deepseek anthropic entry, got %#v", deepseek)
	}
	openai := recorder.List(Filter{Model: "gpt-5.5"})
	if len(openai) != 1 || openai[0].Model != "gpt-5.5" {
		t.Fatalf("expected persisted model metadata, got %#v", openai)
	}
	claude := recorder.List(Filter{Client: "claude"})
	if len(claude) != 1 || claude[0].ClientName != "Claude Code" {
		t.Fatalf("expected persisted client metadata, got %#v", claude)
	}
	cooldown := recorder.List(Filter{Status: "429"})
	if len(cooldown) != 1 || !cooldown[0].CooldownTriggered || len(cooldown[0].RetryChain) != 1 {
		t.Fatalf("expected persisted retry chain and cooldown state, got %#v", cooldown)
	}
}

func TestSQLiteStoreSaveImportsLegacyEntries(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "request_history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	entries := []Entry{{
		ID:       12,
		Time:     time.Now(),
		Level:    "info",
		Provider: "openai",
		Status:   200,
		Message:  "legacy",
	}}
	if err := store.Save(entries); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.List(Filter{Status: "success"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].ID != 12 || loaded[0].Message != "legacy" {
		t.Fatalf("expected imported legacy entry, got %#v", loaded)
	}
}

func TestSQLiteStoreSaveAssignsIDsForLegacyEntriesWithoutIDs(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "request_history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	entries := []Entry{
		{
			Time:    time.Now().Add(-time.Minute),
			Level:   "info",
			Status:  200,
			Message: "first legacy entry",
		},
		{
			Time:    time.Now(),
			Level:   "warn",
			Status:  429,
			Message: "second legacy entry",
		},
	}
	if err := store.Save(entries); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.List(Filter{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected both legacy entries, got %#v", loaded)
	}
	if loaded[0].ID == 0 || loaded[1].ID == 0 || loaded[0].ID == loaded[1].ID {
		t.Fatalf("expected generated unique IDs, got %#v", loaded)
	}
}

func TestSQLiteStoreDailyUsageSummarizesBillableEntries(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "request_history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	recorder, err := NewRecorder(store, 10)
	if err != nil {
		t.Fatal(err)
	}
	usageDate := time.Date(2026, 5, 1, 10, 30, 0, 0, time.Local)
	recorder.Add(Entry{
		Time:         usageDate,
		Level:        "info",
		Method:       "POST",
		Path:         "/v1/chat/completions",
		Provider:     "openai",
		Protocol:     "openai",
		ClientKey:    "codex",
		ClientName:   "Codex",
		Model:        "gpt-5.5",
		Status:       200,
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
		Message:      "request proxied",
	})
	recorder.Add(Entry{
		Time:        usageDate.Add(time.Minute),
		Level:       "info",
		Method:      "CHECK",
		Path:        "/maintenance/current-token-quota-refresh",
		Provider:    "openai",
		Protocol:    "quota-refresh",
		Model:       "codex_auth_json",
		Status:      200,
		TotalTokens: 999,
		Message:     "quota refresh completed",
	})
	recorder.Add(Entry{
		Time:        usageDate.Add(2 * time.Minute),
		Level:       "info",
		Method:      "POST",
		Path:        "/v1/chat/completions",
		Provider:    "openai",
		Status:      200,
		TotalTokens: 88,
		Message:     "missing model",
	})

	usage, err := store.DailyUsage("2026-05-01", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(usage) != 1 {
		t.Fatalf("expected one billable usage row, got %#v", usage)
	}
	if usage[0].Model != "gpt-5.5" || usage[0].RequestCount != 1 || usage[0].InputTokens != 100 || usage[0].OutputTokens != 50 || usage[0].TotalTokens != 150 {
		t.Fatalf("unexpected usage row: %#v", usage[0])
	}
	dates, err := store.DailyUsageDates(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(dates) != 1 || dates[0] != "2026-05-01" {
		t.Fatalf("expected usage date to be listed, got %#v", dates)
	}
}

func TestSQLiteStoreBillingSummaryKeepsLifetimeUsage(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "request_history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	recorder, err := NewRecorder(store, 10)
	if err != nil {
		t.Fatal(err)
	}
	recentDay := time.Now()
	oldDay := recentDay.AddDate(0, 0, -2)
	recorder.Add(Entry{
		Time:         oldDay,
		Level:        "info",
		Method:       "POST",
		Path:         "/v1/chat/completions",
		Provider:     "openai",
		Protocol:     "openai",
		ClientKey:    "deleted-account",
		ClientName:   "Deleted Account",
		Model:        "gpt-5.5",
		Status:       200,
		InputTokens:  1_200_000_000,
		OutputTokens: 800_000_000,
		TotalTokens:  2_000_000_000,
		Message:      "request proxied",
	})
	recorder.Add(Entry{
		Time:         recentDay,
		Level:        "info",
		Method:       "POST",
		Path:         "/v1/chat/completions",
		Provider:     "openai",
		Protocol:     "openai",
		ClientKey:    "active-account",
		ClientName:   "Active Account",
		Model:        "gpt-5.5",
		Status:       200,
		InputTokens:  70,
		OutputTokens: 53,
		TotalTokens:  123,
		Message:      "request proxied",
	})
	recorder.Add(Entry{
		Time:        recentDay,
		Level:       "info",
		Method:      "CHECK",
		Path:        "/maintenance/current-token-quota-refresh",
		Provider:    "openai",
		Protocol:    "quota-refresh",
		Model:       "codex_auth_json",
		Status:      200,
		TotalTokens: 999,
		Message:     "quota refresh completed",
	})

	summary := recorder.BillingSummary(30)
	if summary.RequestCount != 2 || summary.InputTokens != 1_200_000_070 || summary.OutputTokens != 800_000_053 || summary.TotalTokens != 2_000_000_123 {
		t.Fatalf("unexpected lifetime billing summary: %#v", summary)
	}
	if len(summary.DailyRows) != 2 {
		t.Fatalf("expected recent daily rows, got %#v", summary.DailyRows)
	}

	if err := recorder.SetRetentionDays(1); err != nil {
		t.Fatal(err)
	}
	summary = recorder.BillingSummary(30)
	if summary.TotalTokens != 2_000_000_123 {
		t.Fatalf("expected retention to keep lifetime total, got %#v", summary)
	}
	if len(summary.DailyRows) != 1 || summary.DailyRows[0].TotalTokens != 123 {
		t.Fatalf("expected retention to prune only daily rows, got %#v", summary.DailyRows)
	}

	if err := recorder.ClearRequestHistory(); err != nil {
		t.Fatal(err)
	}
	summary = recorder.BillingSummary(30)
	if summary.TotalTokens != 2_000_000_123 {
		t.Fatalf("expected clearing request history to keep billing total, got %#v", summary)
	}

	if err := recorder.ClearDailyUsage(); err != nil {
		t.Fatal(err)
	}
	summary = recorder.BillingSummary(30)
	if summary.RequestCount != 0 || summary.TotalTokens != 0 || len(summary.DailyRows) != 0 {
		t.Fatalf("expected clearing billing usage to reset billing summary, got %#v", summary)
	}
}

func TestSQLiteStoreSummaryAggregatesAllMatchingHistory(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "request_history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	recorder, err := NewRecorder(store, 10)
	if err != nil {
		t.Fatal(err)
	}
	firstDay := time.Date(2026, 5, 1, 10, 30, 0, 0, time.Local)
	secondDay := firstDay.AddDate(0, 0, 1)
	recorder.Add(Entry{
		Time:        firstDay,
		Level:       "info",
		Method:      "POST",
		Path:        "/v1/chat/completions",
		Provider:    "openai",
		ClientKey:   "codex",
		ClientName:  "Codex",
		Model:       "gpt-5.5",
		Status:      200,
		Duration:    10,
		TokenName:   "primary",
		TotalTokens: 100,
		Message:     "request proxied",
	})
	recorder.Add(Entry{
		Time:        secondDay,
		Level:       "warn",
		Method:      "POST",
		Path:        "/v1/chat/completions",
		Provider:    "deepseek",
		ClientKey:   "opencode",
		ClientName:  "OpenCode",
		Model:       "deepseek-v4-pro",
		Status:      429,
		Duration:    20,
		TokenName:   "backup",
		TotalTokens: 25,
		Message:     "rate limited",
	})
	recorder.Add(Entry{
		Time:        secondDay.Add(time.Minute),
		Level:       "info",
		Method:      "POST",
		Path:        "/v1/chat/completions",
		Provider:    "openai",
		ClientKey:   "codex",
		ClientName:  "Codex",
		Model:       "gpt-5.5",
		Status:      200,
		Duration:    30,
		TokenName:   "primary",
		TotalTokens: 50,
		Message:     "request proxied",
	})

	filtered, err := store.Summary(Filter{Provider: "openai", Limit: 1}, 14)
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Total != 2 || filtered.Failed != 0 || filtered.TotalTokens != 150 || filtered.AverageDuration != 20 {
		t.Fatalf("unexpected filtered summary: %#v", filtered)
	}
	if row := dailySummaryForDate(filtered.DailyRows, "2026-05-01"); row.RequestCount != 1 || row.TotalTokens != 100 {
		t.Fatalf("unexpected first daily row: %#v", row)
	}
	if row := dailySummaryForDate(filtered.DailyRows, "2026-05-02"); row.RequestCount != 1 || row.TotalTokens != 50 {
		t.Fatalf("unexpected second daily row: %#v", row)
	}
	if len(filtered.ModelRanks) != 1 || filtered.ModelRanks[0].Label != "gpt-5.5" || filtered.ModelRanks[0].TotalTokens != 150 {
		t.Fatalf("unexpected model ranks: %#v", filtered.ModelRanks)
	}

	all, err := store.Summary(Filter{}, 14)
	if err != nil {
		t.Fatal(err)
	}
	if all.Total != 3 || all.Failed != 1 || all.TotalTokens != 175 || all.FailureRate != 33 {
		t.Fatalf("unexpected full summary: %#v", all)
	}
	if len(all.TokenFailureRanks) != 1 || all.TokenFailureRanks[0].Label != "backup" || all.TokenFailureRanks[0].Count != 1 {
		t.Fatalf("unexpected token failure ranks: %#v", all.TokenFailureRanks)
	}
	if len(all.FailureReasonRanks) != 1 || all.FailureReasonRanks[0].Label != "429__reason_sep__rate limited" {
		t.Fatalf("unexpected failure reason ranks: %#v", all.FailureReasonRanks)
	}
}

func TestSQLiteStoreSummarySurvivesDetailPrune(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "request_history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	recorder, err := NewRecorder(store, 2)
	if err != nil {
		t.Fatal(err)
	}
	baseTime := time.Date(2026, 5, 1, 10, 0, 0, 0, time.Local)
	recorder.Add(Entry{
		Time:        baseTime,
		Level:       "error",
		Method:      "POST",
		Path:        "/v1/chat/completions",
		Provider:    "anthropic",
		ClientKey:   "claude",
		ClientName:  "Claude Code",
		Model:       "claude-sonnet",
		Status:      500,
		Duration:    40,
		TokenName:   "old-token",
		TotalTokens: 10,
		Message:     "old failure",
	})
	recorder.Add(Entry{
		Time:        baseTime.Add(time.Minute),
		Level:       "info",
		Method:      "POST",
		Path:        "/v1/chat/completions",
		Provider:    "openai",
		ClientKey:   "codex",
		ClientName:  "Codex",
		Model:       "gpt-5.5",
		Status:      200,
		Duration:    20,
		TokenName:   "primary",
		TotalTokens: 20,
		Message:     "request proxied",
	})
	recorder.Add(Entry{
		Time:        baseTime.Add(2 * time.Minute),
		Level:       "info",
		Method:      "POST",
		Path:        "/v1/chat/completions",
		Provider:    "deepseek",
		ClientKey:   "opencode",
		ClientName:  "OpenCode",
		Model:       "deepseek-v4",
		Status:      200,
		Duration:    30,
		TokenName:   "backup",
		TotalTokens: 30,
		Message:     "request proxied",
	})

	if entries := recorder.List(Filter{}); len(entries) != 2 {
		t.Fatalf("expected request details to be pruned to latest 2 rows, got %#v", entries)
	}
	all, err := store.Summary(Filter{}, 14)
	if err != nil {
		t.Fatal(err)
	}
	if all.Total != 3 || all.Failed != 1 || all.TotalTokens != 60 || all.AverageDuration != 30 {
		t.Fatalf("expected summary to include pruned details, got %#v", all)
	}
	if row := dailySummaryForDate(all.DailyRows, "2026-05-01"); row.RequestCount != 3 || row.FailedCount != 1 || row.TotalTokens != 60 {
		t.Fatalf("unexpected daily summary after prune: %#v", row)
	}
	filtered, err := store.Summary(Filter{Provider: "anthropic"}, 14)
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Total != 1 || filtered.Failed != 1 || filtered.TotalTokens != 10 {
		t.Fatalf("expected filtered summary to include pruned provider, got %#v", filtered)
	}
}

func TestSQLiteStoreClearRequestHistoryKeepsDailySummary(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "request_history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	recorder, err := NewRecorder(store, 10)
	if err != nil {
		t.Fatal(err)
	}
	baseTime := time.Date(2026, 5, 2, 10, 0, 0, 0, time.Local)
	recorder.Add(Entry{
		Time:        baseTime,
		Level:       "info",
		Method:      "POST",
		Path:        "/v1/chat/completions",
		Provider:    "openai",
		ClientKey:   "codex",
		ClientName:  "Codex",
		Model:       "gpt-5.5",
		Status:      200,
		Duration:    10,
		TokenName:   "primary",
		TotalTokens: 25,
		Message:     "request proxied",
	})
	recorder.Add(Entry{
		Time:        baseTime.Add(time.Minute),
		Level:       "warn",
		Method:      "POST",
		Path:        "/v1/chat/completions",
		Provider:    "openai",
		ClientKey:   "codex",
		ClientName:  "Codex",
		Model:       "gpt-5.5",
		Status:      429,
		Duration:    30,
		TokenName:   "backup",
		TotalTokens: 35,
		Message:     "rate limited",
	})
	if err := recorder.ClearRequestHistory(); err != nil {
		t.Fatal(err)
	}

	if entries := recorder.List(Filter{}); len(entries) != 0 {
		t.Fatalf("expected request details to be cleared, got %#v", entries)
	}
	summary, err := store.Summary(Filter{}, 14)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Total != 2 || summary.Failed != 1 || summary.TotalTokens != 60 || summary.AverageDuration != 20 {
		t.Fatalf("expected daily summary to survive history clear, got %#v", summary)
	}
	if row := dailySummaryForDate(summary.DailyRows, "2026-05-02"); row.RequestCount != 2 || row.FailedCount != 1 || row.TotalTokens != 60 {
		t.Fatalf("unexpected daily summary after history clear: %#v", row)
	}
}

func dailySummaryForDate(rows []DailySummary, date string) DailySummary {
	for _, row := range rows {
		if row.Date == date {
			return row
		}
	}
	return DailySummary{}
}

func TestRecorderRetentionPrunesHistoryAndDailyUsageByDate(t *testing.T) {
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "request_history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	recorder, err := NewRecorder(store, 10)
	if err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().AddDate(0, 0, -20)
	recentTime := time.Now()
	recorder.Add(Entry{
		Time:        oldTime,
		Level:       "info",
		Method:      "POST",
		Path:        "/v1/chat/completions",
		Provider:    "openai",
		Model:       "gpt-5.4",
		Status:      200,
		TotalTokens: 100,
		Message:     "old",
	})
	recorder.Add(Entry{
		Time:        recentTime,
		Level:       "info",
		Method:      "POST",
		Path:        "/v1/chat/completions",
		Provider:    "openai",
		Model:       "gpt-5.5",
		Status:      200,
		TotalTokens: 200,
		Message:     "recent",
	})
	if err := recorder.SetRetentionDays(14); err != nil {
		t.Fatal(err)
	}

	entries := recorder.List(Filter{})
	if len(entries) != 1 || entries[0].Model != "gpt-5.5" {
		t.Fatalf("expected only recent history after retention, got %#v", entries)
	}
	dates, err := store.DailyUsageDates(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(dates) != 1 || dates[0] != recentTime.Format("2006-01-02") {
		t.Fatalf("expected only recent usage date after retention, got %#v", dates)
	}
	summary, err := store.Summary(Filter{}, 30)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Total != 1 || summary.TotalTokens != 200 {
		t.Fatalf("expected only recent request summary after retention, got %#v", summary)
	}
	if row := dailySummaryForDate(summary.DailyRows, oldTime.Format("2006-01-02")); row.RequestCount != 0 {
		t.Fatalf("expected old request summary to be pruned, got %#v", row)
	}
}

func TestSQLiteStoreMigratesExistingRowsWithoutClientColumns(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "request_history.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
CREATE TABLE request_history (
  id INTEGER PRIMARY KEY,
  time TEXT NOT NULL,
  level TEXT NOT NULL,
  method TEXT,
  path TEXT,
  provider TEXT,
  protocol TEXT,
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
)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO request_history (id, time, level, provider, protocol, model, status, retry_chain, message) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		1, time.Now().Format(time.RFC3339Nano), "info", "openai", "openai", "gpt-5.5", 200, "[]", "legacy request"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	loaded, err := store.List(Filter{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].ClientKey != "" || loaded[0].Message != "legacy request" {
		t.Fatalf("expected legacy row to load with empty client fields, got %#v", loaded)
	}
}
