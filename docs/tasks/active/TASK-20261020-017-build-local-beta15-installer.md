# TASK-20261020-017: 构建 Beta.15 本地 production 安装包

## Status

In Progress

## Created

2026-10-20

## Request

完成当前修复后构建新的 Beta production 版本，供用户自行本地更新。

## Goal

交付包含当前已提交改动的 Windows amd64 production Beta 安装包，验证版本、来源和 SHA256；不自动安装或公开发布。

## Scope

版本递增、发布构建与质量门、资产／来源／版本校验、隔离启动验证及交付记录。

## Out of Scope

- 不安装、替换或停止用户当前生产程序，不修改真实账号、凭据、聊天历史或客户端配置。
- 不上传发布、不推送 Git、不覆盖旧版本发布资产。
- 不提交既有 Atria 草稿或其他无关变更。

# Analysis

- 当前最新本地包为 Beta.14；Beta.15 目录不存在，沿用完整 SemVer／数字 Windows 版本映射，保留安装身份。
- 使用既有发布脚本、NSIS、原生版本与资产校验，源码先提交冻结，再用 `git archive` 导出受跟踪源码到独立目录。
- 纳入 TASK-014／015／016：额度展示与 Token 块、Google 额度持久化、River 队列／均衡与失败冷却、Claude Messages 长会话修复。
- 初步确认 18 个旧发布资产及一个在运行的 production 进程；记录保护基线后不停止、不覆盖或回滚该进程／安装目录。
- 真实安装覆盖及原聊天续接不冒充构建验证；本次仅交付用户手动更新。

# Implementation Plan

1. 核对版本／构建规则，保护旧资产和当前安装状态。
2. 更新版本并提交明确的构建源，使用现有 production 流程构建和验证。
3. 校验安装包、摘要和来源，进行隔离启动验证，交付文件并归档任务。

# Implementation Result

已同步应用／前端完整版本至 Beta.15，通过原脚本校验 Windows 数字资源映射；新增随包发布说明和三种版本格式读取回归。待冻结源码并完成隔离构建。

# Verification

- 保护基线已记录：18 个旧发布资产、3 个当前安装文件及 1 个 production 进程；均不由构建任务更改。
- `scripts/set-release-version.ps1 -Version v2.0.0-beta.15` 通过。
- 定向 Prettier 检查和 38 项当前发行说明测试通过，`internal/version` 测试通过。
- 完整质量门、NSIS 打包及产物校验待执行。

# Git

待本地提交。

# Result

待完成。
