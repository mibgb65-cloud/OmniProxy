# TASK-20261118-009: 构建 v2.3.1-beta.26 本地安装包

## Status

Completed

## Created

2026-11-18 15:06

## Request

用户在 TASK-20261118-007（账号级拒绝换号）与 TASK-20261118-008（家族名 + 档位在选服务商前解析）完成后回复「继续」，按既有节奏构建新的本地 Beta 生产安装包，供本机覆盖更新后验证两处修复。

## Goal

产出 `v2.3.1-beta.26` Windows amd64 生产安装包（未签名、本地使用），包含：

- TASK-20261118-005：Google 订阅缓存命中 Token（`9bdd5d42`，beta.25 已包含但用户尚未安装）；
- TASK-20261118-007：账号级拒绝保留真实错误码并换号（`8e7b1566`）；
- TASK-20261118-008：家族名 + 档位在选服务商前解析到官方变体（`e7c75b81`）。

## Scope

- `docs/releases/v2.3.1-beta.26.md` 版本说明（中英）；
- `internal/version/version.go`、`frontend/package.json` 版本号；
- `scripts/build-release.ps1 -Version v2.3.1-beta.26 -OutputDirectory release-assets\v2.3.1-beta.26`；
- 记录产物大小与 SHA256。

## Out of Scope

- 打标签、推送、公开 Release、代码签名、自动安装或启动应用。
- 数据库迁移（本版本无）。

## Existing Behavior

当前本机运行的是 beta.25 之前的构建；`v2.3.1-beta.25` 安装包已产出但未安装。

## Expected Behavior

安装 `v2.3.1-beta.26` 后：新的 Google 请求显示真实缓存命中；Gemini 家族的“家族名 + 档位”请求由 Google 订阅承接；WorkBuddy 侧的账号级拒绝显示真实错误码并自动换号。

---

# Implementation Plan

1. 写版本说明与版本号，提交 `chore(release)`。
2. 运行构建脚本（内部会执行 `task --force verify` 与 `task --force package`）。
3. 校验产物大小、SHA256 与 Windows 版本资源。
4. 回填构建结果并归档本记录。

---

# Progress

- 2026-11-18 15:06 创建记录，准备版本说明与版本号。
- 2026-11-18 15:12 `2487335e` 提交版本说明与版本号（`chore(release)`）。
- 2026-11-18 15:22 构建完成，`BUILD_EXIT=0`，产物校验通过。

---

# Implementation Result

## Changes

- 新增 `docs/releases/v2.3.1-beta.26.md`（中英版本说明：两处修复、无不兼容变更、本地更新步骤、验证边界）。
- 版本号：`internal/version/version.go`、`frontend/package.json` → `2.3.1-beta.26`。
- 安装包：`release-assets/v2.3.1-beta.26/OmniProxy-Setup-v2.3.1-beta.26-windows-amd64.exe`（21,398,583 bytes，SHA256 `db8818b6d80bf242ad88a56acb436f0335726ecd68ad0f8839f176d6f005c5eb`），附 `.sha256`。
- 构建脚本自动更新的 `build/windows/info.json`：`fixed` 版本 `2.3.1.0`，`ProductVersion` 为 `2.3.1-beta.26`（换行符已还原为 LF）。
- NSIS 基础版本保持 `2.3.1`（`build/windows/nsis/wails_tools.nsh` 未变化）。

## Files Changed

- `docs/releases/v2.3.1-beta.26.md`
- `internal/version/version.go`、`frontend/package.json`
- `build/windows/info.json`（构建脚本生成）
- 产物目录 `release-assets/v2.3.1-beta.26/`（被 `.gitignore` 忽略，不入库）

## Technical Notes

- 构建脚本内部执行 `task --force verify` 与 `task --force package`，并校验 Windows 应用／安装器版本资源与发布资产名。
- 安装包未签名（按项目 `codeSigningPolicy`），未自动安装、未启动应用、未打标签、未推送。

---

# Verification

## Tests

- [x] 构建脚本内置的 `task --force verify`（格式、lint、`tsc --noEmit`、Go 全包、前端 159 文件／977 项测试、Windows 生产构建）
- [ ] 真实请求验证：需要用户安装后发起，构建期不消耗任何订阅／中转额度

## Quality Checks

- [x] Format／Lint／Type Check／Test／Build
- [x] Windows 版本资源校验：`Windows 应用和安装器版本资源校验通过`
- [x] 发布资产校验：`发布资产校验通过：v2.3.1-beta.26`

## Verification Result

```text
scripts\build-release.ps1 -Version v2.3.1-beta.26 -OutputDirectory release-assets\v2.3.1-beta.26
BUILD_EXIT=0
安装器：21,398,583 bytes
SHA256：db8818b6d80bf242ad88a56acb436f0335726ecd68ad0f8839f176d6f005c5eb
Windows 数字版本：2.3.1.0；NSIS 基础版本：2.3.1
```

---

# Documentation

- [x] 版本说明 `docs/releases/v2.3.1-beta.26.md`
- [x] CURRENT
- [ ] 其它：No documentation changes required.

---

# Deviations

- 计划里只写“记录产物大小与 SHA256”，实际同时把构建生成的 `build/windows/info.json` 纳入提交（否则仓库版本资源与安装包不一致）。

---

# Git

## Commit

```text
2487335e chore(release): 准备 v2.3.1-beta.26 本地构建
5ded80c8 chore(release): 记录 v2.3.1-beta.26 构建产物与版本资源
```

---

# Result

`v2.3.1-beta.26` 本地安装包已产出并校验通过，包含 beta.25 的缓存命中修复、账号级拒绝换号与服务商截获修复；未签名、未安装、未启动、未打标签、未推送。

---

# Completed

2026-11-18 15:25

