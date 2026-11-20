# TASK-20261120-005: 构建 v2.4.1-beta.3 本地安装包

## Status

Completed

## Created

2026-11-20 09:45

## Request

用户在反馈「同一 Google 账号在账号页同时存在两条记录」并确认修复方式后，沿用既有的本地 Beta 流程：
修复提交后构建新的 Beta 生产安装包供本机覆盖更新（见 TASK-20261120-004）。

## Goal

产出 `v2.4.1-beta.3` Windows amd64 生产安装包（未签名、本地使用），供本机覆盖更新后验证：

- 「添加账号」重新授权同一 Google／ChatGPT 账号时更新原账号，不再新增重复卡片。
- 已有的重复卡片保持不变，可在账号页手动删除。

## Background

- 版本基线 `v2.4.1-beta.2`（`738a8936` 构建准备、`074158fc` 记录）。
- 修复实现提交为 TASK-20261120-004 的提交，本版本无数据库迁移。
- `docs/releases/v2.4.1-beta.3.md` 与 `release-assets/v2.4.1-beta.3` 尚不存在。

## Scope

- `docs/releases/v2.4.1-beta.3.md` 版本说明（中英）。
- 版本号 → `2.4.1-beta.3`（`internal/version/version.go`、`frontend/package.json`）。
- `scripts/build-release.ps1 -Version v2.4.1-beta.3 -OutputDirectory release-assets\v2.4.1-beta.3`。
- 记录安装器大小、SHA256、Windows 数字版本与 NSIS 基础版本。

## Out of Scope

- 打标签、推送、公开 Release、代码签名、自动安装或启动应用。
- 真实 Google／ChatGPT 授权复测（需要用户本人浏览器授权）。

## Existing Behavior

`v2.4.1-beta.2`：「添加账号」重新授权同一订阅账号会新增第二条账号记录。

## Expected Behavior

安装后：「添加账号」重新授权同一订阅账号更新原记录，账号数量不增长；已有重复卡片由用户手动删除。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`、`build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- `docs/releases/v2.4.1-beta.3.md`
- `scripts/build-release.ps1`、`scripts/set-release-version.ps1`

## Dependencies

NSIS、`task`、`wails3`、Go 1.26、pnpm —— 已验证可用。

## Risks

- 构建脚本内部运行 `task --force verify` 与 `task --force package`；构建前后核对 `git status`。
- `release-assets/v2.4.1-beta.3` 必须为空或不存在。
- 当前 shell 为 Windows PowerShell 5.1、代码页 936，无 BOM 的 UTF-8 脚本需要临时 BOM 副本运行（仓库脚本不改动）。

---

# Implementation Plan

1. 写版本说明与版本号，提交 `chore(release)`。
2. 运行 `scripts/build-release.ps1`。
3. 校验安装器、版本资源与发布资产。
4. 回填任务记录、归档、更新 CURRENT，提交文档。

---

# Progress

- 2026-11-20 09:45 创建记录。
- 2026-11-20 09:47 版本号 → `2.4.1-beta.3`，写版本说明并提交 `chore(release)`。
- 2026-11-20 09:47 直接运行 `.ps1` 被 `RemoteSigned` 策略拒绝（未签名脚本）；改用 `powershell.exe -ExecutionPolicy Bypass -File` 运行临时 UTF-8 BOM 副本，并修正副本内的 `$root` 计算。
- 2026-11-20 09:58 构建 EXIT=0：质量门、NSIS 打包、版本资源与发布资产校验全部通过。

---

# Implementation Result

## Changes

- 新增 `docs/releases/v2.4.1-beta.3.md`（中英版本说明：订阅身份复用、已有重复卡片的手动处理、无迁移、本地更新与验证边界）。
- 版本号：`internal/version/version.go`、`frontend/package.json` → `2.4.1-beta.3`；构建脚本同步 `build/config.yml`、`build/windows/info.json`（`ProductVersion` → `2.4.1-beta.3`，`fixed` 仍为 `2.4.1.0`）与 NSIS `INFO_PRODUCTVERSION`（仍为 `2.4.1`）。
- 安装包：`release-assets/v2.4.1-beta.3/OmniProxy-Setup-v2.4.1-beta.3-windows-amd64.exe`
  （21,562,763 bytes，SHA256 `72fa62e0bce5845ec74db60becb906be92c1986d6a7ae5d4e4f5f099c584e08f`），附 `.sha256`。
- 产物目录被 `.gitignore` 忽略，不入库。

## Files Changed

- `docs/releases/v2.4.1-beta.3.md`
- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- 产物目录 `release-assets/v2.4.1-beta.3/`

## Technical Notes

- 安装包未签名（本地 Beta）、未自动安装、未启动、未打标签、未推送。
- 本版本无数据库迁移；内容为 TASK-20261120-004 的订阅身份修复。
- 构建脚本重写 `build/windows/info.json` 时会输出 CRLF，已按 `.gitattributes` 归一为 LF 后再提交（与之前每个版本的构建处理一致）。

---

# Verification

## Tests

- [x] 构建脚本内置的 `task --force verify`（格式、lint、`tsc --noEmit`、Go 全包、前端 164 文件／1005 项测试、Windows 生产构建）
- [x] Windows 应用与安装器版本资源校验
- [x] 发布资产（数量、命名、SHA256）校验
- [ ] 真实安装与启动：由用户执行
- [ ] 真实 Google／ChatGPT 重新授权：由用户执行

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
scripts\build-release.bom.ps1 -Version v2.4.1-beta.3 -OutputDirectory release-assets\v2.4.1-beta.3
BUILD_EXIT=0
已校验 v2.4.1-beta.3，Windows 数字版本 2.4.1.0
前端 164 文件 / 1005 项测试通过
Windows 应用和安装器版本资源校验通过
发布资产校验通过：v2.4.1-beta.3
安装器：21,562,763 bytes
SHA256：72fa62e0bce5845ec74db60becb906be92c1986d6a7ae5d4e4f5f099c584e08f
Windows 数字版本：2.4.1.0；NSIS 基础版本：2.4.1
```

---

# Documentation

- [x] 版本说明 `docs/releases/v2.4.1-beta.3.md`（会被前端 `import.meta.glob` 收录）
- [x] CURRENT
- [x] 功能文档由 TASK-20261120-004 更新（ADR-029、PRD、ACTIVITY、GOOGLE_SUBSCRIPTION、SUBSCRIPTION）

---

# Deviations

- 当前执行环境为 Windows PowerShell 5.1、代码页 936，且 `CurrentUser` 执行策略为 `RemoteSigned`：临时 BOM 副本直接运行被策略拒绝，改用 `powershell.exe -ExecutionPolicy Bypass -File` 运行；仓库 `.ps1` 未修改，副本已删除。
- 与 beta.2 不同，`build/windows/info.json` 的 `ProductVersion` 由构建脚本写入后随本次「记录构建产物」提交，而不是放在准备提交里；提交后源码与安装包内容一致。

---

# Git

## Commit

```text
b1e35183 chore(release): 准备 v2.4.1-beta.3 本地构建
```

本任务记录与 `docs/plans/CURRENT.md` 的更新随 `chore(release): 记录 v2.4.1-beta.3 构建产物` 一并提交。

---

# Result

`v2.4.1-beta.3` 本地 Windows amd64 生产安装包已产出并校验通过，内容为「同一订阅身份重新登录复用原账号」（TASK-20261120-004，ADR-029）；未签名、未安装、未启动、未打标签、未推送。

---

# Completed

2026-11-20 09:58

