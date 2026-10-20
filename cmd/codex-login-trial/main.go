package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"omniproxy/internal/application/agentconfig"
	"omniproxy/internal/infrastructure/codex"
	"omniproxy/internal/infrastructure/credential"
	"omniproxy/internal/platform/profile"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "旧版登录试验未完成：", err)
		os.Exit(1)
	}
}

func run() error {
	apply := flag.Bool("apply", false, "明确允许写入指定临时或真实 Codex 目录")
	restore := flag.Bool("restore", false, "恢复上一次试验前的文件，外部改动会阻止恢复")
	verify := flag.Bool("verify", false, "写入后通过隔离 app-server 核验登录；只用于临时目录")
	home := flag.String("home", "", "必填：绝对 CODEX_HOME 路径")
	kind := flag.String("profile", "", "必填：production、development 或 test")
	flag.Parse()
	if flag.NArg() != 0 || !filepath.IsAbs(*home) || len(*home) > 2048 || *apply && *restore || *verify && !*apply {
		return errors.New("参数无效；必须明确 profile 和绝对 home，apply 与 restore 互斥")
	}
	unlock, err := lockTrial(*home)
	if err != nil {
		return err
	}
	defer unlock()
	if _, err := profile.New(profile.Kind(*kind)); err != nil {
		return errors.New("无效运行环境")
	}
	if err := os.Setenv("OMNIPROXY_PROFILE", *kind); err != nil {
		return errors.New("无法选择运行环境")
	}
	prof, err := profile.Detect()
	if err != nil {
		return errors.New("读取运行环境失败")
	}
	store, err := credential.New(prof)
	if err != nil {
		return errors.New("初始化加密备份失败")
	}
	service := agentconfig.New(nil, store, agentconfig.Paths{Codex: filepath.Join(*home, "config.toml")}, prof.ProxyPort, nil)
	if *restore {
		if err := service.RestoreCodexFileLogin(); err != nil {
			return err
		}
		fmt.Println("已恢复试验前的 config.toml 和 auth.json，聊天及系统加密登录未改写。")
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	selected, subscriptions, profileRoot, err := selectAccount(ctx, prof, store)
	if err != nil {
		return err
	}
	defer selected.db.Close()
	defer subscriptions.Close()
	fmt.Println("已确认账号池存在已认证且额度可用的 ChatGPT 账号。")
	if !*apply {
		fmt.Println("只读检查完成；未写客户端文件。")
		return nil
	}
	if *verify {
		defaults, err := agentconfig.DefaultPaths()
		if err != nil || sameHome(*home, filepath.Dir(defaults.Codex)) {
			return errors.New("verify 仅允许临时目录，不能启动读取真实默认配置的验证进程")
		}
	}
	_, isolated, err := selected.accounts.SubscriptionProfile(ctx, selected.item.ID)
	if err != nil {
		return errors.New("读取选中账号的隔离授权失败")
	}
	err = subscriptions.UseTokenAt(ctx, selected.item.ID, selected.item.UpdatedAt, func(_ []byte, accountID string) error {
		return codex.UseClientAuth(filepath.Join(profileRoot, isolated), accountID, func(auth []byte) error {
			return service.TryCodexFileLogin(auth, accountID)
		})
	})
	if err != nil {
		return err
	}
	if *verify {
		client, err := codex.OpenFile(ctx, *home)
		if err == nil {
			defer client.Close()
			var identity *codex.Account
			identity, err = client.Account(ctx, false)
			if err == nil && (identity == nil || identity.Type != "chatgpt" || identity.Email != selected.item.AuthEmail) {
				err = errors.New("临时客户端没有读取预期的池账号")
			}
		}
		if err != nil {
			return errors.Join(errors.New("隔离登录验证失败"), service.RestoreCodexFileLogin())
		}
		fmt.Println("隔离 app-server 已读取选定的 ChatGPT 池账号；这不等同于 Desktop 额度门禁验证。")
	}
	fmt.Println("旧版方式已应用：保留 openai，设置 ChatGPT 文件登录；原配置和授权已加密备份。")
	fmt.Println("恢复需对相同 home/profile 使用 -restore；客户端外部修改会阻止覆盖。")
	fmt.Println("未重启 Desktop、未发送模型消息、未改写聊天或系统加密登录；Desktop 需完全重启后验收。")
	return nil
}

func sameHome(first, second string) bool {
	left, leftErr := filepath.EvalSymlinks(first)
	right, rightErr := filepath.EvalSymlinks(second)
	return leftErr != nil || rightErr != nil || strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}
