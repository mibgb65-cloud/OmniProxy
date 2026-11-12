# TASK-20261112-003: 构建本地 v2.2.3-beta.8 生产安装包

## Status

Completed

## Created

2026-11-12 09:45（Asia/Singapore）

## Request

用户要求：提交代码后构建新的 Beta 生产版本，供其本地更新，并完成任务 Markdown 更新。

> 提交代码，然后构建新的beta生产版本我进行本地更新，记得完成任务md更新

## Goal

将「保存当前正在监听的网关端口不再被误判为占用」修复（TASK-20261112-001，`15842462`）
纳入 `v2.2.3-beta.8`，同步版本与双语发行说明，提交版本源码后构建 Windows amd64 生产安装包，
核对版本资源与摘要，并归档任务记录。

## Background

- `v2.2.3-beta.7` 已于 2026-11-11 本地构建（TASK-20261111-002），当前源码版本为 `2.2.3-beta.7`。
- 自 beta.7 版本提交 `07cd91ad` 起的唯一用户可见改动是端口自身占用修复：
  软件开着时保存当前端口（生产 3000）不再被自动避让成 3001，也不再重启监听器。
- 该修复的 `task verify` 中 format／lint／typecheck／Go 全部包／`task build` 通过；前端全量测试
  在默认并发下有既有超时抖动，限 4 worker 重跑 146 文件／893 项全部通过。实现提交 `15842462`。
- 仓库中另有一条未跟踪、未提交的任务记录
  `docs/tasks/active/TASK-20261112-002-google-subscription-agent-chain-parity.md` 与本次无关，
  不纳入本任务，也不参与构建。

## Scope

- 应用／前端／Windows 显示版本由 `2.2.3-beta.7` 提升为 `2.2.3-beta.8`。
- Windows 数字版本保持 `2.2.3.0`，NSIS 基础版本保持 `2.2.3`。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.8.md`（前端 `import.meta.glob` 随包读取）。
- 提交版本源码后执行生产打包，核对版本资源、SHA256 与发布目录。
- 更新任务记录与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不重跑完整质量门：本任务只做版本映射与打包；实现任务的测试与构建结果已记录在 TASK-20261112-001。
- 不推送、不打标签、不公开发布、不签名。
- 不安装、不启动或停止用户实例，不修改真实用户数据及客户端配置。

## Existing Behavior

端口自身占用修复已提交为 `15842462`，任务归档提交为 `940fa1c3`；版本为 `2.2.3-beta.7`，
`release-assets/v2.2.3-beta.8` 不存在。

## Expected Behavior

新安装器包含端口自身占用修复，用户从托盘完全退出后可本地覆盖更新；版本资源与摘要与实际产物一致。

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
- 工作区存在他人未提交的任务记录，提交时必须只暂存本任务文件。

---

# Implementation Plan

1. 创建本记录；确认当前版本与发布目录状态。
2. 更新 `internal/version/version.go`、`frontend/package.json`，运行
   `scripts/set-release-version.ps1 -Version v2.2.3-beta.8` 映射 Windows 资源。
3. 新增双语发行说明并审阅版本变更，仅提交本任务文件。
4. 执行 `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.8/bin`。
5. 复制安装器到 `release-assets/v2.2.3-beta.8/`，生成 SHA256 侧车，运行版本资源与发布资产校验。
6. 填写实际结果，归档任务并提交，更新 CURRENT。

---

# Progress

- 已确认当前版本 `2.2.3-beta.7`，`release-assets/v2.2.3-beta.8` 不存在，NSIS 位于 `C:\Program Files (x86)\NSIS`。
- 版本映射脚本通过：应用／前端／Windows 显示版本 `2.2.3-beta.8`，数字版本 `2.2.3.0`，
  NSIS 基础版本 `2.2.3`（`build/config.yml` 与 `wails_tools.nsh` 值未变，仅 `info.json` 的
  `ProductVersion` 改动）。
- 版本与发行说明先提交为 `dc114884`，再执行生产打包。
- `task --force package` exit 0，完成绑定／图标生成、前端生产构建（含新增发行说明打包）、
  Go `production` 编译和 NSIS 打包。
- Windows 原生版本资源、发布资产目录及 SHA256 校验通过；未签名，未启动、安装或公开发布。

---

# Implementation Result

## Changes

- 应用、前端和 Windows 显示版本统一为 `2.2.3-beta.8`；Windows 数字资源 `2.2.3.0`、
  NSIS 基础版本 `2.2.3` 保持不变。
- 新增双语发行说明 `docs/releases/v2.2.3-beta.8.md`，说明端口自身占用修复的三种表现
  （保存当前端口不再避让、其它端口与真实占用仍照常避让、「恢复默认端口」不重启监听器）、
  本地覆盖更新步骤、版本映射与校验边界。该文件经前端 `import.meta.glob` 随包读取。
- 包含端口自身占用修复（TASK-20261112-001，`15842462`）；没有新增迁移、依赖或业务逻辑变化；
  绑定生成无差异（打包后 `git status` 无受跟踪文件差异）。
- 已保存完整构建日志至 `tmp/local-v2.2.3-beta.8-package.log`。

## Build and Artifacts

构建命令（当前进程 PATH 临时加入 NSIS）：

```text
task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.8/bin
```

| 产物 | 大小（bytes） | SHA256 |
| --- | ---: | --- |
| `release-assets/v2.2.3-beta.8/OmniProxy-Setup-v2.2.3-beta.8-windows-amd64.exe` | 20,611,512 | `cc36c202c52770ecc46a1b0e0936bba8509d6e171c6e5920aeabaf663763002e` |
| `tmp/local-v2.2.3-beta.8/bin/omniproxy.exe` | 23,664,128 | `108958faa1a142055191bbf7a17ed660eeea3aa40d51b26a5158d27c52446dc4` |

安装器摘要侧车与安装器同目录，后缀 `.exe.sha256`（内容为 `<hash>  <文件名>`，含结尾换行，
与 `v2.2.3-beta.7` 一致）。原始 NSIS 安装器位于 `bin/OmniProxy-amd64-installer.exe`。
产物目录与 `tmp/` 不入 Git。

## Technical Notes

- 版本映射仍由 `scripts/set-release-version.ps1` 完成：先手工把 `internal/version/version.go` 与
  `frontend/package.json` 改为完整 SemVer，脚本校验两者一致后再写入 `build/config.yml`、
  `build/windows/info.json` 和 `build/windows/nsis/wails_tools.nsh`。
- 本机 `pwsh` 不在 PATH，脚本需用 `powershell -NoProfile -ExecutionPolicy Bypass -File` 执行。
- 生产构建使用 `-tags production -trimpath -buildvcs=false -ldflags="-w -s -H windowsgui"`，`CGO_ENABLED=0`。
- 发包前工作区存在他人未提交的任务记录 `TASK-20261112-002-…`，与版本提交分开暂存，未被纳入。

---

# Verification

## Tests

- [x] Unit Tests：不重跑；见 TASK-20261112-001（3 项端口回归测试、`internal/transport/wails` 全量通过）。
- [ ] Integration Tests：N/A
- [ ] E2E Tests：N/A

## Quality Checks

- [x] Format：版本提交仅改动版本字面量、`info.json` 与发行说明。
- [x] Lint：未改业务代码，不重跑（TASK-20261112-001 已通过）。
- [x] Type Check：未改业务代码，不重跑（同上）。
- [x] Test：见上，不重跑。
- [x] Build：`task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.8/bin` exit 0。
- [x] Verify：本构建任务不运行 `task verify`；实现任务的验证结果记录在 TASK-20261112-001。

## Verification Result

```text
scripts/set-release-version.ps1 -Version v2.2.3-beta.8
已校验 v2.2.3-beta.8，Windows 数字版本 2.2.3.0

