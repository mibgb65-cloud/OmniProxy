package activity

import (
	"context"
	"io/fs"
	"omniproxy/internal/infrastructure/database"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"
)

func TestDashboardMigrationBackfillMarksMissingHistory(t *testing.T) {
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "migration.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	files := fstest.MapFS{}
	entries, err := os.ReadDir("../../../migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() >= "010_dashboard_buckets.sql" {
			continue
		}
		raw, err := os.ReadFile("../../../migrations/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		files[entry.Name()] = &fstest.MapFile{Data: raw}
	}
	ctx := context.Background()
	if err := database.Migrate(ctx, db, files); err != nil {
		t.Fatal(err)
	}
	start := time.Now().Add(-time.Hour)
	if _, err := db.Exec(`INSERT INTO request_records(id,started_at,day,protocol,transport,state,latency_ms,input_tokens,output_tokens,usage_complete) VALUES('aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',?,'day','responses','http','success',100,20,10,1)`, timeText(start)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE usage_totals SET requests=3,input_tokens=90,output_tokens=30`); err != nil {
		t.Fatal(err)
	}
	all := os.DirFS("../../../migrations")
	if err := database.Migrate(ctx, db, fs.FS(all)); err != nil {
		t.Fatal(err)
	}
	s := New(db)
	info, err := s.Dashboard(ctx, DashboardQuery{Boundaries: []int64{dashboardBucket(start), dashboardBucket(start) + 7200}})
	if err != nil {
		t.Fatal(err)
	}
	if !info.HistoryIncomplete || info.Summary.Requests != 1 || info.Summary.TotalTokens != "30" {
		t.Fatal("missing historical requests were fabricated", info)
	}
	if info.Summary.FirstTokenSamples != 0 || info.Summary.AverageFirstTokenMS != nil || info.Recent[0].FirstTokenMS != nil {
		t.Fatal("legacy total latency was fabricated as first-token latency")
	}
	if err := database.Migrate(ctx, db, all); err != nil {
		t.Fatal(err)
	}
	var requests int64
	if db.QueryRow(`SELECT SUM(requests) FROM dashboard_buckets`).Scan(&requests) != nil || requests != 1 {
		t.Fatal("backfill ran more than once")
	}
}
