package profile

import (
	"os"
	"testing"
)

func TestGatewayAutoStartPersistsAndPreservesOtherSettings(t *testing.T) {
	useTempConfigDir(t)
	cfg, err := LoadConfig(Test)
	if err != nil || cfg.GatewayAutoStart {
		t.Fatalf("default gateway startup: %+v, %v", cfg, err)
	}
	before := Config{DataDir: `D:\data`, UpdateRepo: "example/omniproxy", CloseBehavior: CloseToTray, AutoDownloadUpdates: true}
	if err := SaveConfig(Test, before); err != nil {
		t.Fatal(err)
	}
	for _, enabled := range []bool{true, false} {
		if err := SetGatewayAutoStart(Test, enabled); err != nil {
			t.Fatal(err)
		}
		got, err := LoadConfig(Test)
		if err != nil {
			t.Fatal(err)
		}
		want := before
		want.GatewayAutoStart = enabled
		if got != want {
			t.Fatalf("unrelated settings changed: %+v", got)
		}
	}
}

func TestGatewayAutoStartRejectsMalformedConfig(t *testing.T) {
	useTempConfigDir(t)
	path := ConfigFilePath(Test)
	for _, raw := range []string{`{"gatewayAutoStart":"true"}`, `{"gatewayAutoStart":1}`, `{`} {
		if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := SetGatewayAutoStart(Test, true); err == nil {
			t.Fatal("corrupt config silently replaced")
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != raw {
			t.Fatalf("corrupt config modified: %v", err)
		}
	}
}
