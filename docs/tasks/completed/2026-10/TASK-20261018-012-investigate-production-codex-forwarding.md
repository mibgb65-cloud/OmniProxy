# TASK-20261018-012: 排查生产版 Codex 接入未转发

## Status

Completed

## Created

2026-10-18 20:06

## Request

用户刚测试 Codex 接入未成功转发，要求查看生产端日志并定位原因。

## Goal

只读核对生产进程、配置及请求日志，区分客户端接入、代理路由和上游转发故障。

## Scope

- 生产进程、健康接口、端口、脱敏运行日志。
- SQLite 请求及尝试记录、受管客户端接入状态、无敏感字段的 Codex 配置。
- 对照现有转发和错误处理实现；不修改业务代码、账号、凭据或配置，不重启、不发模型请求。

# Analysis

## Evidence

以下时间均为 2026-10-18 UTC+08:00，来自生产库中 UTC 时间换算。

| 开始时间 | 传输 | 实际上游状态 | 最终请求状态 |
| --- | --- | --- | --- |
| 18:59:09 | WebSocket | 400 / upstream_rejected_request | 502 / upstream_rejected_request |
| 18:59:30 | WebSocket | 400 / upstream_rejected_request | 502 / upstream_rejected_request |
| 19:00:24 | WebSocket | 429 / upstream_rate_limited | 502 / upstream_rejected_request |
| 19:00:45–19:00:51 | WebSocket 及一次 HTTP | 未新增上游尝试 | 429 / account_pool_cooling，共 6 次 |

- 生产库中共 9 条 Responses 请求，均失败，模型为 gpt-6.1-sol；前三条均路由至 builtin_openai 的同一订阅账号。两个 OpenAI 订阅账号启用，但只有实际被选中的账号有模型缓存，其中包含该模型；不能据此断言模型本身不可用。
- WebSocket 路径在 internal/core/proxy/websocket.go:108–109 将所有非 200 上游状态统一包装成 502 / upstream_rejected_request；尝试级记录保留了真实 400、400、429。
- internal/core/proxy/response.go 与 WebSocket 非 200 分支没有提取上游错误正文中的安全错误码／参数；现有落盘证据不能确定 400 的具体拒绝原因，不推测具体参数、模型权限或账号额度。
- 当前生产进程 PID 8124，20:03:52 启动；3890/healthz 确认 production / 2.0.0-beta.2 / ok。检查时该进程只监听 3890，没有监听代理端口 3000。
- 当前 .codex/config.toml 的 model_provider 为 openai，另有指向 http://127.0.0.1:3000/codex/v1 的 omniproxy_deepseek provider 定义，但未在顶层选中。数据库中的 codex 接入记录仍为 enabled，持久记录不能替代当前配置和实际监听状态。
- 当前 v2/logs/omniproxy.log 仅有本次启动的 3 条记录，唯一警告为 managed client catalog refresh failed / refresh_failed；不足以解释此前上游 400。生产库最新 Responses 记录停留在 19:00:51，未发现 20:03 启动后的新请求，不能把历史失败强行等同于用户所说的最新一次测试。

# Implementation Plan

1. 确认实际生产版、数据目录与日志。
2. 使用 SQLite mode=ro 与 query_only 查询无敏感诊断字段，核对配置与端口。
3. 对照错误映射实现，明确已知事实、未证实原因和复现前提。

# Implementation Result

## Changes

- 完成只读排查：历史请求进入了代理，遭遇上游 400 后出现 429，再触发本地账号池冷却。
- 确认 WebSocket 非 200 错误统一映射为 502，导致请求级信息失真；本任务不实施修复。
- 识别当前代理端口未监听、Codex 默认 provider 已非本地代理，且没有新请求落库。
- 未读取凭据文件、令牌或提示词；输出和记录不含账号名称、邮箱、密钥及原始请求内容。

## Files Changed

仅本 Task Record。

## Remaining Risks

上游 400 的具体原因待确认。需要恢复本地代理监听和客户端接入后，经用户实际重试采集安全的上游错误诊断；不得记录完整错误正文或真实提示词。当前检查是时间点快照，不证明重启前或后续的监听及配置状态。

# Verification

- PASS：两次只读生产健康接口核对，Beta.2 / production / ok。
- PASS：生产数据库只读查询请求及尝试表，三次真实上游状态与六次本地冷却相互对应。
- PASS：复查端口、配置和最新请求时间，结果一致。
- PASS：人工对照 WebSocket、HTTP、故障切换代码与记录。
- PASS：Task Record UTF-8 无 BOM 及 git diff 空白检查；五份既有用户未跟踪记录保持不变。
- N/A：未修改代码、配置或生产数据，不运行 task verify，不发送付费模型请求。

# Documentation

除本排查记录外，无文档需要更新。

# Git

Not committed by Agent.

# Result

已查清历史失败链路和当前接入状态，确认错误映射造成的诊断盲区；具体上游 400 原因仍缺少安全错误详情，未声称已经修复。

# Completed

2026-10-18 20:09