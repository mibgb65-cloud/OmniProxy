# TASK-20261020-010: 构建 Beta.13 本地生产安装包

## Status

Completed

## Created

2026-10-20

## Request

用户要求 Google 自动接入优化完成后，构建新的 Beta 生产版本，供其自行本地更新。

## Goal / Scope

- 同步版本为 `2.0.0-beta.13`，编写准确的随包发行说明。
- 包含已提交的 Google 订阅模型自动同步和安全接入优化，继承 Beta.12 已发布到本地的功能。
- 冻结提交，只导出受跟踪源码，在隔离目录运行原发布脚本、完整质量门和 NSIS 打包。
- 校验 Windows 原生版本、production 标签、SHA256 与来源，交付独立的 Beta.13 资产目录。

## Out of Scope

不混入仍在进行的其他任务及未跟踪实验工具，不自动安装、停止或替换生产程序，
不读取真实配置或凭据，不创建发布标签、推送或远端发布，不覆盖旧安装包。

# Analysis

当前 Go／前端／Windows 完整版本为 Beta.12，最新本地安装器同为 Beta.12。
Google 优化已提交于 `161fb82`，任务归档于 `c1eaab6`。
工作区另有其他任务的未提交内容，应与生产构建来源隔离，不能误将实验改动纳入安装器。

已确认沿用现有 Windows 数字版本 `2.0.0.0`、NSIS `2.0.0` 与产品安装身份。
NSIS 已安装，Beta.13 目标发布目录尚不存在；使用 `git archive` 隔离源码，不创建额外工作树。

# Implementation Plan

1. 核对发布规范、原有脚本、版本读取测试及已有资产／生产程序的保护基线。
2. 更新版本与发行说明，验证后独立提交并冻结来源。
3. 导出冻结提交的受跟踪源码，隔离执行既有完整验证和生产打包流程。
4. 校验安装包、来源、数字版本与 SHA256，确认原文件／进程未受影响。
5. 回填结果、更新看板并归档，交付安装包供用户手动更新。

# Progress

- 已同步 Go、前端和 Windows 完整版本为 Beta.13；数字基础版本与安装身份不变。
- 已新增随包发行说明和三种运行版本读取回归，定向 Prettier 与 32 项测试通过。
- 已只读保存 15 个旧资产／已安装程序文件、1 个生产进程的摘要／启动时刻基线；未读取真实配置或凭据。
- 构建源已冻结为 `3b7ab6285ba5f2ab1d1d5d7ef6cdd8f49925392c`，
  已确认 Google 实现提交为其祖先，只导出受跟踪文件到 `tmp/local-beta13-20261020/source/`。
- 初次来源核对通过，未导出未提交网关启动文件、Codex 登录同步试验或 `cmd/` 工具。
- 原发布脚本已退出码 0 完成完整质量门、production 构建、NSIS 打包和内置资产校验。
- 构建后再次核对 1290 项来源，只有 Windows 资源 JSON 的 CRLF／LF 等价差异；无内容不一致或混入的试验文件。
- 独立原生版本、生产标签、SHA256、签名和保护基线校验通过；15 个旧文件及原生产进程保持不变。

# Implementation Result

## Changes

- Go、前端及 Windows 完整版本同步为 Beta.13，保留数字资源 `2.0.0.0`、NSIS `2.0.0` 和既有安装产品身份。
- 新增 Beta.13 随包发行说明与三种版本格式读取回归；覆盖 Google 自动同步、手动限选、默认模型及外部修改保护，保留所有历史说明。
- 从冻结提交只导出受跟踪源码，隔离完成原有完整验证及 Windows amd64 NSIS 生产打包。
- 输出安装器及同名 SHA256 至独立 Beta.13 目录；未覆盖旧资产，不包含工作区未提交的网关启动、Codex 登录试验及 `cmd/` 工具。
- 未运行安装器、停止生产进程、修改真实配置／凭据或发布到远端。

# Verification

- PASS：`scripts/set-release-version.ps1 -Version v2.0.0-beta.13`，完整版本及数字版本一致。
- PASS：版本相关前端文件定向 Prettier，发行说明读取 32 项测试。
- PASS：冻结源码内执行 `scripts/build-release.ps1 -Version v2.0.0-beta.13 -OutputDirectory <workspace>/release-assets/v2.0.0-beta.13`，退出码 0。
- PASS：脚本内 `task --force verify`：格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、前端 114 文件／674 用例、Wails 绑定和 Windows production 构建。
- PASS：脚本内 `task --force package`：production 重建、官方 WebView2 bootstrapper 和 UTF-8 NSIS 安装器生成。
- PASS：脚本内及独立 Windows 原生版本 API：应用 `2.0.0-beta.13`，安装器 `2.0.0`，数字资源均 `2.0.0.0`，ProductName／CompanyName 均为 OmniProxy。
- PASS：`go version -m`：`production`、`windows/amd64`、`CGO_ENABLED=0`，无 `updateqa` 标签。
- PASS：构建前后 1290 项源码／测试／依赖／版本与冻结 Git blob 匹配；资源 JSON 仅换行等价，无未提交实验文件。
- PASS：发布资产格式及实际 SHA256 匹配，目录仅包含安装器和同名 SHA256 文件。
- PASS：旧资产及已安装程序共 15 个文件摘要／大小未变；原生产进程 PID 与启动时刻未变。
- Authenticode：`NotSigned`，本地未签名包不被表述为签名发行。
- 非阻断提示：保留既有 jsdom scrollTo、Vite 大 chunk／Tailwind 耗时及跨平台 Task 的 uname／tail 探测提示；原脚本最终成功，不为打包顺带改动无关构建逻辑。
- 未执行：实际覆盖安装、真实 Agent 重载／推理及生产安装版重启，交由用户本地验收。

# Artifact

- 安装器：`E:/go/new/wails3-v/OmniProxy/release-assets/v2.0.0-beta.13/OmniProxy-Setup-v2.0.0-beta.13-windows-amd64.exe`
- SHA256 文件：安装器同名追加 `.sha256`。
- 大小：19,448,349 bytes。
- SHA256：`86eaa6c00ce058578c36e0c0a48510ebea4d27f84722b9cd730798774d7d9adb`。
- 签名：NotSigned。
- 来源、构建日志、1290 项来源验证及独立产物／保护基线摘要在忽略目录 `tmp/local-beta13-20261020/`。

# Git

`3b7ab6285ba5f2ab1d1d5d7ef6cdd8f49925392c` —
`chore(release): 准备 Beta.13 本地生产安装包`。

只提交本任务版本、说明与测试；其他任务的未提交改动不纳入构建或提交。
仅本地提交，不推送、不创建发布标签；产物验证及归档另作文档提交。

# Documentation

已新增 Beta.13 随包说明，更新 CURRENT 的当前本地版本和任务概览，任务完整记录归档于 completed/2026-10。

# Deviations

None.

# Result

Beta.13 Windows amd64 production 本地安装器及 SHA256 已生成并独立验证，
包含已提交的 Google 自动同步优化和 Beta.12 既有功能，可交由用户手动更新。
旧安装包、已安装生产程序和运行中的生产进程保持原样；未签名、未自动安装或公开发布。

# Completed

2026-10-20（Asia/Singapore）
