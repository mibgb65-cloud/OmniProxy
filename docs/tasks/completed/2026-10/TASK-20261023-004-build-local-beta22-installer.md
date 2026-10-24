# TASK-20261023-004: 构建 Beta.22 本地生产安装包（River 视觉输入）

## Status

Completed

## Created

2026-10-23 18:40（Asia/Singapore）

## Request

TASK-20261023-003（River 图片输入接入）完成后，构建新的 Beta 生产版本供用户在本机手动覆盖更新。不自动安装、不推送、不公开发布。

## Goal

产出 `release-assets/v2.0.0-beta.22/` 下的 Windows amd64 production 安装器及 SHA256，通过版本资源、来源与既有资产保护校验。

## Scope

- 版本同步：`internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`。
- 发布说明：`docs/releases/v2.0.0-beta.22.md`。
- 完整质量门与 NSIS 打包、元数据校验、SHA256 与资产验证。

## Out of Scope

- 不自动安装、不停止或替换正在运行的生产进程。
- 不创建或推送版本标签，不公开发布。
- 不改动 Beta.6–Beta.21 的历史发布资产。
- 不包含真实 River Key 的图片计费实测结论。

---

# Implementation Plan

1. 同步 Beta.22 版本并准备发布说明。
2. `scripts/build-release.ps1 -Version v2.0.0-beta.22`；若脚本受环境限制未整体跑通，按 TASK-20261023-002 记录的同一方式逐项执行等价命令。
3. 核对版本资源、SHA256、来源与 16 个历史资产目录未变。
4. 记录结果并提交。

---

# Progress

- 已在 TASK-20261023-003 完成后开始；工作区仅含该任务变更，无其他用户改动。
- 复核 TASK-20261023-002 记录的三个环境坑（`uname` 缺失、`task` 对 `pnpm install` stderr 报 NativeCommandError、NSIS 未在 PATH），预先准备规避。

---

# Implementation Result

## Changes

- 版本同步至 `2.0.0-beta.22`：`internal/version/version.go`、`frontend/package.json`、
  `build/windows/info.json`（仅 `0409.ProductVersion` 一行，Windows 数字版本维持 `2.0.0.0`）。
- 新增 `docs/releases/v2.0.0-beta.22.md`，说明 River 图片输入、兼容边界与升级后两步操作。
- 产出 `release-assets/v2.0.0-beta.22/` 两个文件，未覆盖任何历史发布目录。

## Files Changed

- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `docs/releases/v2.0.0-beta.22.md`（新增）
- `docs/plans/CURRENT.md`、本记录

## Technical Notes

`set-release-version.ps1` 用 `ConvertTo-Json` 重排 `build/windows/info.json` 并引入 CRLF；
本轮同样手工归一化回原缩进与 LF，使 diff 仅 `ProductVersion` 一行。未修改发布脚本或质量门。
`build/config.yml` 与 `build/windows/nsis/wails_tools.nsh` 的数值版本本就一致，无 diff。

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

- PASS：`set-release-version.ps1 -Version v2.0.0-beta.22` 输出「已校验 v2.0.0-beta.22，Windows 数字版本 2.0.0.0」。
- PASS：`task --force verify` 退出码 0 —— 格式检查 → lint → 类型检查 → Go 全套 → 121 文件／724 项前端测试 → Windows production `go build -tags production`。
- PASS：`task --force package` 退出码 0（makensis 3.12），安装器 19,574,954 bytes。
- PASS：`scripts/test-windows-metadata.ps1 -Version v2.0.0-beta.22` 输出「Windows 应用和安装器版本资源校验通过」。
- PASS：`scripts/verify-release-assets.ps1 -Version v2.0.0-beta.22` 输出「发布资产校验通过」。
- PASS：安装器 19,574,954 bytes，SHA256 `6906f9b334aaa5f408b207658cf54418dee3e7fe59408c811ef2deb89ae78fb8`，`.sha256` 内容与实际哈希一致。
- PASS：`release-assets/` 下 17 个目录各 2 个文件，Beta.6–Beta.21 历史资产未变；Beta.22 为新建目录。
- PASS：`build/windows/info.json` 仅一行 diff、LF，无 BOM。
- 未执行：代码签名、自动安装、停止或替换生产进程；未创建或推送标签，未公开发布。

---

# Documentation

- [x] `docs/releases/v2.0.0-beta.22.md`
- [x] CURRENT

---

# Deviations

`scripts/build-release.ps1` 未整体跑通（与 TASK-20261023-002 同因）：`$ErrorActionPreference='Stop'`
把 `task` 转发 `pnpm install` 的 stderr 当作终止错误，以及 NSIS 未在 PATH。改为在同一工作区、
相同版本下逐项执行该脚本所需命令（`set-release-version` → `task --force verify` →
`task --force package` → `test-windows-metadata` → 资产复制 → SHA256 → `verify-release-assets`），
全部返回 0。未修改发布脚本或质量门。

---

# Git

## Commit

```text
90681e0 docs: 同步 Codex 透明转发决策、产品文档与 2026-10 任务记录
```

版本映射随后续构建演进为 2.0.0-beta.35（9c7d61b）。

---

# Result

Beta.22 production 本地安装包已构建并通过完整质量门、Windows 版本资源与发布资产校验，未签名、未自动安装、未指派标签或公开发布，供用户本地覆盖更新。

---

# Remaining Risks

- 安装包未签名，Windows 会提示未知发布者；机器级安装需要管理员确认。
- 升级后需在路由页对 River 账号重新「同步官方模型」，并在应用内对该 Claude 客户端重新应用配置，否则模型目录仍显示旧的纯文本元数据。
- 真实覆盖安装、数据保留与 Claude 端图片发送效果未经本次构建验证。
