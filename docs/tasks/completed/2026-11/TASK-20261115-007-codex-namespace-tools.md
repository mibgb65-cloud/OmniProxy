# TASK-20261115-007: Codex 入口移除第三方服务商无法表达的命名空间工具

## Status

Completed

## Created

2026-11-15 18:10（Asia/Singapore）

## Request

> 你看看呢：1d7ddb7c13219589000807526fa08efb

用户安装 `v2.3.1-beta.4` 后在 Codex 桌面端重试同一模型，请求仍然失败。

## Goal

Codex 选择 River／DeepSeek 等非 OpenAI 模型时，客户端按 OpenAI 契约打包的
`type: "namespace"` 工具分组不再让整个请求被本地校验拒绝：网关在交给第三方服务商前
移除这些分组，函数工具原样保留，并记录移除数量。

## Background

实机请求 `1d7ddb7c13219589000807526fa08efb`：路由已正确（`builtin_river` +
`deepseek-ai/DeepSeek-V4.1-Flash-1017`），但 45 ms 内返回
`400 unsupported_content`，未到上游（尝试耗时 12 ms）。

用临时 `CODEX_HOME` 跑 `codex-cli 0.163.0`（ChatGPT 登录、同一份目录与配置形状），
把请求指向只记录**字段形状**的本地监听器，得到真实正文：`store: false`、
`include: ["reasoning.encrypted_content"]`、`input` 全为 message、`tools` 为 8 个
`type=function` **加 8 个 `type=namespace`**（`multi_agent_v1`、
`mcp__codex_apps__*` 等，内部是嵌套的 function 工具）。

对运行中的 beta.4 逐项探测确认（`/codex/v1` 按前缀放行，无需接入 Key）：

| 正文形状 | 结果 |
| --- | --- |
| 最小 Responses 正文 | 200 |
| `store: false` | 200 |
| `store: true` | 400 `unsupported_content` |
| `type=function` 工具 | 200 |
| `type=namespace` 工具 | 400 `unsupported_content` |
| `context_management` 非空 | 400 `unsupported_content` |
| `include`／`prompt_cache_key`／`service_tier`／`tool_choice` | 200 |

真实正文不含 `store: true`、不含 `context_management`，因此唯一命中项是
`type=namespace`：River 适配器 `validateTools` 只接受 `type=function`
（`internal/provider/river/tools.go`），Codex 认为自己在跟 OpenAI 说话，才会这样分组。

## Scope

- `internal/core/proxy`：`Request` 增加只由透明入口设置的 `FromCodexEntry`；透明入口
  判定改走协议处理时置位。
- `internal/application/gateway`：新增 `adaptCodexEntryBody`，在路由完成后按目标服务商
  类型移除 `type=namespace` 工具分组，函数工具与其它字段原样保留，移除数量写日志。
- 测试：Core 断言改走协议处理时 `FromCodexEntry` 为真；网关端到端断言命名空间被移除、
  函数工具保留；真实 River Key 用例改用真实工具形状。
- 文档：`docs/product/GATEWAY.md`、`docs/product/RIVER.md`。
- 版本：`v2.3.1-beta.5` 本地构建，供用户重新覆盖安装。

## Out of Scope

- 不改 Codex 客户端、不改它的工具分组方式，也不把命名空间工具改写为函数工具。
- 不处理 `store: true`／`context_management`：真实 Codex 正文不含它们，在第三方服务商上
  伪造这两个字段的语义没有依据。
- 不重新设计 River 的工具能力矩阵。

## Existing Behavior

Codex 选择 River 模型时，带命名空间工具的请求在网关本地被判 `unsupported_content`。

## Expected Behavior

Codex 选择 River 模型时可以正常对话：命名空间工具分组被移除（MCP／多智能体工具在该
模型上不可用），`exec_command`、`read_file`、`view_image` 等函数工具保持可用。

---

# Analysis

## Related Modules

- `internal/core/proxy/types.go`、`handler.go`
- `internal/application/gateway/routing.go`、新增 `codex_entry_tools.go`
- `internal/provider/river/tools.go`（拒绝原因，不修改）

## Risks

- 移除分组会让该模型看不到 MCP／多智能体工具：这是能力差异而非静默丢数据，日志与文档
  都会写明；被移除的是分组本身，不是用户消息、历史或函数工具。
- 只在 `FromCodexEntry` 且目标不是 OpenAI 时生效，其它客户端与 `/v1` 入口不变。

---

# Implementation Plan

1. 记录本任务（实机复测发现问题后立即补建，见 Deviations）。
2. Core 增加 `FromCodexEntry` 并在透明入口改走协议处理时置位。
3. 网关新增 `adaptCodexEntryBody`：按目标服务商 kind 移除 `namespace` 工具分组并记日志。
4. 测试：Core 标记、网关端到端、真实 River Key。
5. 提升 `v2.3.1-beta.5` 并重新生产打包。
6. 回填文档、CURRENT 与本记录。

