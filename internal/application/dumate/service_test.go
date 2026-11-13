package dumate

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"omniproxy/internal/application/account"
	providerdumate "omniproxy/internal/provider/dumate"
)

type fakeAccounts struct {
	items   map[string]account.Item
	stores  map[string]providerdumate.Credentials
	saved   []savedCall
	findErr error
	saveErr error
}

type savedCall struct {
	id, providerID, name string
	credentials          providerdumate.Credentials
}

func newFakeAccounts() *fakeAccounts {
	return &fakeAccounts{items: map[string]account.Item{}, stores: map[string]providerdumate.Credentials{}}
}

func (f *fakeAccounts) Metadata(_ context.Context, id string) (account.Item, error) {
	item, ok := f.items[id]
	if !ok {
		return account.Item{}, account.ErrNotFound
	}
	return item, nil
}

func (f *fakeAccounts) UseDuMateCredentials(_ context.Context, id string, consume func(account.Item, providerdumate.Credentials) error) error {
	item, ok := f.items[id]
	if !ok {
		return providerdumate.Failure(providerdumate.CodeCredentialUnavailable)
	}
	return consume(item, f.stores[id])
}

func (f *fakeAccounts) SaveDuMateAccount(_ context.Context, providerID, id, name string, credentials providerdumate.Credentials) (account.Item, error) {
	f.saved = append(f.saved, savedCall{id: id, providerID: providerID, name: name, credentials: credentials})
	if f.saveErr != nil {
		return account.Item{}, f.saveErr
	}
	if id == "" {
		id = "acct_dumate"
	}
	item := account.Item{ID: id, ProviderID: providerID, Name: name, AuthMethod: account.AuthDuMateCookie, AuthState: "authenticated", AuthAccountID: credentials.AccountID, UpdatedAt: "v1"}
	f.items[id] = item
	f.stores[id] = credentials
	return item, nil
}

func (f *fakeAccounts) FindDuMateAccount(_ context.Context, providerID, subject string) (account.Item, bool, error) {
	if f.findErr != nil {
		return account.Item{}, false, f.findErr
	}
	for _, item := range f.items {
		if item.ProviderID == providerID && item.AuthAccountID == subject {
			return item, true, nil
		}
	}
	return account.Item{}, false, nil
}

func (f *fakeAccounts) CheckDuMateProvider(_ context.Context, providerID string) error {
	if providerID != "builtin_dumate" {
		return account.ErrInvalidInput
	}
	return nil
}

type fakeConsole struct {
	userInfo  func(providerdumate.Credentials) (providerdumate.Identity, error)
	overview  func() (providerdumate.Overview, error)
	bonus     func() (providerdumate.LoginBonus, error)
	claim     func() error
	userCalls int
}

func (f *fakeConsole) UserInfo(_ context.Context, credentials providerdumate.Credentials) (providerdumate.Identity, error) {
	f.userCalls++
	if f.userInfo == nil {
		return providerdumate.Identity{AccountID: credentials.AccountID, DisplayName: "Tester"}, nil
	}
	return f.userInfo(credentials)
}

func (f *fakeConsole) Overview(context.Context, providerdumate.Credentials) (providerdumate.Overview, error) {
	if f.overview == nil {
		return providerdumate.Overview{TotalPoints: 100, UsedPoints: 40}, nil
	}
	return f.overview()
}

func (f *fakeConsole) LoginBonus(context.Context, providerdumate.Credentials) (providerdumate.LoginBonus, error) {
	if f.bonus == nil {
		return providerdumate.LoginBonus{}, nil
	}
	return f.bonus()
}

func (f *fakeConsole) ClaimLoginBonus(context.Context, providerdumate.Credentials) error {
	if f.claim == nil {
		return nil
	}
	return f.claim()
}

func validCredentials() providerdumate.Credentials {
	return providerdumate.Credentials{Cookies: []providerdumate.Cookie{{Name: providerdumate.SessionCookie, Value: "session"}}, Source: providerdumate.SourceManual}
}

func TestImportDesktopUsesRemoteIdentity(t *testing.T) {
	accounts, console := newFakeAccounts(), &fakeConsole{}
	service := New(accounts, console)
	service.desktop = func() (string, error) { return "", providerdumate.ErrDesktopLoginMissing }
	if _, err := service.ImportDesktop(context.Background(), "builtin_dumate"); err == nil || !strings.Contains(err.Error(), "desktop_login_missing") {
		t.Fatalf("missing desktop login = %v", err)
	}
	service.desktop = func() (string, error) { return "C:\\Users\\test\\AppData\\Roaming\\qianfan-desktop-app", nil }
	service.readDesktop = func(string) (providerdumate.Credentials, error) {
		return providerdumate.Credentials{Cookies: validCredentials().Cookies, Source: providerdumate.SourceDesktop, AccountID: "1001", Name: "Tester"}, nil
	}
	result, err := service.ImportDesktop(context.Background(), "builtin_dumate")
	if err != nil {
		t.Fatal(err)
	}
	if result.Mode != "created" || result.Source != providerdumate.SourceDesktop {
		t.Fatalf("result = %+v", result)
	}
	if len(accounts.saved) != 1 || accounts.saved[0].credentials.AccountID != "1001" || accounts.saved[0].name != "Tester" {
		t.Fatalf("saved = %+v", accounts.saved)
	}
	if _, err := service.ImportDesktop(context.Background(), "builtin_openai"); err == nil {
		t.Fatal("non-dumate provider accepted")
	}
}

