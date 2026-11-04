# TASK-20261104-002: 构建包含 Wails v3.1.0 的本地 Beta 生产安装包

## Status

Completed

## Created

2026-11-04

## Request

无需再测试，提交代码，然后构建新的 Beta 生产版本，供用户本地更新，并完成任务 Markdown 更新。

## Goal

为已升级至 Wails `v3.1.0` 的源码分配下一本地 Beta 版本，同步版本元数据并提交，构建可供本地覆盖更新的 Windows amd64 生产安装包，记录实际产物、摘要与验证边界。

## Scope

- 核对当前版本及既有本地 Beta 打包流程，更新版本和内嵌发行说明。
- 提交本任务源码与版本修改，再执行生产构建及 NSIS 打包。
- 仅检查产物元数据、依赖版本及摘要，不重新运行测试或完整质量门。
- 补齐并归档本任务记录，更新 CURRENT。

## Out of Scope

- 重跑上一任务已经通过的测试、lint、完整 `task verify`。
- 自动安装、启动或停止用户应用，修改真实用户数据。
- 打标签、推送远端、公开发布或修改签名信任设置。
- 修改或提交已有无关未跟踪文件。

# Analysis

- 当前应用版本为 `2.2.2`；Wails 升级实现及归档已分别提交为 `b74dd234`、`a317fc1e`。
- 下一本地版本采用 `v2.2.3-beta.1`；本地标签及产物目录均无此版本。Windows 数字版本映射为 `2.2.3.0`，NSIS 基础版本 `2.2.3`。
- 默认发布脚本包含完整质量门，不能直接调用后违反用户“无需再测试”的要求；复用现有版本及打包任务，记录本次跳过测试的明确授权。
- 使用 `task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.1/bin`，明确选择生产标签，并将应用 EXE 输出到隔离目录；NSIS 固定输出 `bin/OmniProxy-amd64-installer.exe`，随后复制到空的新版本资产目录。
- NSIS 已安装。本次沿用既有未签名本地构建方式，不读取私钥、不修改证书信任。已安装应用仍在运行，打包不停止该实例，也不覆盖其安装目录。

# Implementation Plan

1. 确认版本与打包入口，将本任务更新为 In Progress。
2. 同步版本及发行说明，检查差异并提交。
3. 执行生产打包，检查安装包与摘要，记录实际结果和提交号。
4. 完成任务记录与 CURRENT，归档并提交文档。

# Implementation Result

- 应用与前端版本已更新为 `2.2.3-beta.1`，Windows 数字版本为 `2.2.3.0`，NSIS／构建配置基础版本为 `2.2.3`。
- 已新增双语内嵌发行说明 `docs/releases/v2.2.3-beta.1.md`，明确本地包、未重复测试、未签名及人工更新边界；本次无新增数据库迁移。

## Build and Artifacts

- 源码提交：`8399d382a43fceba70b4e6e3d0437d511b929f46`。
- 构建命令：`task --force package DEV=false BIN_DIR=tmp/local-v2.2.3-beta.1/bin`，退出码 0。
- 过程：Go 模块整理、绑定／图标生成、前端生产编译、Go `-tags production` 编译、NSIS 打包；未调用 `verify`、lint 或测试任务。
- 日志：`tmp/local-v2.2.3-beta.1/package.log`。

| 产物 | 大小 |
| --- | --- |
| `release-assets/v2.2.3-beta.1/OmniProxy-Setup-v2.2.3-beta.1-windows-amd64.exe` | 20,555,019 bytes |
| 同目录同名 `.exe.sha256` | 安装器 SHA256 侧车 |
| `tmp/local-v2.2.3-beta.1/bin/omniproxy.exe` | 23,561,216 bytes |

安装器 SHA256：

```text
e11b86bb53ed1e7605afeecaa8e8d5c8cfe983ce7b877ffebb825441b92e5a61
```

生产应用 EXE SHA256：

```text
6b5aa13d85ab4821048ccf4f2bf3de7688df94fd7b605908d2ac8cc49a5655e7
```

## Limits

- 安装包 `Get-AuthenticodeSignature` 状态为 `NotSigned`；未签名，可能出现未知发布者或 SmartScreen 提示。
- 未安装、未停止或启动应用；原有已安装实例保持运行，用户应从托盘完全退出后再覆盖更新。
- 未打标签、未推送、未上传或公开发布，未修改证书信任和真实用户数据。
- Go SDK 继续使用 `1.26.2`，不将本次打包扩展为工具链升级或原生界面回归。

# Progress

- 已确认版本、现有脚本和安装器输出路径；版本提交完成后再开始构建。
- 已执行现有版本映射脚本，随后恢复 Windows 元信息文件的原有缩进与 LF，避免 PowerShell 序列化带来无关格式差异。
- 版本与发行说明已提交为 `8399d382a43fceba70b4e6e3d0437d511b929f46`，随后开始执行独立生产打包任务；未运行测试或 `task verify`。
- 生产编译与 NSIS 一次通过，生成的新版本目录只包含安装器及摘要文件；运行中的已安装实例未改动。
- 已完成 Windows 原生版本资源、Go 构建信息、摘要与未签名状态核对。

# Verification

- 单测、集成测试、完整质量门：按用户明确要求不再执行；沿用上一 Wails 升级任务的结果，不冒称本版本重新通过完整质量门。
- PASS：生产构建与 NSIS 打包，退出码 0；日志中无测试或完整质量门任务调用。
- PASS：`go version -m` 确认 Wails `v3.1.0`、`production` 标签、Go `1.26.2`、Windows amd64。
- PASS：`scripts/test-windows-metadata.ps1` 仅作为只读产物元数据校验，核对应用与安装器的原生版本资源：应用 `ProductVersion=2.2.3-beta.1`，安装器基础版本 `2.2.3`，数字版本均为 `2.2.3.0`，公司／产品身份保持 OmniProxy。此步骤不运行单测或启动 EXE。
- PASS：`scripts/verify-release-assets.ps1` 核对目录仅有安装器及 SHA256 侧车，摘要格式及独立复算一致。
- PASS：仅本任务文件纳入提交，版本映射、UTF-8 无 BOM／LF 及 `git diff --check` 检查通过。
- 保留既有 Taskfile 平台探测 `uname`／`tail` 缺失与 Vite 大 chunk 非阻断提示，未做无关构建重构。

# Documentation

- 已更新双语版本发行说明、CURRENT 与本任务记录。
- 已在本记录写明实际构建产物、摘要、源码提交、用户授权跳过测试及未签名／未安装边界。

# Deviations

按用户要求跳过重复测试；生产构建仍执行正常编译和必要的产物检查。

# Git

```text
8399d382a43fceba70b4e6e3d0437d511b929f46 chore(release): 准备本地 v2.2.3-beta.1 生产安装包
```

已先提交再构建，未打标签、未推送远端。

# Result

已按要求先提交版本源码，再生成包含 Wails `v3.1.0` 的本地 `v2.2.3-beta.1` Windows amd64 生产安装包。未重复测试，必要产物检查通过，任务记录已补齐并归档，供用户手动覆盖更新。

# Completed

2026-11-04（Asia/Singapore）
