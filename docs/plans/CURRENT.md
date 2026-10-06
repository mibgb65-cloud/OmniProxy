# Current Development Status

当前状态看板。完整任务信息见 `docs/tasks/`（Source of Truth）。

## Current Version

0.1.0 — see `internal/version` and `build/config.yml`.

## In Progress

- TASK-20261006-002 标题栏双击最大化与还原

## Recently Completed

- TASK-20260928-010 账号卡片滚动、厂商详情页与全局内容宽度

- TASK-20260928-009 使用弹窗添加提供商账号

- TASK-20260928-008 账号页面与 API Key 账号管理

- TASK-20260928-007 精简提供商页面说明文案

- TASK-20260928-006 测试结果悬浮提示
- TASK-20260928-005 使用真实提供商品牌图标
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

- Settings, About, Providers, and Accounts have live Wails data; gateway and
  other feature pages remain placeholders until their backends land.
- Nine built-in providers are enabled by default and can be switched off;
  custom OpenAI-compatible endpoints can be added. API Key accounts are stored
  locally; provider request adapters and OpenAI subscription login are pending.
- Account API Keys are stored and managed but are not yet used by gateway calls.
  Token Plan authentication, upstream key validation, and account balance queries
  are not connected yet; provider account pages show this state explicitly.
- Provider model names are curated examples rather than account-specific live
  results. The Test action checks network reachability and latency without
  authenticating or invoking a model.
- UI log streaming (Wails events from the logger buffer) is not wired yet.
- Single-instance protection is not wired yet; a manual launch after autostart
  can start a second process.
- E2E (Playwright) is deferred until the first core user flow exists.

## Next Steps

1. Gateway proxy server with request parsing and the first provider adapter
   (OpenAI), followed by the model router.
2. Account scheduler (eligibility, health, cooldown, retry) and automatic
   failover.
3. Wire the UI log stream through Wails events from the logger buffer.
