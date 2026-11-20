# TASK-20261120-007: 构建 v2.4.1-beta.4 本地安装包

## Status

Planned

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

---

# Implementation Result

待完成后填写。

---
