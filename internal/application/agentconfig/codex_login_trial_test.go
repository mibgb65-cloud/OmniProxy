package agentconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

const trialAuth = `{"OPENAI_API_KEY":"synthetic-api","tokens":{"id_token":"synthetic-id","access_token":"synthetic-access","refresh_token":"synthetic-refresh","account_id":"synthetic-account"}}`

func loginTrialFixture(t *testing.T, withAuth bool) (*Service, []byte, []byte) {
	t.Helper()
	root := t.TempDir()
	config := []byte("# keep comment\nmodel_provider='openai'\nmodel='gpt-test'\nopenai_base_url='http://127.0.0.1:3001/v1'\n[features]\nmulti_agent=true\n")
	original := []byte(`{"original":"synthetic-before"}`)
	if err := os.WriteFile(filepath.Join(root, "config.toml"), config, 0600); err != nil {
		t.Fatal(err)
	}
	if withAuth {
		if err := os.WriteFile(filepath.Join(root, "auth.json"), original, 0600); err != nil {
			t.Fatal(err)
		}
	}
	store := &memoryStore{values: map[string][]byte{}}
	return New(nil, store, Paths{Codex: filepath.Join(root, "config.toml")}, 3001, nil), config, original
}

func TestCodexLoginTrialAndRestorePreserveIdentityAndBytes(t *testing.T) {
	for _, withAuth := range []bool{false, true} {
		t.Run(map[bool]string{false: "new-auth", true: "existing-auth"}[withAuth], func(t *testing.T) {
			service, config, original := loginTrialFixture(t, withAuth)
			if err := service.TryCodexFileLogin([]byte(trialAuth), "synthetic-account"); err != nil {
				t.Fatal(err)
			}
			after, err := os.ReadFile(service.paths.Codex)
			var fields map[string]any
			if err != nil || toml.Unmarshal(after, &fields) != nil || fields["model_provider"] != "openai" ||
				fields["cli_auth_credentials_store"] != "file" || fields["forced_login_method"] != "chatgpt" ||
				!strings.Contains(string(after), "# keep comment") || fields["model"] != "gpt-test" {
				t.Fatal("login mode or unrelated configuration lost", err)
			}
			authPath, manifestPath := service.loginTrialPaths()
			receipt, err := os.ReadFile(manifestPath)
			if err != nil || strings.Contains(string(receipt), "synthetic") {
				t.Fatal("manifest exposed credentials")
			}
			if err := service.TryCodexFileLogin([]byte(trialAuth), "synthetic-account"); err == nil {
				t.Fatal("active trial overwritten")
			}
			if err := service.RestoreCodexFileLogin(); err != nil {
				t.Fatal(err)
			}
			restored, err := os.ReadFile(service.paths.Codex)
			if err != nil || string(restored) != string(config) {
				t.Fatal("original config not restored byte-for-byte")
			}
			auth, err := os.ReadFile(authPath)
			if withAuth && (err != nil || string(auth) != string(original)) || !withAuth && !os.IsNotExist(err) {
				t.Fatal("original auth existence or bytes not restored")
			}
			if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
				t.Fatal("trial receipt left after successful restore")
			}
			if len(service.store.(*memoryStore).values) != 0 {
				t.Fatal("protected backup left after restore")
			}
		})
	}
}

func TestCodexLoginRestorePreservesExternalChangesAndBackup(t *testing.T) {
	for _, target := range []string{"config.toml", "auth.json"} {
		t.Run(target, func(t *testing.T) {
			service, _, _ := loginTrialFixture(t, true)
			if err := service.TryCodexFileLogin([]byte(trialAuth), "synthetic-account"); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(filepath.Dir(service.paths.Codex), target)
			if err := os.WriteFile(path, []byte("external-change"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := service.RestoreCodexFileLogin(); err == nil || err.Error() != "agent: config_changed" {
				t.Fatal("external changes were overwritten", err)
			}
			body, err := os.ReadFile(path)
			if err != nil || string(body) != "external-change" || len(service.store.(*memoryStore).values) != 1 {
				t.Fatal("external file or recovery backup lost")
			}
		})
	}
}

func TestCodexLoginTrialRejectsWrongProviderWorkspaceAndEndpoint(t *testing.T) {
	for _, body := range []string{
		`model_provider="custom"` + "\n" + `openai_base_url="http://127.0.0.1:3001/v1"`,
		`model_provider="openai"` + "\n" + `openai_base_url="https://example.invalid/v1"`,
		`model_provider="openai"` + "\n" + `openai_base_url="http://127.0.0.1:3000/v1"`,
		`model_provider="openai"` + "\n" + `openai_base_url="http://127.0.0.1:3001/v1"` + "\n" + `forced_chatgpt_workspace_id="different"`,
		`model_provider="openai"` + "\n" + `openai_base_url="http://127.0.0.1:3001/v1"` + "\n" + `profile="managed"`,
	} {
		service, _, _ := loginTrialFixture(t, false)
		if err := os.WriteFile(service.paths.Codex, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if err := service.TryCodexFileLogin([]byte(trialAuth), "synthetic-account"); err == nil {
			t.Fatal("conflicting client configuration accepted")
		}
		if len(service.store.(*memoryStore).values) != 0 {
			t.Fatal("invalid trial created a credential backup")
		}
	}
}
