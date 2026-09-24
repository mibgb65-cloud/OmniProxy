# Technology Stack

The technology baseline was selected in `技术架构选择.md` (project root) and
recorded as ADR-001. This document describes the current state; the complete
dependency list lives in `go.mod` / `go.sum` and `frontend/package.json` /
`frontend/pnpm-lock.yaml`.

## Runtime

| Item        | Choice                                  | Status      |
| ----------- | --------------------------------------- | ----------- |
| Language    | Go 1.26 (`go.mod` pins `go 1.26.0`)     | in use      |
| Node.js     | 22 (`.nvmrc`, `engines` in package.json)| in use      |
| Package mgr | pnpm (lockfile committed)               | in use      |
| Task runner | Taskfile (`task`)                       | in use      |

## Languages

- Go — all backend code.
- TypeScript — all frontend code (`.ts` / `.tsx`); plain JavaScript business
  code is not allowed.

## Frameworks

- Wails 3 (v3.0.0-beta.25) — desktop shell only. Core must not import it.
- React 19 + Vite 8 — frontend SPA.
- chi — router/middleware for the local Control API. No large web framework
  (Gin/Fiber/Echo are explicitly out).
- net/http + database/sql + log/slog — standard library first.

## UI

- Tailwind CSS v4 (via `@tailwindcss/vite`)
- shadcn/ui (new-york style, neutral base) + Radix UI (`radix-ui` package)
- Lucide React — the only icon library
- TanStack Router (file-based routing, `src/app/routes/`)
- TanStack Query — server state (accounts, providers, quota, usage, …)
- Zustand — UI state only (sidebar, theme, transient selections)
- TanStack Table — planned for account/provider/request tables
- React Hook Form + Zod — planned for account/provider/routing forms
- Apache ECharts — planned for usage/quota charts (aggregation happens in Go)

## Storage

- SQLite via `database/sql` with the pure-Go `modernc.org/sqlite` driver
  (no CGO, matching the Wails Windows build defaults).
- SQL migrations in `migrations/`, applied by
  `internal/infrastructure/database`; `schema_version` tracks applied
  migrations. Schema changes never happen outside migrations.

## Networking

- Wails typed bindings — desktop UI ↔ Go (generated into `frontend/bindings`).
- Wails events — real-time state push to the UI.
- Local HTTP Control API (chi, loopback only) — external clients.
- net/http transport for upstream AI provider calls.

## Build Tools

- Vite 8 (frontend dev server, HMR, production build).
- Wails 3 CLI (`wails3 dev`, `wails3 build`, `wails3 generate bindings`).
- Taskfile as the single entry point (`task dev|build|test|lint|verify|…`).
- NSIS installer via the Wails build system (`task package`).

## Testing

- Go: standard `testing` + `net/http/httptest`, co-located `_test.go` files.
- Frontend: Vitest + Testing Library (jsdom), co-located `*.test.ts(x)`.
- E2E: Playwright — planned, added with the first core user flow.

## Major Dependencies

| Dependency                | Purpose                              | Why                                                    |
| ------------------------- | ------------------------------------ | ------------------------------------------------------ |
| `wailsapp/wails/v3`       | Desktop shell, bindings, events      | Chosen desktop framework (ADR-001)                     |
| `go-chi/chi/v5`           | Control API router/middleware        | Small, composable; no heavyweight web framework        |
| `modernc.org/sqlite`      | SQLite driver (pure Go)              | No CGO toolchain needed; matches Wails Windows builds  |
| `react` / `react-dom`     | UI library                           | Project standard                                       |
| `@tanstack/react-router`  | File-based routing                   | Type-safe routing, first-class TanStack integration    |
| `@tanstack/react-query`   | Server state                         | Cache/loading/error/invalidate for backend data        |
| `zustand`                 | UI state                             | Minimal store; server data stays in Query              |
| `@wailsio/runtime`        | Wails JS runtime + Vite plugin       | Required by the Wails frontend toolchain               |
| `tailwindcss`             | Utility CSS                          | Project standard                                       |
| `radix-ui`                | Accessible headless primitives       | Backs shadcn/ui components                             |
| `lucide-react`            | Icons                                | Single icon library                                    |
| `class-variance-authority`, `clsx`, `tailwind-merge` | Component variants / class merging | Standard shadcn/ui companions             |

Deferred until the feature that needs them lands: `@tanstack/react-table`,
`react-hook-form` + `zod`, `echarts`, Playwright. Adding a dependency earlier
than its first use is not allowed (see CODING_RULES).

## Platform Requirements

- Windows 10/11 x64 with the WebView2 runtime — the current target.
- macOS and Linux: not targeted yet; the Go core and the runtime profile
  design keep them possible (DPAPI credential storage is the main
  platform-specific piece and will sit behind `internal/infrastructure/credential`,
  to be created with the account feature).

## Signing & Distribution (planned)

- Code signing: SignPath.
- Installer: NSIS via the Wails build system.
- Updates: GitHub Releases + Wails updater with SHA-256 verification.
