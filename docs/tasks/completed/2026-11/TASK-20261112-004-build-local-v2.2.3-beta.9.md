# TASK-20261112-004: 构建本地 v2.2.3-beta.9 生产安装包

## Status

In Progress

## Created

2026-11-12 11:05（Asia/Singapore）

## Request

用户要求：提交代码后构建新的 Beta 生产版本，供其本地更新，并完成任务 Markdown 更新。

> 提交代码后，然后构建新的beta生产版本我进行本地更新，记得完成任务md更新

## Goal

将 Google 订阅 Agent 链路对齐（TASK-20261112-002，`d4af73e9`）纳入 `v2.2.3-beta.9`，
同步版本与双语发行说明，提交版本源码后构建 Windows amd64 生产安装包，核对版本资源与摘要，
并归档任务记录。

## Background

- `v2.2.3-beta.8` 已于 2026-11-12 本地构建（TASK-20261112-003 构建任务，版本提交 `dc114884`），
  当前源码应用／前端版本仍为 `2.2.3-beta.8`。
- 自 beta.8 版本提交 `dc114884` 起的用户可见改动只有 TASK-20261112-002
  （`d4af73e9`，50 个文件）及其两份文档回填提交（`af053226`、`9ea7c54b`）。
- 该实现任务的 `task verify` 通过（147 文件／895 项前端、全部 Go 包、Windows production 构建），
  真机对照已完成：原文 system prompt 得到不可重试的 `400 prompt_turned_away`，按词打散后与
  删去命中行均为 `200`。
- 工作区另有一条未跟踪、与本次无关的旧记录
  `docs/tasks/completed/2026-10/TASK-20261029-010-subq-provider.md`，保持原样，不纳入构建或提交。

## Scope

- 应用／前端／Windows 显示版本由 `2.2.3-beta.8` 提升为 `2.2.3-beta.9`。
- Windows 数字版本保持 `2.2.3.0`，NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.9.md`（前端 `import.meta.glob` 随包读取）。
- 提交版本源码后执行生产打包，核对版本资源、SHA256 与发布目录。
- 更新任务记录与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不重跑完整质量门：本任务只做版本映射与打包；测试与构建结果已记录在 TASK-20261112-002。
- 不推送、不打标签、不公开发布、不签名。
- 不安装、不启动或停止用户实例，不修改真实用户数据及客户端配置。

## Existing Behavior

Google 订阅 Agent 链路对齐已提交为 `d4af73e9`，文档回填为 `af053226`、`9ea7c54b`；
版本仍为 `2.2.3-beta.8`，`release-assets/v2.2.3-beta.9` 不存在。

## Expected Behavior

新安装器包含 Google 订阅 Agent 链路对齐与「按原文发送 system prompt」设置项，用户从托盘完全退出后
可本地覆盖更新；版本资源与摘要与实际产物一致。

---

# Analysis

## Related Modules

- `internal/version/version.go`
- `frontend/package.json`
- `build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- `docs/releases/`、`docs/plans/CURRENT.md`

## Dependencies

沿用 Wails v3.1.0、Task 3.53.1、Go 1.26.2、pnpm 12.6.0、NSIS 及现有版本映射／产物校验脚本。

## Risks

- 本地安装器未签名，可能触发未知发布者或 SmartScreen 提示。
- 版本与资源映射不一致或复用旧产物会导致错误交付；使用 `--force` 生产构建并核对原生版本资源与 SHA256。
- 旧版本进程若未完全退出，覆盖安装会被安装器拒绝或提示占用；发布说明要求先从托盘退出。
- 用户当前运行的是修复前的旧二进制（端口 3000），需完全退出后才能覆盖更新。
- 工作区存在他人未提交的旧任务记录，提交时必须只暂存本任务文件。

---

# Implementation Plan

1. 创建本记录；确认当前版本与发布目录状态。
2. 更新 `internal/version/version.go`、`frontend/package.json`，运行
   `scripts/set-release-version.ps1 -Version v2.2.3-beta.9` 映射 Windows 资源。
