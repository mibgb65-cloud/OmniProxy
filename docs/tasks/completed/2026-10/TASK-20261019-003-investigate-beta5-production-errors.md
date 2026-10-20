# TASK-20261019-003: 排查 Beta.5 生产端近期错误请求

## Status

Completed

## Created

2026-10-19 11:08

## Request

查看目前生产端日志，出现了很多错误请求。

## Scope

只读检查当前生产健康、日志与近期请求统计，按版本／时间／模型／传输／状态码及失败阶段归类，依据现有源码定位原因与后续修复范围。

## Out of Scope

不重启生产或当前 Agent，不改共享配置、账号、额度开关、数据库或发送登记；不重放真实请求，不在记录或输出保存提示词、图片、凭据、账号邮箱和原始敏感错误。

# Analysis

已只读确认生产运行 2.0.0-beta.5，控制端健康正常；需区分旧版历史失败、客户端重试、上游拒绝与本地处理缺陷。

# Implementation Plan

1. 读取日志字段与只读数据库表结构，仅输出允许的诊断元数据。
2. 聚合最近错误、核对同时间成功请求与运行版本，关联现有转发／自动激活路径。
3. 针对证据做最小本地验证，报告根因、影响与修复建议；不擅自改变生产行为。

# Progress

- 当前 main 与远端一致，既有未跟踪记录保持不变。

# Implementation Result

## Findings

- 生产 Beta.5 自 2026-10-19 10:50:33（新加坡）启动，Control API 健康正常，schema=34。
- 升级后当前统计快照为 12 个网关请求：gpt-6-luna／Responses HTTP stream=true 共 9 次 502 invalid_upstream_response、1 次 499 request_cancelled；gpt-6.1-sol 与 gpt-5.6-luna／WebSocket 各 1 次 200，均有完整用量。
- 9 次失败发生于 11:01:35–11:05:16 的连续序列。每个 request 只有 attempt_index=1，真实上游状态均为 200，最终下游为 502；不能把它们解释为一次网关请求内的多账号重试。后续请求紧接前次结束，更符合客户端连续重试，但现有元数据不能证明属于同一客户端会话。
- 全部 9 次日志 upstream_media_type=missing／upstream_content_type_inferred=false。依据 chatgpt/response_type.go 和 core/proxy/response.go，可定位到缺失媒体类型的首事件推断未通过，HTTP 流式消费端随后拒绝。现有日志没有区分首事件 JSON、事件类型／ID、前缀上限、EOF／读失败，不能进一步断言是合法 SSE 兼容漏判、模型权限错误或真实异常响应；不能宣称已找到并修复最终根因。
- 同一账号在同一时段两个其他模型成功，当前没有新 401／403／429 证据，因此不能直接归因于账号失效、配额耗尽或网关整体不可用。模型目录包含 gpt-6-luna 也不等于已经核验其本次调用权限。
- 数据库在本次 Beta.5 启动前另有 54 条 status>=400 的历史记录（包括取消），不能把历史全部算作新版本故障。
- managed client catalog refresh failed 是另一条配置刷新警告。只读比较确认 Codex 配置与受管 applied_hash 不一致，而目录自身匹配；当前三个 OpenAI 模型的客户端目录仍为 text-only。现有外部配置保护会拒绝刷新，需用户预览并受控重新应用，不能绕过保护影响当前 Agent；该事实不等于上述 502 的最终原因。
- 自动激活日志 10:56 记录 sent=1，10:57 后 activated=2／pending=0／failed=0；生产观测表出现已确认目标。没有本次自动激活失败的日志证据；本 Agent 没有触发新查询、模型发送或登记变更。

## Suggested Follow-up

优先对首事件判定增加不含正文的固定原因枚举（读失败、EOF、前缀超限、JSON／事件／ID 校验等），并在独立环境复现 gpt-6-luna HTTP 响应。只有识别到合法协议变体后才补兼容，不无条件信任缺失 Content-Type 的 200；若实际为错误响应则保留受控错误分类，不通过盲目换号或重放掩盖。真实重放会消耗额度，本轮没有执行。

临时可停止失败模型客户端侧的连续重试，使用同环境已有成功记录的模型；不能保证其他请求条件或权限相同。

# Verification

- PASS：生产 healthz、日志 JSON 元数据、SQLite mode=ro 请求／尝试表与匿名聚合交叉核对；未读取或保存对话正文、图片、原始响应及凭据。
- PASS：受管哈希按实际 digest（存在标记加内容 SHA256）只读比对，没有调用可能写配置的 RefreshCatalog。
- PASS：go test ./internal/provider/chatgpt ./internal/core/proxy -run 'Test(ResponseTypeInference|Inference|ResponseHeader)' -count=1。
- 本轮未改产品代码；纯诊断记录不运行完整 task verify，不用现有模拟回归冒充真实失败已复现。
- 已知边界：历史响应正文未保存，不能靠现有日志补出具体首事件；本轮未做真实上游探测／付费重放或线上诊断插桩。

# Git

Not committed by Agent. 本轮仅只读排查，诊断记录保留在工作区；未提交或推送，不改已发布版本。

# Result

已确认新版当前错误集中在 gpt-6-luna HTTP 的缺失响应类型／首事件校验路径，而非所有模型转发失效；分离旧历史、客户端连续重试、配置刷新警告与自动激活状态。最终响应形态仍需安全诊断或隔离复现，尚未修复；当前生产、账号与 Agent 配置保持不变。

# Completed

2026-10-19 11:11
