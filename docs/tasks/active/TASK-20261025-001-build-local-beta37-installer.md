# TASK-20261025-001: 构建 Beta.37 本地生产安装包

## Status

In Progress（安装包已构建并校验，等待用户本地覆盖安装验收）

## Created

2026-10-25 09:10（Asia/Singapore）

## Request

用户要求：构建新的 Beta 生产版本，用于本机覆盖更新。

## Goal

产出 `release-assets/v2.0.0-beta.37/` 安装器与 SHA256，通过完整质量门、Windows 版本资源与
发布资产校验，供用户手动覆盖更新。

## Background

- 相对 Beta.36 的改动只有已归档的
  [TASK-20261024-022](../completed/2026-10/TASK-20261024-022-passthrough-quota-failover.md)：
  Codex 透明入口（客户端实际使用的 `/codex/v1`）在执行前额度拒绝时换号。
- 无数据库迁移；Beta.36 的迁移 043（Oxion AI 内置服务商）已在用户当前版本中。
- 关键修复：额度拒绝判定上提到 `core/proxy` 共用、透明转首事件 4 KiB 有界预读命中即按真实
  `resets_at` 冷却换号、透明转发改尝试级记账（修掉「换号即 503 `accounting_unavailable`」）、
  冷却遵循上游 `Retry-After`、请求结束后按需去抖刷新额度观察。

## Scope

- 版本同步至 `2.0.0-beta.37`，新增 `docs/releases/v2.0.0-beta.37.md`。
- 完整质量门 → 打包 → Windows 版本资源 → 发布资产与 SHA256 校验。
- 更新 `docs/plans/CURRENT.md` 的当前版本段落。

## Out of Scope

- 不自动安装、不打标签、不推送、不代码签名。
- 不发送真实订阅账号请求；不验收原生窗口视觉与实机 Codex 客户端换号效果。

---

# Implementation Plan

1. 同步 `internal/version/version.go` 与 `frontend/package.json` 到 `2.0.0-beta.37`，
   运行 `scripts/set-release-version.ps1` 映射 Windows 数字版本资源。
2. 写发布说明，执行 `task --force verify`。
3. 执行 `task --force package`（PATH 里加 NSIS），校验 Windows 版本资源。
4. 复制安装器到 `release-assets/v2.0.0-beta.37/`，生成 SHA256 并运行资产校验。
