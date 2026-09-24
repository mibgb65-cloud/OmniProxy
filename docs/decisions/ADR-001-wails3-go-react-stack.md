# ADR-001: Wails 3 + Go Core + React Frontend Stack

## Status

Accepted

## Context

OmniProxy is a full rewrite of an existing local AI API gateway and
multi-account management tool. The previous implementation's technology
baseline needed a deliberate, documented decision before any feature code was
written, because the choice constrains every later change: desktop framework,
backend language, HTTP stack, frontend stack, UI system, state management,
database access, logging, credential storage, testing and CI.

The requirements that drove the decision:

- A Windows desktop application with a native window, system tray, single
  instance, autostart and auto-update.
- A Go backend capable of running as a standalone core (proxy, scheduler,
  routing) independent of any UI.
- A typed, low-friction bridge between Go and the UI.
- A local HTTP Control API for external clients (CLI, agents, MCP) that shares
  logic with the UI path.
- Local-first storage (SQLite) with OS-level credential protection.
- A modern, type-safe frontend that stays maintainable as the UI grows.

## Decision

Adopt the stack documented in `技术架构选择.md` (project root):

- **Desktop:** Wails 3 (beta) with the WebView2 runtime. Wails is responsible
  for the desktop shell only; the core must never depend on it.
- **Backend:** Go with `net/http` + `chi` for the Control API,
  `database/sql` + SQLite (pure-Go `modernc.org/sqlite` driver) for storage,
  and `log/slog` for structured logging.
- **Frontend:** React + TypeScript built with Vite; package manager pnpm;
  Tailwind CSS + shadcn/ui + Radix UI + Lucide React; TanStack Router for
  routing; TanStack Query for server state; Zustand for UI state; TanStack
  Table, React Hook Form + Zod and Apache ECharts for the feature UIs.
- **Communication:** Wails typed bindings for the UI, Wails events for
  real-time state, and a loopback HTTP Control API for external clients; both
  entry points reuse the same application services.
- **Security:** credentials in the OS credential store (DPAPI on Windows),
  database stores only `credential_ref`; a central log Redactor.
- **Storage:** SQLite with append-only SQL migrations tracked in
  `schema_version`.
- **Testing:** Go `testing` + `httptest`; Vitest + Testing Library for the
  frontend; Playwright for E2E when the first core flow exists.
- **CI/CD:** GitHub Actions, Taskfile as the single command entry point,
  SignPath for signing, NSIS installer, GitHub Releases + Wails updater.

## Alternatives Considered

- **Electron / Tauri:** rejected — larger runtime footprint and a second
  language/runtime for the shell; the project explicitly excludes them.
- **Vue / Angular:** rejected — the team standard is React + TypeScript.
- **Gin / Fiber / Echo:** rejected — chi plus `net/http` covers routing and
  middleware without a heavyweight framework.
- **PostgreSQL / MongoDB / Redis:** rejected — a local-first single-user tool
  does not need an external datastore; SQLite is sufficient and portable.
- **Redux / MobX:** rejected — TanStack Query (server state) plus Zustand (UI
  state) covers the needs with far less boilerplate.
- **CGO SQLite driver (`mattn/go-sqlite3`):** rejected — the Wails Windows
  build defaults to `CGO_ENABLED=0`; the pure-Go driver keeps builds
  toolchain-free.

## Reasoning

- Go gives a genuinely reusable core: the gateway, scheduler and routing logic
  can be tested and later reused behind a CLI or the Control API without a UI.
- Wails 3 provides the required desktop capabilities (tray, single instance,
  autostart, events, updater) while keeping the UI in standard web technology.
- Typed bindings plus events remove an entire class of UI/backend integration
  bugs and make polling unnecessary.
- SQLite plus migrations gives a reproducible, reviewable schema evolution
  with zero operational overhead.
- The dependency list stays small on purpose: every addition must earn its
  place (see `docs/development/CODING_RULES.md`).

## Consequences

- The core must not import Wails or React packages; this is enforced by
  package layout and review.
- UI code must not access SQLite or the credential store directly.
- The HTTP API and the bindings must share application services; duplicated
  business logic across transports is a defect.
- `main.go` stays at the repository root because the Wails v3 build tooling
  builds the root main package; all other Go code lives under `internal/`.
- Beta-version desktop framework: Wails 3 API churn is accepted as a risk and
  upgrades are deliberate, tested steps.
- Pure-Go SQLite avoids CGO but increases binary size somewhat; accepted for
  build simplicity.
