# TASK-20261021-004: 构建 Beta.17 本地 production 安装包

## Status

In Progress

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
- 同步应用／前端版本为 Beta.17；发行说明准确记录 Schema 修复和独立 429 验证边界。
- 后续隔离构建和验证待执行。

# Verification

- `scripts/set-release-version.ps1 -Version v2.0.0-beta.17`：PASS。
- `go test ./internal/version`：PASS。
- 前端 `package.json` Prettier 检查、版本发行说明与关于页 45 项定向测试：PASS。
- `git diff --check`：PASS；Windows 资源脚本的 CRLF／LF 提示无内容影响。
- 完整质量门、production 打包和最终产物验证待执行。

# Documentation

更新发行说明与 CURRENT；不改变产品范围、架构或技术栈。

# Git

待记录构建源提交；仅提交本任务文件。

# Result

构建准备中。
