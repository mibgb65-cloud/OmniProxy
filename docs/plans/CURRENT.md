# Current Development Status

当前状态看板。完整任务信息见 `docs/tasks/`（Source of Truth）。

## Current Version

0.1.0 — see `internal/version` and `build/config.yml`.

## In Progress

- (none)

## Recently Completed

- TASK-20260924-006 Settings page: data storage management
- TASK-20260924-005 Task Record workflow
- TASK-20260924-004 Migrate i18n to i18next + react-i18n
- TASK-20260924-003 Frontend i18n (Chinese/English)
- TASK-20260924-002 State file/function size limits in coding rules
- TASK-20260924-001 Project initialization foundation

## Known Issues

- The dashboard is the only page with live data; the other nine pages are
  placeholders until their backends land.
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
