# TASK-20261110-004: 构建本地 v2.2.3-beta.6 生产安装包

## Status

Completed

## Created

2026-11-10 09:45（Asia/Singapore）

## Request

用户要求提交代码后，构建新的 Beta 生产版本供其本地更新，并完成任务 Markdown 更新。

## Goal

将已完成的界面调整（TASK-20261110-003，`d383ccc7`）纳入 `v2.2.3-beta.6`，同步版本与双语发行说明，提交后构建 Windows amd64 生产安装包并记录实际产物。

## Background

- `v2.2.3-beta.5` 已于 2026-11-10 本地构建（TASK-20261110-002），构建前源码版本为 `2.2.3-beta.5`。
- 本版本包含：移除仪表盘／用量页／请求页的手动刷新按钮；日志页脚「回到最新」按钮与「自动跟随最新」提示共用固定 32px 行高，消除页面跳动。
- 该改动的全量前端测试（146 文件／893 项，限 4 worker）、Go 全部包测试与生产构建已通过，见 TASK-20261110-003。

## Scope

- 应用／前端／Windows 显示版本由 `2.2.3-beta.5` 提升为 `2.2.3-beta.6`。
- Windows 数字版本保持 `2.2.3.0`，NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明，提交版本源码后执行生产打包。
- 核对生产产物的版本资源、SHA256 和发布目录，更新任务记录及 CURRENT。

## Out of Scope

- 不重跑完整质量门：本任务只做发布版本映射与打包；实现任务的全量前端测试、Go 测试和生产构建已通过。
- 不推送、不打标签、不公开发布、不签名。
- 不安装、不启动或停止用户实例，不修改真实用户数据及客户端配置。

## Existing Behavior

界面调整已提交为 `d383ccc7`，任务归档提交为 `45c75ccf`；版本为 `2.2.3-beta.5`，`release-assets/v2.2.3-beta.6` 不存在。

## Expected Behavior

新安装器包含上述界面调整，用户可从托盘完全退出后本地覆盖更新；版本资源与摘要与实际产物一致。

# Analysis

## Related Modules

- `internal/version/version.go`
- `frontend/package.json`
- `build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- `docs/releases/`、`docs/plans/CURRENT.md`

## Dependencies

沿用 Wails v3.1.0、Task 3.53.1、Go、pnpm、NSIS 及现有版本映射／产物校验脚本。

## Risks

- 本地安装器未签名，可能触发未知发布者或 SmartScreen 提示。
- 版本与资源映射不一致或复用旧产物会导致错误交付；使用 `--force` 生产构建并核对原生版本资源与 SHA256。

# Implementation Plan

1. 创建本记录；确认当前版本与发布目录状态。
2. 更新源码版本、发行说明与 CURRENT，运行 `scripts/set-release-version.ps1` 映射 Windows 资源。
3. 审阅并提交版本变更，排除无关文件。
4. 执行 `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.6/bin`。
5. 核对产物版本、大小和摘要，归档任务并提交最终结果。

# Progress

- 已确认当前版本 `2.2.3-beta.5`、`release-assets/v2.2.3-beta.6` 不存在，NSIS 位于 `C:\Program Files (x86)\NSIS`。
- 版本映射脚本通过：应用／前端／Windows 显示版本 `2.2.3-beta.6`，数字版本 `2.2.3.0`，NSIS 基础版本 `2.2.3`。
- 版本与发行说明先提交为 `375e5333`，再执行生产打包。
- `task --force package` exit 0，完成绑定／图标生成、前端生产构建、Go `production` 编译和 NSIS 打包。
- Windows 原生版本资源、发布资产目录及 SHA256 校验通过；未签名，未启动、安装或公开发布。

# Implementation Result

## Changes

- 应用、前端和 Windows 显示版本统一为 `2.2.3-beta.6`；Windows 数字资源 `2.2.3.0`、NSIS 基础版本 `2.2.3` 保持不变。
- 新增双语发行说明，说明三页手动刷新按钮移除、日志页脚固定行高、自动更新机制、本地覆盖更新及质量门边界。
- 包含界面修复提交 `d383ccc7`；没有新增迁移、依赖或业务逻辑变化；绑定生成无差异。
- 已保存完整构建日志至 `tmp/local-v2.2.3-beta.6-package.log`。

## Build and Artifacts

构建命令（当前进程 PATH 临时加入 NSIS）：

```text
task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.6/bin
```

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.2.3-beta.6/OmniProxy-Setup-v2.2.3-beta.6-windows-amd64.exe` | 20,599,143 | `e691d8caef25d49ccc542663dc496f14d60201ba05f2fe6139bd4399cc373ca5` |
| `tmp/local-v2.2.3-beta.6/bin/omniproxy.exe` | 23,644,160 | `df6e67a3e5a486fea3eadfcc369b84615607a6633a6d768adc5ce20b0289b4b7` |