3. 新增双语发行说明并审阅版本变更，仅提交本任务文件。
4. 执行 `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.9/bin`。
5. 复制安装器到 `release-assets/v2.2.3-beta.9/`，生成 SHA256 侧车，运行版本资源与发布资产校验。
6. 填写实际结果，归档任务并提交，更新 CURRENT。

---

# Progress

- 已确认当前版本 `2.2.3-beta.8`，`release-assets/v2.2.3-beta.9` 不存在，NSIS 位于 `C:\Program Files (x86)\NSIS`。
- 版本映射脚本通过：应用／前端／Windows 显示版本 `2.2.3-beta.9`，数字版本 `2.2.3.0`，
  NSIS 基础版本 `2.2.3`（`build/config.yml` 与 `wails_tools.nsh` 值未变，仅 `info.json` 的
  `ProductVersion` 改动）。
- 版本与发行说明先提交为 `9f2d825d`，再执行生产打包。
- `task --force package` exit 0，完成绑定／图标生成、前端生产构建（含新增发行说明打包）、
  Go `production` 编译和 NSIS 打包。
- Windows 原生版本资源、发布资产目录及 SHA256 校验通过；未签名，未启动、安装或公开发布。

---

# Implementation Result

## Changes

- 应用、前端和 Windows 显示版本统一为 `2.2.3-beta.9`；Windows 数字资源 `2.2.3.0`、
  NSIS 基础版本 `2.2.3` 保持不变。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.9.md`，说明 Google 订阅 Agent 链路对齐的用户可见后果
  （不再误报 `429`、原文 prompt 得到 `400 prompt_turned_away`、默认按词打散并存「按原文发送 system prompt」
  开关、6 小时同 prompt 预判跳过、容量 429 按重置时间休息并在有候选时换号、模型目录按权益过滤、
  重新授权后才有档位信息）、本地覆盖更新步骤与校验边界。该文件经前端 `import.meta.glob` 随包读取。
- 包含 TASK-20261112-002 的实现（`d4af73e9`，50 个文件）与两份文档回填提交；本次版本准备没有
  新增迁移、依赖或业务逻辑变化；绑定生成无差异（打包后 `git status` 无受跟踪文件差异）。
- 已保存完整构建日志至 `tmp/local-v2.2.3-beta.9-package.log`。

## Build and Artifacts

构建命令（当前进程 PATH 临时加入 NSIS）：

```text
task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.9/bin
```

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.2.3-beta.9/OmniProxy-Setup-v2.2.3-beta.9-windows-amd64.exe` | 20,678,269 | `64221dbddc193f2f3f4cddebc2f977b5f47f29eec1b7b78dd130933a22f9d42f` |
| `tmp/local-v2.2.3-beta.9/bin/omniproxy.exe` | 23,738,368 | `f1f4a5c4700f532ba18351f9bfda9ecfcde1e195ad18dbf3263b40fd1f728bf4` |

安装器摘要侧车与安装器同目录，后缀 `.exe.sha256`（内容为 `<hash>  <文件名>`，含结尾换行，无 BOM，
与 `v2.2.3-beta.8` 一致）。原始 NSIS 安装器位于 `bin/OmniProxy-amd64-installer.exe`。
产物目录与 `tmp/` 不入 Git。

## Technical Notes

- 版本映射仍由 `scripts/set-release-version.ps1` 完成：先手工把 `internal/version/version.go` 与
  `frontend/package.json` 改为完整 SemVer，脚本校验两者一致后再写入 `build/config.yml`、
  `build/windows/info.json` 和 `build/windows/nsis/wails_tools.nsh`。
- 本机 `pwsh` 不在 PATH，脚本需用 `powershell -NoProfile -ExecutionPolicy Bypass -File` 执行。
- 生产构建使用 `-tags production -trimpath -buildvcs=false -ldflags="-w -s -H windowsgui"`，`CGO_ENABLED=0`。
- 打包日志中的 `uname`／`tail` 缺失提示与 `-V` 提示来自 Taskfile 的 Windows 分支探测，构建仍 exit 0，
  与既有 Beta 构建一致。
- 工作区存在他人未提交的旧记录 `TASK-20261029-010-subq-provider.md`，与版本提交分开暂存，未被纳入。

---

# Verification

## Tests

