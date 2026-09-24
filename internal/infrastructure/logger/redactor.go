package logger

import (
	"regexp"
	"strings"
)

// masked is the replacement used for every redacted value.
const masked = "***"

// sensitiveKeys are matched case-insensitively in "key: value" and
// "key=value" pairs inside log messages. Longer keys must precede their
// substrings (e.g. "access-token" before "token") so the most specific
// alternative wins.
var sensitiveKeys = []string{
	"authorization",
	"api-key", "api_key", "apikey", "x-api-key",
	"access-token", "access_token",
	"refresh-token", "refresh_token",
	"cookie", "set-cookie", "sessionid", "session-id", "session-token",
	"password", "passwd",
	"secret", "client-secret", "client_secret",
	"credential", "credentials",
	"private-key", "private_key",
	"token", "auth", "jwt",
}

// sensitiveKeySubstrings are matched inside structured log attribute keys
// (after lowercasing and '-' -> '_'). They are deliberately narrower than
// sensitiveKeys so that benign keys such as "token_count" survive.
var sensitiveKeySubstrings = []string{
	"secret", "password", "passwd", "credential", "cookie",
	"authorization", "private_key", "api_key", "apikey",
	"access_token", "refresh_token", "session_token", "auth_token",
}

// Redactor masks credentials and other sensitive data before it reaches any
// log sink. Every log line produced by OmniProxy passes through a Redactor
// (技术架构选择.md §18); individual providers must never decide on their own
// whether a credential may be printed.
type Redactor struct {
	keyValue *regexp.Regexp
	bearer   *regexp.Regexp
	apiKey   *regexp.Regexp
	jwt      *regexp.Regexp
}

// NewRedactor creates a Redactor with the default sensitive-key set.
func NewRedactor() *Redactor {
	keyPattern := `(?i)\b(` + strings.Join(sensitiveKeys, "|") + `)\b\s*[:=]\s*["']?(?:(?:bearer|basic|token)\s+)?["']?([A-Za-z0-9\-._~+/=]+)["']?`
	return &Redactor{
		keyValue: regexp.MustCompile(keyPattern),
		bearer:   regexp.MustCompile(`(?i)\b(bearer|basic)\s+([A-Za-z0-9\-._~+/]{6,}={0,2})`),
		apiKey:   regexp.MustCompile(`\bsk-[A-Za-z0-9\-_]{12,}`),
		jwt:      regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{6,}\.[A-Za-z0-9_-]{6,}\.[A-Za-z0-9_-]{6,}`),
	}
}

// Redact returns s with every detected secret replaced by "***".
func (r *Redactor) Redact(s string) string {
	if s == "" {
		return s
	}
	s = r.keyValue.ReplaceAllString(s, "$1="+masked)
	s = r.bearer.ReplaceAllString(s, "$1 "+masked)
	s = r.apiKey.ReplaceAllString(s, masked)
	s = r.jwt.ReplaceAllString(s, masked)
	return s
}

// SensitiveKey reports whether a structured log attribute key looks like it
// carries a credential. Callers use it to decide before logging; the logger
// itself also applies it to every attribute.
func (r *Redactor) SensitiveKey(key string) bool {
	k := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	for _, s := range sensitiveKeySubstrings {
		if strings.Contains(k, s) {
			return true
		}
	}
	return false
}
