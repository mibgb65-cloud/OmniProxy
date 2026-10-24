package account

import (
	"context"
	"errors"
	"testing"
)

func TestCodexTokenRefreshCannotOverwriteReimportOrOtherMember(t *testing.T) {
	for _, mode := range []string{"reimport", "other_member", "stale_refresh", "disabled"} {
		t.Run(mode, func(t *testing.T) {
			service, _ := codexService(t)
			ctx := context.Background()
			original := codexAuthFixture(t, "workspace", "member@example.invalid", "team", "access-1", "refresh-1")
			item, err := service.ImportCodexAuth(ctx, "builtin_openai", "test", original)
			if err != nil {
				t.Fatal(err)
			}
			var stored []byte
			if err := service.UseCodexAuth(ctx, item.ID, func(raw []byte) error {
				stored = append([]byte(nil), raw...)
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			defer clear(stored)
			refreshed := codexAuthFixture(t, "workspace", "member@example.invalid", "team", "access-2", "refresh-2")
			switch mode {
			case "reimport":
				if _, err := service.ImportCodexAuth(ctx, "builtin_openai", "test", refreshed); err != nil {
					t.Fatal(err)
				}
			case "other_member":
				refreshed = codexAuthFixture(t, "workspace", "other@example.invalid", "team", "access-other", "refresh-other")
			case "stale_refresh":
				if err := service.ReplaceCodexAuthTokensAt(ctx, item.ID, item.UpdatedAt, stored, refreshed); err != nil {
					t.Fatal(err)
				}
			case "disabled":
				if _, err := service.db.ExecContext(ctx, `UPDATE accounts SET enabled=0 WHERE id=?`, item.ID); err != nil {
					t.Fatal(err)
				}
			}
			if err := service.ReplaceCodexAuthTokensAt(ctx, item.ID, item.UpdatedAt, stored, refreshed); !errors.Is(err, ErrConflict) {
				t.Fatal("过期或异成员续期结果覆盖了当前授权", err)
			}
		})
	}
}
