package version

import "strings"

// Compare 按 SemVer 比较版本，保留旧调用对缺省 minor／patch 和无效字符串的兼容。
func Compare(a, b string) int {
	left, okA := parse(a)
	right, okB := parse(b)
	if !okA || !okB {
		return strings.Compare(strings.TrimSpace(a), strings.TrimSpace(b))
	}
	for i := range left.core {
		if result := compareNumber(left.core[i], right.core[i]); result != 0 {
			return result
		}
	}
	if left.pre == "" && right.pre != "" {
		return 1
	}
	if left.pre != "" && right.pre == "" {
		return -1
	}
	if left.pre == right.pre {
		return 0
	}
	lp, rp := strings.Split(left.pre, "."), strings.Split(right.pre, ".")
	for i := 0; i < min(len(lp), len(rp)); i++ {
		if lp[i] == rp[i] {
			continue
		}
		ln, rn := numeric(lp[i]), numeric(rp[i])
		if ln && rn {
			return compareNumber(lp[i], rp[i])
		}
		if ln {
			return -1
		}
		if rn {
			return 1
		}
		return strings.Compare(lp[i], rp[i])
	}
	if len(lp) < len(rp) {
		return -1
	}
	return 1
}

type semanticVersion struct {
	core  [3]string
	pre   string
	parts int
}

// Valid 检查远端标签是否为完整 SemVer，禁止无效标签参与更新判断。
func Valid(value string) bool { v, ok := parse(value); return ok && v.parts == 3 }

// IsPrerelease 用于版本对应的默认更新渠道。
func IsPrerelease(value string) bool { v, ok := parse(value); return ok && v.pre != "" }

func parse(value string) (semanticVersion, bool) {
	var out semanticVersion
	if len(value) > 256 {
		return out, false
	}
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	core, build, hasBuild := strings.Cut(value, "+")
	if hasBuild && !identifiers(build, false) {
		return out, false
	}
	core, pre, hasPre := strings.Cut(core, "-")
	if hasPre && !identifiers(pre, true) {
		return out, false
	}
	parts := strings.Split(core, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return out, false
	}
	out.core = [3]string{"0", "0", "0"}
	for i, part := range parts {
		if !numeric(part) || len(part) > 1 && part[0] == '0' {
			return out, false
		}
		out.core[i] = part
	}
	out.pre, out.parts = pre, len(parts)
	return out, true
}

func identifiers(value string, prerelease bool) bool {
	for _, part := range strings.Split(value, ".") {
		if part == "" || prerelease && numeric(part) && len(part) > 1 && part[0] == '0' {
			return false
		}
		for _, c := range part {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}

func numeric(value string) bool {
	if value == "" {
		return false
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func compareNumber(a, b string) int {
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return strings.Compare(a, b)
}
