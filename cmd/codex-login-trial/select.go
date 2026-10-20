package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/subscription"
	"omniproxy/internal/infrastructure/credential"
	_ "omniproxy/internal/infrastructure/database"
	"omniproxy/internal/platform/profile"
)

type candidate struct {
	db       *sql.DB
	accounts *account.Service
	item     account.Item
}

func selectAccount(ctx context.Context, prof *profile.Profile, store *credential.Store) (candidate, *subscription.Service, string, error) {
	path := filepath.ToSlash(prof.DatabasePath())
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	dsn := (&url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro&_pragma=query_only(1)&_pragma=busy_timeout(5000)"}).String()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return candidate{}, nil, "", errors.New("读取账号池失败")
	}
	keep := false
	defer func() {
		if !keep {
			db.Close()
		}
	}()
	accounts := account.NewService(db, store)
	items, err := accounts.List(ctx)
	if err != nil {
		return candidate{}, nil, "", errors.New("读取账号列表失败")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return candidate{}, nil, "", errors.New("读取隔离授权根目录失败")
	}
	root := filepath.Join(cache, "OmniProxy-CodexAuth", string(prof.Kind))
	service := subscription.New(accounts, root, nil)
	best, remaining := account.Item{}, 0
	failed := 0
	for _, item := range items {
		if !item.Enabled || item.AuthMethod != account.AuthSubscription || item.AuthState != "authenticated" || item.AuthPlan == "" {
			continue
		}
		quota, err := service.ReadQuota(ctx, item.ID)
		if err != nil {
			failed++
			continue
		}
		available, known := availableQuota(quota)
		if !known {
			failed++
			continue
		}
		if available > remaining {
			best, remaining = item, available
		}
	}
	if failed > 0 {
		fmt.Printf("有 %d 个候选账号未能确认完整额度，已跳过，不推算其可用性。\n", failed)
	}
	if best.ID == "" {
		service.Close()
		return candidate{}, nil, "", errors.New("未确认任何额度可用的合格 ChatGPT 账号；未改动客户端")
	}
	keep = true
	return candidate{db: db, accounts: accounts, item: best}, service, root, nil
}

func availableQuota(quota subscription.Quota) (int, bool) {
	if quota.Weekly == nil || quota.FiveHourEnabled && quota.FiveHour == nil {
		return 0, false
	}
	remaining := quota.Weekly.RemainingPercent
	if remaining < 0 || remaining > 100 {
		return 0, false
	}
	if quota.FiveHourEnabled {
		if quota.FiveHour.RemainingPercent < 0 || quota.FiveHour.RemainingPercent > 100 {
			return 0, false
		}
		remaining = min(remaining, quota.FiveHour.RemainingPercent)
	}
	return remaining, true
}
