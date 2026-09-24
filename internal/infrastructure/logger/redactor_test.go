package logger

import "testing"

func TestRedactKeyValuePairs(t *testing.T) {
	r := NewRedactor()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "authorization bearer header",
			input: "Authorization: Bearer sk-ant-api03-abcdef123456789",
			want:  "Authorization=***",
		},
		{
			name:  "api key assignment",
			input: "api_key=sk-proj-1234567890abcdef",
			want:  "api_key=***",
		},
		{
			name:  "quoted password",
			input: `password: "SuperSecret123"`,
			want:  "password=***",
		},
		{
			name:  "cookie header stops at semicolon",
			input: "cookie: sessionid=abc123def456; other=keep",
			want:  "cookie=***; other=keep",
		},
		{
			name:  "access token equals",
			input: "access_token=gho_16C7e42F292c6912E7710c838347Ae178B4a",
			want:  "access_token=***",
		},
		{
			name:  "standalone bearer token",
			input: "sending bearer abc123def456ghi789 to upstream",
			want:  "sending bearer *** to upstream",
		},
		{
			name:  "standalone sk style api key",
			input: "leaked key sk-1234567890abcdefXYZ in config",
			want:  "leaked key *** in config",
		},
		{
			name:  "jwt",
			input: "jwt=eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N",
			want:  "jwt=***",
		},
		{
			name:  "normal text untouched",
			input: "request completed provider=openai model=gpt-4 status=200 duration=1.2s",
			want:  "request completed provider=openai model=gpt-4 status=200 duration=1.2s",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.Redact(tt.input); got != tt.want {
				t.Errorf("Redact(%q)\n = %q\nwant %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSensitiveKey(t *testing.T) {
	r := NewRedactor()
	sensitive := []string{"authorization", "api_key", "x-api-key", "access_token", "refresh_token", "password", "client_secret", "cookie", "Authorization"}
	benign := []string{"provider", "model", "status", "duration", "request_id", "token_count", "account_id", "error_code"}
	for _, k := range sensitive {
		if !r.SensitiveKey(k) {
			t.Errorf("SensitiveKey(%q) = false, want true", k)
		}
	}
	for _, k := range benign {
		if r.SensitiveKey(k) {
			t.Errorf("SensitiveKey(%q) = true, want false", k)
		}
	}
}
