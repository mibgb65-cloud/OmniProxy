# TASK-20261013-002: 同步主分支并修复开发启动时序

## Status

In Progress

## Created

2026-10-13（Asia/Singapore）

## Request

我需要你处理一下这个 git，如果文档能提交就提交吧，先同步到最新分支，然后修复 `task dev`。

## Goal

将现有安全、相关的文档提交到 Git，使本地 `main` 包含最新代码与文档；永久修复普通 `task dev` 在前端就绪前启动原生应用的问题，并完成验证和提交。

## Scope

- 检查并提交当前工作区的既有文档。
- 将本地 `main` 快进到最新实现，保留其他分支及 worktree。
- 修复开发启动等待逻辑，覆盖延迟就绪、超时和错误路径。
- 更新构建说明和任务看板，运行 `task verify` 与实际开发启动验收。

## Out of Scope

新增远端、推送、改写提交历史、删除旧分支或 worktree、修复业务功能中的独立问题。

## Analysis

当前分支为 `codex/account-details-layout`，最新代码提交为 `44fdbc4`；本地 `main` 为其祖先，落后 33 个提交。未配置远端。70 项未提交内容均为 Markdown 文档，用户已授权纳入本次 Git 整理。

原有 Wails 配置在开发构建后并发启动 Vite 和原生应用；Wails 应用连接前端仅重试 10 次，每次等待 500 毫秒，今天两次启动因前端尚未就绪而失败。临时等待脚本已经证明先确认前端可用再启动应用能恢复开发环境。

方案比较：仓库 PowerShell 启动脚本可复用临时方案，但会增加独立的脚本验证链路。选择在 Go 桌面适配层、创建 Wails 应用前等待前端就绪，使用现有标准库与 Go 测试；通过构建标签使生产版保持直接启动。等待仅允许本机开发地址，整体上限 120 秒，单次请求有超时且不跟随重定向。

## Implementation Plan

1. 审查文档、编码、敏感信息和 Git 差异，提交现有文档及本任务记录。
2. 切换到本地 `main`，使用 `--ff-only` 同步最新实现和文档。
3. 比较启动等待方案，选择与当前 Windows 技术栈兼容的最小实现，保留 Wails 热更新及进程生命周期。
4. 验证延迟就绪、连接失败及超时行为，运行完整 `task verify`。
5. 用普通 `task dev` 启动并检查窗口、HMR、Control API；更新相关文档并提交修复，归档记录。

## Risks

开发进程需要退出以释放端口和二进制；必须按项目路径识别，保留其他应用实例。等待机制必须有明确超时和失败输出，不能无限等待或泄露凭据。

## Implementation Result

待完成后填写。

## Verification

文档提交前检查：71 项 Markdown（含本任务记录）未发现敏感凭据模式，相对文件链接均有效；发现一份既有记录的 BOM／换行问题，提交前规范化为 UTF-8 无 BOM、LF。后续代码验证待完成后填写。

## Documentation

计划更新 `docs/development/BUILD.md`、`docs/plans/CURRENT.md` 和本任务记录。

## Deviations

None.

## Git

待提交后填写。

## Result

待完成后填写。
