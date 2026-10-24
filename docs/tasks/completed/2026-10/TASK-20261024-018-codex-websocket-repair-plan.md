# TASK-20261024-018: 编写 Codex WebSocket 握手超时修复计划

## Status

Completed

## Created

2026-10-24（Asia/Singapore）

## Request

用户在只读排查后要求：“那你帮我写一个计划 plan 的 md 吧，该怎么修改修复 bug 优化，你写吧”。

本次授权仅编写计划，不实施业务代码、客户端配置、账号授权或运行环境变更。

## Goal

形成可执行的 Markdown 修复计划，明确证据边界、方案取舍、修改顺序、安全约束、测试与验收标准。

## Scope

- 新建 `docs/plans/PLAN-20261024-codex-websocket-handshake.md`。
- 更新本任务记录及 `docs/plans/CURRENT.md` 中的任务概览。

## Out of Scope

- 实施计划中的代码修复。
- 修改真实 Codex 配置、登录、凭据、数据库或应用开关。
- 构建、安装、重启、发布或提交现有工作区中的其他改动。

# Analysis

以前一轮只读排查的代码与本机日志为依据，区分“已确认的握手超时链路”与“仍缺阶段耗时的具体慢步骤”。
不得把现有单元测试通过写成实机 WebSocket 已修复，也不得把文档任务完成写成修复已完成。

# Implementation Plan

1. 核对相关任务、产品约束与现有入口／凭据准备代码。
2. 编写修复计划，包含取消语义、握手预算、凭据快路径、并发隔离、旧配置清理和回归测试。
3. 校验 Markdown 格式、相对链接、命令与工作区变更范围。
4. 记录实际结果，归档本次文档任务；后续修复保持待实施。

# Implementation Result

- 新建[修复计划](../../../plans/PLAN-20261024-codex-websocket-handshake.md)，明确已确认事实与尚缺耗时证据的边界。
- 分阶段列出诊断与取消语义、握手总预算、两种凭据来源的快路径与并发协调、旧配置清理和最小连接可观测性。
- 补充修改范围、方案取舍、回归矩阵、定向包测试／质量门命令、性能建议、CLI／Desktop 实机验收和安全回退要求。
- 在 `docs/plans/CURRENT.md` 的 Next Steps 中登记待实施计划；未修改业务代码、配置、账号或运行状态。
- 文档任务已完成；后续业务修复需另建实施任务，不将本任务的 Completed 误读为故障已修复。

# Verification

- 新文档使用 UTF-8 无 BOM，相对链接检查通过。
- 新计划与任务记录的 Prettier 检查通过；仅对新计划执行格式化，未整体格式化既有看板。
- 人工核对计划中的 Go 包路径与 `task verify` 入口存在；命令被明确标记为后续实施步骤。
- 本次只有文档变更，未重新运行 Go／前端测试或 `task verify`；前一轮排查结果作为历史依据注明，不冒充本次验证。
- 已检查变更范围；保留工作区其他任务的既有修改。

# Documentation

- 新增 `docs/plans/PLAN-20261024-codex-websocket-handshake.md`。
- 更新 `docs/plans/CURRENT.md` 的计划概览。
- 本次不修改产品／架构／ADR 的现行契约；需要的同步工作列入后续实施计划。

# Git

Not committed by Agent.

# Result

已完成可执行的 Markdown 修复计划。业务修复、构建和实机验收均未开始，不声称重连问题已解决。

# Completed

2026-10-24（Asia/Singapore）
