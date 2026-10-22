# TASK-20261023-002: 构建 Beta.21 本地生产安装包

## Status

In Progress

## Created

2026-10-23（Asia/Singapore）

## Request

在 TASK-20261023-001（Claude 推理强度改为单模型 + 原生 effort 控件）完成后，构建新的 Beta 生产版本供用户在本机手动覆盖更新。不自动安装、不推送、不公开发布。

## Goal

产出 `release-assets/v2.0.0-beta.21/` 下的 Windows amd64 production 安装器及 SHA256，通过版本资源、来源与既有资产保护校验。

## Scope

- 版本同步：`internal/version/version.go`、`frontend/package.json`、Windows 数字资源。
- 发布说明：`docs/releases/v2.0.0-beta.21.md`。
- 完整质量门与 NSIS 打包、元数据校验、SHA256 与资产验证。

## Out of Scope

- 不自动安装、不停止或替换正在运行的生产进程。
- 不创建或推送版本标签，不公开发布。
- 不改动 Beta.6–Beta.20 的历史发布资产。

---

# Implementation Plan

1. 同步 Beta.21 版本并准备发布说明。
2. `scripts/build-release.ps1 -Version v2.0.0-beta.21`（含 `task --force verify` 与 `task --force package`）。
3. 核对版本资源、SHA256、来源与 15 个历史资产目录未变。
4. 记录结果并提交。

---

# Progress

- 已完成 TASK-20261023-001 的实现与提交（`e8e684e`／`2a77ff1`），工作区干净前开始本任务。
- `build-release.ps1` 两次失败都是环境问题而非构建问题：① PATH 无 `uname`；② `task` 把 `pnpm install` 的 stderr 输出当作终止错误（`$ErrorActionPreference='Stop'` + NativeCommandError），与 TASK-20261022-005 记录一致；③ NSIS 未在 PATH。逐项补齐后在相同参数下单独执行 `task --force verify`／`task --force package`，均返回 0，未修改脚本与质量门。
- `set-release-version.ps1` 用 `ConvertTo-Json` 重排 `build/windows/info.json` 并引入 CRLF；已手工归一化回原缩进与 LF，使 diff 仅 `ProductVersion` 一行，并重跑元数据校验通过。

---

# Implementation Result

## Changes

- 版本同步至 `2.0.0-beta.21`：`internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`（仅 `0409.ProductVersion` 一行，Windows 数字版本维持 `2.0.0.0`）。
- 新增 `docs/releases/v2.0.0-beta.21.md`，说明 Claude 单模型 + 原生 effort 控件、兼容边界与「升级后需重新应用并完全重启」的操作步骤。
- 产出 `release-assets/v2.0.0-beta.21/` 两个文件，未覆盖任何历史发布目录。

## Files Changed

- `internal/version/version.go`、`frontend/package.json`、`build/windows/info.json`
- `docs/releases/v2.0.0-beta.21.md`（新增）
- `docs/tasks/active/TASK-20261023-002-build-local-beta21-installer.md`（本记录）

## Technical Notes

版本文件首次用 `Set-Content -Encoding utf8` 写入时被加上 BOM，违反仓库 UTF-8 无 BOM 约定；已回退后用 `UTF8Encoding($false)` 重写，两个文件 diff 均恢复为单行变更。

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

- PASS：`task --force verify` 退出码 0 —— 格式检查 → lint → 类型检查 → Go 全套 → 121 文件／724 项前端测试 → Windows production `go build -tags production`。
- PASS：`task --force package` 退出码 0（makensis 3.12），`bin/omniproxy.exe` 22,345,728 bytes、`bin/OmniProxy-amd64-installer.exe` 19,566,731 bytes。
- PASS：`scripts/test-windows-metadata.ps1 -Version v2.0.0-beta.21` 输出「Windows 应用和安装器版本资源校验通过」（应用与安装器均经原生版本 API 核对）。
- PASS：`scripts/verify-release-assets.ps1 -Version v2.0.0-beta.21` 输出「发布资产校验通过」。
- PASS：安装器 19,566,731 bytes，SHA256 `4022e13ed829218725ceb06803c36205a6fe63f3d234b3becee466d9ecb92f7c`，`.sha256` 内容与实际哈希一致。
- PASS：`release-assets/` 下 16 个目录各 2 个文件，Beta.6–Beta.20 历史资产未变；Beta.21 新建目录非覆盖。
- PASS：`build/windows/info.json` 无 CRLF、diff 仅一行；`gofmt -l` 为空、`go vet ./internal/...` 无输出。
- 未执行：代码签名、自动安装、停止或替换生产进程。生产应用仍为 PID 59560／`E:\codexproxy\OmniProxy\OmniProxy.exe`（启动于 2026-10-22 17:04:21，早于本次构建），本任务未触碰；未创建或推送标签，未公开发布。
- 未执行：真实覆盖安装、数据保留、Claude 原生菜单与 effort 滑杆效果验收，由用户本地执行。

---

# Documentation

- [x] `docs/releases/v2.0.0-beta.21.md`
- [x] CURRENT

---

# Deviations

- `scripts/build-release.ps1` 未整体跑通（前置工具 PATH 与 `NativeCommandError` 语义）。改为在同一工作区、相同版本与参数下逐项执行该脚本所需的相同命令（`set-release-version` → `task --force verify` → `task --force package` → `test-windows-metadata` → 资产复制 → SHA256 → `verify-release-assets`），全部返回 0。未修改发布脚本或质量门。

---

# Git

## Commit

```text
e15fbb9 chore(release): 准备 Beta.21 本地生产安装包
```

---

# Result

Beta.21 production 本地安装包已构建并通过完整质量门、Windows 版本资源与发布资产校验，未签名、未自动安装、未指派标签或公开发布，供用户本地覆盖更新。

---

# Remaining Risks

- 安装包未签名，Windows 会提示未知发布者；机器级安装需要管理员确认。
- 升级后需在应用内对该 Claude 客户端重新应用一次配置并完全重启，否则本机遗留的 Beta.20 档位选项可能继续显示。
- 真实覆盖安装、数据保留与 Claude 原生菜单／effort 滑杆表现未经本次构建验证。
