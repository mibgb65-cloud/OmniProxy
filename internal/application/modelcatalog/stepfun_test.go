package modelcatalog

import (
	"context"
	"io"
	"net/http"
	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/stepfun"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type stepRoundTrip func(*http.Request) (*http.Response, error)

func (f stepRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestStepModelSyncCannotWriteOldBillingVersion(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "step.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	accounts := account.NewService(db, googleStore{})
	a, err := accounts.Create(ctx, account.CreateInput{ProviderID: "builtin_stepfun", Name: "Step", APIKey: "placeholder-step-key"})
	if err != nil {
		t.Fatal(err)
	}
	s := New(db, accounts, provider.NewService(db), nil)
	items, err := s.List(ctx)
	if err != nil || len(items) != 4 || items[0].Source != "preset" || items[0].FetchedAt != "" {
		t.Fatal("preset pretended to be discovery")
	}
	s.stepfun = stepfun.New(stepRoundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://api.stepfun.com/v1/models" {
			t.Fatal("ordinary discovery path changed")
		}
		options := *a.StepFun
		options.BillingMode = "step_plan"
		if _, err := accounts.Update(ctx, account.UpdateInput{ID: a.ID, Name: a.Name, StepFun: &options}); err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"step-5-preview"}]}`))}, nil
	}))
	if _, err := s.Sync(ctx, a.ID); err == nil {
		t.Fatal("old ordinary result committed after switching to subscription")
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM account_models WHERE account_id=?`, a.ID).Scan(&count); err != nil || count != 0 {
		t.Fatal("stale discovery became active")
	}
	items, err = s.List(ctx)
	if err != nil || len(items) != 5 {
		t.Fatal("subscription presets unavailable")
	}
	if _, err := s.Sync(ctx, a.ID); err == nil {
		t.Fatal("subscription synchronized ordinary model API")
	}
}