func TestImportCookieValidatesAndMapsFailures(t *testing.T) {
	accounts := newFakeAccounts()
	console := &fakeConsole{userInfo: func(providerdumate.Credentials) (providerdumate.Identity, error) {
		return providerdumate.Identity{}, providerdumate.SignInFailure()
	}}
	service := New(accounts, console)
	if _, err := service.ImportCookie(context.Background(), "builtin_dumate", "no-cookies"); err == nil || !strings.Contains(err.Error(), "cookie_invalid") {
		t.Fatalf("invalid cookie = %v", err)
	}
	if _, err := service.ImportCookie(context.Background(), "builtin_dumate", "bce-sessionid=abc"); err == nil || !strings.Contains(err.Error(), "sign_in_expired") {
		t.Fatalf("expired cookie = %v", err)
	}
	console.userInfo = nil
	result, err := service.ImportCookie(context.Background(), "builtin_dumate", "bce-sessionid=abc; bce-user-info=csrf")
	if err != nil || result.Source != providerdumate.SourceManual {
		t.Fatalf("result = (%+v, %v)", result, err)
	}
	encoded, _ := json.Marshal(result)
	if strings.Contains(string(encoded), "bce-sessionid") {
		t.Fatal("cookie escaped into import result")
	}
}

func TestPointsFallsBackToRetainedSnapshot(t *testing.T) {
	accounts, console := newFakeAccounts(), &fakeConsole{}
	accounts.items["acct_dumate"] = account.Item{ID: "acct_dumate", ProviderID: "builtin_dumate", AuthMethod: account.AuthDuMateCookie, AuthState: "authenticated", AuthAccountID: "1001"}
	accounts.stores["acct_dumate"] = validCredentials()
	service := New(accounts, console)
	live, err := service.Points(context.Background(), "acct_dumate")
	if err != nil || live.State != "live" || live.Points == nil || live.Points.RemainingPoints != 60 {
		t.Fatalf("live = (%+v, %v)", live, err)
	}
	console.overview = func() (providerdumate.Overview, error) {
		return providerdumate.Overview{}, providerdumate.Failure(providerdumate.CodeConsoleUnavailable)
	}
	stale, err := service.Points(context.Background(), "acct_dumate")
	if err != nil || stale.State != "stale" || stale.ErrorCode != providerdumate.CodeConsoleUnavailable || stale.Points == nil || stale.Points.RemainingPoints != 60 {
		t.Fatalf("stale = (%+v, %v)", stale, err)
	}
	if _, err := service.Points(context.Background(), "missing"); err == nil {
		t.Fatal("missing account accepted")
	}
}

func TestCheckinOutcomes(t *testing.T) {
	accounts := newFakeAccounts()
	accounts.items["acct_dumate"] = account.Item{ID: "acct_dumate", ProviderID: "builtin_dumate", AuthMethod: account.AuthDuMateCookie, AuthState: "authenticated", AuthAccountID: "1001"}
	accounts.stores["acct_dumate"] = validCredentials()
	console := &fakeConsole{}
	service := New(accounts, console)
	console.bonus = func() (providerdumate.LoginBonus, error) {
		return providerdumate.LoginBonus{HasIssued: true, TotalTimes: 5}, nil
	}
	if result, err := service.Checkin(context.Background(), "acct_dumate"); err != nil || result.Outcome != "done" || result.TotalTimes != 5 {
		t.Fatalf("done = (%+v, %v)", result, err)
	}
	bonusCalls := 0
	console.bonus = func() (providerdumate.LoginBonus, error) {
		bonusCalls++
		if bonusCalls == 1 {
			return providerdumate.LoginBonus{HasIssued: false, TotalPoints: 10}, nil
		}
		return providerdumate.LoginBonus{HasIssued: true, TotalPoints: 510, TotalTimes: 6}, nil
	}
	if result, err := service.Checkin(context.Background(), "acct_dumate"); err != nil || result.Outcome != "claimed" || result.Credit != 500 || result.TotalTimes != 6 {
		t.Fatalf("claimed = (%+v, %v)", result, err)
	}
	console.bonus = func() (providerdumate.LoginBonus, error) {
		return providerdumate.LoginBonus{HasIssued: false}, nil
	}
	if result, err := service.Checkin(context.Background(), "acct_dumate"); err != nil || result.Outcome != "pending" {
		t.Fatalf("pending = (%+v, %v)", result, err)
	}
	console.claim = func() error { return providerdumate.SignInFailure() }
	if result, err := service.Checkin(context.Background(), "acct_dumate"); err != nil || result.Outcome != "failed" || result.ErrorCode != "sign_in_expired" {
		t.Fatalf("failed = (%+v, %v)", result, err)
	}
	console.claim = nil
	console.bonus = func() (providerdumate.LoginBonus, error) {
		return providerdumate.LoginBonus{}, providerdumate.SignInFailure()
	}
	if result, err := service.Checkin(context.Background(), "acct_dumate"); err != nil || result.ErrorCode != "sign_in_expired" {
		t.Fatalf("expired = (%+v, %v)", result, err)
	}
}

func TestCheckinRejectsForeignAccount(t *testing.T) {
	accounts := newFakeAccounts()
	accounts.items["acct_other"] = account.Item{ID: "acct_other", ProviderID: "builtin_dumate", AuthMethod: account.AuthAPIKey}
	service := New(accounts, &fakeConsole{})
	if _, err := service.Checkin(context.Background(), "acct_other"); err == nil {
		t.Fatal("non-cookie account accepted")
	}
	if _, err := service.Checkin(context.Background(), "missing"); err == nil {
		t.Fatal("missing account accepted")
	}
}
