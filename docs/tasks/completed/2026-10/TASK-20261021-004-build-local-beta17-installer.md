# TASK-20261021-004: 构建 Beta.17 本地 production 安装包

## Status

Completed

## Created

2026-10-21（Asia/Singapore）

## Request

用户要求构建新的 Beta 生产版本，由用户自行进行本地更新。

## Goal

交付包含 Gemini 工具 JSON Schema 方言修复的 Windows amd64 production Beta.17
安装包，验证完整质量门、构建来源、原生版本和 SHA256。

## Scope

- 完整版本递增到 `2.0.0-beta.17`，保持 Windows 数字版本和安装身份兼容。
- 添加准确的发行说明；冻结源码后隔离执行既有发布构建脚本。
- 核对新资产、编译参数、运行时版本及既有资产／安装文件保护基线，归档交付。

## Out of Scope

- 不自动安装、停止或替换正在运行的生产程序。
- 不修改真实账号、凭据、客户端配置或聊天，不执行真实推理。
- 不公开发布、推送 Git 或创建发布标签，不覆盖旧发布资产。
- 不提交既有 Atria 草稿或其他无关内容。

# Analysis

- 当前版本和最新本地包为 Beta.16，目标 Beta.17 尚不存在。
- 修复提交 `f4ae785` 将 Gemini 工具定义从受限 `parameters` 改为 `parametersJsonSchema`，
  保留完整约束；同内核默认工具真实 A/B 400→200。进一步 Desktop 内核验收遇到 429，
  本次构建不宣称解除限流或完成真实桌面调用验收。
- 沿用 `scripts/build-release.ps1`，在冻结提交的 `git archive` 隔离源码中执行；
  不将未跟踪文件、真实配置或凭据复制到构建目录。
- 构建前保存旧发布资产、已安装文件及生产进程只读基线，完成后复核。

# Implementation Plan

1. 保存保护基线，同步版本和发行说明，检查并提交冻结构建源。
2. 导出受跟踪源码，运行完整发布脚本和必要补充验证。
3. 校验来源、产物及隔离运行时版本，复核保护基线，归档交付。

# Implementation Result

- 已保存保护基线：22 个旧发布资产、3 个安装文件及 1 个运行中的生产进程。
- 应用／前端完整版本与 Windows ProductVersion 同步为 `2.0.0-beta.17`，数字资源
  保持 `2.0.0.0`，沿用既有安装身份；发行说明准确记录 Schema 修复和独立 429 验证边界。
- 源码冻结为 `0bd390af47545943220c3018cfef7167cf82793d`，已确认包含 Gemini
  修复提交 `f4ae785`。仅用 `git archive` 导出受跟踪源码，Atria 未跟踪草稿不进入构建。
- 在 `tmp/local-beta17-20261021/source/` 使用独立 TEMP／TMP 运行完整发布脚本，
  完成质量门、production 构建和 NSIS 打包；不复制真实配置或凭据，不覆盖旧资产。
- 最终安装包写入独立 `release-assets/v2.0.0-beta.17/`，已完成原生版本、编译参数、
  源码来源、SHA256、隔离运行时版本与保护基线复核。

## Artifacts

- 安装器：
  `E:/go/new/wails3-v/OmniProxy/release-assets/v2.0.0-beta.17/OmniProxy-Setup-v2.0.0-beta.17-windows-amd64.exe`
- 校验文件：同名追加 `.sha256`。
- 大小：19,537,034 bytes。
- SHA256：`22d4459d340ebcabf43e7f5f8aa9cf569d3d3cfe2a2f2e6f7a4644b8cb271f0e`。
- 签名状态：`NotSigned`。
- 构建日志、源码归档和来源清单、隔离启动与保护基线记录保存在忽略目录
  `tmp/local-beta17-20261021/`，不作为公开发布资产。

# Verification

- `scripts/set-release-version.ps1 -Version v2.0.0-beta.17`：PASS。
- `go test ./internal/version`：PASS。
- 前端 `package.json` Prettier 检查、版本发行说明与关于页 45 项定向测试：PASS。
- `git diff --check`：PASS；Windows 资源脚本的 CRLF／LF 提示无内容影响。
- 冻结源码的版本／数字资源／发行说明及 UTF-8 无 BOM 核对：PASS。
- 隔离源码内 `go test ./cmd/...`、`go vet ./cmd/...`：PASS，未执行登录试验命令。
- `scripts/build-release.ps1 -Version v2.0.0-beta.17 -OutputDirectory <workspace>/release-assets/v2.0.0-beta.17`：
  PASS，退出码 0。
- `task --force verify`：PASS，包括格式、Go vet／staticcheck、ESLint、TypeScript、
  Go 测试、118 个文件／703 项前端测试及 Windows production 构建。
- `task --force package`：PASS，production 重建及 NSIS 打包通过，未运行安装器。
- 构建后 1,734 项源码文本／配置与冻结 Git blob 核对：PASS；
  仅 `build/windows/info.json` 为 CRLF／LF 等价差异，没有实际内容变化或私有文件混入。
- 新资产目录仅含安装器及同名 SHA256，文件名、大小和摘要校验：PASS。
- Windows 原生 API 验证应用完整版本 Beta.17、安装器数字版本 `2.0.0`、
  双方数字资源 `2.0.0.0` 及产品身份：PASS。
- `go version -m` 验证 `production`、`windows/amd64`、`CGO_ENABLED=0` 且无 `updateqa`：
  PASS。
- 最终 production 二进制在独立 test profile、数据／配置／WebView 目录及 43117／43917
  端口启动，`/healthz` 返回 `2.0.0-beta.17`／test：PASS。
  验收后仅关闭路径匹配的自有进程，未发送真实推理。
- 22 个旧发布资产及 3 个安装文件的大小／SHA256、原生产 PID 19884 路径和启动时间：
  两次复核均保持不变。
- 既有 jsdom `scrollTo`、Vite 大 chunk／插件耗时提示不影响门禁通过，没有为消除提示引入无关修改。
- 真实覆盖安装、数据保留及 Gemini 桌面会话调用仍由用户本地更新后验收。

# Documentation

已更新 Beta.17 发行说明与 CURRENT；不改变产品范围、架构或技术栈。

# Deviations

None.

# Git

## Source Commit

`0bd390af47545943220c3018cfef7167cf82793d chore(release): 准备 Beta.17 本地生产安装包`

仅提交本任务文件；未跟踪 Atria 草稿保持原样。未推送或创建发布标签。

# Result

已交付包含 Gemini 工具 JSON Schema 修复的 Beta.17 Windows amd64 production 本地
安装包和 SHA256。完整质量门、打包、构建来源／原生版本／摘要与隔离启动验证通过；
旧发布资产和已安装文件、原生产进程未变。

未自动安装或公开发布，安装包未签名；Google 429 限制不属于本版承诺解决范围，
实际覆盖更新与模型调用留给用户验收。

# Completed

2026-10-21（Asia/Singapore）
