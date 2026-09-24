package version

import "testing"

func TestCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"v1.0.0", "1.0.0", 0},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"1.1.0", "1.0.9", 1},
		{"2.0.0", "1.9.9", 1},
		{"1.2", "1.2.0", 0},
		{"1.2.3-beta.1", "1.2.3", 0},
		{"0.1.0", "0.1.0", 0},
		{"0.10.0", "0.9.0", 1},
	}
	for _, tt := range tests {
		if got := Compare(tt.a, tt.b); got != tt.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
		// The comparison must be antisymmetric for parseable versions.
		if got := Compare(tt.b, tt.a); got != -tt.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", tt.b, tt.a, got, -tt.want)
		}
	}
}

func TestCompareFallsBackToStringCompare(t *testing.T) {
	if got := Compare("nightly", "stable"); got != -1 {
		t.Errorf("Compare(nightly, stable) = %d, want -1", got)
	}
	if got := Compare("abc", "abc"); got != 0 {
		t.Errorf("Compare(abc, abc) = %d, want 0", got)
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	for _, v := range []string{"", "x.y.z", "1..2", "1.2.3.4", "-1.0.0"} {
		if _, ok := parse(v); ok {
			t.Errorf("parse(%q) succeeded, want failure", v)
		}
	}
}
