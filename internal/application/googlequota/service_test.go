package googlequota

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/google"
)

func fixture(t *testing.T) (*Service, *sql.DB) {
	t.Helper()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "google.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a", "b"} {
		if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at) VALUES(?,'builtin_google',?,'private-ref','now','v1')`, id, id); err != nil {
			t.Fatal(err)
		}
	}
	return New(db), db
}

func TestProjectGroupingCredentialIsolationAndSafeObservations(t *testing.T) {
	s, db := fixture(t)
	ctx := context.Background()
	for _, id := range []string{"a", "b"} {
		if err := s.Bind(ctx, BindingInput{AccountID: id, ProjectID: "shared-project", AccountUpdatedAt: "v1"}); err != nil {
			t.Fatal(err)
		}
		if err := s.Record(ctx, id, "v1", "gemini-test", google.Failure{Category: "rate_limited", RetrySeconds: 30}); err != nil {
			t.Fatal(err)
		}
	}
	resources, err := s.List(ctx)
	if err != nil || len(resources) != 1 || len(resources[0].AccountIDs) != 2 || len(resources[0].Observations) != 1 || resources[0].Verified || resources[0].OfficialRemaining != nil {
		t.Fatal("shared project incorrectly represented", err)
	}
	encoded, _ := json.Marshal(resources)
	if strings.Contains(string(encoded), "private-ref") {
		t.Fatal("private credential version exposed")
	}
	if _, err := db.Exec(`UPDATE accounts SET name='renamed',updated_at='v2' WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	resources, err = s.List(ctx)
	if err != nil || len(resources) != 1 {
		t.Fatal("rename invalidated binding")
	}
	if err := s.Record(ctx, "a", "v1", "gemini-new", google.Failure{Category: "payment_required"}); err != ErrChanged {
		t.Fatal("in-flight stale result accepted", err)
	}
	if _, err := db.Exec(`UPDATE accounts SET credential_ref='new-ref',updated_at='v3' WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	resources, err = s.List(ctx)
	if err != nil || len(resources) != 2 {
		t.Fatal("new credentials retained old binding")
	}
	for _, resource := range resources {
		if resource.ID == "account:a" && (resource.ProjectID != "" || len(resource.Observations) != 0) {
			t.Fatal("stale key resource data retained")
		}
	}
	if err := s.Bind(ctx, BindingInput{AccountID: "a", ProjectID: "shared-project", AccountUpdatedAt: "v2"}); err != ErrChanged {
		t.Fatal("stale binding edit accepted")
	}
}

func TestBindingValidationAndExpiredObservation(t *testing.T) {
	s, db := fixture(t)
	ctx := context.Background()
	for _, project := range []string{"SHORT", "abc", strings.Repeat("a", 31), "project/id", "-project", "project-"} {
		if s.Bind(ctx, BindingInput{AccountID: "a", ProjectID: project, AccountUpdatedAt: "v1"}) != ErrInvalid {
			t.Error("invalid project accepted")
		}
	}
	if s.Record(ctx, "a", "v1", "gemini-test", google.Failure{Category: "rate_limited", Metric: "private error text"}) != ErrInvalid {
		t.Fatal("unvalidated metric accepted")
	}
	if err := s.Record(ctx, "a", "v1", "gemini-test", google.Failure{Category: "daily_exhausted"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE google_resource_observations SET retry_at='2020-01-01T00:00:00Z'`); err != nil {
		t.Fatal(err)
	}
	resources, err := s.List(ctx)
	if err != nil || !resources[0].Observations[0].Stale || resources[0].OfficialRemaining != nil {
		t.Fatal("expired observation treated as recovered")
	}
	if err := s.Bind(ctx, BindingInput{AccountID: "a", ProjectID: "valid-project", AccountUpdatedAt: "v1"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Bind(ctx, BindingInput{AccountID: "a", AccountUpdatedAt: "v1"}); err != nil {
		t.Fatal("unbind failed")
	}
}

func TestDailyResetAccountsForDST(t *testing.T) {
	for _, test := range []struct{ now, want string }{
		{"2026-03-08T07:30:00Z", "2026-03-08T08:00:00Z"},
		{"2026-03-08T09:00:00Z", "2026-03-09T07:00:00Z"},
		{"2026-11-01T08:00:00Z", "2026-11-02T08:00:00Z"},
	} {
		now, _ := time.Parse(time.RFC3339, test.now)
		if NextDailyReset(now).Format(time.RFC3339) != test.want {
			t.Error("incorrect Pacific reset", test.now)
		}
	}
}
