# TASK-20261116-008: Codex 桌面端无 call_id 的工具结果导致 River 流中断

## Status

Completed

## Created

2026-11-16 11:30（Asia/Singapore）

## Request

> codex在使用deepseek-v4.1flash-1017时依旧报错，你看看这个是什么情况，看看能不能修复好这个问题

用户附请求页截图：`v2.3.1-beta.10` 下 2026-11-16 11:08:45–11:08:57 的 6 次
Codex 请求全部失败（模型 `deepseek-ai/DeepSeek-V4.1-Flash-1017`、档位 `high`、
账号在 `rv_1fjif`／`rv_ZQPU6`／`rv_10k6M` 之间轮换），重试 5 次后放弃。

## Goal

定位这批失败的真实原因并修复，使 Codex 客户端使用 River 的
`DeepSeek-V4.1-Flash-1017` 时不再出现「请求已失败／未报告」的流中断。

## Background

请求库与网关日志给出的共同点（2026-11-16 03:08:45Z–03:08:57Z，即 11:08 本地）：

| 字段 | 值 |
| --- | --- |
| 客户端／协议 | `codex`／`responses`，`stream=1`，`effort=high` |
| 路由／账号 | `builtin_river` + `deepseek-ai/DeepSeek-V4.1-Flash-1017`，三个账号轮换 |
| 上游 | 200，`upstream_media_type: sse`，0 Token，`usage_state: unknown` |
| 错误码 | `upstream_stream_failed`，耗时 580–2618 ms |
| 诊断字段 | 无 `stream_error_side`、无 `upstream_stream_kind` —— 说明没有数据行被判失败，是读流提前结束 |

Codex 自己的日志（`~/.codex/logs_2.sqlite`）同一窗口记录：

```text
codex_http_client  Request completed url=http://127.0.0.1:3000/codex/v1/responses status=200 OK
codex_core::responses_retry  stream disconnected - retrying sampling request (1/5)...
  sampling_error=stream disconnected before completion: Transport error: network error: error decoding response body
```

失败线程 `01a1df22-8be9-7473-bcad-a9bf248a5365`（桌面端，cwd `E:\go\高保真设计图\edumail`）
在 rollout 中三个回合全部是这条报错，即该线程从创建起从未成功；同一时间窗内
`gpt-6.1-sol`（`builtin_openai` 透明转发）与 `harness-desktop` 的 DeepSeek Chat 请求全部成功，
所以不是账号、限流或网络整体故障。

## Root Cause

Codex 桌面端会把「Companion Page」上下文作为 `function_call_output` 输入项注入请求，
但这类伪工具结果**没有 `call_id`**（rollout 里是 `name=codex_apps_open_page`、
`namespace=agent`、无 `call_id`）：

```json
{"type":"function_call_output","id":"fco_...","name":"codex_apps_open_page",
 "namespace":"agent","output":"{\"page_id\":null}"}
```

网关当前把 Codex 正文原样转给 River `/v1/responses`。River 收到缺少 `call_id` 的
工具结果后**先返回 200 与 SSE 开头，随后直接中断连接**（无 SSE 错误事件、无终结事件）；
网关 `copyStream` 读到 EOF，记录 `upstream_stream_failed`，并按既有规则
`panic(http.ErrAbortHandler)` 断开下游 → Codex 侧看到
`error decoding response body`，重试 5 次后放弃。

真实 River Key 上的对照实验（把 2026-11-15 抓到的等价正文的 `input` 换成不同形态）：

| 变体 | 内容 | 结果 |
| --- | --- | --- |
| A | 抓包正文原样 | 200 + `response.completed` |
| B／E1 | 含 `call_id: null` 的 `function_call_output`（E1 为 rollout 原样项） | 200 后 185 字节处中断（0.6 s） |
| C／E2 | 同一批项，仅把 `call_id` 换成字符串 | 200 + `response.completed` |
| D | 只有普通 developer／user 文本 | 200 + `response.completed` |
| G | 孤儿工具结果但 `call_id` 是合法字符串 | 200 + `response.completed` |
| F | 真实 MCP `function_call` + 对应 `function_call_output` | 200 + `response.completed` |

