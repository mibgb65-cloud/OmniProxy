package version

import (
	"strconv"
	"strings"
)

// Compare compares two semantic versions (with or without a leading "v").
// It returns -1 when a < b, 0 when a == b and 1 when a > b. Pre-release
// suffixes are ignored: 1.2.3-beta compares equal to 1.2.3. Versions that
// cannot be parsed fall back to a plain string comparison.
func Compare(a, b string) int {
	pa, okA := parse(a)
	pb, okB := parse(b)
	if !okA || !okB {
		return strings.Compare(strings.TrimSpace(a), strings.TrimSpace(b))
	}
	for i := range pa {
		switch {
		case pa[i] < pb[i]:
			return -1
		case pa[i] > pb[i]:
			return 1
		}
	}
	return 0
}

// parse extracts the numeric major/minor/patch triple from a version string.
func parse(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexByte(v, '-'); i >= 0 { // drop pre-release suffix
		v = v[:i]
	}
	if i := strings.IndexByte(v, '+'); i >= 0 { // drop build metadata
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return out, false
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
