# TASK-20261025-016: 构建 Beta.44 本地生产安装包

## Status

Completed

## Created

2026-10-25 22:30（Asia/Shanghai）

## Request

> 代码提交后，构建新的beta生产版本我进行本地更新

## Goal

提交本次改动后产出 `release-assets/v2.0.0-beta.44/` 安装器与 SHA256，通过完整质量门、
Windows 版本资源与发布资产校验，供用户手动覆盖更新。

## Background

相对 Beta.43（安装器 20,194,569 bytes，SHA256 `968ca082…7a8`）新增一个已完成任务：

- [TASK-20261025-015](../completed/2026-10/TASK-20261025-015-network-proxy-dialog-layout.md)：
  网络代理弹窗布局重做——560px 单栏整块滚动改为 880px 两栏分区（左「代理服务器」、
  右「使用代理的提供商」）、固定头部与页脚、提供商列表独立滚动 + 名称筛选 + 三态计数摘要。

纯前端改动，无 Go 逻辑、无数据库迁移、无新增依赖、无绑定变化。

## Scope

- 提交 Beta.44 之前的全部改动（弹窗组件、测试、i18n、文档、任务记录归档）。
- 版本同步至 `2.0.0-beta.44`（`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json` 的 `ProductVersion`；Windows 数字版本仍 `2.0.0.0`）。
- 新增 `docs/releases/v2.0.0-beta.44.md`。
- 完整质量门 → 打包 → Windows 版本资源 → 发布资产与 SHA256 校验。
- 更新 `docs/plans/CURRENT.md`。

## Out of Scope

- 不自动安装、不打标签、不推送、不代码签名。
- 不停止或替换用户正在运行的生产程序；安装由用户手动覆盖执行。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `scripts/set-release-version.ps1`、`scripts/build-release.ps1`
- `scripts/test-windows-metadata.ps1`、`scripts/verify-release-assets.ps1`
- `docs/releases/`

## Dependencies

- Go 1.26、Node 22、pnpm、Task、Wails beta.30、staticcheck、NSIS（`C:\Program Files (x86)\NSIS`）。

## Risks

- `makensis` 不在 PATH：`build-release.ps1` 会自行补 `C:\Program Files (x86)\NSIS`。
- `set-release-version.ps1` 会用 `ConvertTo-Json` 重写 `build/windows/info.json`；以它为准，
  与 Beta.41 起的做法一致。
- 安装器未签名，覆盖安装时 SmartScreen 可能提示未知发布者。
- 本版无迁移，回退到 Beta.43 安全；弹窗的新布局回退后回到旧单栏界面。

---

# Implementation Plan

1. 提交功能改动与任务记录归档。
2. 版本提升到 `2.0.0-beta.44`（`version.go` + `package.json` 手工改，其余交给脚本同步）。
3. 新增 `docs/releases/v2.0.0-beta.44.md`，更新 `docs/plans/CURRENT.md`。
4. 提交版本与文档。
5. `scripts/build-release.ps1 -Version v2.0.0-beta.44 -OutputDirectory release-assets/v2.0.0-beta.44`
   （内部执行 `task --force verify`、`task --force package`、元数据校验、SHA256 与资产校验）。
6. 回填本记录、提交并归档。

---

# Progress

- UI 改动提交 `3d831c8`（`feat(frontend): 网络代理弹窗改为两栏分区与固定页脚`）。
- TASK-20261025-015 已归档到 `docs/tasks/completed/2026-10/`（`574434a`）。
- 版本与发布说明提交 `e68b510`（`chore(release): 2.0.0-beta.44 版本与发布说明`）。
- `task --force verify` exit 0（`tmp/verify-beta44.log`，132 文件／809 项前端测试）。
- `task --force package` exit 0，安装器 20,198,610 bytes。
- Windows 版本资源、发布资产与 SHA256 校验通过。

---

# Implementation Result

- 版本同步至 `2.0.0-beta.44`：`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json` 的 `ProductVersion`（`scripts/set-release-version.ps1` 同步，
  Windows 数字版本仍 `2.0.0.0`，`build/config.yml` 与 `wails_tools.nsh` 未发生变化）。
