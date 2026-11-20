# TASK-20261120-009: 构建 v2.4.1-beta.5 本地安装包

## Status

Completed

## Created

2026-11-20 20:12

## Request

用户要求：完成代理协议确认修复后，提交代码，构建新的 Beta 生产版本供本机覆盖更新，并更新任务记录。

## Goal

产出 `v2.4.1-beta.5` Windows amd64 生产安装包（未签名、本地使用），供本机覆盖更新后验证：

- 远端纯 SOCKS5 代理配成 HTTP 时，请求不再挂到超时，首次请求内自动切到 SOCKS5 并记住结果。

## Background

- 版本基线 `v2.4.1-beta.4`（`5efedf02` 记录构建产物）。
- 功能实现提交 `ab16e41c`（TASK-20261120-008）、记录哈希 `e845b69c`。
- 本版本无数据库迁移。
- `docs/releases/v2.4.1-beta.5.md` 与 `release-assets/v2.4.1-beta.5` 尚不存在。

## Scope

- `docs/releases/v2.4.1-beta.5.md` 版本说明（中英）。
- 版本号 → `2.4.1-beta.5`（`internal/version/version.go`、`frontend/package.json`）。
- `scripts/build-release.ps1 -Version v2.4.1-beta.5 -OutputDirectory release-assets\v2.4.1-beta.5`。
- 记录安装器大小、SHA256、Windows 数字版本与 NSIS 基础版本。

## Out of Scope

- 打标签、推送、公开 Release、代码签名、自动安装或启动应用。
- 桌面窗口内的实机目视检查（由用户安装后确认）。

## Existing Behavior

`v2.4.1-beta.4`：网络代理里把纯 SOCKS5 端口配成 `http` 时，提供商请求挂到调用方超时且无错误码。

## Expected Behavior

安装后：同一配置在首次请求内自愈到 SOCKS5，之后按识别结果建池；「网络代理」卡片显示实际协议。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`、`build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- `docs/releases/v2.4.1-beta.5.md`
- `scripts/build-release.ps1`、`scripts/set-release-version.ps1`、`scripts/test-windows-metadata.ps1`、`scripts/verify-release-assets.ps1`

## Dependencies

NSIS、`task`、`wails3`、Go 1.26、pnpm —— 已在本机验证可用。

## Risks

- 构建脚本内部运行 `task --force verify` 与 `task --force package`；构建前后核对 `git status`。
- `release-assets/v2.4.1-beta.5` 必须为空或不存在。
- 当前 shell 为 Windows PowerShell 5.1、代码页 936、`RemoteSigned` 策略：仓库 `.ps1` 是 UTF-8 无 BOM，
  PS 5.1 会按 ANSI 解析含中文的脚本并报语法错误，必须用临时 UTF-8 BOM 副本运行。

---

# Implementation Plan

1. 写版本说明与版本号，提交 `chore(release)`。
2. 运行 `scripts/build-release.ps1`。
3. 校验安装器、版本资源与发布资产。
4. 回填任务记录、归档、更新 CURRENT，提交文档。

---

# Progress

- 2026-11-20 20:12 创建记录，版本号 → `2.4.1-beta.5`，提交 `d6cee390`。
- 2026-11-20 20:13 第一次构建失败：`set-release-version.ps1` 按 ANSI 解析，中文 `throw` 文本触发语法错误。
- 2026-11-20 20:14 改为四个脚本（`build-release`、`set-release-version`、`test-windows-metadata`、
  `verify-release-assets`）的 BOM 副本，放在仓库根的临时目录 `scripts-bom/` 以保持 `$PSScriptRoot` 与
  根目录推导不变；构建 EXIT=0，随后删除临时目录。

---

# Implementation Result

## Changes

- 新增 `docs/releases/v2.4.1-beta.5.md`（中英版本说明：远端代理显式协议确认与自愈、握手预算 4 秒、
  协议变化清除识别结果、无迁移、本地更新与验证边界）。
