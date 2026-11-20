# TASK-20261120-007: 构建 v2.4.1-beta.4 本地安装包

## Status

Completed

## Created

2026-11-20 10:50

## Request

用户在用量统计的模型行提出合并推理档位变体（TASK-20261120-006）后，沿用既有本地 Beta 流程：
修复提交后构建新的 Beta 生产安装包供本机覆盖更新。

## Goal

产出 `v2.4.1-beta.4` Windows amd64 生产安装包（未签名、本地使用），供本机覆盖更新后验证：

- 用量页「按提供商与模型」把 `gemini-4-neon-high`／`-xhigh` 等档位变体合并为一行。
- 用量页模型占比、模型每日用量与仪表盘模型 Token 占比同样按模型家族归并。

## Background

- 版本基线 `v2.4.1-beta.3`（`b1e35183` 构建准备、`d8a9db9f` 记录）。
- 功能实现提交为 TASK-20261120-006 的提交，本版本无数据库迁移。
- `docs/releases/v2.4.1-beta.4.md` 与 `release-assets/v2.4.1-beta.4` 尚不存在。

## Scope

- `docs/releases/v2.4.1-beta.4.md` 版本说明（中英）。
- 版本号 → `2.4.1-beta.4`（`internal/version/version.go`、`frontend/package.json`）。
- `scripts/build-release.ps1 -Version v2.4.1-beta.4 -OutputDirectory release-assets\v2.4.1-beta.4`。
- 记录安装器大小、SHA256、Windows 数字版本与 NSIS 基础版本。

## Out of Scope

- 打标签、推送、公开 Release、代码签名、自动安装或启动应用。
- 真实请求复测（本版本只改统计读取，仍由用户安装后确认桌面窗口显示）。

## Existing Behavior

`v2.4.1-beta.3`：同一 Gemini 模型的 `-high`／`-xhigh` 等档位变体在统计中各占一行。

## Expected Behavior

安装后：同一模型的档位变体合并为一行，请求数与 Token 相加；合并行悬浮可看到被合并的具体型号；
前六名与「其他」按合并后的家族计算。

---

# Analysis

## Related Modules

- `internal/version/version.go`、`frontend/package.json`、`build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`
- `docs/releases/v2.4.1-beta.4.md`
- `scripts/build-release.ps1`、`scripts/set-release-version.ps1`

## Dependencies

NSIS、`task`、`wails3`、Go 1.26、pnpm —— 已验证可用。

## Risks

- 构建脚本内部运行 `task --force verify` 与 `task --force package`；构建前后核对 `git status`。
- `release-assets/v2.4.1-beta.4` 必须为空或不存在。
- 当前 shell 为 Windows PowerShell 5.1、代码页 936、`RemoteSigned` 策略，需用临时 UTF-8 BOM 副本
  + `-ExecutionPolicy Bypass` 运行构建脚本（仓库脚本不改动）。

---

# Implementation Plan

1. 写版本说明与版本号，提交 `chore(release)`。
2. 运行 `scripts/build-release.ps1`。
3. 校验安装器、版本资源与发布资产。
4. 回填任务记录、归档、更新 CURRENT，提交文档。

---

# Progress

- 2026-11-20 10:50 创建记录。
- 2026-11-20 10:51 版本号 → `2.4.1-beta.4`，写版本说明并提交 `chore(release)`。
- 2026-11-20 10:58 构建 EXIT=0：质量门、NSIS 打包、版本资源与发布资产校验全部通过。

---

# Implementation Result

## Changes

- 新增 `docs/releases/v2.4.1-beta.4.md`（中英版本说明：模型家族归并、证据规则、无迁移、本地更新与验证边界）。
- 版本号：`internal/version/version.go`、`frontend/package.json` → `2.4.1-beta.4`；构建脚本同步
  `build/config.yml`、`build/windows/info.json`（`ProductVersion` → `2.4.1-beta.4`，`fixed` 仍为 `2.4.1.0`）
  与 NSIS `INFO_PRODUCTVERSION`（仍为 `2.4.1`）。
- 安装包：`release-assets/v2.4.1-beta.4/OmniProxy-Setup-v2.4.1-beta.4-windows-amd64.exe`
  （21,592,674 bytes，SHA256 `b4738184894be37825aff16fe0d261b64c408e85aab66519681c1fc6239e558e`），附 `.sha256`。
- 产物目录被 `.gitignore` 忽略，不入库。

## Files Changed

- `docs/releases/v2.4.1-beta.4.md`
- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- 产物目录 `release-assets/v2.4.1-beta.4/`

## Technical Notes

- 安装包未签名（本地 Beta）、未自动安装、未启动、未打标签、未推送。
- 本版本无数据库迁移；内容为 TASK-20261120-006 的用量统计模型家族归并（ADR-030）。
- 构建脚本重写 `build/windows/info.json` 时会输出 CRLF，已按 `.gitattributes` 归一为 LF 后再提交。

---

# Verification

## Tests

- [x] 构建脚本内置的 `task --force verify`（格式、lint、`tsc --noEmit`、Go 全包、前端 164 文件／1006 项测试、Windows 生产构建）
- [x] Windows 应用与安装器版本资源校验
- [x] 发布资产（数量、命名、SHA256）校验
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
scripts\build-release.bom.ps1 -Version v2.4.1-beta.4 -OutputDirectory release-assets\v2.4.1-beta.4
BUILD_EXIT=0
已校验 v2.4.1-beta.4，Windows 数字版本 2.4.1.0
前端 164 文件 / 1006 项测试通过
Windows 应用和安装器版本资源校验通过
发布资产校验通过：v2.4.1-beta.4
安装器：21,592,674 bytes
SHA256：b4738184894be37825aff16fe0d261b64c408e85aab66519681c1fc6239e558e
Windows 数字版本：2.4.1.0；NSIS 基础版本：2.4.1
```

---

# Documentation

- [x] 版本说明 `docs/releases/v2.4.1-beta.4.md`（会被前端 `import.meta.glob` 收录）
- [x] CURRENT
- [x] 功能文档由 TASK-20261120-006 更新（ADR-030、ACTIVITY、DASHBOARD、PRD）

---

# Deviations

- 当前执行环境为 Windows PowerShell 5.1、代码页 936，且 `CurrentUser` 执行策略为 `RemoteSigned`：
  构建脚本改用临时 UTF-8 BOM 副本 + `-ExecutionPolicy Bypass` 运行；仓库 `.ps1` 未修改，副本已删除。
- `build/windows/info.json` 的 `ProductVersion` 由构建脚本写入后随本次「记录构建产物」提交。

---

# Git

## Commit

```text
48a599e0 chore(release): 准备 v2.4.1-beta.4 本地构建
```

本任务记录与 `docs/plans/CURRENT.md` 的更新随 `chore(release): 记录 v2.4.1-beta.4 构建产物` 一并提交。

---

# Result

`v2.4.1-beta.4` 本地 Windows amd64 生产安装包已产出并校验通过，内容为用量统计按模型家族合并推理档位变体
（TASK-20261120-006，ADR-030）；未签名、未安装、未启动、未打标签、未推送。

---

# Completed

2026-11-20 10:58

