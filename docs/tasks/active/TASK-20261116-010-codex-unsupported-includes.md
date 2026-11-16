# TASK-20261116-010: 移除 Codex 无法表达的 include（output.encrypted_content）修复 River 流中断

## Status

In Progress

## Created

2026-11-16 14:38（Asia/Singapore）

## Request

> 还是有问题，不行，你看看吧

用户附请求页截图：`v2.3.1-beta.11` 下 2026-11-16 13:53:47–13:53:56 的 6 次 Codex
请求全部失败（`deepseek-ai/DeepSeek-V4.1-Flash-1017`、`high`、`rv_1fjif`／`rv_ZQPU6`／
`rv_10k6M` 轮换），并且 14:33 复测仍然失败。

## Goal

定位新一批失败的真实原因并修复，使 Codex 桌面端选择 River 模型时不再每轮必失败。

## Background

beta.11 的修复仍然有效（日志确认 `codex orphan tool outputs dropped count=2`、
`codex namespace tools dropped count=7`），但这批请求依旧
`200 + sse + 0 Token + upstream_stream_failed`，且新增
`upstream_stream_kind=incomplete_stream`：上游在 SSE 开头之后中断，网关无数据行错误。

失败线程 `01a1e2eb-080d-7df3-8ed2-c5677197042b` 是全新会话，只有 4 条消息
（桌面端上下文 35 790 字节 + AGENTS.md + 时间上下文 + 用户「你好」），用这些消息加
旧工具集复放**成功**，说明触发点不在输入内容。

用本地中继抓到桌面端真实正文（102 957 字节）后确认：Codex 桌面端已升级到
0.169 一代内核，请求新增

```json
"include": ["reasoning.encrypted_content", "output.encrypted_content"]
```

## Root Cause

真实 River Key 对照实验（同一份抓包正文，仅改 `include`）：

| `include` | 结果 |
| --- | --- |
| `["reasoning.encrypted_content","output.encrypted_content"]`（桌面端原样） | 200 后 185 字节处中断（0.76 s） |
| `["reasoning.encrypted_content"]` | 200 + `response.completed` |
| 无 `include` | 200 + `response.completed` |
| `["output.encrypted_content"]` | 200 后 185 字节处中断（0.83 s） |

`output.encrypted_content` 是 OpenAI 的加密输出能力；River 的 Responses 实现收到后
会在发出 SSE 开头之后中断连接。网关原样转发该字段，因此 Codex 新一轮会话必然失败。
加密输出本来也无法从第三方服务商获得，移除请求不损失任何可用能力。

## Scope

- `internal/application/gateway/codex_entry_tools.go`：Codex 入口改走协议路径、目标服务商
  不是 OpenAI 时，从 `include` 中移除目标服务商无法表达的取值
  （`output.encrypted_content`），保留其余取值；全部移除时删除该字段；记录
  `codex unsupported includes dropped` 与数量，正文无改动时保持字节不变。
- 测试：过滤逻辑单测、Codex 入口到上游的正文断言、真实 Key live 用例加入该字段。
- 文档与版本：`GATEWAY.md`、`RIVER.md`、`v2.3.1-beta.12` 发行说明、`CURRENT.md`、
  `version.go`、`frontend/package.json`；随后构建 `v2.3.1-beta.12`。

## Out of Scope

- 不改变「已开始生成后不切换账号、不自动重放」规则，不改 OpenAI 透明转发路径。
- 不猜测其它我们无法核实的 `include` 取值：只移除本次真实触发失败的那一个。
- 不改 WebSocket 路径（网关仍按既有策略拒绝 WebSocket）。

## Existing Behavior

Codex 0.169 的请求带 `output.encrypted_content` 时，网关原样转给 River，River 先返回
`200` 与 SSE 开头再中断；客户端重试 5 次全部失败。

## Expected Behavior

- Codex 入口 → 非 OpenAI 服务商：移除 `output.encrypted_content` 等无法表达的 `include`
  取值，保留 `reasoning.encrypted_content`；移除数量写入日志。
- Codex 入口 → OpenAI：正文完全不变。
- 抓包正文与新会话在真实 River Key 上正常完成。

---

# Analysis

## Related Modules

- `internal/application/gateway/codex_entry_tools.go`（`adaptCodexEntryBody`）
- 测试：`codex_entry_tools_test.go`、`passthrough_provider_test.go`、`live_codex_test.go`

## Dependencies

N/A

## Risks

- 移除 `include` 取值会改变模型/客户端可见能力：只移除第三方服务商本来就无法提供的
  加密输出请求；如果将来某个第三方服务商支持该能力，需要重新验证后再放行。
- 只对 `FromCodexEntry` + 非 OpenAI 生效，`/v1` 直连与 OpenAI 路径不受影响。
- 日志只记录固定枚举值与数量，不含用户内容。

---

# Implementation Plan

1. 记录本任务与抓包/对照证据。
2. `codex_entry_tools.go`：新增 `withoutUnsupportedIncludes`，接入适配流程与日志。
3. 测试：单测、网关整合测试、live 用例。
4. 文档与版本，`task verify`，真实 Key 复放，打包 `v2.3.1-beta.12`。

---

# Progress

