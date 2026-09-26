package profile

import "testing"

func TestParseCloseBehavior(t *testing.T) {
	for _, tc := range []struct {
		stored string
		want   string
		valid  bool
	}{
		{stored: "", want: CloseExit, valid: true},
		{stored: CloseExit, want: CloseExit, valid: true},
		{stored: CloseToTray, want: CloseToTray, valid: true},
		{stored: "hide", valid: false},
	} {
		got, err := ParseCloseBehavior(tc.stored)
		if (err == nil) != tc.valid || got != tc.want {
			t.Errorf("ParseCloseBehavior(%q) = %q, %v; want %q, valid=%t", tc.stored, got, err, tc.want, tc.valid)
		}
	}
}

func TestSetCloseBehaviorPreservesOtherConfig(t *testing.T) {
	useTempConfigDir(t)
	before := Config{DataDir: `D:\data`, UpdateRepo: "example/omniproxy"}
	if err := SaveConfig(Test, before); err != nil {
		t.Fatal(err)
	}
	if err := SetCloseBehavior(Test, CloseToTray); err != nil {
		t.Fatal(err)
	}
	got, err := LoadConfig(Test)
	if err != nil {
		t.Fatal(err)
	}
	if got.DataDir != before.DataDir || got.UpdateRepo != before.UpdateRepo || got.CloseBehavior != CloseToTray {
		t.Fatalf("stored config = %+v", got)
	}
	if err := SetCloseBehavior(Test, "unexpected"); err == nil {
		t.Fatal("invalid close behavior should fail")
	}
	after, err := LoadConfig(Test)
	if err != nil || after != got {
		t.Fatalf("invalid change modified config: %+v, %v", after, err)
	}
}