结论：只有「工具结果类输入项的 `call_id` 缺失／为空／非字符串」会触发 River 中断；
命名空间分组、`namespace`／`internal_chat_message_metadata_passthrough` 等额外字段
与合法 `call_id` 的工具结果都不受影响。

## Scope

本任务包含：

- `internal/application/gateway/codex_entry_tools.go`：Codex 入口改走协议路径、目标服务商
  不是 OpenAI 时，除既有「移除 `type: "namespace"` 工具分组」外，再移除 `input` 中
  `call_id` 不是非空字符串的 `function_call_output`／`custom_tool_call_output`，并记录
  `codex orphan tool outputs dropped` 与数量。
- `internal/provider/river/validation.go`：River 的 Responses 校验把缺少 `call_id` 的工具
  结果判为 `unsupported_content`（400），使非 Codex 入口的同类正文得到明确错误，而不是
  「200 后中断」；Codex 入口在适配后不会命中这条校验。
- `internal/core/proxy`：读流在没有终结事件时结束，记录可核实的分类
  （`upstream_stream_kind=incomplete_stream`），供下次直接判断失败侧。
- 测试、文档、版本与 `v2.3.1-beta.11` 本地构建。

## Out of Scope

本任务明确不处理：

- 不改变「已开始生成后不切换账号、不自动重放」的既有故障切换规则。
- 不改变「已提交响应头后断开连接」的下游中断方式（不新增 `response.failed` 事件）。
- 不改 OpenAI 透明转发路径：OpenAI 自己接受这些伪项，改动只会带来无谓差异。
- 不恢复 Companion Page 在第三方服务商上的能力：分组工具同样已被移除，代价写进文档。

## Existing Behavior

Codex 桌面端在带 Companion Page 的线程里向 River 发请求时，网关把无 `call_id` 的
工具结果原样上送，River 返回 200 后中断，客户端重试 5 次全部失败；网关日志只有
`upstream_stream_failed`，无法从记录判断是读流中断。

## Expected Behavior

- Codex 入口 → 非 OpenAI 服务商：移除无法表达的孤儿工具结果（与移除命名空间工具同一条
  路径），请求正常完成；移除数量写入日志，能力差异不静默。
- 合法的 `function_call_output`／`custom_tool_call_output`（带 `call_id`）原样保留。
- 非 Codex 入口出现同类孤儿工具结果时，River 适配器返回 `400 unsupported_content`，
  不再产生「200 后中断」。
- 上游流在终结事件前结束时，日志带 `upstream_stream_kind=incomplete_stream`。

---

# Analysis

## Related Modules

- `internal/application/gateway/codex_entry_tools.go`（`adaptCodexEntryBody`）
- `internal/application/gateway/routing.go`（路由完成后调用适配的位置）
- `internal/provider/river/validation.go`（`textMessages`）
- `internal/core/proxy/stream.go`、`response.go`、`trace.go`（流失败分类与日志）
- 测试：`internal/application/gateway/passthrough_provider_test.go`、
  `internal/provider/river/*_test.go`、`internal/core/proxy/stream_error_event_test.go`

## Dependencies

N/A

## Risks

- 移除输入项会改变模型可见上下文：被移除的只是没有对应 `function_call` 的伪结果，
  且这些工具分组本来就没有下发，信息价值为零；仍按既有做法写日志并更新文档。
- 适配只作用于 `FromCodexEntry` + 非 OpenAI 目标，`/v1` 直连与 OpenAI 路径不变。
- 新增的 River 校验会拒绝一类此前「看起来能发出去」的正文：这类正文实际会让上游中断，
  明确 400 是更好的行为，且不影响任何合法正文。
- 新增日志字段只增加分类，不改变客户端可见内容与状态码。

---

# Implementation Plan

1. 记录本任务，回填根因与对照实验。
2. `codex_entry_tools.go`：抽出输入项清理逻辑与日志计数，扩展文档注释。
3. `river/validation.go`：`textMessages` 校验工具结果项的 `call_id`。
4. `stream.go`／`response.go`／`trace.go`：读流无终结事件时标记分类。
5. 测试：Codex 入口适配（保留合法项、移除孤儿项）、River 校验、流分类。
6. 文档与版本：`GATEWAY.md`、`RIVER.md`（如需要）、`v2.3.1-beta.11` 发行说明、
   `CURRENT.md`、`version.go`、`frontend/package.json`。