- 新增发布说明 `docs/releases/v2.0.0-beta.44.md`（弹窗两栏布局、固定页脚、筛选与计数摘要、
  行为不变说明、无迁移可回退、已知边界）。
- `release-assets/v2.0.0-beta.44/`：
  - `OmniProxy-Setup-v2.0.0-beta.44-windows-amd64.exe` 20,198,610 bytes
  - SHA256 `913b5050903a36ee9700d99864d347b8bd86aaadeaf45b6b3404b9f5a75f6884`
- 相对 Beta.43 的 20,194,569 bytes 增加 4,041 bytes（弹窗组件重写、3 个新测试、5 个 i18n 键；
  无 Go 逻辑、无迁移、无新增依赖）。
- `bin/omniproxy.exe` 23,066,112 bytes；安装器 Total size 20,198,610／80,415,156 bytes（25.1%）。

# Verification

## Tests

- [x] Unit Tests（Go 全量）
- [x] Integration Tests（既有适配器／网关／透明转发用例）
- [ ] E2E Tests（无独立 E2E 套件）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task --force verify   exit 0（tmp/verify-beta44.log）
  format / vet / staticcheck / ESLint / tsc / go test / vitest / production build 全部通过
  前端 132 文件 / 809 项测试
task --force package  exit 0（`scripts/build-release.ps1` 内执行，NSIS 输出 Total size 20,198,610 / 80,415,156 bytes，25.1%）
scripts/test-windows-metadata.ps1   PASS「Windows 应用和安装器版本资源校验通过」
scripts/verify-release-assets.ps1   PASS「发布资产校验通过：v2.0.0-beta.44」
```

- 未执行：代码签名、自动安装、停止或替换生产进程、打标签、推送。

## Remaining Risks

- 安装器未签名，Windows SmartScreen 仍可能提示未知发布者。
- 弹窗两栏布局只在浏览器预览（深色／浅色／760px 窄屏）核对过，需安装后在本机原生窗口再确认一次。
- 本版无迁移，回退到 Beta.43 安全。

---

# Documentation

- [x] Release notes（`docs/releases/v2.0.0-beta.44.md`）
- [x] CURRENT（Current Version 与 In Progress）

---

# Deviations

- `scripts/build-release.ps1` 修两处（本任务范围内，属发布工具）：

  1. `task --force verify/package` 会把进度行写到 stderr，其中包含 Windows 上
     `uname` 不存在的提示；在 `$ErrorActionPreference='Stop'` 下 PowerShell 把原生 stderr
     当成终止错误，脚本在质量门还没开跑就退出（日志只有一行「已校验 …」）。
     改为在两次调用前后临时降为 `Continue`，并先取 `$LASTEXITCODE` 再判失败。
  2. `[IO.Path]::GetFullPath($path,$base)` 重载在 Windows PowerShell 5.1 不存在
     （MethodCountCouldNotFindBest），改为先 `Join-Path` 再单参数 `GetFullPath`。

- 由于 1 的失败，实际执行分两次：脚本跑到打包与元数据校验成功后在 2 处中断，随后用与脚本
  完全相同的步骤（建空目录 → 复制 → SHA256 → `verify-release-assets.ps1`）完成收尾。

---

# Git

## Commit

```text
3d831c8 feat(frontend): 网络代理弹窗改为两栏分区与固定页脚
574434a docs(task): 记录 TASK-20261025-015 并归档
e68b510 chore(release): 2.0.0-beta.44 版本与发布说明
```

---

# Result

Beta.44 本地生产安装包已产出并通过质量门、Windows 版本资源、SHA256 与资产校验。用户运行
`release-assets/v2.0.0-beta.44/OmniProxy-Setup-v2.0.0-beta.44-windows-amd64.exe` 覆盖更新，
首次启动无迁移；网络代理弹窗即为两栏新布局。

---

# Completed


2026-10-25 22:50（Asia/Shanghai）