安装器摘要侧车与安装器同目录，后缀 `.exe.sha256`。原始 NSIS 安装器位于 `bin/OmniProxy-amd64-installer.exe`。产物目录与 `tmp/` 不入 Git。

## Technical Notes

- 版本映射仍由 `scripts/set-release-version.ps1` 完成：`internal/version/version.go` 与 `frontend/package.json` 先改为完整 SemVer，脚本再写入 `build/config.yml`、`build/windows/info.json` 和 `build/windows/nsis/wails_tools.nsh`。
- 生产构建使用 `-tags production -trimpath -buildvcs=false -ldflags="-w -s -H windowsgui"`，`CGO_ENABLED=0`。

# Verification

## Tests

- [x] Unit Tests：不重跑；见 TASK-20261110-003 的全量前端 146 文件／893 项（限 4 worker）与 Go 全部包测试。
- [ ] Integration Tests：N/A
- [ ] E2E Tests：N/A

## Quality Checks

- [x] Format：版本提交仅改动版本字面量与文档。
- [x] Lint：未改业务代码，不重跑（实现任务已通过）。
- [x] Type Check：未改业务代码，不重跑（实现任务已通过）。
- [x] Test：见上，不重跑。
- [x] Build：`task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.6/bin` exit 0。
- [x] Verify：本构建任务不运行 `task verify`；实现任务的 `task verify` 结果与限制记录在 TASK-20261110-003。

## Verification Result

```text
scripts/set-release-version.ps1 -Version v2.2.3-beta.6
已校验 v2.2.3-beta.6，Windows 数字版本 2.2.3.0

task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.6/bin
exit 0

scripts/test-windows-metadata.ps1 -Application tmp/.../omniproxy.exe -Installer bin/OmniProxy-amd64-installer.exe -Version v2.2.3-beta.6
Windows 应用和安装器版本资源校验通过

scripts/verify-release-assets.ps1 -Version v2.2.3-beta.6 -AssetDirectory release-assets/v2.2.3-beta.6
发布资产校验通过：v2.2.3-beta.6
```

- 前端生产构建（含 TypeScript 编译）通过；Go 构建信息为 `-tags=production`、`GOOS=windows`、`GOARCH=amd64`、`CGO_ENABLED=0`、Wails `v3.1.0`。
- 应用与安装器 Authenticode 状态均为 `NotSigned`。
- 打包后 `git status` 仅有本次未跟踪的预置任务文件，无额外受跟踪文件差异。

# Documentation

- 新增 `docs/releases/v2.2.3-beta.6.md`。
- 更新 `docs/plans/CURRENT.md` 的本地构建状态。
- 本记录已填写实际构建结果并归档。

# Deviations

- 未调用 `scripts/build-release.ps1`：该脚本会强制执行 `task --force verify`，而本次是本地上 Beta 增量打包，实现任务的全量前端与 Go 测试已通过且默认并发下存在无关既有测试的 5 秒超时。改为按仓库已有本地 Beta 流程手动执行版本映射、`task --force package`、原生版本资源与发布资产校验，与 `v2.2.3-beta.5` 的做法一致。

# Git

版本提交：`375e5333`（`chore(release): 准备本地 v2.2.3-beta.6 生产版本`）。

界面修复提交：`d383ccc7`（`fix(ui): 移除手动刷新按钮并稳定日志页脚高度`）。

先提交版本再打包；本记录与 CURRENT 的最终归档单独提交。无关未跟踪的 `TASK-20261029-010-subq-provider.md` 保持原样，未纳入本次提交。未打标签、未推送、未公开发布。

# Result

`v2.2.3-beta.6` Windows amd64 本地 Beta 生产安装器已生成，包含手动刷新按钮移除与日志页脚固定行高，版本资源与摘要核对通过。安装器未签名、未自动安装、未启动；用户可先从托盘完全退出并备份数据，再手动覆盖更新。安装后界面观感由用户本地验收。

---

# Completed

2026-11-10 09:43
