# TASK-20261119-007: 构建 v2.4.1-beta.1 本地安装包

## Status

Completed

## Created

2026-11-19 17:18

## Request

用户在 TASK-20261119-006（可选记录请求正文并导出请求记录 XLSX）完成后要求：提交代码，构建新的 Beta 生产版本供本地更新，并更新任务记录。

## Goal

产出 `v2.4.1-beta.1` Windows amd64 生产安装包（未签名、本地使用），供本机覆盖更新后立即使用：

- TASK-20261119-006：默认关闭的请求正文记录、详情回看与请求记录 XLSX 导出（`3b412e7`）。

## Background

功能与任务记录已提交（`3b412e7` 功能、`e5e550f7` 记录）。版本基线为已发布的 `2.4.0`；`docs/releases/v2.4.1-beta.1.md` 与 `release-assets/v2.4.1-beta.1` 均不存在。

工作区只剩既有的未跟踪文件（`.gotmp/` 本地缓存与 `docs/tasks/completed/2026-10/TASK-20261029-010-subq-provider.md`），不属于本任务。

## Scope

- `docs/releases/v2.4.1-beta.1.md` 版本说明（中英）；
- 版本号 → `2.4.1-beta.1`（`internal/version/version.go`、`frontend/package.json`）；
- 运行 `scripts/build-release.ps1 -Version v2.4.1-beta.1 -OutputDirectory release-assets\v2.4.1-beta.1`；
  脚本同步 `build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`；
- 记录产物大小、SHA256、Windows 数字版本与 NSIS 基础版本。

## Out of Scope

- 打标签、推送、公开 Release、代码签名、自动安装或启动应用；
- 真实账号或真实模型请求（由用户安装后自行使用）；
- 新的产品功能。

## Existing Behavior

本机运行 `v2.4.0`：请求明细只保存模型、Token 等元数据，不保存正文，没有请求记录导出。

## Expected Behavior

安装后：

- 「设置 → 请求与日志」出现「记录请求正文」开关，默认关闭；开启后新请求保存客户端正文（每条最多 256 KiB）。
- 请求详情显示「请求内容」区块；未记录时明确说明。
- 请求页可按当前筛选导出 XLSX，包含模型与请求正文列。
- 首次启动执行迁移 053（纯追加），既有账号、路由与统计保留。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`、`build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- `docs/releases/v2.4.1-beta.1.md`
- `scripts/build-release.ps1`、`scripts/set-release-version.ps1`、`scripts/test-windows-metadata.ps1`、`scripts/verify-release-assets.ps1`

## Dependencies

- NSIS、`task`、`wails3`、Go 1.26、pnpm —— 上一轮构建已验证可用。

## Risks

- 构建脚本内部运行 `task --force verify` 与 `task --force package`；构建前后核对 `git status`。
- `release-assets/v2.4.1-beta.1` 必须不存在或为空，脚本拒绝覆盖。
- 版本资源文件由脚本改写，构建后确认没有意外换行或非本任务改动。

---

# Implementation Plan

1. 写版本说明，修改源码与前端版本号，提交 `chore(release)`。
2. 运行 `scripts/build-release.ps1`，等待完整质量门与打包。
3. 校验安装器大小、SHA256、Windows 版本资源与发布资产。
4. 回填任务记录、移动到 completed、更新 CURRENT，提交文档。

---

# Progress

- 2026-11-19 17:18 创建记录，写版本说明并同步源码／前端／Windows 版本资源。
- 2026-11-19 17:19 `set-release-version.ps1` 通过；提交 `b338fca1`（准备构建）。
- 2026-11-19 17:26 `scripts/build-release.ps1` EXIT=0：完整质量门、NSIS 打包、版本资源与发布资产校验全部通过。

---

# Implementation Result

## Changes

- 新增 `docs/releases/v2.4.1-beta.1.md`（中英版本说明：请求正文记录、详情回看、XLSX 导出、隐私与边界、迁移 053、本地更新与验证边界）。
- 版本号：`internal/version/version.go`、`frontend/package.json` → `2.4.1-beta.1`；`build/config.yml` → `2.4.1`；`build/windows/info.json` 的 `fixed` 为 `2.4.1.0`、`ProductVersion` 为 `2.4.1-beta.1`；`build/windows/nsis/wails_tools.nsh` 的 `INFO_PRODUCTVERSION` 为 `2.4.1`。
- 安装包：`release-assets/v2.4.1-beta.1/OmniProxy-Setup-v2.4.1-beta.1-windows-amd64.exe`
  （21,533,202 bytes，SHA256 `ea8fa2ce31511864075b49ef611e519fd316b46b8c088d5953ee1d6617388c82`），附 `.sha256`。
- 产物目录被 `.gitignore` 忽略，不入库。

## Files Changed

- `docs/releases/v2.4.1-beta.1.md`
- `internal/version/version.go`、`frontend/package.json`
- `build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- 产物目录 `release-assets/v2.4.1-beta.1/`

## Technical Notes

- 构建脚本再次运行 `set-release-version.ps1` 会把 `build/windows/info.json` 写成 CRLF；已还原为 LF，内容与该提交一致。
- 安装包未签名（按项目自签策略，本地 Beta 不签名）、未自动安装、未启动、未打标签、未推送。
- 本版本包含迁移 053（纯追加）：首次启动自动执行，默认不记录正文。

---

# Verification

## Tests

- [x] 构建脚本内置的 `task --force verify`（格式、lint、`tsc --noEmit`、Go 全包、前端 163 文件／1002 项测试、Windows 生产构建）
- [x] Windows 应用与安装器版本资源校验
- [x] 发布资产（数量、命名、SHA256）校验
- [ ] 真实安装与启动：由用户执行
- [ ] 桌面端实机目视检查：未执行

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
scripts\build-release.ps1 -Version v2.4.1-beta.1 -OutputDirectory release-assets\v2.4.1-beta.1
BUILD_EXIT=0
已校验 v2.4.1-beta.1，Windows 数字版本 2.4.1.0
前端 163 文件 / 1002 项测试通过
Windows 应用和安装器版本资源校验通过
发布资产校验通过：v2.4.1-beta.1
安装器：21,533,202 bytes
SHA256：ea8fa2ce31511864075b49ef611e519fd316b46b8c088d5953ee1d6617388c82
Windows 数字版本：2.4.1.0；NSIS 基础版本：2.4.1
```

---

# Documentation

- [x] 版本说明 `docs/releases/v2.4.1-beta.1.md`（同时被前端 `import.meta.glob` 收录，用于更新完成提示）
- [x] CURRENT
- [x] 其它：TASK-20261119-006 已更新产品文档与 ADR-028，本轮无新增

---

# Deviations

None. 构建一次通过；`build/windows/info.json` 的 CRLF 仅出现在工作区，已还原为 LF，内容与提交一致。

---

# Git

## Commit

```text
b338fca1 chore(release): 准备 v2.4.1-beta.1 本地构建
```

本任务记录与 `docs/plans/CURRENT.md` 的更新随 `chore(release): 记录 v2.4.1-beta.1 构建产物` 一并提交。

---

# Result

`v2.4.1-beta.1` 本地 Windows amd64 生产安装包已产出并校验通过，内容为 TASK-20261119-006（`3b412e7`）的可选请求正文记录与请求记录 XLSX 导出；未签名、未安装、未启动、未打标签、未推送。首次启动执行迁移 053（纯追加），正文记录默认关闭。

---

# Completed

2026-11-19 17:28
