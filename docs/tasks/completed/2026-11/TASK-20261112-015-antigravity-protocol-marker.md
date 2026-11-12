# TASK-20261112-015: Antigravity 请求缺协议标记导致 unsupported_protocol

## Status

In Progress

## Created

2026-11-12 21:58（Asia/Singapore）

## Request

用户安装 `v2.2.3-beta.13` 后重试仍然失败（“Agent terminated due to error”）。需要继续定位。

## Goal

- 修复 Antigravity 请求进入转发层时协议为空、被判为 `unsupported_protocol` 的问题。
- 补回归测试，确保桌面端 Chat 请求在解析层就带上 `chat` 协议。

## Background

`v2.2.3-beta.13` 已修复家族模型名解析，请求明细显示模型已换到 `gemini-4-argon-high`、
档位 `high`，但状态码变成 `400 unsupported_protocol`：

```text
2026-11-12T13:36:58.376Z  input_model=gemini-4-argon  model=gemini-4-argon-high  status=400  code=unsupported_protocol  effort=high
```

本地直接对网关复现，任意 Google 模型（含精确 ID `gemini-3.8-flash-high`）都是同样的
`400 unsupported_protocol`：

```text
POST /v1/chat/completions {"model":"gemini-4-argon","messages":[…],"stream":false}
{"error":{"code":"unsupported_protocol", …}}
```

根因：`ParseClientRequest` 对 `client == "antigravity"` 走 `ParseAntigravityRequest` →
`parseChatRequest`，而 `parseChatRequest` 只填 Model／Stream／Body／Effort，**不填
`Protocol`**；普通客户端的 `/v1/chat/completions` 走 `ParseProtocolRequest`，那里显式
`Protocol = "chat"`，所以只有桌面端漏了。转发层 `forwardGoogleAgent` 见
`request.Protocol != "chat"` 且不是 responses／messages，直接拒绝。

## Scope

- `internal/core/proxy/request.go`：`parseChatRequest` 统一设置 `Protocol = "chat"`。
- `internal/core/proxy/antigravity_client_test.go`：新增协议回归测试。

## Out of Scope

- 不改路由、目录、适配器或其它客户端行为。
- 不重新打包；由后续构建任务完成。

## Existing Behavior

Antigravity 请求到达转发层时 `Protocol == ""`，所有 Google 模型请求 400。

## Expected Behavior

Antigravity Chat 请求 `Protocol == "chat"`，正常进入 Google 订阅转发路径。

---

# Analysis

## Related Modules

- `internal/core/proxy/request.go`、`agent_request.go`
- `internal/application/gateway/google_agent.go`、`internal/provider/antigravity/chat_request.go`

## Dependencies

无新增依赖。

## Risks

- `parseChatRequest` 同时服务 `ParseRequest`／`ParseConvertedChatRequest`／
  `ParseAntigravityRequest`，三者都是 Chat 语义，显式设置协议不会改变其它调用方行为。
- 修复后请求会真正到达上游，可能暴露下一层错误（内容、工具或档位），需要按错误码继续跟进。

---

# Implementation Plan

1. 创建本记录。
2. `parseChatRequest` 设置 `Protocol = "chat"` 并注明原因。
3. 补测试：桌面端与普通 Chat 解析都得到 `chat`，档位与 stream 不丢失。
4. 跑 Go 全量与 `task verify`，填写结果并提交。
5. 需要重新打包才能让用户验证。

---

# Progress

- 2026-11-12 21:58：创建记录；本地复现任意 Google 模型 400 `unsupported_protocol`，
  定位到 `parseChatRequest` 未填 `Protocol`。

---

# Implementation Result

## Changes

- `internal/core/proxy/request.go`：`parseChatRequest` 在解析时就写入
  `request.Protocol = "chat"`，与 `ParseProtocolRequest` 的 Chat 分支保持一致。
- `internal/core/proxy/antigravity_client_test.go`：新增
  `TestAntigravityParseKeepsChatProtocol`，断言
  `ParseClientRequest("antigravity", "/v1/chat/completions", …)` 与 `ParseRequest` 都得到
  `chat`，且 `reasoning_effort`／`stream` 不丢失。

## Not Changed

- 路由、目录、档位变体与适配器实现。
- 其它客户端与协议（Responses／Messages 仍由 `ParseProtocolRequest` 处理）。

---

# Verification

## Tests

- [x] Unit Tests：`go test ./internal/core/proxy/...` 通过。
- [x] Go 全量：`go test . ./internal/...` exit 0。
- [x] Integration Tests：隔离实例上 5 组真实上游请求全部 200（见下）。
- [ ] E2E Tests：桌面端真机重试待用户安装新构建

## Quality Checks

- [x] Format：`gofmt -l` 无输出。
- [x] Lint：`go vet . ./internal/...` exit 0。
- [x] Type Check：`task --force verify` 内含前端 typecheck，exit 0。
- [x] Test：见上。
- [x] Build：`task --force verify` exit 0。

## Verification Result

```text
go test ./internal/core/proxy/...   → ok
go test . ./internal/...            → exit 0
go vet . ./internal/...             → exit 0
gofmt -l internal/core/proxy internal/application/gateway → 无输出
task --force verify                 → exit 0
```

- 修复前的复现：`curl POST /v1/chat/completions` 任意 Google 模型都返回
  `400 unsupported_protocol`。
- 修复后端到端验证（隔离实例：`OMNIPROXY_PROFILE=development`、
  `OMNIPROXY_DATA_DIR` 指向生产数据的**副本**、端口 3011，不触碰用户正在运行的实例）：

```text
gemini-4-argon（无档位）           → 200  model=gemini-4-argon-high  20.5s
gemini-4-argon + effort=medium      → 200  model=gemini-4-argon-medium 28.1s
claude-opus-4-6                     → 200  model=claude-opus-4-6-thinking
gpt-oss-120b                        → 200  model=gpt-oss-120b-medium
gemini-3.8-flash + low, stream=true → 200  SSE，model=gemini-3.8-flash-low
```

- 期间一次 `gemini-4-argon-medium` 返回 502 `operation_failed`，耗时正好 30 s，重试同样
  参数 28.1 s 成功；属于上游慢响应撞上 30 秒普通推理时限，不是模型不可用。探针实例与
  复制的凭据目录已删除。

---

# Documentation

- [x] PRIMARY：无需修改产品文档
- [x] PRD／Architecture／Tech Stack／Coding Rules／Testing／Build／ADR：无需修改

---

# Git

## Commit

```text
02e2a304 fix(proxy): 标记 Antigravity Chat 请求协议并准备 v2.2.3-beta.14
```

修复与紧接着的版本映射在同一个提交里；归档提交 `4a530c8c`。未打标签、未推送。

---

# Result

Antigravity 桌面端请求在解析层就带上 `chat` 协议，转发层不再把它当成未知协议拒绝；
家族名解析与档位处理的既有修复保持不变。

---

# Completed

2026-11-12 22:02
