package codex

import (
	"strings"
	"testing"
)

func TestLauncherOverridesDoNotChangeLoginIdentityOrIssuer(t *testing.T) {
	for key, value := range map[string]string{
		"CODEX_INTERNAL_ORIGINATOR_OVERRIDE": "Codex Desktop",
		"CODEX_APP_SERVER_LOGIN_CLIENT_ID":   "launcher-client",
		"CODEX_APP_SERVER_LOGIN_ISSUER":      "https://launcher.invalid",
	} {
		t.Setenv(key, value)
	}
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:17890")
	t.Setenv("CODEX_CA_CERTIFICATE", "test-ca.pem")
	t.Setenv("SSL_CERT_FILE", "test-ssl.pem")
	environment := map[string]string{}
	for _, entry := range isolatedEnvironment("isolated-home") {
		key, value, _ := strings.Cut(entry, "=")
		environment[strings.ToUpper(key)] = value
	}
	for _, key := range []string{"CODEX_INTERNAL_ORIGINATOR_OVERRIDE", "CODEX_APP_SERVER_LOGIN_CLIENT_ID", "CODEX_APP_SERVER_LOGIN_ISSUER"} {
		if _, inherited := environment[key]; inherited {
			t.Fatalf("launcher override inherited: %s", key)
		}
	}
	if environment["CODEX_HOME"] != "isolated-home" || environment["HTTPS_PROXY"] != "http://127.0.0.1:17890" || environment["CODEX_CA_CERTIFICATE"] != "test-ca.pem" || environment["SSL_CERT_FILE"] != "test-ssl.pem" {
		t.Fatal("account isolation damaged network configuration")
	}
}
