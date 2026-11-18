# TASK-20261118-009: 构建 v2.3.1-beta.26 本地安装包

## Status

In Progress

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

---
