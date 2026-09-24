# Current Development Status

## Current Version

0.1.0 (initialization) — see `internal/version` and `build/config.yml`.

## Current Goal

Frontend milestone 1: internationalization (Chinese/English) is done; next is
wiring the first real feature pages once backends land.

## In Progress

- Nothing in progress. Frontend i18n is complete and verified.

## Recently Completed

- Frontend i18n: i18next + react-i18n with typed zh/en dictionaries
  (`frontend/src/i18n`), language switcher in the sidebar, language persisted
  in the Zustand UI store (localStorage). All existing UI copy migrated to
  translation keys; document `lang`/title follow the selection.
- Wails 3 + React + Vite project scaffold, restructured to the planned layout
  (`internal/…`, `frontend/src/…`; `main.go` stays at the root per Wails
  tooling).
- Runtime profiles (production/development/test) with isolated data dirs and
  ports, `.env` support (`internal/platform/profile`).
- Structured logging with central Redactor, rotating file sink and UI buffer
  (`internal/infrastructure/logger`).
- SQLite open + embedded SQL migration runner with `schema_version`
  (`internal/infrastructure/database`, `migrations/001_initial.sql`).
- Local Control API skeleton (chi, loopback, `/healthz`) and the Wails
  `AppService` bindings (`AppInfo`, `Ping`) sharing startup/shutdown.
- Frontend shell: Tailwind v4 + shadcn/ui, TanStack Router (all planned pages
  routed, placeholders), TanStack Query, Zustand, dashboard showing live
  backend info, Vitest + ESLint + Prettier.
- Unified commands via Taskfile (`task dev/build/test/lint/format/typecheck/
  verify`) and GitHub Actions CI.
- Full documentation set (PRD, architecture, tech stack, coding/Git/testing/
  build rules, ADR-001) and the first commit on `main`.
- Verified end to end: `task verify` passes and the production binary was
  smoke-tested (starts, binds the Control API, serves `/healthz`, writes logs,
  keeps dev/prod data directories separate).

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
   account management (`internal/application/provider`, `internal/core/…`),
   with the account/provider pages wired to real bindings.
3. Gateway proxy server with request parsing and the first provider adapter
   (OpenAI), followed by the model router.
4. Account scheduler (eligibility, health, cooldown, retry) and automatic
   failover.
5. Wire the UI log stream through Wails events from the logger buffer.
