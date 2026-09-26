# Current Development Status

当前状态看板。完整任务信息见 `docs/tasks/`（Source of Truth）。

## Current Version

0.1.0 — see `internal/version` and `build/config.yml`.

## In Progress

- (none)

## Recently Completed

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

- Settings and About have live Wails data; account, provider, gateway and other
  feature pages remain placeholders until their backends land.
- No application feature services exist yet; `AppService` and `/healthz` are
  the only entry points.
- UI log streaming (Wails events from the logger buffer) is not wired yet.
- E2E (Playwright) is deferred until the first core user flow exists.

## Next Steps

1. Credential store (Windows DPAPI) behind `internal/infrastructure/credential`.
2. First application service + core feature: provider configuration and
   account management, with the account/provider pages wired to real bindings.
3. Gateway proxy server with request parsing and the first provider adapter
   (OpenAI), followed by the model router.
4. Account scheduler (eligibility, health, cooldown, retry) and automatic
   failover.
5. Wire the UI log stream through Wails events from the logger buffer.
