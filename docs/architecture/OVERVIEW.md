# Architecture Overview

## System Overview

OmniProxy is a local desktop application built as three layers:

```text
┌────────────────────────────────────────────┐
│ Desktop Shell (Wails 3 + React frontend)   │  internal/transport/wails + frontend/
├────────────────────────────────────────────┤
│ Application Services (use cases)           │  internal/application/<feature>
├────────────────────────────────────────────┤
│ Core (gateway, routing, scheduler, …)      │  internal/core/…
├────────────────────────────────────────────┤
│ Infrastructure (db, credentials, logger)   │  internal/infrastructure/…
└────────────────────────────────────────────┘
```

The Core must stay independent of Wails and React: if both were deleted, the
Core would still compile and run behind the local HTTP Control API.

## Major Components

| Component            | Location                              | Responsibility                                            |
| -------------------- | ------------------------------------- | --------------------------------------------------------- |
| Desktop shell        | `internal/transport/wails`            | Wails app, window, typed bindings, lifecycle              |
| Control API          | `internal/transport/httpapi`          | Local loopback HTTP API (chi) for CLI/agents/MCP          |
| Application services | `internal/application/<feature>`      | Use-case orchestration shared by both transports          |
| Core                 | `internal/core/…`                     | Gateway, router, scheduler, quota, usage, model registry  |
| Provider adapters    | `internal/provider/<name>`            | Per-provider capability behind one adapter interface      |
| Client integrations  | `internal/integration/<client>`       | Detect/read/backup/apply/restore/validate client configs  |
| Infrastructure       | `internal/infrastructure/…`           | SQLite database + migrations, credential store, logger    |
| Runtime profile      | `internal/platform/profile`           | Production/development/test isolation (ports, data dirs)  |
| Frontend             | `frontend/src/…`                      | React SPA: pages, features, components, hooks, services; all user-facing copy goes through `frontend/src/i18n` (zh/en) |

## Component Responsibilities

### Desktop shell (`internal/transport/wails`)

Creates the Wails application and the main window, registers bound services
(the `AppService`), and manages startup/shutdown. It contains no business
logic. It is the only package allowed to import the Wails runtime.

On Windows, the shell also owns the system tray and the window-close hook.
Closing the main window exits by default; a persisted setting can hide it to
the tray instead. The tray menu restores the window or quits through the normal
shutdown path. Login autostart uses Wails' per-user Windows registration and
is exposed only in the production profile.

### Control API (`internal/transport/httpapi`)

A chi router bound to `127.0.0.1:<control port>`. External clients use it to
query health and (later) drive the gateway. The desktop UI does **not** call
it; the UI uses Wails typed bindings.

### Application services (`internal/application/<feature>`)

One package per feature area (account, gateway, provider, quota, routing,
settings). Both transports call into these services; neither transport
duplicates business logic.

> Current state: `internal/application/provider` manages nine default-enabled
> built-in providers and custom OpenAI-compatible endpoints in SQLite and is
> exposed through Wails bindings. It also provides a bounded, unauthenticated
> endpoint reachability check. `internal/application/account` manages API Key
> account metadata and protected credentials through Wails bindings. The Control
> API still exposes only `/healthz`; gateway adapters have not landed yet.

账号单个创建接受空名称并生成 Key 前最多 8 个 Unicode 字符的名称；批量创建只接受提供商 ID 和 1–100 个 Keys，先完整校验，再通过 SQLite 事务统一保存元数据。任一写入失败时回滚本批元数据并清理已写入的凭证；清理失败会向调用方返回错误。自动名称包含用户明确要求的 Key 前缀，不得将名称写入日志。

前端新增账号弹窗提供单个／批量方式，批量每行一个 Key、不提供名称输入；总览和详情共享 Radix AlertDialog 删除确认，只有确认后才调用删除服务。