- 版本号：`internal/version/version.go`、`frontend/package.json` → `2.4.1-beta.5`；构建脚本同步
  `build/config.yml`（`version: "2.4.1"`）、`build/windows/info.json`（`ProductVersion` → `2.4.1-beta.5`，
  `fixed` 仍为 `2.4.1.0`）与 NSIS `INFO_PRODUCTVERSION`（仍为 `2.4.1`）。
- 安装包：`release-assets/v2.4.1-beta.5/OmniProxy-Setup-v2.4.1-beta.5-windows-amd64.exe`
  （21,598,293 bytes，SHA256 `a9abe118ca031d3b9cd733b54374fc70cc301e0a8936c6176cd41d6500cb1593`），附 `.sha256`。
- 产物目录被 `.gitignore` 忽略，不入库。

## Files Changed

- `docs/releases/v2.4.1-beta.5.md`
- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `docs/tasks/completed/2026-11/TASK-20261120-009-build-local-v2.4.1-beta.5.md`、`docs/plans/CURRENT.md`
- 产物目录 `release-assets/v2.4.1-beta.5/`

## Technical Notes

- 安装包未签名（本地 Beta）、未自动安装、未启动、未打标签、未推送。
- 本版本无数据库迁移；功能内容为 TASK-20261120-008 的显式代理协议确认与自愈（`ab16e41c`）。
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
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts-bom\build-release.ps1 -Version v2.4.1-beta.5 -OutputDirectory release-assets\v2.4.1-beta.5
BUILD_EXIT=0
已校验 v2.4.1-beta.5，Windows 数字版本 2.4.1.0
前端 164 文件 / 1006 项测试通过
Windows 应用和安装器版本资源校验通过
发布资产校验通过：v2.4.1-beta.5
安装器：21,598,293 bytes
SHA256：a9abe118ca031d3b9cd733b54374fc70cc301e0a8936c6176cd41d6500cb1593
Windows 数字版本：2.4.1.0；NSIS 基础版本：2.4.1
```

安装器文件属性复核：`ProductVersion=2.4.1`、`FileVersion=2.4.1`（NSIS 基础版本，与 beta.4 一致）；
`build/config.yml` → `2.4.1`，`build/windows/nsis/wails_tools.nsh` → `INFO_PRODUCTVERSION "2.4.1"`。

---

# Documentation

- [x] 版本说明 `docs/releases/v2.4.1-beta.5.md`
- [x] CURRENT
- [x] 功能文档由 TASK-20261120-008 更新（`docs/product/NETWORK_PROXY.md`）

---

# Deviations

- 构建环境限制比上一版本记录更严：Windows PowerShell 5.1 按 ANSI 解析 UTF-8 无 BOM 的脚本，
  `set-release-version.ps1` 里的中文 `throw` 文本直接导致语法错误。第一次只给 `build-release.ps1`
  加 BOM 仍然失败；最终把四个被调用脚本都做成 BOM 副本放到仓库根的 `scripts-bom/`（保证
  `$PSScriptRoot` 推导的仓库根不变）后一次通过，临时目录已删除，仓库脚本未改动。
- `build/windows/info.json` 的 `ProductVersion` 由构建脚本写入，随本次「记录构建产物」提交。

---

# Git

## Commit

```text
d6cee390 chore(release): 准备 v2.4.1-beta.5 本地构建
```

本任务记录、`build/windows/info.json` 与 `docs/plans/CURRENT.md` 的更新随
`chore(release): 记录 v2.4.1-beta.5 构建产物` 一并提交。

---

# Result

`v2.4.1-beta.5` 本地 Windows amd64 生产安装包已产出并校验通过，内容为显式代理协议首次使用前握手确认与
必要时自愈（TASK-20261120-008，`ab16e41c`）；未签名、未安装、未启动、未打标签、未推送。
用户需从托盘完全退出 OmniProxy、备份数据目录后运行安装器覆盖安装。

---

# Completed

2026-11-20 20:22
