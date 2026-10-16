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

- Wails 3 (v3.0.0-beta.28) — desktop shell only. Core must not import it.
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
- React Hook Form + Zod — forms (settings, provider config, routing rules);
  `@hookform/resolvers/zod` wires the schema in. Backend validation stays
  authoritative; frontend schemas guard the obvious cases.
- Apache ECharts — planned for usage/quota charts (aggregation happens in Go)
- Internationalization: `i18next` + `react-i18n` (`frontend/src/i18n`). The
  Chinese dictionary (`locales/zh.ts`) is the key source of truth; the English
  dictionary is type-checked against it, and an i18next module augmentation
  makes `t()` accept only known keys (compile error on typos). The selected
  language lives in the Zustand UI store and is persisted to localStorage;
  `I18nProvider` syncs store changes into i18next. Keys are flat with
  `keySeparator: false`.

## Storage

- SQLite via `database/sql` with the pure-Go `modernc.org/sqlite` driver
  (no CGO, matching the Wails Windows build defaults).
- SQL migrations in `migrations/`, applied by
  `internal/infrastructure/database`; `schema_version` tracks applied
  migrations. Schema changes never happen outside migrations.
- Opaque credential bundles are stored as per-user DPAPI-protected files in
  profile-specific data directories (`internal/infrastructure/credential`,
  ADR-002). SQLite stores only `credential_ref` values.
- Provider configurations use `providers` and `internal/application/provider`.
  Migration `002_builtin_providers.sql` persists nine default-enabled built-in
  rows and marks existing user-created rows as custom; custom endpoints use the
  OpenAI-compatible kind. The UI uses generated Wails bindings and TanStack
  Query to manage switches and custom configurations. Curated model names are
  kept in the built-in catalog for display; a credential-free HTTP HEAD probe
  measures endpoint reachability and response-header latency without redirects.
  Migration `019_builtin_atria.sql` adds Atria as the tenth preset without
  resetting existing state. Its native HTTP adapter uses the existing standard
  library transport and proxy interfaces; no new SDK dependency is added.
- API Key accounts use the existing `accounts` table and
  `internal/application/account`. The account list exposes metadata only;
  `credential_ref` stays in SQLite while the API Key is saved in the profile's
  DPAPI-protected credential store. The desktop UI manages accounts through
  generated Wails bindings and TanStack Query.

- 迁移 `029_builtin_river.sql` 增加第十一家 River；标准库原生 Chat／Responses、既有 Messages 转换与账号目录缓存，不新增 SDK 或依赖。

- ChatGPT / Codex 订阅授权复用本机官方 Codex CLI 的 stdio app-server，
  不新增 SDK 依赖；授权令牌由官方组件的强制 keyring 管理，DPAPI 仅保存
  隔离配置定位。SQLite 仍只保存元数据和 credential_ref（ADR-003）。

## Networking

- Wails typed bindings — desktop UI ↔ Go (generated into `frontend/bindings`).
- Wails events — real-time state push to the UI.
- Local HTTP Control API (chi, loopback only) — external clients.
- net/http transport for upstream AI provider calls.
- MiMo 沿用标准库原生 Chat／Responses／Messages HTTP／SSE 和 `api-key`，不新增 SDK、依赖或数据库迁移；模型发现复用现有账号快照表。

- 阶跃星辰四类账号通道复用标准库与 DPAPI，迁移 021／022 维护配置版本及请求归属；无新增 SDK 或依赖。

