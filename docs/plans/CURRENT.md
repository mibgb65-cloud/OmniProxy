# Current Development Status

当前状态看板。完整任务信息见 `docs/tasks/`（Source of Truth）。

## Current Version

0.1.0 — see `internal/version` and `build/config.yml`.

## In Progress

- TASK-20260928-005 使用真实提供商品牌图标

## Recently Completed

- TASK-20260928-004 内置提供商默认启用与自定义提供商
- TASK-20260928-003 九家提供商预设与配置管理
- TASK-20260928-002 Windows 凭证存储基础设施
- TASK-20260928-001 修复主界面滚动布局与滚动条交互
- TASK-20260926-008 开机启动与关闭到托盘设置
- TASK-20260926-007 按参考稿优化存储设置页面
- TASK-20260926-006 设置页数据存储占用图与完整目录统计
- TASK-20260926-005 为侧栏文字增加逐字收展动画
- TASK-20260926-004 修复侧栏收展时图标位移与垂直跳动
- TASK-20260926-003 恢复收缩侧栏的展开按钮并居中图标
- TASK-20260926-002 修复侧边栏展开时短暂出现滚动条
- TASK-20260926-001 自绘标题栏与亮暗模式
- TASK-20260924-007 About page (version, runtime info, update check)
- TASK-20260924-006 Settings page: data storage management
- TASK-20260924-005 Task Record workflow
- TASK-20260924-004 Migrate i18n to i18next + react-i18n
- TASK-20260924-003 Frontend i18n (Chinese/English)
- TASK-20260924-002 State file/function size limits in coding rules
- TASK-20260924-001 Project initialization foundation

## Known Issues

- Settings, About, and Providers have live Wails data; account, gateway and
  other feature pages remain placeholders until their backends land.
- Nine built-in providers are enabled by default and can be switched off;
  custom OpenAI-compatible endpoints can be added. Provider request adapters,
  API Key accounts, and OpenAI subscription login are not connected yet.
- Provider model names are curated examples rather than account-specific live
  results. The Test action checks network reachability and latency without
  authenticating or invoking a model.
- UI log streaming (Wails events from the logger buffer) is not wired yet.
- Single-instance protection is not wired yet; a manual launch after autostart
  can start a second process.
- E2E (Playwright) is deferred until the first core user flow exists.

## Next Steps

1. Account management with API Key credentials, using the credential store and
   provider configurations already in place; wire the Accounts page to bindings.
2. Gateway proxy server with request parsing and the first provider adapter
   (OpenAI), followed by the model router.
3. Account scheduler (eligibility, health, cooldown, retry) and automatic
   failover.
4. Wire the UI log stream through Wails events from the logger buffer.
