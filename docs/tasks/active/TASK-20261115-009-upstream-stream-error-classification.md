# TASK-20261115-009: 区分上游 SSE 错误事件与流中断并记录可核实的失败原因

## Status

In Progress

## Created

2026-11-15 19:20（Asia/Singapore）

## Request

> 还是有问题，你看看。

用户安装 `v2.3.1-beta.5` 后在 Codex 桌面端发送一条消息，请求明细里
18:47:25–18:47:36 的 6 次请求全部是 `upstream_stream_failed`（模型
`deepseek-ai/DeepSeek-V4.1-Flash-1017`，账号在 `rv_1fjif`／`rv_ZQPU6`／`rv_10k6M`
之间轮换）。

## Goal

把「上游在 SSE 里主动报错」与「上游流被截断／内容不合语法」区分开：前者记为
`upstream_error_event`，后者保持 `upstream_stream_failed`，并把可核实的分类（事件
`type` 与上游错误码，不含说明文字）写进网关日志，使下次失败可以直接判断是上游故障
还是网关缺陷。

## Background

6 次失败记录的共同点：

| 字段 | 值 |
| --- | --- |
| 协议／客户端 | `responses`／`codex` |
| 路由 | `builtin_river` + `deepseek-ai/DeepSeek-V4.1-Flash-1017` |
| 上游状态 | 200，`upstream_media_type: sse`，`upstream_content_type_inferred: false` |
| 用量 | `input_tokens=0`、`output_tokens=0`、`usage_state: unknown` |
| 计时 | 首次 1747 ms，其余 663／678／678／682／691 ms |
| 日志附加字段 | 无 `stream_error_side` |

`stream_error_side` 只在透明转发的分块路径写入（`passthrough_send.go`），没有该字段说明
这批请求走的是协议路径，失败由 `copyStream` → `streamLineState` 判定；同窗口日志里这 6 次
请求前各有 `codex namespace tools dropped count=9`，与协议路径一致。0 Token 且已发出 200
说明上游只交付了开头的事件就被判失败。

排除工作（全部针对运行中的 beta.5，`/codex/v1` 按前缀放行，无需接入 Key）：

| 探测 | 结果 |
| --- | --- |
| 用真实 `CODEX_HOME` 跑 `codex-cli 0.163.0`，仅把 base URL 指到本地转发监听器，抓取与桌面端等价的正文（8 个 `namespace` + 11 个 `function` 工具、`store:false`、`include`、`prompt_cache_key`、完整 `client_metadata`） | 抓到 103 372 字节正文，且该轮对话成功 |
| 上述正文经网关顺序重放 18 次 | 18/18 收到 `response.completed` |
| 桌面端独有字段逐个加入探测：`personality`、`reasoning.summary`、`text.verbosity`、`service_tier`、`truncation`、`instructions`、全部同时加入 | 全部 200 完成 |
| 20 路并发重放 | 10 次 429 限流、10 次成功（`upstream_rate_limited` 路径，不是本错误码） |

同时确认：进入失败窗口前 Codex 先做了 5 次 WebSocket 握手（`websocket_disabled` 403，
18:47:20.6–25.6），随后 6 次 POST 之间的间隔按 1.4 s／1.9 s／1.4 s／4.2 s 退避 ——
是客户端重试节奏，不是网关行为。

结论：路由与正文形状已经正确（beta.5 的修复有效），这 11 秒内的失败来自上游对 200 SSE
返回错误或提前结束。当前实现把两类原因合并成 `upstream_stream_failed`，日志也没有可核实的
分类，客户端因此只看到 `stream disconnected before completion ... error decoding response body`，
无法判断是上游故障还是网关缺陷。

## Scope

- `internal/core/proxy`：把单条 SSE 数据行的判定拆开并分类；分类结果写进 `Trace`。
- `internal/core/proxy/trace.go`：新增日志字段 `upstream_stream_kind`、
  `upstream_event_type`、`upstream_error_code`（短、净化、不含说明文字）。
- 测试：分类与日志字段的单元测试，更新既有「首个事件即失败」用例的期望错误码。
- 文档：`docs/product/GATEWAY.md`、`docs/releases/v2.3.1-beta.6.md`、`docs/plans/CURRENT.md`。
- 版本：`v2.3.1-beta.6` 本地构建，供用户覆盖安装后复测。

## Out of Scope

- 不改变「已开始生成后不切换账号、不自动重放」的既有规则，也不改故障切换集合。
- 不改 WebSocket 路径的失败分类（该路径另有 `copySocketResponse`，本次不动）。
- 不把上游的说明文字回显给客户端：错误响应仍只有稳定错误码和内置安全说明。
- 不根据上游错误码自动改写请求或模型。

## Existing Behavior

上游在 SSE 里报错与流被截断都记为 `upstream_stream_failed`，日志只有该错误码，
没有可核实的原因分类。

## Expected Behavior

- 上游 SSE 明确报错（顶层 `error` 或 `response.error` 非空）→ `upstream_error_event`（502），
  日志带事件 `type` 与上游错误码。
- 数据行不是合法 JSON，或合法但不是 JSON 对象 → 仍为 `upstream_stream_failed`（502），
  日志带 `invalid_json`／`unexpected_shape` 分类。
- 两种情况下客户端收到的内容与状态码行为不变。

---

# Analysis

## Related Modules

- `internal/core/proxy/response.go`：`hasUpstreamError`（WebSocket 与 JSON 响应路径共用）
- `internal/core/proxy/stream.go`：`copyStream`／`streamLineState`
- `internal/core/proxy/trace.go`：`diagnosticLogFields`
- `internal/core/proxy/stream_headers_test.go`：首个事件即失败的既有断言

## Root Cause

`streamLineState` 把三种情况折成同一个错误码：非法 JSON、合法但不是对象、上游明确报错。
`hasUpstreamError` 还兼任「不是 JSON 对象也算失败」的判定，使调用方无法区分原因，
日志也没有任何可以区分的字段。

## Risks

- 新增错误码会出现在请求明细里：前端只展示代码本身（无 per-code 文案映射），因此不需要
  前端改动；发行说明与 `GATEWAY.md` 会写明新代码含义。
- 上游错误码属于上游文本：只保留可打印短码（≤40 字符）并经过统一 Redactor，不保存
  `message`／`param` 等说明字段。
- 分类只在 `copyStream` 生效，WebSocket 路径与 JSON 响应路径行为不变。

---

# Implementation Plan

1. 记录本任务。
2. `response.go`：新增 `inspectStreamData` 返回可核实的失败分类，`hasUpstreamError` 改为
   复用它，保持既有调用点语义。
3. `stream.go`：`streamLineState` 按分类返回 `upstream_error_event`／`upstream_stream_failed`
   并把分类写入 `Trace`。
4. `trace.go`：新增三个诊断字段并补齐线程安全写入。
5. 测试：分类表驱动用例、Trace 字段用例、更新 `TestStreamImmediateFailureStaysUncommitted`。
6. 文档与版本：`GATEWAY.md`、beta.6 发行说明、`CURRENT.md`、`version.go`、`package.json`。
7. `task verify`；打包 `v2.3.1-beta.6`；提交并回填本记录。

---

# Progress

- 2026-11-15 19:20：完成定位与排除，建立本记录。

---

# Implementation Result

（实施后填写）

---

# Verification

（实施后填写）

---

# Documentation

（实施后填写）

---

# Deviations

（实施后填写）

---

# Git

## Commit

```text
（实施后填写）
```

---

# Result

（实施后填写）

---

# Completed

（实施后填写）