- 2026-11-16 14:34：抓到桌面端真实正文（4 次相同请求，102 957 字节）。
- 2026-11-16 14:37：`include` 对照实验定位到 `output.encrypted_content`。
- 2026-11-16 14:38：建立本记录，恢复用户 Codex 配置并停止中继。

---

# Implementation Result

## Changes

- `internal/application/gateway/codex_entry_tools.go`：新增 `codexUnsupportedIncludes`
  （当前只有 `output.encrypted_content`）与 `withoutUnsupportedIncludes`；`adaptCodexEntryBody`
  在移除命名空间分组、孤儿工具结果之后，从 `include` 数组里移除这些取值，全部被移除时
  删除 `include` 字段，并记录 `codex unsupported includes dropped`（`count` 与 `values`）。
  保留项按原始 JSON 写回，非字符串元素不动，正文无改动时保持字节不变。
- 测试：
  - `internal/application/gateway/codex_entry_tools_test.go`：过滤逻辑表驱动
    （只移除不支持项、未知取值与非字符串元素保留、缺失／null／非数组不改动）。
  - `internal/application/gateway/passthrough_provider_test.go`：Codex 入口完整链路断言
    上游 `include` 只剩 `["reasoning.encrypted_content"]`。
  - `internal/application/gateway/live_codex_test.go`：真实 Key 用例按 Codex 0.169 形状
    同时发送两个取值，要求完整回复。
- 文档：`docs/product/GATEWAY.md`、`docs/product/RIVER.md`、
  `docs/releases/v2.3.1-beta.12.md`、`docs/plans/CURRENT.md`、版本号 `2.3.1-beta.12`。

## Files Changed

- `internal/application/gateway/codex_entry_tools.go`
- `internal/application/gateway/codex_entry_tools_test.go`
- `internal/application/gateway/passthrough_provider_test.go`
- `internal/application/gateway/live_codex_test.go`
- `internal/version/version.go`、`frontend/package.json`
- `docs/product/GATEWAY.md`、`docs/product/RIVER.md`、`docs/plans/CURRENT.md`
- `docs/releases/v2.3.1-beta.12.md`（新增）

## Technical Notes

- 只移除已用真实 Key 复现失败的取值；其它 `include` 取值继续透传，避免替客户端猜测能力。
- 适配仍只对 `FromCodexEntry` + 非 OpenAI 目标生效，OpenAI 透明转发正文不变。
- 日志只写固定枚举值与数量，不含用户内容。

---

# Verification

## Tests

- [x] Unit Tests：`TestWithoutUnsupportedIncludesDropsOnlyUnsupportedValues`、
  `TestWithoutUnsupportedIncludesKeepsUnexpectedShape` 通过；`go test
  ./internal/application/gateway/ -count=1` 全部通过。
- [x] Integration Tests：`TestCodexEntryRoutesNonOpenAIModelThroughProtocolPath`
  （非流式与流式）断言上游 `include` 已被过滤。
- [x] E2E Tests：真实 River Key；抓包正文（102,957 字节）在新构建隔离实例上两次
  `200` + `response.completed`；`TestLiveCodexRiverModel` 连续 2 次通过。

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

复放对照（真实 River Key，同一份抓包正文）：

| 场景 | 修复前（beta.11） | 修复后（新构建隔离实例） |
| --- | --- | --- |
| 桌面端原始正文（含 `output.encrypted_content`） | 200 后 185 字节处中断（13:53 与 14:33 两轮，每轮 6 次） | 200 + `response.completed`，两次 |
| `include` 只留 `reasoning.encrypted_content` | 200 + `response.completed` | 200 + `response.completed` |
| 无 `include` | 200 + `response.completed` | 200 + `response.completed` |
| `include` 只留 `output.encrypted_content` | 200 后 185 字节处中断 | 200 + `response.completed` |

未验证：安装新构建后由真实 Codex 桌面端在同一线程重试（需要用户安装并重启 Codex）。

---

# Documentation

- [x] `docs/product/GATEWAY.md`：`include` 过滤与日志字段
- [x] `docs/product/RIVER.md`：River 对 `output.encrypted_content` 的行为
- [x] `docs/releases/v2.3.1-beta.12.md`：根因、修复与验证边界
- [x] `docs/plans/CURRENT.md`：当前状态
- [x] 其它文档无需修改

---

# Deviations

- 原计划只记录 13:53 的失败窗口。实际排查中用户又复测了一轮（14:33，4 次相同请求），
  借助网关前的本地中继抓到 102,957 字节真实正文，才定位到 `include`；中继期间临时改过
  `~/.codex/config.toml` 的 `openai_base_url`，取证后已从备份恢复。
- 原计划先按「工具形状」怀疑，复放证明输入与工具都与成功路径等价；真实原因是 Codex
  0.169 新增的 `include` 取值，与 beta.11 修复的孤儿工具结果是两个独立触发点。

---

# Git

## Commit

```text
Not committed by Agent.
```

---

# Result

Codex 0.169 起选择 River 模型每轮必失败的根因已定位并修复：网关在 Codex 入口改走协议
路径时移除第三方服务商无法表达的 `include` 取值（`output.encrypted_content`），保留
`reasoning.encrypted_content`，并记录 `codex unsupported includes dropped`。抓包正文在新构建
隔离实例上两次 `200` + `response.completed`，`task verify` EXIT=0。

---

# Completed

2026-11-16 14:55（Asia/Singapore）



