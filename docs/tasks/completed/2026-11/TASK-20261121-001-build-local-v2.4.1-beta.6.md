# TASK-20261121-001: 构建 v2.4.1-beta.6 本地安装包

## Status

Completed

## Created

2026-11-21 11:05

## Request

用户要求：完成 Antigravity 提前响应头修复后，提交代码，构建新的 Beta 生产版本供本机覆盖更新，并更新任务记录。

## Goal

产出 `v2.4.1-beta.6` Windows amd64 生产安装包（未签名、本地使用），供本机覆盖更新后验证：

- Antigravity 桌面端等待上游响应头超过 60 秒时不再出现「已取消／未报告」；
- 网关在受理 3 秒后先下发 `200` 与 SSE 响应头，并按 10 秒心跳保活；
- 其它客户端的错误状态码语义不变。

## Background

- 版本基线 `v2.4.1-beta.5`（`82171576` 记录构建产物）。
- 功能实现提交 `ac4b6c06`（TASK-20261120-010）、记录哈希 `2ca280e4`。
- 本版本无数据库迁移。
- `docs/releases/v2.4.1-beta.6.md` 与 `release-assets/v2.4.1-beta.6` 尚不存在。

## Scope

- `docs/releases/v2.4.1-beta.6.md` 版本说明（中英）。
- 版本号 → `2.4.1-beta.6`（`internal/version/version.go`、`frontend/package.json`）。
- `scripts/build-release.ps1 -Version v2.4.1-beta.6 -OutputDirectory release-assets\v2.4.1-beta.6`。
- 记录安装器大小、SHA256、Windows 数字版本与 NSIS 基础版本。

## Out of Scope

- 打标签、推送、公开 Release、代码签名、自动安装或启动应用。
- 桌面窗口内的实机目视检查（由用户安装后确认）。

## Existing Behavior

`v2.4.1-beta.5`：Antigravity 的流式请求要等 provider 拿到上游响应头后才可能写响应；上游超过 60 秒不回响应头时，桌面端按响应头超时断开，请求页出现「已取消／未报告」，4 秒后重发的同一请求通常成功。

## Expected Behavior

安装后：Antigravity 请求在受理 3 秒后就能看到 `200` 与 SSE 响应头，思考期间每 10 秒收到 `: keep-alive`；上游响应头到达后正常转发，不再触发 60 秒响应头超时。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`、`build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- `docs/releases/v2.4.1-beta.6.md`
- `scripts/build-release.ps1`、`scripts/set-release-version.ps1`、`scripts/test-windows-metadata.ps1`、`scripts/verify-release-assets.ps1`

## Dependencies

NSIS、`task`、`wails3`、Go 1.26、pnpm —— 已在本机验证可用。

## Risks

- 构建脚本内部运行 `task --force verify` 与 `task --force package`；构建前后核对 `git status`。
- `release-assets/v2.4.1-beta.6` 必须为空或不存在。
- Windows PowerShell 5.1 按 ANSI 解析 UTF-8 无 BOM 的仓库脚本，含中文 `throw` 文本时会报语法错误；
  沿用上一版本的 BOM 副本方案（临时目录 `scripts-bom/`，保持 `$PSScriptRoot` 推导的仓库根不变）。

---

# Implementation Plan

1. 写版本说明与版本号，提交 `chore(release)`。
2. 运行 `scripts/build-release.ps1`（BOM 副本）。
3. 校验安装器、版本资源与发布资产。
4. 回填任务记录、归档、更新 CURRENT，提交文档。

---

# Progress

- 2026-11-21 11:05 创建记录，写版本说明与版本号，提交 `336b861c`。
- 2026-11-21 11:07 第一次构建失败：仓库 `.ps1` 为 UTF-8 无 BOM，Windows PowerShell 5.1 按 ANSI
  解析，中文 `throw` 文本触发语法错误。
- 2026-11-21 11:08 生成四个脚本的 BOM 副本到仓库根 `scripts-bom/`（保持 `$PSScriptRoot` 推导的
  仓库根不变）；直接调用仍被 `RemoteSigned` 执行策略拒绝，改用
  `powershell.exe -NoProfile -ExecutionPolicy Bypass -File` 运行。
- 2026-11-21 11:12 构建 EXIT=0，安装器与发布资产校验通过；删除临时目录 `scripts-bom/`。

---

# Implementation Result

## Changes