7. `task verify`；真实 River Key 用失败正文形状验证；打包 `v2.3.1-beta.11`；提交并回填记录。

---

# Progress

- 2026-11-16 11:12：从请求库与 Codex 日志取到失败窗口，确认上游为 200 SSE 后中断。
- 2026-11-16 11:25：用真实 River Key 做 7 组对照，定位到无 `call_id` 的工具结果项。
- 2026-11-16 11:30：建立本记录。
- 2026-11-16 11:34：实现 Codex 入口清理、River 校验、流分类与三组测试；单元测试通过。
- 2026-11-16 11:38：隔离实例 + 真实 Key 复放失败正文，发现第二个缺陷
  （`response.incomplete` 未被当作终结事件，完整交付后仍中断连接）；同批修复。
- 2026-11-16 11:42：真实失败正文在新构建上两次 `200` + `response.completed`；
  `task verify` EXIT=0。

---

# Implementation Result

## Changes

- `internal/application/gateway/codex_entry_tools.go`：`adaptCodexEntryBody` 拆成
  「移除命名空间工具分组」与「移除孤儿工具结果」两步，各自只在确有移除时才重写正文并记日志
  （`codex namespace tools dropped`／`codex orphan tool outputs dropped` 与数量）。
  孤儿判定只看类型 `function_call_output` 与 `call_id` 是否为非空字符串，其余输入项、
  函数工具与其它字段原样保留；没有改动时正文保持字节不变。
- `internal/provider/river/validation.go`：Responses 输入校验要求 `function_call_output`
  的 `call_id` 为非空字符串（`supportedToolCallID`），同类正文在非 Codex 入口返回
  `400 unsupported_content`，不再「200 后中断」。
- `internal/core/proxy/stream.go`：`response.incomplete` 与 `response.completed`／
  `message_stop` 一样按终结事件处理；读流在终结事件前结束时（非单行超限）记录
  `upstream_stream_kind=incomplete_stream`。
- `internal/core/proxy/response.go`：新增 `streamDataIncompleteStream` 分类常量。
- 测试：
  - `internal/application/gateway/codex_entry_tools_test.go`（新增）：孤儿结果表驱动
    （缺失／null／空串／空白／非字符串都被移除，合法调用与结果保留）；非数组 `input`
    与缺失 `tools` 不改动。
  - `internal/application/gateway/passthrough_provider_test.go`：Codex 入口完整链路断言
    上游收到的 `input` 只剩 3 项、成对调用／结果未被改写、孤儿结果没有出现在上游。
  - `internal/provider/river/tool_output_validation_test.go`（新增）：四种无
    `call_id` 形状均 `400 unsupported_content`，成对调用／结果通过。
  - `internal/core/proxy/stream_error_event_test.go`：`response.incomplete` 为终结事件、
    整条流以它收尾时转发成功；终结事件前结束的流记录 `incomplete_stream`。
  - `internal/application/gateway/live_codex_test.go`：真实 Key 的 Codex 入口用例加入
    成对调用／结果与一条孤儿 Companion Page 结果；读取失败时保留已收到字节而不是直接
    `Fatal`，便于定位。
- 文档：`docs/product/GATEWAY.md`、`docs/product/RIVER.md`、
  `docs/releases/v2.3.1-beta.11.md`、`docs/plans/CURRENT.md`、版本号 `2.3.1-beta.11`。

## Files Changed

- `internal/application/gateway/codex_entry_tools.go`
- `internal/application/gateway/codex_entry_tools_test.go`（新增）
- `internal/application/gateway/passthrough_provider_test.go`
- `internal/application/gateway/live_codex_test.go`
- `internal/provider/river/validation.go`
- `internal/provider/river/tool_output_validation_test.go`（新增）
- `internal/core/proxy/stream.go`、`response.go`、`stream_error_event_test.go`
- `internal/version/version.go`、`frontend/package.json`
- `docs/product/GATEWAY.md`、`docs/product/RIVER.md`、`docs/plans/CURRENT.md`
- `docs/releases/v2.3.1-beta.11.md`（新增）

## Technical Notes

