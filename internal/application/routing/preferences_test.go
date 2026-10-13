package routing

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestPreferenceInputAndResourceValidation(t *testing.T) {
	s, _ := preferenceFixture(t)
	ctx := context.Background()
	for _, input := range [][2]string{
		{"", "backup_ds"}, {"bad model", "backup_ds"}, {strings.Repeat("x", 129), "backup_ds"},
		{"deepseek-flash", "bad\nprovider"}, {"deepseek-flash", strings.Repeat("x", 129)},
		{"deepseek-flash", "missing"}, {"deepseek-flash", "builtin_openai"}, {"unknown", "backup_ds"},
	} {
		if err := s.SetPreferredProvider(ctx, input[0], input[1]); err == nil {
			t.Fatalf("accepted invalid preference %q", input)
		}
	}
}

func TestPreferenceCountLimitAndRecovery(t *testing.T) {
	s, db := preferenceFixture(t)
	_, err := db.Exec(`WITH RECURSIVE numbers(value) AS (SELECT 1 UNION ALL SELECT value+1 FROM numbers WHERE value<999)
INSERT INTO model_provider_preferences(model,provider_id,updated_at) SELECT 'saved_'||value,'builtin_deepseek','now' FROM numbers;
INSERT INTO model_provider_preferences(model,provider_id,updated_at) VALUES('deepseek-flash','builtin_deepseek','now');`)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := s.SetPreferredProvider(ctx, "deepseek-v4-pro", "backup_ds"); !errors.Is(err, ErrLimitReached) {
		t.Fatal("preference limit not enforced", err)
	}
	if err := s.SetPreferredProvider(ctx, "deepseek-flash", "backup_ds"); err != nil {
		t.Fatal("existing preference cannot be updated at limit", err)
	}
	if err := s.SetPreferredProvider(ctx, "saved_1", ""); err != nil {
		t.Fatal("stale preference cannot be cleared", err)
	}
	if err := s.SetPreferredProvider(ctx, "deepseek-v4-pro", "backup_ds"); err != nil {
		t.Fatal("freed preference slot cannot be reused", err)
	}
}

func TestPreferenceReadFailureDoesNotFallBackSilently(t *testing.T) {
	s, db := preferenceFixture(t)
	if _, err := db.Exec(`DROP TABLE model_provider_preferences`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Resolve(context.Background(), "deepseek-flash"); err == nil {
		t.Fatal("configuration failure was treated as no preference")
	}
}
