# TASK-20261111-002: 构建本地 v2.2.3-beta.7 生产安装包

## Status

Completed

## Created

2026-11-11 22:35（Asia/Singapore）

## Request

用户要求：提交代码后构建新的 Beta 生产版本，供其本地更新，并完成任务 Markdown 更新。

## Goal

将单实例与窗口唤起修复（TASK-20261111-001，`5d6fef11`）纳入 `v2.2.3-beta.7`，同步版本与双语发行说明，提交版本源码后构建 Windows amd64 生产安装包，核对版本资源与摘要，并归档任务记录。

## Background

- `v2.2.3-beta.6` 已于 2026-11-10 本地构建（TASK-20261110-004），当前源码版本为 `2.2.3-beta.6`。
- 本版本包含：同一 profile 只允许一个 OmniProxy 进程；重复启动在读取配置、打开数据库、执行迁移与初始化日志之前被命名互斥量拦下，只把已有窗口带到前台并以 0 退出。
- 该改动的 `task verify` exit 0（53 个 Go 测试包、前端 146 文件／893 项、Windows production 编译），实现提交 `5d6fef11`，任务归档提交 `fa81c4ef`，见 TASK-20261111-001。

## Scope

- 应用／前端／Windows 显示版本由 `2.2.3-beta.6` 提升为 `2.2.3-beta.7`。
- Windows 数字版本保持 `2.2.3.0`，NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明，提交版本源码后执行生产打包。
- 核对生产产物的版本资源、SHA256 与发布目录，更新任务记录及 CURRENT。

## Out of Scope

- 不重跑完整质量门：本任务只做版本映射与打包；实现任务的 `task verify` 已 exit 0。
- 不推送、不打标签、不公开发布、不签名。
- 不安装、不启动或停止用户实例，不修改真实用户数据及客户端配置。

## Existing Behavior

单实例修复已提交为 `5d6fef11`，任务归档提交为 `fa81c4ef`；版本为 `2.2.3-beta.6`，`release-assets/v2.2.3-beta.7` 不存在。

## Expected Behavior

新安装器包含单实例与窗口唤起修复，用户从托盘完全退出后可本地覆盖更新；版本资源与摘要与实际产物一致。

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

---

# Implementation Plan

1. 创建本记录；确认当前版本与发布目录状态。
2. 更新源码版本、发行说明与 CURRENT，运行 `scripts/set-release-version.ps1` 映射 Windows 资源。
3. 审阅并提交版本变更，排除无关文件。
4. 执行 `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.7/bin`。
5. 复制安装器到 `release-assets/v2.2.3-beta.7/`，生成 SHA256 侧车，运行版本资源与发布资产校验。
6. 填写实际结果，归档任务并提交。

---

# Progress

- 已确认当前版本 `2.2.3-beta.6`，`release-assets/v2.2.3-beta.7` 不存在，NSIS 位于 `C:\Program Files (x86)\NSIS`。
- 版本映射脚本通过：应用／前端／Windows 显示版本 `2.2.3-beta.7`，数字版本 `2.2.3.0`，NSIS 基础版本 `2.2.3`。
- 版本与发行说明先提交为 `07cd91ad`，再执行生产打包。
- `task --force package` exit 0，完成绑定／图标生成、前端生产构建、Go `production` 编译和 NSIS 打包。
- Windows 原生版本资源、发布资产目录及 SHA256 校验通过；未签名，未启动、安装或公开发布。

---

# Implementation Result

## Changes

- 应用、前端和 Windows 显示版本统一为 `2.2.3-beta.7`；Windows 数字资源 `2.2.3.0`、NSIS 基础版本 `2.2.3` 保持不变。
- 新增双语发行说明，说明单实例行为、重复启动的窗口唤起、profile 隔离、自动更新机制、本地覆盖更新与质量门边界。
- 包含单实例修复提交 `5d6fef11`（任务 TASK-20261111-001）；没有新增迁移、依赖或业务逻辑变化；绑定生成无差异。
- 已保存完整构建日志至 `tmp/local-v2.2.3-beta.7-package.log`。

## Build and Artifacts

构建命令（当前进程 PATH 临时加入 NSIS）：