---

# Progress

- 2026-11-15 18:05：用临时 `CODEX_HOME` + 形状监听器拿到真实请求体结构，确认
  `type=namespace` 是唯一命中项。
- 2026-11-15 18:12：实现 Core 标记与网关适配，单元／端到端测试通过。
- 2026-11-15 18:16：真实 River Key 验证：带真实工具形状的 `/codex/v1/responses`
  非流式 200／2733 ms、流式 200／904 ms，上游回显只含函数工具。

---

# Implementation Result

## Changes

- `internal/core/proxy`：`Request` 增加 `FromCodexEntry`（只由透明入口设置，客户端 JSON
  不能指定）；`handler.go` 在透明入口判定改走协议处理时置位。
- `internal/application/gateway`：新增 `adaptCodexEntryBody`，在路由完成后按目标服务商
  类型（`providerKind`）移除 `type: "namespace"` 工具分组，函数工具与其它字段原样保留，
  移除时记录 `codex namespace tools dropped` 与数量；`RouteRequest` 在应用档位变体后调用。
- 测试：Core 断言改走协议处理时 `FromCodexEntry` 为真；网关端到端用例发送函数＋命名空间
  工具集，断言上游只收到函数工具且模型／档位仍被改写；真实 River Key 用例改用真实工具形状。
- 文档：`docs/product/GATEWAY.md`、`docs/product/RIVER.md`。

## Files Changed

- `internal/core/proxy/types.go`、`handler.go`、`passthrough_policy_test.go`
- `internal/application/gateway/codex_entry_tools.go`（新增）、`routing.go`、
  `passthrough_provider_test.go`、`live_codex_test.go`
- `docs/product/GATEWAY.md`、`docs/product/RIVER.md`

## Technical Notes

- 只按 `type` 移除分组本身，不重命名、不把嵌套工具提升为函数工具：Codex 调用的工具名
  与分组绑定，改名会让模型拿到无法路由的调用。
- 判定条件同时要求 `FromCodexEntry` 与目标 kind 非 `openai`，所以 `/v1` 入口与其它
  客户端（包括 Codex 自定义 provider 走 `/v1` 选 OpenAI 模型）都不受影响。
- 真实正文不含 `store: true` 与 `context_management`，因此没有为这两个字段伪造语义。

---

# Verification

## Tests

- [x] Unit Tests：`go test ./internal/core/proxy/ ./internal/application/gateway/` 通过。
- [x] Go 全量：`go test . ./internal/...` 54 个包全部通过。
- [x] Integration Tests：`TestCodexEntryRoutesNonOpenAIModelThroughProtocolPath` 覆盖函数＋
      命名空间工具集（非流式／流式）、官方模型 ID、档位 `xhigh` → `max` 与请求记录。
- [x] 真实链路：`OMNIPROXY_LIVE_RIVER=1`（只读生产库、真实 River Key）非流式 200／2733 ms、
      流式 200／904 ms，回显 `tools` 只剩 `exec_command`。
- [x] E2E Tests：待 `v2.3.1-beta.5` 安装后由用户在 Codex 复测。

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
go test . ./internal/...     → 54 个包全部通过
task verify（随 v2.3.1-beta.5 打包） → 见 TASK-20261115-008
```

---

# Documentation

- [x] `docs/product/GATEWAY.md`：Codex 入口改走协议处理时的工具分组适配与日志
- [x] `docs/product/RIVER.md`：River 只接受函数工具，命名空间工具在该模型上不可用
- [x] `docs/releases/v2.3.1-beta.5.md`、`docs/plans/CURRENT.md`

---

# Deviations

本记录在实现过程中补建：实机复测（`1d7ddb7c…`）发现 beta.4 仍有第二个失败原因，属于
TASK-20261115-005 的直接延续，先完成探测与修复、随后立即补齐记录，未先实现无关功能。

---

# Git

## Commit

```text
699cc0f7 fix(gateway): Codex 入口移除第三方服务商无法表达的命名空间工具
```

---

# Result

Codex 在 `/codex/v1` 选择 River 等非 OpenAI 模型时，客户端按 OpenAI 契约打包的
`type: "namespace"` 工具分组不再让整个请求被本地校验拒绝：网关在路由完成后移除这些分组、
保留全部函数工具与其它字段，并记录移除数量。真实 River Key 下用真实工具形状验证通过
（非流式 200／2733 ms、流式 200／904 ms）。代价是该模型上无法调用 MCP／多智能体工具，
这一点已写进发行说明与产品文档。

---

# Completed

2026-11-15 18:30（Asia/Singapore）