task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.8/bin
exit 0

scripts/test-windows-metadata.ps1 -Application tmp/local-v2.2.3-beta.8/bin/omniproxy.exe -Installer bin/OmniProxy-amd64-installer.exe -Version v2.2.3-beta.8
Windows 应用和安装器版本资源校验通过

scripts/verify-release-assets.ps1 -Version v2.2.3-beta.8 -AssetDirectory release-assets/v2.2.3-beta.8
发布资产校验通过：v2.2.3-beta.8
```

- 前端生产构建（含 TypeScript 编译）与 Wails 绑定生成通过；Go 构建使用 `-tags production`、
  `GOOS=windows`、`GOARCH=amd64`、`CGO_ENABLED=0`、Wails `v3.1.0`。
- 应用与安装器 Authenticode 状态均为 `NotSigned`。
- 打包后 `git status` 仅有本任务的未跟踪记录与两份既有无关文件
  （`TASK-20261029-010-subq-provider.md`、`TASK-20261112-002-…`），无额外受跟踪文件差异。

---

# Documentation

- [x] PRD：无需修改（本版本不含产品范围变化）
- [x] Architecture：无需修改（端口判定说明已在 TASK-20261112-001 更新 `GATEWAY.md`）
- [x] Tech Stack：无需修改
- [x] Coding Rules：无需修改
- [x] Testing：无需修改
- [x] Build：无需修改
- [x] ADR：无需修改
- [x] CURRENT：已更新本地构建状态，并新增 `docs/releases/v2.2.3-beta.8.md`

---

# Deviations

- 未调用 `scripts/build-release.ps1`：该脚本会强制执行 `task --force verify`，而本次是本地上 Beta
  增量打包，且完整质量门在前端默认并发下会命中既有超时抖动。改为按仓库既有本地 Beta 流程手动执行
  版本映射、`task --force package`、原生版本资源与发布资产校验，与 `v2.2.3-beta.6`／`beta.7` 一致。
- 原计划直接运行 `pwsh` 调用脚本；本机 `pwsh` 不在 PATH，改用 `powershell -NoProfile -ExecutionPolicy Bypass -File`。

---

# Git

## Commit

```text
dc114884 chore(release): 准备本地 v2.2.3-beta.8 生产版本
```

版本提交包含 `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
与 `docs/releases/v2.2.3-beta.8.md`。

端口修复提交：`15842462`（`fix(gateway): 保存当前正在监听的端口不再被误判为占用`），
任务归档提交 `940fa1c3`。先提交版本再打包；本记录与 CURRENT 的最终归档单独提交。
无关未跟踪文件保持原样，未纳入提交。未打标签、未推送、未公开发布。

---

# Result

`v2.2.3-beta.8` Windows amd64 本地 Beta 生产安装器已生成，包含「保存当前端口不再被误判为占用」
修复，版本资源与摘要核对通过。安装器未签名、未自动安装、未启动；用户可先从托盘完全退出并备份数据，
再手动覆盖更新。安装后保存当前端口是否保持 3000、以及网关是否不重启，由用户在实机验收。

---

# Completed

2026-11-12 10:05
