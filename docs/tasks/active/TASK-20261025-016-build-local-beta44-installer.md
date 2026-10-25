# TASK-20261025-016: 构建 Beta.44 本地生产安装包

## Status

In Progress

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
- TASK-20261025-015 已归档到 `docs/tasks/completed/2026-10/`。
