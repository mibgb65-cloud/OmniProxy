package routing

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestStepFunSettingsPersistValidateAndFailClosed(t *testing.T) {
	s, db := fixture(t)
	ctx := context.Background()
	if settings, err := s.GetStepFunSettings(ctx); err != nil || settings.PreferredBillingMode != "step_plan" {
		t.Fatal("default order missing", settings, err)
	}
	for _, mode := range []string{"payg", "step_plan"} {
		if settings, err := s.SetStepFunPreferredBillingMode(ctx, mode); err != nil || settings.PreferredBillingMode != mode {
			t.Fatal("save failed", settings, err)
		}
		if settings, err := New(db, nil, nil).GetStepFunSettings(ctx); err != nil || settings.PreferredBillingMode != mode {
			t.Fatal("order not persisted", settings, err)
		}
	}
	for _, invalid := range []string{"", "api_key", "token_plan", "STEP_PLAN", " payg", "payg\x00", strings.Repeat("x", 10000)} {
		if _, err := s.SetStepFunPreferredBillingMode(ctx, invalid); !errors.Is(err, ErrInvalidInput) {
			t.Fatal("invalid mode accepted")
		}
	}
	if settings, err := s.GetStepFunSettings(ctx); err != nil || settings.PreferredBillingMode != "step_plan" {
		t.Fatal("invalid input changed persisted order", err)
	}
	if settings, err := s.GetMiMoSettings(ctx); err != nil || settings.PreferredBillingMode != "token_plan" {
		t.Fatal("StepFun changes affected MiMo preference", settings, err)
	}
	if _, err := db.Exec(`PRAGMA ignore_check_constraints=ON; UPDATE stepfun_settings SET preferred_billing_mode='corrupt'`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetStepFunSettings(ctx); err == nil {
		t.Fatal("corrupt order silently defaulted")
	}
	if _, err := db.Exec(`DELETE FROM stepfun_settings`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetStepFunSettings(ctx); err == nil {
		t.Fatal("missing order silently defaulted")
	}
	if _, err := s.SetStepFunPreferredBillingMode(ctx, "payg"); err == nil {
		t.Fatal("missing record accepted save")
	}
	if _, err := New(nil, nil, nil).GetStepFunSettings(ctx); err == nil {
		t.Fatal("missing database accepted")
	}
	if _, err := New(nil, nil, nil).SetStepFunPreferredBillingMode(ctx, "payg"); err == nil {
		t.Fatal("missing database accepted save")
	}
}
