# TASK-20260924-007: About Page - Version, Runtime Info, Update Check

## Status

Completed

## Created

2026-09-24 15:52

## Request

“在左侧的设置选项卡下面新增一个关于选项按钮，里面显示应用版本，运行信息，检查更新等内容。”

## Goal

侧边栏「设置」下方新增「关于」入口，页面包含：

1. 应用版本（版本号、应用标识、描述）
2. 运行信息：运行配置、端口、数据/日志目录、Go 与 Wails 版本、平台、运行时长、SQLite 版本
3. 检查更新：对照 GitHub Releases 检查新版本；未配置发布仓库时给出明确说明

## Background

- 技术栈文档规划更新体系为 GitHub Releases + Wails Updater（含 SHA-256 校验）；本任务实现其中“版本检查”一环，下载/安装更新留待后续独立任务。
- 项目当前没有配置 GitHub 远程仓库，因此更新检查在未配置仓库时必须明确告知用户，而不是假装成功。
- 版本比较、GitHub Releases 客户端均以标准库实现，可测试（httptest）。

## Scope

- Go：`internal/version` 语义化版本比较；`internal/infrastructure/update` GitHub Releases 检查客户端；`profile.Config` 增加 `updateRepo`；AppService 新增 `RuntimeInfo` 与 `CheckForUpdate` 绑定
- 前端：`/about` 路由与页面（运行信息卡、检查更新卡）、侧边栏导航项、i18n 中英案文、运行时长计时显示、测试

## Out of Scope

- 更新下载、SHA-256 校验、安装（Wails Updater 集成，后续任务）
- Release Notes 展示（仅提供发布页链接）
- Release/CI 发布流程

## Existing Behavior

侧边栏无「关于」入口；`/about` 路由不存在；无版本检查能力（`AppInfo` 仅含基础版本号）。

## Expected Behavior

- 侧边栏「设置」下方出现「关于」；页面展示应用名称/版本/描述
- 运行信息卡展示配置、端口、目录、Go/Wails 版本、平台架构、运行时长（每秒刷新）、SQLite 版本
- 「检查更新」按钮：已配置仓库时展示 当前版本/最新版本/发布页链接/是否可用；未配置时说明原因；网络失败时展示错误

---

# Analysis

## Related Modules

- `internal/version`（版本比较）
- `internal/infrastructure/update`（新增：GitHub Releases 客户端）
- `internal/platform/profile`（Config.UpdateRepo）
- `internal/transport/wails`（RuntimeInfo / CheckForUpdate 绑定）
- `frontend/src/app/routes/about.tsx`、`frontend/src/features/about/**`、`frontend/src/components/common/app-sidebar.tsx`、`frontend/src/i18n/locales/*`

## Dependencies

- GitHub Releases API（`api.github.com`），仅检查时访问；标准库 `net/http`
- Wails 版本号通过 `debug.ReadBuildInfo()` 依赖信息获取，避免硬编码与 go.mod 脱节

## Risks

- 未配置仓库时按钮必须给出可解释的“未配置”状态，而非伪造成功（产品诚信）
- GitHub API 限流/网络异常需有明确错误状态
- Low：只读外部 API，不影响本地数据

---

# Implementation Plan

1. `internal/version/compare.go`：语义化版本比较（剥离 v 前缀与预发布后缀，数值三段比较）+ 测试
2. `profile.Config` 增加 `UpdateRepo`
3. `internal/infrastructure/update/checker.go`：Latest(ctx) 获取最新 Release（可注入 BaseURL 供测试）+ 测试
4. `internal/transport/wails/about.go`：RuntimeInfo（含 startedAt、SQLite 版本、构建信息）与 CheckForUpdate（unconfigured/up-to-date/available/error 四态）+ 测试
5. 重新生成 bindings
6. 前端：use-about（query+mutation）、runtime-card（含每秒 uptime）、update-card、路由、侧边栏项、formatDuration、i18n
7. 前端测试；task verify；提交；完成本记录

---

# Progress