- 新增 `docs/releases/v2.4.1-beta.6.md`（中英版本说明：Antigravity 提前响应头与保活、范围限定、
  提交后失败的语义、与 beta.1 修复的区别、无迁移、本地更新与验证边界）。
- 版本号：`internal/version/version.go`、`frontend/package.json` → `2.4.1-beta.6`；构建脚本同步
  `build/config.yml`（`version: "2.4.1"`）、`build/windows/info.json`（`ProductVersion` → `2.4.1-beta.6`，
  `fixed` 仍为 `2.4.1.0`）与 NSIS `INFO_PRODUCTVERSION`（仍为 `2.4.1`）。
- 安装包：`release-assets/v2.4.1-beta.6/OmniProxy-Setup-v2.4.1-beta.6-windows-amd64.exe`
  （21,604,916 bytes，SHA256 `0555bcf42ad2581c6510005c49744c5d485bee0857a1e29290ab7c431e43cc26`），附 `.sha256`。
- 产物目录被 `.gitignore` 忽略，不入库。

## Files Changed

- `docs/releases/v2.4.1-beta.6.md`
- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `docs/tasks/completed/2026-11/TASK-20261121-001-build-local-v2.4.1-beta.6.md`、`docs/plans/CURRENT.md`
- 产物目录 `release-assets/v2.4.1-beta.6/`

## Technical Notes

- 安装包未签名（本地 Beta）、未自动安装、未启动、未打标签、未推送。
- 本版本无数据库迁移；功能内容为 TASK-20261120-010 的 Antigravity 提前响应头与保活（`ac4b6c06`）。
- 构建脚本重写 `build/windows/info.json` 会输出 CRLF，按 `.gitattributes`（`*.json text eol=lf`）归一后再提交。
- 前端版本说明通过 `import.meta.glob("../../../../docs/releases/v*.md")` 收录，新文件已随本次前端构建打进包内。

---

# Verification

## Tests

- [x] 构建脚本内置的 `task --force verify`（格式、lint、`tsc --noEmit`、Go 全包、前端 164 文件／1006 项测试、Windows 生产构建）
- [x] Windows 应用与安装器版本资源校验（`test-windows-metadata.ps1`）
- [x] 发布资产（数量、命名、SHA256）校验（`verify-release-assets.ps1`）
- [ ] 真实安装与启动：由用户执行
- [ ] 桌面窗口目视检查：未执行

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts-bom\build-release.ps1 -Version v2.4.1-beta.6 -OutputDirectory release-assets\v2.4.1-beta.6
BUILD_EXIT=0
已校验 v2.4.1-beta.6，Windows 数字版本 2.4.1.0
前端 164 文件 / 1006 项测试通过
Windows 应用和安装器版本资源校验通过
发布资产校验通过：v2.4.1-beta.6
安装器：21,604,916 bytes
SHA256：0555bcf42ad2581c6510005c49744c5d485bee0857a1e29290ab7c431e43cc26
Windows 数字版本：2.4.1.0；NSIS 基础版本：2.4.1
```

---

# Documentation

- [x] 版本说明 `docs/releases/v2.4.1-beta.6.md`
- [x] CURRENT
- [x] 功能文档由 TASK-20261120-010 更新（`docs/product/GATEWAY.md`）

---

# Deviations

- 构建环境与上一版本相同：Windows PowerShell 5.1 按 ANSI 解析 UTF-8 无 BOM 的仓库脚本。BOM 副本
  首次直接执行被 `RemoteSigned` 拒绝，最终用 `powershell.exe -NoProfile -ExecutionPolicy Bypass -File`
  运行；临时目录 `scripts-bom/` 已删除，仓库脚本未改动。
- `build/windows/info.json` 的 `ProductVersion` 由构建脚本写入，随本次「记录构建产物」提交。

---

# Git

## Commit

```text
336b861c chore(release): 准备 v2.4.1-beta.6 本地构建
```

本任务记录、`build/windows/info.json` 与 `docs/plans/CURRENT.md` 的更新随
`chore(release): 记录 v2.4.1-beta.6 构建产物` 一并提交。

---

# Result

`v2.4.1-beta.6` 本地 Windows amd64 生产安装包已产出并校验通过，内容为 Antigravity 流式请求在等待
上游响应头期间提前下发响应头并保活（TASK-20261120-010，`ac4b6c06`）；未签名、未安装、未启动、
未打标签、未推送。用户需从托盘完全退出 OmniProxy、备份数据目录后运行安装器覆盖安装。

---

# Completed

2026-11-21 11:15
