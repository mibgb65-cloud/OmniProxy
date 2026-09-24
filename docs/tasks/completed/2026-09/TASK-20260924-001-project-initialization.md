# TASK-20260924-001: Project Initialization Foundation

## Status

Completed

## Created

2026-09-24 15:10

## Request

按照《新项目初始化规范.md》的要求，以及本项目选择的《技术架构选择.md》技术栈，进行项目的初始化。

## Goal

建立可持续维护、可复现、可验证、适合多 Agent 协作的项目基础：目录结构、文档体系、工具链、质量门禁与 CI，使后续功能开发有一个稳定基线。

## Background

仓库初始为空（仅两份规范文档与 `git init` 结果，无提交）。技术栈基线已在《技术架构选择.md》中确定：Wails 3 桌面外壳 + Go Core + React 前端。

## Scope

- Wails 3 项目脚手架并按目标布局重构
- Go 后端骨架：运行时 Profile、结构化日志（含 Redactor）、SQLite + 迁移、Wails 绑定与 Control API
- 前端工程体系：Vite + React + TS、Tailwind + shadcn/ui、TanStack Router/Query、Zustand、Vitest/ESLint/Prettier
- 配置文件：.gitignore/.gitattributes/.editorconfig/.env.example/.nvmrc
- Taskfile 统一命令与 GitHub Actions CI
- 完整文档体系（README、AGENTS、docs 全套、ADR-001）

## Out of Scope

- 具体业务功能（Provider 适配、调度器、网关等）
- E2E（Playwright）与 Release 自动化
- macOS/Linux 平台支持

## Existing Behavior

空项目，无任何代码与文档。

## Expected Behavior

`task verify` 全流程通过；生产二进制可启动并通过冒烟测试；文档齐全；CI 可运行。

---

# Analysis

## Related Modules

- `main.go`（Wails 入口）
- `internal/platform/profile`、`internal/infrastructure/{logger,database}`、`internal/transport/{wails,httpapi}`、`internal/version`
- `frontend/src/**`
- `migrations/`、`build/`、`.github/`

## Dependencies

- Wails 3 CLI（v3.0.0-beta.25）、Go 1.26、Node 22、pnpm、Task、staticcheck

## Risks

- Wails v3 为 beta 版本，API 可能变动（接受为已知风险）
- `modernc.org/sqlite` 为纯 Go 驱动，编译体积较大（接受）

---

# Implementation Plan

1. 安装工具链（wails3、pnpm、task、staticcheck）
2. `wails3 init` 脚手架到临时目录，合并并按目标布局重构
3. 实现 Go 骨架各包并编写测试
4. 搭建前端工程并接入全部选定依赖
5. 创建配置文件、Taskfile、CI
6. 编写完整文档体系
7. 运行 `task verify`，冒烟测试生产二进制
8. 首次提交

---

# Progress

- 确认 Wails v3 构建工具链要求 main 包位于仓库根目录（`go build` 直接构建根目录），决定 `main.go` 保留根目录、其余代码全部进入 `internal/`
- 确认 `build/ios` 模板文件在桌面平台不可编译，Go 命令精确指定 `.` 与 `./internal/...`
- 冒烟测试发现控制端口被占用会导致应用整体退出，改为降级模式（UI 可用、错误明确记录）

---

# Implementation Result

## Changes

- Wails 3 + React + Vite 脚手架合并至根目录，`main.go` 保持根目录（Wails 约定），业务代码位于 `internal/`
- 运行时 Profile：production/development/test 隔离（端口、数据目录、AppID），支持 `.env` 与环境变量覆盖
- 结构化日志：`log/slog` + 中央 Redactor + 轮转文件 + UI 缓冲三个 sink
- SQLite：`database/sql` + 纯 Go 驱动，嵌入 `migrations/` 的迁移运行器（`schema_version` 追踪）
- Control API（chi，仅回环，`/healthz`）与 Wails `AppService`（`AppInfo`/`Ping`）共享启停
- 前端外壳：Tailwind v4 + shadcn/ui、TanStack Router（10 个页面路由）、TanStack Query、Zustand、Dashboard 展示真实绑定数据
- Taskfile 统一命令（dev/build/test/lint/format/typecheck/verify）与 CI
- 文档全套：README、AGENTS.md、docs/{README,product/PRD,architecture/{OVERVIEW,TECH_STACK},development/{CODING_RULES,GIT_WORKFLOW,TESTING,BUILD},plans/CURRENT,decisions/{README,ADR-001}}

## Files Changed

- `main.go`、`internal/**`、`frontend/**`、`migrations/001_initial.sql`、`build/config.yml`、`Taskfile.yml`、`.github/workflows/ci.yml`、`.gitignore`、`.gitattributes`、`.editorconfig`、`.env.example`、`.nvmrc`、`README.md`、`AGENTS.md`、`docs/**`

## Technical Notes

- SQLite 驱动选型 `modernc.org/sqlite`：Wails Windows 构建默认 `CGO_ENABLED=0`，纯 Go 驱动无需 CGO 工具链
- `frontend/dist/.gitkeep` + gitignore 反转规则：保证新克隆上 `go build`/`go test` 可编译（main 包 embed frontend/dist）
- `frontend/bindings/` 为生成代码但提交入库：Wails v3 模板默认不忽略，且使 `tsc`/ESLint/Vitest 在无 Wails 工具链时可用

---

# Verification

## Tests

- [x] Unit Tests（Go：profile/logger/database/httpapi；Frontend：Vitest）
- [ ] Integration Tests — N/A（初始化阶段）
- [ ] E2E Tests — N/A（按规范推迟）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task verify
PASS
```

附加冒烟测试：生产二进制启动后 `/healthz` 返回 `{"status":"ok","version":"0.1.0"}`，dev/prod 数据目录隔离，日志结构化落盘。

---

# Documentation

- [x] PRD
- [x] Architecture
- [x] Tech Stack
- [x] Coding Rules
- [ ] Testing — 已创建
- [x] Build
- [x] ADR（ADR-001）
- [x] CURRENT

---

# Deviations

- 原计划：`main.go` 置于 `cmd/omniproxy/`（技术架构文档推荐布局）。实际：保留根目录。原因：Wails v3 构建 Taskfile 直接构建根 main 包，移动会破坏 `task build`/`task dev`。
- 原计划：创建 `scripts/` 目录。实际：未创建。原因：Taskfile 已清晰覆盖全部命令，避免空目录。
- TanStack Table / React Hook Form + Zod / ECharts / Playwright 未接入：遵循“不为不存在的需求提前引入依赖”，待首个使用功能落地时引入。

---

# Git

## Commit

```text
4b6d924 chore: initialize OmniProxy project foundation
```

---

# Result

项目基础建成：`task verify` 全绿，生产二进制冒烟测试通过，文档与 CI 就绪，可以开始功能开发。

---

# Completed

2026-09-24 11:45

---

> Retroactive record: 本记录于 Task Record 工作流建立时补建（见 TASK-20260924-005），工作本身于 2026-09-24 上午完成。