- 确认 Wails 版本经 `debug.ReadBuildInfo()` 依赖列表获取；发现测试二进制不含模块依赖信息（deps=0），真实构建二进制正常（已用独立程序验证），测试断言相应放宽并注释
- 路由树在新路由添加后需重新生成（vite 构建触发），否则 `createFileRoute("/about")` 类型报错

---

# Implementation Result

## Changes

- `internal/version/compare.go`：`Compare(a, b)` 语义化比较（v 前缀、预发布后缀、build metadata 处理；不可解析时回退字符串比较）
- `internal/infrastructure/update/checker.go`：`Checker.Latest(ctx)` 查询 GitHub `/releases/{owner}/{repo}/releases/latest`（Accept/User-Agent 头、10s 超时、404/5xx/损坏负载/不可达均有明确错误；BaseURL 可注入）
- `internal/platform/profile/config.go`：`Config.UpdateRepo`（"owner/name"，空 = 未配置）
- `internal/transport/wails/about.go`：`RuntimeInfo`（版本/名称/描述/AppID/profile/目录/端口/Go 版本/Wails 版本/平台/SQLite 版本/启动时间）与 `CheckForUpdate`（四态：up-to-date/available/unconfigured/error）；`newUpdateChecker` 包变量作为测试接缝
- `internal/transport/wails/app.go`：AppService 记录 `startedAt`
- 前端：`features/about`（use-about、about-view、runtime-card、update-card）、`app/routes/about.tsx`、侧边栏「关于」导航项（Info 图标，位于设置下方）、`lib/format.ts` 增加 `formatDuration`、services 增加两个封装、i18n 新增 nav.about + 31 个 about.* 键（中英）

## Files Changed

- Go：`internal/version/{compare.go,compare_test.go}`、`internal/infrastructure/update/{checker.go,checker_test.go}`、`internal/platform/profile/config.go`、`internal/transport/wails/{about.go,about_test.go,app.go}`
- 前端：`frontend/src/features/about/*`、`frontend/src/app/routes/about.tsx`、`frontend/src/app/routeTree.gen.ts`（重新生成）、`frontend/src/components/common/app-sidebar.tsx`、`frontend/src/lib/{format.ts,format.test.ts}`、`frontend/src/services/omniproxy.ts`、`frontend/src/i18n/locales/{zh.ts,en.ts}`、`frontend/bindings/**`（重新生成）

## Technical Notes

- 更新检查为“只读发现”一环：不下载、不安装；与下载/安装解耦，后续 Wails Updater 任务在其上构建
- uptime 由前端基于 `startedAt`（RFC3339）每秒计算，`HH:MM:SS` 格式语言中立，无需 i18n
- 发布页链接使用 `target="_blank" rel="noreferrer"`（Wails WebView 中新窗口打开外链）

---

# Verification

## Tests

- [x] Unit Tests（Go：version 比较 3 例、update checker 5 例、about 5 例；Frontend：formatDuration 3 例、AboutView 4 例）
- [ ] Integration Tests — N/A
- [ ] E2E Tests — N/A

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
PASS（Go 全部通过；前端 36 个测试通过）
```

---

# Documentation

- [ ] PRD — 无需更新（关于页属于基础桌面能力）
- [ ] Architecture — 无需更新（模块边界未变）
- [x] Tech Stack — 无需更新（更新检查属于既定架构的首个落地，TECH_STACK 已描述该体系）
- [ ] Coding Rules — 无需更新
- [ ] Testing — 无需更新
- [ ] Build — 无需更新
- [ ] ADR — 无需更新（沿用既定更新架构）
- [x] CURRENT（随任务完成更新）

---

# Deviations

- 原计划：Wails 版本断言必须等于依赖版本。实际：测试放宽为非空（测试二进制不含依赖信息，真实二进制正常）。原因：Go 测试二进制的 `debug.ReadBuildInfo().Deps` 为空是既有行为，已在代码注释与测试注释中说明。
- 其余按计划完成。

---

# Git

## Commit

```text
docs: establish Task Record workflow
```

（本任务的 commit hash 由紧随其后的记录提交回填。）

---

# Result

「关于」页完成：应用版本、完整运行信息（含实时运行时长）与检查更新（四态，未配置时明确说明）。`task verify` 全绿。

---

# Completed

2026-09-24 16:05