- 只在 `FromCodexEntry` 且目标服务商类型不是 `openai` 时适配，`/v1` 直连与 OpenAI 透明
  转发路径完全不变；孤儿判定与命名空间移除共用同一服务商类型查询结果。
- River 的拒绝码沿用既有 `textMessages` 语义（能表达与否）→ `unsupported_content`，
  没有为一种坏形状新增错误码。
- `incomplete_stream` 与请求状态码组合仍能区分失败侧：客户端主动取消时最终代码是
  `request_cancelled`（`safeFailure` 依据请求上下文判定），因此日志字段本身不声明出错侧。

---

# Verification

## Tests

- [x] Unit Tests：`codex_entry_tools_test.go`、`tool_output_validation_test.go`、
  `stream_error_event_test.go` 新增用例；`go test ./internal/core/proxy/
  ./internal/provider/river/ ./internal/application/gateway/ -count=1` 通过。
- [x] Integration Tests：`TestCodexEntryRoutesNonOpenAIModelThroughProtocolPath`
  覆盖 Codex 入口 → River 适配的完整链路（上游正文断言）。
- [x] E2E Tests：真实 River Key + 真实生产数据启动隔离实例，复放失败线程原始正文
  （13 个 rollout 输入项，含 4 条孤儿结果）两次，均 `200` + `response.completed`；
  `TestLiveCodexRiverModel`（`OMNIPROXY_LIVE_RIVER=1`）连续 3 次全部通过。

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task verify → EXIT=0（155 文件／948 项前端测试、全部 Go 包、Windows production 构建）
```

复放对照（真实 River Key，同一正文）：

| 场景 | 修复前（已安装 beta.10） | 修复后（新构建隔离实例） |
| --- | --- | --- |
| 抓包正文原样 | 200 + `response.completed` | 200 + `response.completed` |
| 失败线程原始正文（4 条孤儿结果） | 200 后 185 字节处中断（11:08 连续 6 次） | 200 + `response.completed`，两次 |
| 加回字符串 `call_id` | 200 + `response.completed` | 200 + `response.completed` |

未验证：安装新构建后由真实 Codex 桌面端在同一线程重试（需要用户安装并重启 Codex）。

---

# Documentation

- [x] `docs/product/GATEWAY.md`：孤儿工具结果移除、终结事件与 `incomplete_stream` 分类
- [x] `docs/product/RIVER.md`：River 对工具结果 `call_id` 的要求与两种入口的行为
- [x] `docs/releases/v2.3.1-beta.11.md`：根因、修复、验证边界
- [x] `docs/plans/CURRENT.md`：当前状态
- [x] 其它文档无需修改

---

# Deviations

- 原计划：River 适配器把同类正文判为 `400 invalid_request`。实际方案：沿用
  `textMessages` 既有的「内容能否表达」语义，返回 `400 unsupported_content`，
  不为一种坏形状新增错误码；测试与文档按实际行为记录。
- 原计划：本任务只修孤儿工具结果与流分类。实际在隔离实例复放时发现第二个缺陷：
  `response.incomplete`（命中 `max_output_tokens`／内容过滤的正常收尾）没有被当作终结
  事件，完整交付后网关仍按截断中断连接，客户端同样看到 `error decoding response body`。
  该缺陷与本次症状同类且改动极小，已一并在本任务修复并补测试。
- 原计划只做单元测试验证；实际增加了真实 Key 的隔离实例复放与 live 用例脚本改动，
  因为该缺陷只有在真实上游流上才会出现（本地假上游不会中断连接）。

---

# Git

## Commit

```text
69c372b1 fix(gateway): 移除 Codex 桌面端无 call_id 的工具结果，修复 River 流中断
```

---

# Result

Codex 桌面端使用 River `DeepSeek-V4.1-Flash-1017` 的流中断已修复：网关在 Codex 入口改走
协议路径时移除没有 `call_id` 的 Companion Page 伪工具结果（记
`codex orphan tool outputs dropped`），River 适配器对非 Codex 入口的同类正文返回
`400 unsupported_content`；`response.incomplete` 按正常终结事件转发；终结事件前结束的流
记 `upstream_stream_kind=incomplete_stream`。真实失败正文在新构建上两次
`200` + `response.completed`，`task verify` EXIT=0。

---

# Completed

2026-11-16 11:48（Asia/Singapore）
