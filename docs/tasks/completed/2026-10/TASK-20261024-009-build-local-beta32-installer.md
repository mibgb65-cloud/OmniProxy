# TASK-20261024-009: 构建 Beta.32 本地生产安装包

## Status

Completed（2026-10-27 随 v2.0.0 正式发布归档）

## Created

2026-10-24 11:20

## Request

用户要求：构建新的 Beta 生产版本，供本机覆盖更新。

## Goal

产出 `release-assets/v2.0.0-beta.32/` 安装器与 SHA256，通过完整质量门、Windows 版本资源与发布资产校验。

## Background

- 本版功能差异是 TASK-20261024-008：Codex 入口改回 v1 形态的透明转发（路径／头部／正文原样、WebSocket 真正桥到上游），登录改由网关拥有（`codex_auth_json` 池账号、自行续期并回写、一键配置把池账号写成客户端登录且可恢复）。
- 数据库结构变化：迁移 042 给 `codex_settings` 增加 `upstream_base_url`。
- 工作区仍含其他在办任务的未提交改动（模型目录、River 图片等），随包打包但不代为提交。

## Scope

- 版本同步至 `2.0.0-beta.32` 并新增 `docs/releases/v2.0.0-beta.32.md`。
- 完整质量门 → 打包 → 版本资源 → 资产与 SHA256 校验。
- 更新 `docs/plans/CURRENT.md`。

## Out of Scope

- 不提交其他在办任务的改动；不自动安装、不打标签、不推送。

---

# Implementation Result

- 版本同步至 `2.0.0-beta.32`（`internal/version/version.go`、`frontend/package.json`，
  由 `scripts/set-release-version.ps1` 同步 Windows 数字版本 2.0.0.0 到
  `build/config.yml`、`build/windows/info.json`、`build/windows/nsis/wails_tools.nsh`）。
- 新增发布说明 `docs/releases/v2.0.0-beta.32.md`。
- 前端三个 `CodexSettings` 测试夹具补上 `upstreamBaseUrl`（Go 结构体新增字段导致的
  类型不匹配），并执行 Prettier 格式化。
- 首次打包后按 v1 对照补齐三处转发缺口，随后重建安装包：
  1. WebSocket 握手透传客户端头（`originator`／`session_id`／`User-Agent`／
     `OpenAI-Beta`），握手自身管理的头与 `Origin` 由网关重建；
  2. WS 握手被明确拒绝（401／429／5xx）时换同池账号重拨，取到上游后才应答客户端；
  3. 入口同时接受 `…/backend-api/codex/…`（v1 的另一种客户端基址）。
  另补齐 v1 会删除的 `X-Goog-Api-Key`。
- `release-assets/v2.0.0-beta.32/`：安装器 19,779,433 bytes，
  SHA256 `bacb7db125caac8258bdac388335c79d6178173a7b1531b09d349ab24f8315d2`。
- 发布目录总数 27（Beta.31 之前 26 个），历史资产未变。

# Verification

- PASS：`task --force verify` exit 0（126 文件 / 750 项前端测试、Go 全部包、
  Windows production 构建；日志 `tmp/verify-beta32.log`）。
- PASS：`task --force package` exit 0（日志 `tmp/package-beta32.log`）。
- PASS：`scripts/test-windows-metadata.ps1` 输出「Windows 应用和安装器版本资源校验通过」。
- PASS：`scripts/verify-release-assets.ps1` 输出「发布资产校验通过：v2.0.0-beta.32」。
- 未执行：代码签名、自动安装、停止或替换生产进程、打标签、推送。
- 环境说明：本机 `task` 需要 `C:\Program Files\Git\usr\bin`（`uname`）与
  `E:\nodejs`（`pnpm`）在 PATH 中，打包还需要 `${env:ProgramFiles(x86)}\NSIS`；
  `scripts/build-release.ps1` 在本机因 `$ErrorActionPreference='Stop'` 被 task 的 stderr
  中断，本次改为等价的分步命令执行。

# Result

Beta.32 production 本地安装包已构建并校验通过，未签名、未自动安装，供用户本地覆盖更新。

# Git

## Commit

```text
90681e0 docs: 同步 Codex 透明转发决策、产品文档与 2026-10 任务记录
```

版本映射随后续构建演进为 2.0.0-beta.35（9c7d61b）。

---

---

# Archival (v2.0.0)

- 原状态：In Progress（安装包已构建并校验，等待用户本地覆盖安装验收）
- 2026-10-27：v2.0.0 正式版已公开（标签源码 `c9c48b4a31988a6cc3fa2f4b1d423e61bd10f71b`，
  Actions 完整质量门与 Windows 构建通过），本任务交付内容已随该版本交付。
- 归档依据：归档时工作区无未提交代码改动，本记录引用的实现提交均为上述发布提交的祖先提交。
- 未执行项（实机客户端／原生窗口视觉／真实覆盖安装验收等）不伪造为已验收，转为发布后
  跟进事项，见 `docs/plans/CURRENT.md` 的 Post-release Follow-ups。
