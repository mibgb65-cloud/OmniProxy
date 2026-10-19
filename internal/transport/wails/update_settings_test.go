package wails

import (
	"testing"

	"omniproxy/internal/platform/profile"
)

func TestUpdateChannelDefaultsAndUserChoice(t *testing.T) {
	for _, tc := range []struct{ current, saved, want string }{
		{"2.0.0-beta.1", "", "beta"}, {"2.0.0", "", "stable"}, {"2.0.0-beta.1", "stable", "stable"}, {"2.0.0", "beta", "beta"},
	} {
		if got := updatePreferences(profile.Config{UpdateChannel: tc.saved}, tc.current); got.Channel != tc.want {
			t.Errorf("%+v => %+v", tc, got)
		}
	}
	svc, _ := newTestService(t)
	initial := profile.Config{DataDir: "C:/retained", CloseBehavior: "tray"}
	if err := profile.SaveConfig(profile.Test, initial); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetUpdateChannel("nightly"); err == nil {
		t.Fatal("invalid channel accepted")
	}
	if _, err := svc.SetUpdateChannel("stable"); err != nil {
		t.Fatal(err)
	}
	stored, err := profile.LoadConfig(profile.Test)
	if err != nil || stored.DataDir != initial.DataDir || stored.CloseBehavior != initial.CloseBehavior || stored.UpdateChannel != "stable" {
		t.Fatalf("lost existing config: %+v %v", stored, err)
	}
}

func TestAutomaticDownloadPreferenceIsOptInAndPreservesConfig(t *testing.T) {
	svc, _ := newTestService(t)
	initial, err := svc.GetUpdatePreferences()
	if err != nil || initial.AutoDownload {
		t.Fatalf("automatic download must default to off: %+v %v", initial, err)
	}
	check := false
	original := profile.Config{DataDir: "C:/retained", CloseBehavior: "tray", UpdateChannel: "stable", AutoCheckUpdates: &check}
	if err := profile.SaveConfig(profile.Test, original); err != nil {
		t.Fatal(err)
	}
	for _, enabled := range []bool{true, false} {
		prefs, err := svc.SetAutomaticUpdateDownload(enabled)
		if err != nil || prefs.AutoDownload != enabled || prefs.AutoCheck {
			t.Fatalf("preferences=%+v err=%v", prefs, err)
		}
		stored, err := profile.LoadConfig(profile.Test)
		if err != nil || stored.AutoDownloadUpdates != enabled || stored.DataDir != original.DataDir || stored.CloseBehavior != original.CloseBehavior || stored.UpdateChannel != original.UpdateChannel {
			t.Fatalf("unrelated config changed: %+v %v", stored, err)
		}
	}
}