```text
task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.7/bin
```

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.2.3-beta.7/OmniProxy-Setup-v2.2.3-beta.7-windows-amd64.exe` | 20,608,748 | `6b1260117d1179973b23ccb2702a8981e7d924ca090997ef59b23c41119b2da4` |
| `tmp/local-v2.2.3-beta.7/bin/omniproxy.exe` | 23,659,008 | `09dd238f9b8e1d9aee66390ef192124b3439a931ee4f41138f9b03a3a44feeb9` |

安装器摘要侧车与安装器同目录，后缀 `.exe.sha256`。原始 NSIS 安装器位于 `bin/OmniProxy-amd64-installer.exe`。产物目录与 `tmp/` 不入 Git。

## Technical Notes

- 版本映射仍由 `scripts/set-release-version.ps1` 完成：`internal/version/version.go` 与 `frontend/package.json` 先改为完整 SemVer，脚本再写入 `build/config.yml`、`build/windows/info.json` 和 `build/windows/nsis/wails_tools.nsh`。本次 `build/config.yml` 与 NSIS 基础版本已是 `2.2.3`，仅 `info.json` 的 `ProductVersion` 变化。
- 生产构建使用 `-tags production -trimpath -buildvcs=false -ldflags="-w -s -H windowsgui"`，`CGO_ENABLED=0`。

---

# Verification

## Tests

- [x] Unit Tests：不重跑；见 TASK-20261111-001 的 `task verify`（53 个 Go 测试包、前端 146 文件／893 项、Windows production 编译，exit 0）。
- [ ] Integration Tests：N/A
- [ ] E2E Tests：N/A

## Quality Checks

- [x] Format：版本提交仅改动版本字面量、`info.json` 与发行说明。
- [x] Lint：未改业务代码，不重跑（实现任务 `task verify` 已通过）。
- [x] Type Check：未改业务代码，不重跑（实现任务 `task verify` 已通过）。
- [x] Test：见上，不重跑。
- [x] Build：`task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.7/bin` exit 0。
- [x] Verify：本构建任务不运行 `task verify`；实现任务的 `task verify` 结果记录在 TASK-20261111-001。

## Verification Result

```text
scripts/set-release-version.ps1 -Version v2.2.3-beta.7
已校验 v2.2.3-beta.7，Windows 数字版本 2.2.3.0

task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.7/bin
exit 0

scripts/test-windows-metadata.ps1 -Application tmp/local-v2.2.3-beta.7/bin/omniproxy.exe -Installer bin/OmniProxy-amd64-installer.exe -Version v2.2.3-beta.7
Windows 应用和安装器版本资源校验通过

scripts/verify-release-assets.ps1 -Version v2.2.3-beta.7 -AssetDirectory release-assets/v2.2.3-beta.7
发布资产校验通过：v2.2.3-beta.7
```

- 前端生产构建（含 TypeScript 编译）与 Wails 绑定生成通过；Go 构建使用 `-tags production`、`GOOS=windows`、`GOARCH=amd64`、`CGO_ENABLED=0`、Wails `v3.1.0`。
- 应用与安装器 Authenticode 状态均为 `NotSigned`。
- 打包后 `git status` 仅有本任务的未跟踪记录与既有无关文件 `TASK-20261029-010-subq-provider.md`，无额外受跟踪文件差异。

---

# Documentation

- [x] PRD：无需修改（本版本不含产品范围变化）
- [x] Architecture：无需修改（单实例架构说明已在 TASK-20261111-001 更新）
- [x] Tech Stack：无需修改
- [x] Coding Rules：无需修改
- [x] Testing：无需修改
- [x] Build：无需修改
- [x] ADR：无需修改
- [x] CURRENT：已更新本地构建状态，并新增 `docs/releases/v2.2.3-beta.7.md`

---

# Deviations

- 未调用 `scripts/build-release.ps1`：该脚本会强制执行 `task --force verify`，而本次是本地上 Beta 增量打包，实现任务的 `task verify` 已 exit 0。改为按仓库既有本地 Beta 流程手动执行版本映射、`task --force package`、原生版本资源与发布资产校验，与 `v2.2.3-beta.6` 的做法一致。

---

# Git

## Commit

```text
07cd91ad chore(release): 准备本地 v2.2.3-beta.7 生产版本
```

版本提交包含 `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json` 与 `docs/releases/v2.2.3-beta.7.md`。

单实例修复提交：`5d6fef11`（`fix(app): 单实例运行并在重复启动时唤起已有窗口`），任务归档提交 `fa81c4ef`。

先提交版本再打包；本记录与 CURRENT 的最终归档单独提交。无关未跟踪的 `TASK-20261029-010-subq-provider.md` 保持原样，未纳入本次提交。未打标签、未推送、未公开发布。

---

# Result

`v2.2.3-beta.7` Windows amd64 本地 Beta 生产安装器已生成，包含单实例守卫与重复启动窗口唤起，版本资源与摘要核对通过。安装器未签名、未自动安装、未启动；用户可先从托盘完全退出并备份数据，再手动覆盖更新。安装后「再次启动只唤起已有窗口」的实际观感由用户在实机验收。

---

# Completed

2026-11-11 22:48
