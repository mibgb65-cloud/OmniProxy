# TASK-20261023-006: 构建 Beta.23 本地生产安装包（本地 Agent 图片目录）

## Status

Completed

## Created

2026-10-23 19:40（Asia/Singapore）

## Request

TASK-20261023-005（本地 Agent 客户端目录公布图片能力）完成后，重新构建 Beta 生产版本供本地覆盖更新。Beta.22 打包于该修复之前，不包含本次改动。

## Goal

产出 `release-assets/v2.0.0-beta.23/` 下的 Windows amd64 production 安装器及 SHA256，通过版本资源、来源与既有资产保护校验。

## Scope

- 版本同步：`internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`。
- 发布说明：`docs/releases/v2.0.0-beta.23.md`。
- 完整质量门与 NSIS 打包、元数据校验、SHA256 与资产验证。

## Out of Scope

- 不自动安装、不停止或替换正在运行的生产进程。
- 不创建或推送版本标签，不公开发布。
- 不改动 Beta.6–Beta.22 的历史发布资产。

---

# Implementation Plan

1. 同步 Beta.23 版本并准备发布说明。
2. `set-release-version.ps1` → `task --force verify` → `task --force package` → 元数据校验 → 资产复制 → SHA256 → 资产校验（沿用 TASK-20261023-004 已验证的逐项执行方式）。
3. 核对版本资源、SHA256 与 17 个历史资产目录未变。

---

# Progress

- 已在 TASK-20261023-005 完成后开始；其最终质量门 exit 0。

---

# Implementation Result

## Changes

- 版本同步至 `2.0.0-beta.23`：`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json`（仅 `0409.ProductVersion` 一行，Windows 数字版本维持 `2.0.0.0`）。
- 新增 `docs/releases/v2.0.0-beta.23.md`，说明本地 Agent 客户端图片目录、承接 Beta.22 的 River 图片转发、
  兼容边界与升级后三步操作。
- 产出 `release-assets/v2.0.0-beta.23/` 两个文件，未覆盖任何历史发布目录。

## Files Changed

- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `docs/releases/v2.0.0-beta.23.md`（新增）
- `docs/plans/CURRENT.md`、本记录

## Technical Notes

`set-release-version.ps1` 再次用 `ConvertTo-Json` 重排 `build/windows/info.json` 并引入 CRLF
（14 行改动）；按 TASK-20261023-004 的同样方式手工归一化回原缩进与 LF，diff 恢复为一行。
`build/config.yml` 与 `build/windows/nsis/wails_tools.nsh` 数值版本本就一致，无 diff。
未修改发布脚本或质量门。

---

# Verification

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

- PASS：`set-release-version.ps1 -Version v2.0.0-beta.23` 输出「已校验 v2.0.0-beta.23，Windows 数字版本 2.0.0.0」。
- PASS：`task --force verify` 退出码 0 —— 格式、lint、类型检查、Go 全套、121 文件／724 项前端测试、Windows production 构建。
- PASS：`task --force package` 退出码 0（makensis 3.12），安装器 19,577,322 bytes。
- PASS：`test-windows-metadata.ps1 -Version v2.0.0-beta.23` 输出「Windows 应用和安装器版本资源校验通过」。
- PASS：`verify-release-assets.ps1 -Version v2.0.0-beta.23` 输出「发布资产校验通过」。
- PASS：安装器 19,577,322 bytes，SHA256 `4baa576391909fdfbea3b826573621c48bf9a199d9fbd196f8d57a0eaacf216d`；
  独立重算与 `.sha256` 记录值一致。
- PASS：`release-assets/` 下 18 个目录各 2 个文件；Beta.6–Beta.22 历史资产未变（`git status --short release-assets` 为空）。
- PASS：`build/windows/info.json` 仅一行 diff、LF、无 BOM。
- 未执行：代码签名、自动安装、停止或替换生产进程；未创建或推送标签，未公开发布。

---

# Documentation

- [x] `docs/releases/v2.0.0-beta.23.md`
- [x] CURRENT

---

# Deviations

`scripts/build-release.ps1` 仍未能整体跑通（与 TASK-20261023-002／004 同因：`task` 转发
`pnpm install` stderr 触发 NativeCommandError，以及 NSIS 需显式前置 PATH）。按脚本所需命令逐项执行，
全部返回 0。未修改发布脚本。

---

# Git

## Commit

```text
90681e0 docs: 同步 Codex 透明转发决策、产品文档与 2026-10 任务记录
```

版本映射随后续构建演进为 2.0.0-beta.35（9c7d61b）。

---

# Result

Beta.23 production 本地安装包已构建并通过完整质量门、Windows 版本资源与发布资产校验，未签名、未自动安装、未指派标签或公开发布，供用户本地覆盖更新。本版包含 Beta.22 的 River 图片转发与本轮本地 Agent 客户端图片目录修复。

---

# Remaining Risks

- 安装包未签名，Windows 会提示未知发布者；机器级安装需要管理员确认。
- 升级后需对 River 账号重新「同步官方模型」，并重新应用 Claude 与 Harness／OpenCode／Pi／Goose 配置，目录才会出现图片能力。
- 真实覆盖安装、数据保留与各客户端实际发图／识别效果未经本次构建验证。
