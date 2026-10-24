# TASK-20261024-006: 构建 Beta.31 本地生产安装包

## Status

In Progress（安装包已构建并校验，等待用户本地覆盖安装验收）

## Created

2026-10-24 04:45（Asia/Singapore）

## Request

用户要求：按三点反馈改完后构建新的 Beta 生产版本，供其本机覆盖更新。

## Goal

产出 `release-assets/v2.0.0-beta.31/` 安装器与 SHA256，通过完整质量门、Windows 版本资源与发布资产校验。

## Background

- 本版功能差异是 TASK-20261024-005：`none → low` 档位映射、目录构建期折叠与重名去重、River 条目在 Codex 声明图片。
- 工作区仍含其他在办任务（River 图片、客户端图片能力）的未提交改动，随包打包但不代为提交。

## Scope

- 版本同步至 `2.0.0-beta.31` 并新增 `docs/releases/v2.0.0-beta.31.md`。
- `task --force verify` → `task --force package` → 版本资源 → 资产与 SHA256 校验。
- 更新 `docs/plans/CURRENT.md`。

## Out of Scope

- 不提交其他在办任务的改动；不自动安装、不打标签、不推送。

---

# Implementation Result

- 版本同步至 `2.0.0-beta.31`；新增发布说明。
- `release-assets/v2.0.0-beta.31/`：安装器 19,661,392 bytes，SHA256 `48661800ce9fc50458d988e33ff57282c18623693514f8c9bf64eac0e483175c`。
- 发布目录 26 个，全部各 2 个文件；历史资产未变。

# Verification

- PASS：`task --force verify` exit 0（126 文件 / 750 项前端测试、Go 全部包、Windows production 构建；日志 `tmp/verify-beta31.log`）。
- PASS：`task --force package` exit 0（日志 `tmp/package-beta31.log`）。
- PASS：`scripts/test-windows-metadata.ps1` 输出「Windows 应用和安装器版本资源校验通过」。
- PASS：`scripts/verify-release-assets.ps1` 输出「发布资产校验通过」。
- 未执行：代码签名、自动安装、停止或替换生产进程、打标签、推送。

# Result

Beta.31 production 本地安装包已构建并校验通过，未签名、未自动安装，供用户本地覆盖更新。
