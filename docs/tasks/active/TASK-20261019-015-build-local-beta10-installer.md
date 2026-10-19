# TASK-20261019-015: 构建 Beta.10 本地生产安装包

## Status

In Progress

## Created

2026-10-19

## Request

构建新的 Beta 生产版本，供用户自行进行本地更新。

## Goal

交付包含额度查询／凭据锁修复的 Windows amd64 production Beta.10 安装包及校验信息，不自动安装或远端发布。

## Scope

- 按现有发布约定统一 Beta.10 版本和本地更新说明。
- 从已提交源码隔离构建，执行 `task verify`、NSIS 打包与资产／版本校验。
- 输出到 `release-assets/v2.0.0-beta.10/`，记录来源提交及 SHA256。

## Out of Scope

- 不自动更新、停止或替换现有生产／开发程序，不改动真实配置与凭据。
- 不推送、不打远端标签或创建公开发行；无证书时不宣称已签名。

# Analysis

当前本地安装包最新为 2.0.0-beta.9，本轮修复提交为 `ab6f5a6`（任务归档 `422fe4c`）。递增为 2.0.0-beta.10，沿用既有机器级安装身份和目录；Go／前端／ProductVersion 使用完整版本，Windows 数字版本与 NSIS 保持 2.0.0.0／2.0.0。

采用现有 `scripts/build-release.ps1`，从本任务版本提交导出的隔离源码运行，不覆盖主工作区 bin，不引入新脚本、依赖或打包捷径。脚本强制运行完整门禁与 NSIS，检查原生元数据并生成 SHA256；本机无签名时明确标记未签名。只提交本任务文件，已有未跟踪任务记录不纳入。

# Implementation Plan

1. 检查版本映射、更新说明及本地隔离打包流程。
2. 更新版本、测试预期及内置更新说明，提交本次构建源。
3. 隔离执行完整门禁、NSIS 打包和资产校验。
4. 同步 CURRENT、实际结果与任务归档，交付安装包路径。

# Implementation Result

已同步 Go／前端／Windows ProductVersion 为 2.0.0-beta.10，增加本地随包说明及普通版本、v 前缀、构建元数据三种读取回归；待隔离构建与资产验证。

# Verification

- PASS：`scripts/set-release-version.ps1 -Version v2.0.0-beta.10`，完整／数字版本映射一致。
- PASS：前端版本文件及随包说明回归的定向 Prettier 检查。
- PASS：`pnpm exec vitest run src/features/about/current-release.test.ts`，23 项测试。
- 待执行：隔离源码完整门禁、NSIS 打包与资产验证。

# Documentation

待更新。

# Git

待本地提交；不推送。

# Result

待完成。
