package modelcatalog

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/workbuddyai"
)

// failingWorkBuddy 模拟 /v3/config 被上游拒绝的 WorkBuddy 订阅通道。
type failingWorkBuddy struct{}

func (failingWorkBuddy) Models(context.Context, string) ([]workbuddyai.Model, string, error) {
	return nil, "", workbuddyai.Failure(workbuddyai.CodeUnauthorized)
}

// 手动同步失败对外必须只有一句 unavailable，但底层稳定码要留给日志：
// 否则界面上永远只有一个「同步失败」，定位不了是授权、网络还是适配器问题。
func TestWorkBuddySyncFailureKeepsStableCodeForDiagnostics(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "workbuddy.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,auth_method,auth_state,auth_account_id)VALUES('wb','builtin_workbuddy_ai','WB','protected-ref',1,'now','now','workbuddy_login','authenticated','uid-1')`); err != nil {
		t.Fatal(err)
	}
	service := New(db, account.NewService(db, nil), provider.NewService(db), nil)
	service.SetWorkBuddySubscription(failingWorkBuddy{})

	models, err := service.Sync(ctx, "wb")
	if err == nil || !errors.Is(err, ErrUnavailable) || err.Error() != "models: unavailable" {
		t.Fatalf("sync error = %v, want models: unavailable", err)
	}
	if models != nil {
		t.Fatalf("models = %+v, want nil", models)
	}
	if code := workbuddyai.ErrorCode(service.LastSyncError()); code != workbuddyai.CodeUnauthorized {
		t.Fatalf("diagnostic code = %q, want %q", code, workbuddyai.CodeUnauthorized)
	}
}