- Kimi 普通 API 同样使用标准库与 DPAPI，迁移 023 管理账号地区与目录失效；原生协议按官方模型能力开放，不新增 SDK 或依赖。
- MiniMax 双平台按量／订阅复用标准库原生 HTTP／SSE、DPAPI 与已有查询队列，迁移 024 保存账号选项和目录版本；订阅资源使用独立公开 DTO 与固定官方入口，无新增 SDK 或依赖。
- 小米 MiMo Token Plan 复用已有标准库原生 HTTP／SSE 和 DPAPI，迁移 025 保存账号计费通道／cn、sgp、ams 集群及目录版本；通道隔离路由沿用现有结构，无新增 SDK 或依赖。
- 迁移 026 保存 MiMo 应用级默认计费顺序；路由与客户端目录复用既有组件，原通道名仅兼容解析，无新增依赖。
- 迁移 027 独立保存 StepFun 默认计费顺序，与 MiMo 共用合格账号筛选及前端设置交互，无新增依赖。
- 迁移 028 增加 Kimi 普通／Code 计费选项及独立计费／地区顺序，复用原生 HTTP／SSE、DPAPI、路由及设置组件，不新增依赖。

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

客户端配置适配增加 `gopkg.in/yaml.v3 v3.0.1`（已有 go.sum 来源，保留 Harness YAML 节点／
注释）及 `github.com/tailscale/hujson`（固定伪版本，正确解析和增量修改 OpenCode JSONC）。
两者只用于已有本地配置文件，不引入客户端 SDK 或 Agent 运行时。

| Dependency                | Purpose                              | Why                                                    |
| ------------------------- | ------------------------------------ | ------------------------------------------------------ |
| `wailsapp/wails/v3`       | Desktop shell, bindings, events      | Chosen desktop framework (ADR-001)                     |
| `go-chi/chi/v5`           | Control API router/middleware        | Small, composable; no heavyweight web framework        |
| `modernc.org/sqlite`      | SQLite driver (pure Go)              | No CGO toolchain needed; matches Wails Windows builds  |
| `golang.org/x/sys/windows`| Windows DPAPI calls                  | Existing module dependency; typed OS API wrappers      |
| `github.com/pelletier/go-toml/v2` | 保留字段的客户端 TOML 合并 | 解析官方语法位置，避免误改提供商身份和无关配置 |
| `github.com/coder/websocket` | Codex Responses WebSocket | 已有 Wails 传递依赖，供代理协议使用 |
| `filippo.io/age` | 官方 CodexAuth 加密存储读取 | 标准 age 库兼容官方密文，避免自行实现密码算法；仅后端短期解密 |
| `react` / `react-dom`     | UI library                           | Project standard                                       |
| `@tanstack/react-router`  | File-based routing                   | Type-safe routing, first-class TanStack integration    |
| `@tanstack/react-query`   | Server state                         | Cache/loading/error/invalidate for backend data        |
| `zustand`                 | UI state                             | Minimal store; server data stays in Query              |
| `@wailsio/runtime`        | Wails JS runtime + Vite plugin       | Required by the Wails frontend toolchain               |

Wails Go 模块、CLI 与前端 `@wailsio/runtime` 固定使用同一 beta.28 版本；CI 与 README 的 CLI 安装命令同步固定版本。
| `tailwindcss`             | Utility CSS                          | Project standard                                       |
| `radix-ui`                | Accessible headless primitives       | Backs shadcn/ui components                             |
| `lucide-react`            | Icons                                | Single icon library                                    |
| `class-variance-authority`, `clsx`, `tailwind-merge` | Component variants / class merging | Standard shadcn/ui companions             |
| `react-hook-form`, `zod`, `@hookform/resolvers` | Form state + runtime validation    | Project-standard form stack (TECH_STACK)                 |

Deferred until the feature that needs them lands: `@tanstack/react-table`,
`echarts`, Playwright. Adding a dependency earlier than its first use is not
allowed (see CODING_RULES).

## Platform Requirements

- Windows 10/11 x64 with the WebView2 runtime — the current target.
- macOS and Linux: not targeted yet; the Go core and runtime profile design
  keep them possible. The current `internal/infrastructure/credential` DPAPI
  implementation is Windows-specific.

## Signing & Distribution (planned)

- Code signing: SignPath.
- Installer: NSIS via the Wails build system.
- Updates: GitHub Releases + Wails updater with SHA-256 verification.