- [x] Unit Tests：不重跑；见 TASK-20261112-002（全部 Go 包通过）。
- [ ] Integration Tests：N/A
- [ ] E2E Tests：N/A

## Quality Checks

- [x] Format：版本提交仅改动版本字面量、`info.json` 与发行说明。
- [x] Lint：未改业务代码，不重跑（TASK-20261112-002 已通过）。
- [x] Type Check：未改业务代码，不重跑（同上）。
- [x] Test：见上，不重跑。
- [x] Build：`task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.9/bin` exit 0。
- [x] Verify：本构建任务不运行 `task verify`；实现任务的验证结果记录在 TASK-20261112-002。

## Verification Result

```text
scripts/set-release-version.ps1 -Version v2.2.3-beta.9
已校验 v2.2.3-beta.9，Windows 数字版本 2.2.3.0

task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.9/bin
exit 0

scripts/test-windows-metadata.ps1 -Application tmp/local-v2.2.3-beta.9/bin/omniproxy.exe -Installer bin/OmniProxy-amd64-installer.exe -Version v2.2.3-beta.9
Windows 应用和安装器版本资源校验通过

scripts/verify-release-assets.ps1 -Version v2.2.3-beta.9 -AssetDirectory release-assets/v2.2.3-beta.9
发布资产校验通过：v2.2.3-beta.9
```

- 前端生产构建（含 TypeScript 编译）与 Wails 绑定生成通过（2728 个模块，`dist` 产物已打包）；
  Go 构建使用 `-tags production`、`GOOS=windows`、`GOARCH=amd64`、`CGO_ENABLED=0`、Wails `v3.1.0`。
- 应用与安装器 Authenticode 状态均为 `NotSigned`（`Get-AuthenticodeSignature`）。
- 打包后 `git status` 仅有本任务的未跟踪记录与一份既有无关文件
  （`TASK-20261029-010-subq-provider.md`），无额外受跟踪文件差异。

---

# Documentation

- [x] PRD：无需修改（本版本不含产品范围变化）
- [x] Architecture：无需修改（Google 订阅链路已在 TASK-20261112-002 更新
  `docs/product/GOOGLE_SUBSCRIPTION.md` 与 ADR-013）
- [x] Tech Stack：无需修改
- [x] Coding Rules：无需修改
- [x] Testing：无需修改
- [x] Build：无需修改
- [x] ADR：无需修改
- [x] CURRENT：已更新本地构建状态，并新增 `docs/releases/v2.2.3-beta.9.md`

---

# Deviations

- 未调用 `scripts/build-release.ps1`：该脚本会强制执行 `task --force verify`，而本次是本地上 Beta
  增量打包，且完整质量门在前端默认并发下会命中既有超时抖动。改为按仓库既有本地 Beta 流程手动执行
  版本映射、`task --force package`、原生版本资源与发布资产校验，与 `v2.2.3-beta.6`／`beta.7`／`beta.8` 一致。
- 原计划直接运行 `pwsh` 调用脚本；本机 `pwsh` 不在 PATH，改用 `powershell -NoProfile -ExecutionPolicy Bypass -File`。

---

# Git

## Commit

```text
9f2d825d chore(release): 准备本地 v2.2.3-beta.9 生产版本
```

版本提交包含 `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
与 `docs/releases/v2.2.3-beta.9.md`。

功能提交：`d4af73e9`（`feat(google): 对齐 Google 订阅 Agent 请求形态与拒绝处理`），
文档回填提交 `af053226`、`9ea7c54b`。先提交版本再打包；本记录与 CURRENT 的最终归档单独提交
（归档提交见本记录所在提交历史）。无关未跟踪文件保持原样，未纳入提交。未打标签、未推送、未公开发布。

---

# Result

`v2.2.3-beta.9` Windows amd64 本地 Beta 生产安装器已生成，包含 Google 订阅 Agent 链路对齐
（TASK-20261112-002），版本资源与摘要核对通过。安装器未签名、未自动安装、未启动；
用户可先从托盘完全退出（当前运行的仍是旧构建）并备份数据，再手动覆盖更新。安装后 Agent 客户端
是否不再收到 `429`、以及「按原文发送 system prompt」开关的效果，由用户在实机验收。

---

# Completed

2026-11-12 11:28
