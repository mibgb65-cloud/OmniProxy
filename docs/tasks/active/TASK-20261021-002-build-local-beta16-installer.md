# TASK-20261021-002: 构建 Beta.16 本地 production 安装包

## Status

In Progress

## Created

2026-10-21（Asia/Singapore）

## Request

用户要求完成修复后构建新的 Beta 生产版本，由用户自行进行本地更新。

## Goal

交付包含 Google 订阅错误分类修复的 Windows amd64 production Beta.16 安装包，验证质量门、来源、原生版本及 SHA256。

## Scope

- 完整版本递增至 `2.0.0-beta.16`，保持 Windows 数字版本及安装身份兼容。
- 新增准确的发行说明，冻结源码后隔离构建、打包并校验新资产。
- 记录产物、验证结果及已知限制，更新当前版本看板。

## Out of Scope

- 不自动安装、停止或替换现有生产应用，不修改真实账号、凭据、聊天或客户端配置。
- 不创建／推送发布标签，不公开发布，不覆盖旧发布资产。
- 不提交既有 Atria 草稿或其他无关变更，不执行真实 Google 推理。

# Analysis

- 当前版本和最新本地包为 Beta.15，目标 Beta.16 尚不存在。
- Google 修复提交为 `f214c79`，只纠正普通端点 HTTP 400 被误报成 401，保留真实认证和限流语义；不宣称真实 Google 授权或推理已恢复。
- 沿用 `scripts/build-release.ps1`：版本核对 → `task --force verify` → production 构建／NSIS → 原生元数据与资产校验。
- 源码先提交冻结，仅通过 `git archive` 导出受跟踪文件到独立目录；不复制真实配置、凭据或无关未跟踪文件。
- 构建前保存旧发布资产、当前安装文件及进程的只读基线，完成后复核；生产安装验收留给用户。

# Implementation Plan

1. 保存保护基线，更新版本、发行说明并提交明确的构建源。
2. 从冻结源码隔离构建，执行既有完整发布脚本和必要补充验证。
3. 核对新资产来源／摘要／原生版本及 production 编译参数，复核保护基线，归档并交付。

# Implementation Result

- 版本与 Windows 完整资源已更新至 `2.0.0-beta.16`，数字版本保持 `2.0.0.0`，安装身份未变。
- 已补充发行说明，准确区分 Google 错误分类修复与真实认证／限流恢复，保留数据和手动更新说明。
- 构建保护基线已保存至忽略目录 `tmp/local-beta16-20261021/`：20 个旧发布资产、3 个安装文件及 1 个运行中的生产进程。
- 待从提交后的冻结源码隔离执行完整发布构建。

# Verification

- `scripts/set-release-version.ps1 -Version v2.0.0-beta.16`：PASS。
- `go test ./internal/version`：PASS。
- 前端 `package.json` Prettier 检查及发行说明 38 项测试：PASS。
- `git diff --check`：PASS；Windows 版本资源脚本的 CRLF／LF 提示不涉及内容差异。
- 完整质量门、production 构建／NSIS、资产与来源检查：待执行。

# Git

待提交。

# Result

待完成。