账号详情的卡片与列表复用时间显示、启停、删除和密钥预览组件，展示偏好保存在 Zustand。查询缓存仅存新的密钥接口返回的脱敏字符串。复制由 Wails 适配层在受控凭证回调内直接写入系统剪贴板，复制方法不返回密钥内容；解密字节在回调结束后清理。较短密钥的预览全部遮掩，凭证读取失败时提供重试，余额未接入时继续显示待接入。

账号编辑复用统一弹窗，只提交名称和可选的新 Key。替换时先写新凭证，再通过事务与原引用／更新时间条件切换数据库引用；提交失败清理新凭证，原凭证不受影响。提交成功后清理旧凭证，清理失败由 UpdateResult 的警告标志报告，避免将已更新误报为未保存。前端提交结束清理 mutation 输入状态，并刷新元数据和脱敏预览。

### Core (`internal/core/…`)

Pure Go business logic: the proxy server, request parsing, model routing,
account scheduling, quota tracking and usage collection. No HTTP framework
types, no Wails types, no database types in its public API.

### Infrastructure

- `infrastructure/database`: opens SQLite via `database/sql` (pure-Go
  `modernc.org/sqlite` driver) and applies embedded SQL migrations from
  `migrations/`. The schema only changes through migrations; `schema_version`
  records what has been applied.
- `infrastructure/logger`: structured `log/slog` setup with a central
  Redactor; sinks are console, rotating file and an in-memory UI buffer.
- Credential storage: per-user DPAPI-protected files under each Windows
  profile's data directory (ADR-002). The database stores only a
  `credential_ref`, never a plaintext secret.

### Runtime profile (`internal/platform/profile`)

Production, development and test profiles resolve their own app ID, data
directory, proxy port, control port and log directory. Dev and production
never share a data directory. The active profile comes from the build tag
(`production` for `wails3 build`) and can be overridden with
`OMNIPROXY_PROFILE`.

## Data Flow

Request path (target architecture):

```text
AI client
  → HTTP Gateway (core/proxy)
  → Request parser
  → Model router (core/routing)
  → Account scheduler (core/scheduler)
  → Provider adapter (provider/<name>)
  → HTTP transport
  → AI provider
```

Response path:

```text
AI provider
  → Provider adapter
  → Response transformer
  → Usage collector (core/usage)
  → client
```

UI path:

```text
React
  → Wails typed binding (frontend/bindings, generated)
  → Application service
  → Core / Infrastructure
```

External client path:

```text
CLI / agent / MCP
  → HTTP Control API (loopback)
  → Application service (same services as the bindings)
  → Core / Infrastructure
```

Real-time updates reach the UI through Wails events (e.g. `proxy:started`,
`request:completed`, `quota:updated`), not through polling.

## External Dependencies

- AI providers: OpenAI, Anthropic, Google, DeepSeek, Zhipu AI, Moonshot AI/Kimi,
  MiniMax, Xiaomi MiMo, StepFun, and OpenAI-compatible endpoints. Nine named
  provider kinds are persisted and enabled by default, with user-controlled
  switches. Custom OpenAI-compatible endpoints can be added separately;
  request adapters are still pending.
- Local AI clients whose configs are managed: Codex, Claude Code, Gemini CLI,
  OpenCode.
- GitHub Releases + Wails updater for application updates.
- Windows DPAPI for local credential protection; Keychain / Secret Service
  adapters may be added when those platforms are supported.

## Architecture Boundaries

1. Core does not depend on Wails.
2. Core does not depend on React.
3. The UI never touches SQLite directly.
4. The UI never touches the credential store.
5. Providers never render UI.
6. The HTTP API and the Wails bindings share the same application services.
7. Business logic never lives inside React components.
8. Credentials never reach logs (central Redactor).
9. Real-time state uses events, not polling.
10. New providers are added as adapters, not by modifying the gateway core.

These boundaries are enforced by review; several are additionally enforced by
package layout (`internal/…` import rules).
