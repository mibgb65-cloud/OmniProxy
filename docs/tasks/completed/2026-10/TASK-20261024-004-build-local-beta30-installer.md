# TASK-20261024-004: 构建 Beta.30 本地生产安装包

## Status

Completed（2026-10-27 随 v2.0.0 正式发布归档）

## Created

2026-10-24 03:40（Asia/Singapore）

## Request

用户要求：修完两个推理强度问题后构建新的 Beta 生产版本，供其本机覆盖更新。

## Goal

产出 `release-assets/v2.0.0-beta.30/` 安装器与 SHA256，通过完整质量门、Windows 版本资源与发布资产校验。

## Background

- Beta.29 已安装并验证过固定入口前缀（`~/.codex/config.toml` 已是 `…/codex/v1`）。
- 本版相对 Beta.29 的功能差异是 TASK-20261024-003：Codex 档位词汇映射、Google 变体折叠与旧 ID 兼容。
- 工作区仍含其他在办任务（River 图片、客户端图片能力）的未提交改动，随包打包但不代为提交。

## Scope

- 版本同步至 `2.0.0-beta.30`（`internal/version/version.go`、`frontend/package.json`、`build/windows/info.json` 的 `0409.ProductVersion`）。
- 新增 `docs/releases/v2.0.0-beta.30.md`。
- `task --force verify` → `task --force package` → 版本资源 → 资产与 SHA256 校验。
- 更新 `docs/plans/CURRENT.md`。

## Out of Scope

- 不提交其他在办任务的改动。
- 不自动安装、不打标签、不推送。

---

# Implementation Result

- 版本同步至 `2.0.0-beta.30`；新增 `docs/releases/v2.0.0-beta.30.md`（档位词汇映射、Google 变体折叠、兼容边界）。
- 产出 `release-assets/v2.0.0-beta.30/`：安装器 19,656,610 bytes，SHA256 `46b4148813c5b8e0775f8d24e0720143c72f5134616a9e4a4bc58c8050c89bd1`。
- 发布目录共 25 个，全部各 2 个文件；历史资产未变。

# Verification

- PASS：`task --force verify` exit 0（126 文件 / 750 项前端测试、Go 全部包、Windows production 构建；日志 `tmp/verify-beta30.log`）。
- PASS：`task --force package` exit 0（日志 `tmp/package-beta30.log`）。
- PASS：`scripts/test-windows-metadata.ps1` 输出「Windows 应用和安装器版本资源校验通过」。
- PASS：`scripts/verify-release-assets.ps1` 输出「发布资产校验通过」。
- 未执行：代码签名、自动安装、停止或替换生产进程、打标签、推送。

# Result

Beta.30 production 本地安装包已构建并校验通过，未签名、未自动安装，供用户本地覆盖更新。

---

# Archival (v2.0.0)

- 原状态：In Progress（打包与校验进行中）
- 2026-10-27：v2.0.0 正式版已公开（标签源码 `c9c48b4a31988a6cc3fa2f4b1d423e61bd10f71b`，
  Actions 完整质量门与 Windows 构建通过），本任务交付内容已随该版本交付。
- 归档依据：归档时工作区无未提交代码改动，本记录引用的实现提交均为上述发布提交的祖先提交。
- 未执行项（实机客户端／原生窗口视觉／真实覆盖安装验收等）不伪造为已验收，转为发布后
  跟进事项，见 `docs/plans/CURRENT.md` 的 Post-release Follow-ups。
