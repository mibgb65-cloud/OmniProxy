package subscription

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/core/netproxy"
	"omniproxy/internal/infrastructure/codex"
)

func (c *testClient) Models(context.Context) ([]codex.Model, error) {
	// 模拟真实 model/list：只投影公开名称和档位，没有 contextWindow。
	return []codex.Model{{ID: "gpt-test", Name: "GPT Test", Metadata: catalog.Metadata{IsDefault: true}}}, nil
}

func modelFixture(t *testing.T) (*fixture, State, string) {
	t.Helper()
	f := newFixture(t)
	view, client, home := f.begin(t, "")
	client.authorize("models@example.test")
	saved := awaitState(t, f.manager, view.ID, "completed")
	f.manager.codexVersion = func(context.Context) (string, error) { return "codex-cli 0.166.0-alpha.5.1", nil }
	f.manager.readToken = func(_ context.Context, path, id, email string, consume func([]byte, string) error) error {
		if path != home || id != saved.Account.AuthAccountID || email != saved.Account.AuthEmail {
			t.Error("model discovery crossed the isolated account boundary")
		}
		return consume([]byte("scoped-qa-only"), id)
	}
	return f, saved, home
}

func TestModelsMergesOnlyExactAuthenticatedNativeCapacity(t *testing.T) {
	f, saved, _ := modelFixture(t)
	calls := 0
	f.manager.modelWindows = func(ctx context.Context, key []byte, id, version string) (map[string]int64, error) {
		calls++
		if string(key) != "scoped-qa-only" || id != saved.Account.AuthAccountID || version != "0.166.0-alpha.5.1" {
			t.Error("wrong native catalog identity or version")
		}
		if scope, ok := netproxy.ScopeFrom(ctx); !ok || scope.ProviderID != saved.Account.ProviderID || scope.ProviderKind != "openai" {
			t.Error("native discovery lost provider proxy scope")
		}
		return map[string]int64{"gpt-test": 500000, "not-visible": 1000000}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	models, err := f.manager.Models(ctx, saved.Account.ID)
	if err != nil || calls != 1 || len(models) != 1 || models[0].Metadata.ContextWindow == nil ||
		*models[0].Metadata.ContextWindow != 500000 || !models[0].Metadata.IsDefault || models[0].Name != "GPT Test" {
		t.Fatal("native capacity was not merged without changing visible models", err)
	}
}

func TestModelsRejectsMissingCapacityAndSanitizesErrors(t *testing.T) {
	for _, test := range []struct {
		name    string
		windows map[string]int64
		err     error
	}{
		{"absent", nil, nil},
		{"different slug", map[string]int64{"gpt-other": 1000000}, nil},
		{"zero", map[string]int64{"gpt-test": 0}, nil},
		{"oversize", map[string]int64{"gpt-test": 10000001}, nil},
		{"upstream", nil, errors.New("private-capacity-data")},
	} {
		t.Run(test.name, func(t *testing.T) {
			f, saved, _ := modelFixture(t)
			f.manager.modelWindows = func(context.Context, []byte, string, string) (map[string]int64, error) {
				return test.windows, test.err
			}
			models, err := f.manager.Models(context.Background(), saved.Account.ID)
			if err == nil || models != nil || strings.Contains(err.Error(), "private") {
				t.Fatal("unsafe capacity accepted or private error leaked", err)
			}
		})
	}
}

func TestModelsRejectsAccountChangeBetweenDiscoveryAndCapacity(t *testing.T) {
	f, saved, _ := modelFixture(t)
	f.manager.codexVersion = func(context.Context) (string, error) {
		if _, err := f.db.Exec(`UPDATE accounts SET updated_at='changed',auth_account_id='another-workspace' WHERE id=?`, saved.Account.ID); err != nil {
			t.Fatal(err)
		}
		return "codex-cli 0.163.0", nil
	}
	f.manager.modelWindows = func(context.Context, []byte, string, string) (map[string]int64, error) {
		t.Error("replaced account reached capacity source")
		return nil, nil
	}
	if models, err := f.manager.Models(context.Background(), saved.Account.ID); err == nil || models != nil {
		t.Fatal("changed account received old model metadata")
	}
}
