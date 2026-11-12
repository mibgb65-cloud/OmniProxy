# TASK-20261112-017: Antigravity 会话撞 30 秒响应头上限导致 502

## Status

In Progress

## Created

2026-11-12 22:00（Asia/Singapore）

## Request

用户安装 `v2.2.3-beta.14` 后仍然失败（“Agent terminated due to error”）。需要继续定位。

## Goal

- 让订阅思考型模型（Gemini 4 Argon 等）在首 token 之前思考超过 30 秒时不再被判成网络失败。
- 桌面端长时间流式会话不再被普通请求的 120 秒时限截断。
- 其它客户端与请求类型保持原时限。

## Background

beta.14 安装后（进程 21:51 启动），请求明细全部变成：

```text
13:51:47  model=gemini-4-argon-high  status=502  code=operation_failed  latency=31100ms  stream=1  effort=high
13:52:22  …  30100ms
13:52:58  …  30107ms
13:53:41  …  30103ms
13:54:33  …  30119ms
```

每次都在 30.1 秒左右失败，且只发生在 Antigravity（其它客户端正常）。协议与模型解析都已正确
（`model=gemini-4-argon-high`、`effort=high`、`provider=builtin_google`）。

根因：`internal/provider/antigravity/client.go` 的 `New()` 把
`ResponseHeaderTimeout` 设为 30 秒。订阅的思考型模型在第一个 token 之前可能先想几十秒到
几分钟，流式请求也要等第一个分片才有响应头；超过 30 秒就被传输层中止，映射成
`operation_failed` 502。上一轮隔离实例里 20–28 秒成功的请求也贴近这条线，说明这是常态而非偶发。

## Scope

- `internal/provider/antigravity/client.go`：响应头等待改为 5 分钟常量。
- `internal/core/proxy/compaction_deadline.go`：新增 `clientRequestTimeout`，Antigravity 至少 10 分钟。
- `internal/core/proxy/handler.go`：使用 `clientRequestTimeout`。
- 相应单元测试。

## Out of Scope

- 不改上游请求形状、模型目录、路由与档位处理。
- 不改其它客户端的超时。

## Existing Behavior

Antigravity 会话在 30 秒被上游响应头超时中止，返回 502。

## Expected Behavior

首 token 较慢的思考型请求正常完成；长时间流式会话不被 120 秒截断；连接不可达仍由
TLS／拨号超时快速失败。

---

# Analysis

## Related Modules

- `internal/provider/antigravity/client.go`
- `internal/core/proxy/handler.go`、`compaction_deadline.go`

## Dependencies

无新增依赖。

## Risks

- 放宽响应头等待后，真正无响应的上游会占用连接更久；连接与 TLS 握手仍是 10 秒，且总体
  请求时限为 10 分钟。
- 放宽客户端时限只对 `antigravity` 生效，其它客户端仍用 120 秒。

---

# Implementation Plan

1. 记录（本轮在改动后补写，见 Deviations）。
2. 响应头等待 30 秒 → 5 分钟，集中为常量并注明原因。
3. `clientRequestTimeout`：Antigravity 至少 10 分钟，保持压缩请求的更宽时限。
4. 补测试：常量下限、`http.Client.Timeout` 必须为 0、各客户端时限选择。
5. 隔离实例验证：构造一个超过 30 秒的真实流式请求。
6. 跑 `task verify`，填写结果并提交。

---

# Progress

- 2026-11-12 22:00：定位到 30 秒 `ResponseHeaderTimeout`；完成改动与测试。
- 隔离实例验证：60.3 秒的流式请求 200（修复前必然 30 秒失败），9.1 秒的常速请求仍正常。

---

# Implementation Result

## Changes

- `internal/provider/antigravity/client.go`：新增 `responseHeaderTimeout = 5 * time.Minute`，
  `New()` 使用它；TLS 握手仍 10 秒，`http.Client.Timeout` 仍为 0（否则会切断长流）。
- `internal/core/proxy/compaction_deadline.go`：新增 `antigravityTimeout = 10 * time.Minute`
  与 `clientRequestTimeout(client, request)`；压缩请求的更宽时限保持不变。
- `internal/core/proxy/handler.go`：请求上下文改用 `clientRequestTimeout`。
- 测试：`antigravity_timeout_test.go`（只放宽 Antigravity、配置更长时不压低、压缩请求仍放宽）；
  `client_test.go` 增加响应头时限下限与整请求时限为 0 的断言。

## Not Changed

- 上游请求形状、目录、路由、档位与其它客户端超时。

---

# Verification

## Tests

- [x] Unit Tests：`go test ./internal/core/proxy/... ./internal/provider/antigravity/...` 通过。
- [x] Go 全量：`go test . ./internal/...` 通过（见下）。
- [x] Integration Tests：隔离实例上一个 60.3 秒的真实流式请求返回 200。
- [ ] E2E Tests：桌面端真机重试待用户安装新构建。

## Quality Checks

- [x] Format：`gofmt -l` 无输出。
- [x] Lint：`go vet . ./internal/...` exit 0。
- [x] Test：见上。
- [x] Verify：`task --force verify` exit 0。

## Verification Result

```text
go test ./internal/core/proxy/... ./internal/provider/antigravity/...  → ok / ok
go test . ./internal/...                                              → exit 0
go vet . ./internal/...                                               → exit 0

隔离实例（端口 3011，生产数据副本，未触碰运行中的应用）：
gemini-4-argon + 约 8 KB system，stream=true  → 200，60.3 s，model=gemini-4-argon-high
gemini-3.8-flash + low，stream=true           → 200，9.1 s，model=gemini-3.8-flash-low
```

- 修复前同样的长请求会在 30.1 秒返回 `operation_failed`，与用户请求明细完全一致。

---

# Documentation

- [x] PRIMARY：无需修改产品文档
- [x] PRD／Architecture／Tech Stack／Coding Rules／Testing／Build／ADR：无需修改

---

# Deviations

- 本轮直接沿用上一轮的连续诊断，代码改动先于本记录写出，记录随后补齐（含实际改动与验证）。

---

# Git

## Commit

（实现提交与归档提交待回填）

---

# Result

Antigravity 会话不再被 30 秒响应头超时打断：思考型模型可以想几分钟，流式会话最长 10 分钟；
其它客户端时限不变。

---

# Completed

2026-11-12 22:06
