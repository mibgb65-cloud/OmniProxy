package codex

import (
	"context"
	"errors"
	"strings"
	"time"
)

// ErrRefreshRequired 要求调用方仅在必要时交给官方组件续期。
// 隔离登录不能自行旋转 refresh token 却不回写官方存储，否则下一次续期会失效。
var ErrRefreshRequired = errors.New("codex: refresh_required")

// UseFreshToken 核验同一份受保护授权的成员、工作区与有效期，结束即清零副本。
// 不新增影子令牌缓存；官方存储变化、注销或删除后自然不能继续借出旧授权。
func UseFreshToken(ctx context.Context, home, expectedID, expectedEmail string, consume func([]byte, string) error) error {
	if consume == nil || expectedID == "" || expectedEmail == "" {
		return ErrProtocol
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return useStoredAuthContext(ctx, home, func(data []byte) error {
		access, accountID, err := freshAccess(ctx, expectedID, expectedEmail, data, time.Now())
		if err != nil {
			return err
		}
		defer clear(access)
		if err := ctx.Err(); err != nil {
			return err
		}
		return consume(access, accountID)
	})
}

func freshAccess(ctx context.Context, expectedID, expectedEmail string, data []byte, now time.Time) ([]byte, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	auth, err := ParseAuthFile(data)
	if err != nil || auth.AccountID != expectedID || expectedEmail == "" ||
		!strings.EqualFold(strings.TrimSpace(auth.Email), strings.TrimSpace(expectedEmail)) {
		return nil, "", ErrProtocol
	}
	if AuthNeedsRefresh(string(data), now) {
		return nil, "", ErrRefreshRequired
	}
	return []byte(auth.AccessToken), auth.AccountID, nil
}
