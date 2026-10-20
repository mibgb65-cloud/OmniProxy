# TASK-20261020-003: 修复 Responses 长会话重连失败

## Status

Completed

## Created

2026-10-20

## Request

用户要求正式修复上轮排查发现的长会话中断后反复 `invalid_request` 问题。

## Goal / Scope

- 修复合法 Responses 长上下文受到 512 项消息上限误拒绝的问题，保留有界输入和协议校验。
- 保留 WebSocket 校验的具体安全错误分类，防止流错误误报为凭据不可用。
- 添加隔离回归测试，完成 `task verify` 和文档同步。

不处理截图中的额度快照持久化需求，不读取或修改真实账号／凭据，不重放真实模型请求，不替换或重启生产程序。

## Analysis

上轮诊断已离线复现：512 项通过，513 项 WebSocket 请求在调用上游前被拒绝为 `invalid_request`；历史重建由 510 项增至 516／517 项后触发同一限制。首次流中断的底层原因尚未确认。

相关诊断：`docs/tasks/completed/2026-10/TASK-20261020-002-investigate-interrupted-codex-turn.md`。

方案比较：仅提高 512 项阈值会在另一段合法长历史上重复失败；删减历史会破坏工具关联与会话语义。本轮选择仅解除 Responses／compact 的消息条目限制，以既有 16 MiB、JSON 深度、并发及上游上下文约束保护资源，Chat／Messages 的 512 项约束不变。已通过全局 JSON 校验的数组无需再次逐项解码以计数。

流故障在 Core 读写边界转换成固定 `proxy.Error`，避免调整凭据访问的既有错误语义或盲目重试已经开始的生成；WebSocket 解析失败保留安全错误码并记录不含正文的拒绝摘要。

## Implementation Plan

1. 核对 Responses／Messages 输入边界、WebSocket 校验及流错误传播，比较最小兼容修复方案。
2. 实施 Responses 专属输入校验和安全错误处理，补齐长会话、重连、边界及凭据分类回归。
3. 运行定向验证与 `task verify`，更新文档、审查差异、仅提交本任务文件并归档。

## Implementation Result

- Responses／compact 数组输入仅在全局有界 JSON 校验后接受，不再次解码计数；Chat／Messages 的 512 项约束保持不变。
- WebSocket 协议解析保留具体安全错误码，增加不含正文的固定拒绝摘要。
- HTTP／WebSocket 读流和写客户端边界返回固定分类，超长 SSE 行返回 `response_too_large`，不更改真实凭据失败或取消／超时语义。
- 新增合成历史回归：512／513／516／517／2048 项 HTTP／compact、510 项请求部分输出后中断／516 项完整重连／518 项合并续写，以及输入安全、错误安全和真实网关记账／不重放测试。

### 安全与兼容性

- WebSocket 在反序列化／重编码之前验证原始帧，避免顶层重复字段被合并后绕过校验；合并后完整上下文继续验证，既有超大帧、图片合并超限与未知 response ID 回归保持通过。
- 不丢弃或改写会话条目，不提高 16 MiB／32 层／并发／SSE 容量边界，不改变上游模型上下文限制。
- 保留 `gateway_error` 错误事件结构和 HTTP 已提交响应头后的真实状态；具体校验码不再统一降级成 `invalid_request`。HTTP 已输出 SSE 后可保留 200，但逻辑请求仍记为失败，尝试保留真实上游 200 与固定错误码。
- Core 不依赖 Wails／React／基础设施，不新增依赖、配置、迁移或自动上游调用。无需架构决策变更。

## Verification

- PASS：`go test ./internal/core/proxy ./internal/application/gateway ./internal/provider/chatgpt -count=1`。
- PASS：关键 `SocketLongHistoryReconnectAfterPartialStream`、`SocketValidationPreservesSafeCodeAndLogsRejection`、`CodexStreamIOFailuresKeepSafeClassificationAndSingleAttempt`、`FreshAccountEligibilityAndCredentialErrors` 回归连续 10 轮。
- PASS：`task verify BIN_DIR=tmp/responses-recovery-20261020/verify-bin`，exit 0。格式检查、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、109 文件／629 项前端测试、Wails 绑定生成、真实前端生产构建及 Windows amd64 production 构建全部通过。
- PASS：完整门禁包含原有超大请求／合并上下文、取消／超时、固定账号与不重放回归；新增网关集成使用隔离 SQLite 和内存上游，真实凭据失败仍为原有分类。
- PASS：暂存差异审查、`git diff --cached --check`、本任务文件 UTF-8 无 BOM 检查。所有修改源码／测试文件均少于 200 行，生成绑定及依赖文件没有漂移。
- 非阻塞既有提示：Windows Task 环境缺少部分 Unix 命令、jsdom `scrollTo` 未实现及 Vite 大 chunk 警告；没有通过屏蔽警告或删改无关测试让门禁通过。
- 验证产物：`tmp/responses-recovery-20261020/verify-bin/omniproxy.exe`，22,083,072 字节，SHA256 `9a34d857f0c8d1a957aa32b938b4a77c94e841f35691bb766c55cf4ad7337254`。验证输出与运行程序路径隔离，未启动此产物或部署生产。安装版生产进程 PID 31376 保持不变；工作区开发进程由 PID 19300 变为 42284，可能由已有开发监视器响应源码修改重建，不能据此宣称生产更新。
- 验证边界：未读取真实凭据、重放原始聊天或调用真实模型；首次历史流中断的底层诱因仍不可由旧日志还原，不能把本轮离线恢复与分类回归当成真实上游网络永不中断的保证。

## Documentation

- 更新 Gateway 协议容量、WebSocket 重连与错误分类说明。
- 更新 CURRENT 的任务状态并归档本记录。
- PRD、技术栈、架构及 ADR 无需变更；不修改原额度快照任务或其他已有未跟踪文件。

## Git

`a467ffeb2467e59fb9a45aa3ca6327d5c0b615b7` — `fix(gateway): 修复 Responses 长会话重连与流错误分类`。

任务归档另行文档提交；未推送，不纳入其他已有未跟踪文件。

## Result

源码修复、隔离恢复回归、完整质量门与本地提交已完成。生产仍运行原版本，需要后续更新运行程序才会生效；未擅自继续原额度快照需求。

## Completed

2026-10-20
