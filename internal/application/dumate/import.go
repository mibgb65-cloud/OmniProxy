package dumate

import (
	"context"
	"errors"

	"omniproxy/internal/application/account"
	"omniproxy/internal/core/netproxy"
	providerdumate "omniproxy/internal/provider/dumate"
)

// ImportResult 是导入后的账号与来源；不包含 Cookie 或 token。
type ImportResult struct {
	Account account.Item `json:"account"`
	// Mode 为 created 或 updated，表示这次导入是否复用了同一远端身份。
	Mode string `json:"mode"`
	// Source 为 desktop、browser 或 manual。
	Source string `json:"source"`
}

// ImportDesktop 只读复用本机搭子桌面端登录态，并用远端身份核对归属。
func (s *Service) ImportDesktop(ctx context.Context, providerID string) (ImportResult, error) {
	if s.accounts.CheckDuMateProvider(ctx, providerID) != nil {
		return ImportResult{}, failure("not_dumate_provider")
	}
	dir, err := s.desktop()
	if err != nil {
		if errors.Is(err, providerdumate.ErrDesktopLoginMissing) {
			return ImportResult{}, failure("desktop_login_missing")
		}
		return ImportResult{}, failure("desktop_login_invalid")
	}
	credentials, err := s.readDesktop(dir)
	if err != nil {
		if errors.Is(err, providerdumate.ErrDesktopLoginMissing) {
			return ImportResult{}, failure("desktop_login_missing")
		}
		return ImportResult{}, failure("desktop_login_invalid")
	}
	return s.save(ctx, providerID, credentials)
}

// ImportCookie 接受用户从浏览器复制的 Cookie 请求头或手动粘贴的字符串。
func (s *Service) ImportCookie(ctx context.Context, providerID, raw string) (ImportResult, error) {
	if s.accounts.CheckDuMateProvider(ctx, providerID) != nil {
		return ImportResult{}, failure("not_dumate_provider")
	}
	cookies, err := providerdumate.ParseCookies(raw)
	if err != nil || providerdumate.Validate(cookies) != nil {
		return ImportResult{}, failure("cookie_invalid")
	}
	credentials := providerdumate.Credentials{Cookies: cookies, Source: providerdumate.SourceManual}
	return s.save(ctx, providerID, credentials)
}

// save 先用远端 user/info 核对身份，再按身份创建或更新账号。
func (s *Service) save(ctx context.Context, providerID string, credentials providerdumate.Credentials) (ImportResult, error) {
	verifyCtx, cancel := context.WithTimeout(netproxy.WithProvider(ctx, providerID, providerdumate.Kind), importTimeout)
	defer cancel()
	identity, err := s.console.UserInfo(verifyCtx, credentials)
	if err != nil {
		return ImportResult{}, failure(importCode(err))
	}
	credentials.AccountID, credentials.Name = identity.AccountID, identity.DisplayName
	existing, found, err := s.accounts.FindDuMateAccount(ctx, providerID, identity.AccountID)
	if err != nil {
		return ImportResult{}, failure("invalid_account")
	}
	id := ""
	if found {
		id = existing.ID
	}
	item, err := s.accounts.SaveDuMateAccount(ctx, providerID, id, identity.DisplayName, credentials)
	if err != nil {
		return ImportResult{}, failure(saveCode(err))
	}
	mode := "created"
	if found {
		mode = "updated"
	}
	// 重新导入后旧的积分快照属于上一份凭据，不能继续展示。
	s.forget(item.ID)
	return ImportResult{Account: item, Mode: mode, Source: credentials.Source}, nil
}

// importCode 把控制台失败映射为导入流程的固定错误码。
func importCode(err error) string {
	if providerdumate.SignInExpired(err) {
		return "sign_in_expired"
	}
	if code := providerdumate.ErrorCode(err); code != "" {
		return code
	}
	return "console_unavailable"
}

// saveCode 保留身份冲突等固定码，其余存储失败统一报 credential_unavailable。
func saveCode(err error) string {
	if code := providerdumate.ErrorCode(err); code != "" {
		return code
	}
	if errors.Is(err, account.ErrConflict) {
		return "identity_changed"
	}
	return "credential_unavailable"
}
