# TASK-20261120-005: 构建 v2.4.1-beta.3 本地安装包

## Status

Planned

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

---

# Implementation Result

待完成后填写。

---
