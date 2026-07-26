package history

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newPruneTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "request_history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return store
}

func historyRowCount(t *testing.T, store *SQLiteStore) int {
	t.Helper()
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM request_history`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func queryPlan(t *testing.T, store *SQLiteStore, query string, args ...any) string {
	t.Helper()
	rows, err := store.db.Query("EXPLAIN QUERY PLAN "+query, args...)
	if err != nil {
		t.Fatalf("explain %q: %v", query, err)
	}
	defer rows.Close()

	details := []string{}
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		if err := rows.Scan(&id, &parent, &notUsed, &detail); err != nil {
			t.Fatal(err)
		}
		details = append(details, detail)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return strings.Join(details, " | ")
}

func TestRecorderDefersPruningBetweenIntervals(t *testing.T) {
	store := newPruneTestStore(t)
	recorder, err := NewRecorder(store, 2)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		recorder.Add(Entry{Time: time.Now(), Level: "info", Message: "entry"})
	}
	// Pruning scans the whole table, so the row cap is allowed to overshoot
	// until the interval elapses.
	if got := historyRowCount(t, store); got != 5 {
		t.Fatalf("expected pruning to be deferred, got %d rows", got)
	}
	// Reads stay correct while the table is over the cap.
	if listed := recorder.List(Filter{}); len(listed) != 2 {
		t.Fatalf("expected reads to honour the cap, got %d entries", len(listed))
	}

	recorder.pruneEvery = 1
	recorder.Add(Entry{Time: time.Now(), Level: "info", Message: "entry"})
	if got := historyRowCount(t, store); got != 2 {
		t.Fatalf("expected the cap to be enforced once the interval elapsed, got %d rows", got)
	}
}

func TestNewRecorderEnforcesCapAtStartup(t *testing.T) {
	store := newPruneTestStore(t)
	for i := 0; i < 6; i++ {
		if err := store.Append(Entry{ID: int64(i + 1), Time: time.Now(), Level: "info"}); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := NewRecorder(store, 2); err != nil {
		t.Fatal(err)
	}
	if got := historyRowCount(t, store); got != 2 {
		t.Fatalf("expected startup to enforce the cap, got %d rows", got)
	}
}

func TestPruneBeforeDateKeepsCutoffDayEntries(t *testing.T) {
	store := newPruneTestStore(t)
	rows := []struct {
		id int64
		at string
	}{
		{1, "2026-06-30T23:59:59Z"},
		{2, "2026-07-01T00:00:00Z"},
		{3, "2026-07-01T12:00:00Z"},
		{4, "2026-07-02T00:00:00Z"},
	}
	for _, row := range rows {
		at, err := time.Parse(time.RFC3339, row.at)
		if err != nil {
			t.Fatal(err)
		}
		if err := store.Append(Entry{ID: row.id, Time: at, Level: "info"}); err != nil {
			t.Fatal(err)
		}
	}

	if err := store.PruneBeforeDate("2026-07-01"); err != nil {
		t.Fatal(err)
	}
	// Everything before the cutoff day goes; the cutoff day itself survives.
	if got := historyRowCount(t, store); got != 3 {
		t.Fatalf("expected the cutoff day to survive, got %d rows", got)
	}
}

func TestRequestHistoryQueriesUseIndexes(t *testing.T) {
	store := newPruneTestStore(t)
	for _, testCase := range []struct {
		name  string
		query string
	}{
		{"provider filter", `SELECT id FROM request_history WHERE provider = ? COLLATE NOCASE`},
		{"client filter", `SELECT id FROM request_history WHERE client_key = ? COLLATE NOCASE`},
		{"level filter", `SELECT id FROM request_history WHERE level = ? COLLATE NOCASE`},
		{"token filter", `SELECT id FROM request_history WHERE token_id = ? COLLATE NOCASE`},
		{"token relabel", `SELECT id FROM request_history WHERE token_id = ?`},
		{"retention prune", `DELETE FROM request_history WHERE time < ?`},
	} {
		plan := queryPlan(t, store, testCase.query, "value")
		if !strings.Contains(plan, "USING INDEX") && !strings.Contains(plan, "USING COVERING INDEX") {
			t.Fatalf("%s: expected an index lookup, got %q", testCase.name, plan)
		}
	}
}
