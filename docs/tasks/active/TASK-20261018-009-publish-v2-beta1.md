# TASK-20261018-009: 发布首个 v2 Beta

## Status

In Progress

## Created

2026-10-18（Asia/Singapore）

## Request

用户确认「可以，你做吧」，授权将已验收的 v2 main 推送到 `mibgb65-cloud/OmniProxy`，创建 `v2.0.0-beta.1` 标签，通过自动流程公开发布首个预发布版本。

## Goal

GitHub Pre-release 包含 CI 构建的 Windows amd64 安装器及匹配 SHA256，能通过实际下载校验；保留旧版 master 和 v1.3.1 Latest 状态。

## Scope

- 配置当前仓库 origin，仅正常推送新 main／版本标签，不强推或改写历史。
- 跟进 main CI、标签 Release 工作流；必要时修复首次远端构建的问题，不绕过质量门。
- 核对版本、发布说明、两项资产和下载校验、Beta／stable 查询及旧版状态。
- 记录真实远端链接、提交、验证和边界并归档；五份既有未跟踪任务记录不纳入提交。

## Analysis

发布候选 `1950b22`，应用版本 `2.0.0-beta.1`。安装生命周期及最终代码质量门已由 TASK-20261018-008 验收。当前本地没有 remote；先前只读核对远端默认分支为 master，没有 main 或 v2 标签，当前正式 Latest 为 v1.3.1，账号具备 push 权限。本次用户授权明确包含实际远端发布，扩展了上个任务「不发布远端」的范围。

再次核对确认：master 为 `56448d20ccd797bc32aae1c02a5e2f2030b8b6c8`，stable Release ID 为 `392117295`（v1.3.1），main 与目标标签不存在。仅推送当前仓库 main，未发现其提交历史包含 `.env`、PFX／P12 或常见 SSH 私钥文件；不纳入未跟踪的既有用户记录。

## Implementation Plan

1. 再核对远端引用、权限、候选和资产；提交发布准备记录。
2. 推送 main，跟进实际 CI，成功后创建并正常推送 v2.0.0-beta.1 标签。
3. 跟进 Release 全流程，下载最终资产并核对 SHA256／原生版本，检查预发布及 Latest／master 保留。
4. 更新任务和状态文档，提交归档并同步 main。

## Implementation Result

待完成。

## Verification

待完成。

## Git

待记录。

## Result

待完成。
