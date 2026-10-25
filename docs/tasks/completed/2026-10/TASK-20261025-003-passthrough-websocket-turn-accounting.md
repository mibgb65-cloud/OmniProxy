# TASK-20261025-003: 透明转发 WebSocket 逐轮请求与用量记账

## Status

Completed

## Created

2026-10-25 09:53

## Request

> 还有就是在 codex 使用 websocket 的时候，这个请求的记录以及 token 的记录是不是没法进入到记录里面？

诊断结论：是。旧协议入口（`/v1/responses` 的 protocol socket）每个 `response.create` 帧建一条请求并解析
上游 `response.completed` 的 usage，历史库里 2260 条 WS 成功记录的 token 齐全；但切换到 Codex 透明转发后，
[`passthrough_socket.go`](../../../internal/core/proxy/passthrough_socket.go) 只做双向原样中继，
**既不建请求记录也不解析用量**，连换号的 attempt 明细都没有。TASK-20261024-015 的 Remaining Risks 已登记该缺口。

## Goal

透明转发 WebSocket 连接上，每一轮 `response.create` 都产生与 HTTP 同构的请求记录：
账号／模型／transport=websocket／首字耗时／上游 usage，失败轮记录真实错误码。

## Scope

- 新增 `internal/core/proxy/passthrough_socket_accounting.go`：在已转发帧上旁路记账（不改字节）。
- `relaySocket` 增加记账钩子；`dialPassthroughSocket` 返回本次成功尝试，供父记录绑定账号身份。
- 新增测试：一轮成功带 usage、连接提前断开记取消、`generate:false` 预热不建记录。

## Out of Scope

- 不改变握手、换号、冷却与 `session.Done` 的既有语义。
- 不实现连接级「连接寿命」记录（按轮记账，与旧协议入口一致）。
- 不改前端请求页字段（`protocol / transport` 已能显示 websocket）。

## Existing Behavior

透明转发 WebSocket：帧原样中继，请求列表看不到这条流量，Token 用量不增长。

## Expected Behavior

- 每轮 `response.create`（不含 `generate:false`）→ 一条 `transport=websocket` 记录，
  `BeginAttempt` 记录真实账号与状态码，`response.completed/incomplete` 到达时写入 input/output/cached tokens。
- 未收到终结事件就断开 → 该轮记为 `request_cancelled`（499），与 HTTP 侧语义一致。
- 中转字节不变；记账失败才中断该轮（与 HTTP 透明转发「记账不可用即 503」一致）。

---

# Analysis

## Related Modules

- `internal/core/proxy/passthrough_socket.go`（中继）
- `internal/core/proxy/trace.go` / `attempt.go` / `usage.go` / `first_token.go`（既有记账原语）
- `internal/application/activity/observer.go`（落库，不需改动）

## Dependencies

N/A

## Risks

- 旁路解析带来开销：只解析文本帧，且只在帧首判定事件类型；不改帧内容。
- 一轮多帧累积：usage 只取终结事件（`ParseUsage` 对流式 responses 只认 `response.completed/incomplete`）。
- 记账失败中断连接的风险：仅在 `accounting_unavailable` 与 usage 入库失败时中断，与既有 HTTP 路径一致。

---

# Implementation Plan

1. `dialPassthroughSocket` 返回 `(conn, attempt, error)`，把成功握手的账号快照带到连接层。
2. 新增 `socketAccounting`：`clientFrame`（开轮）、`upstreamFrame`（首字＋终结＋usage）、`close`（结束未完成轮）。
3. `relaySocket` 在两条方向插入钩子；客户端帧先记账再转发，上游帧先转发再记账。
4. 单测：成功轮带 usage、断开记取消、预热不建记录。

---

# Progress

- 根因定位：`passthrough_socket.go` 全文件无 `beginTrace`；对照 `websocket.go:166` 每轮建 trace。

---

# Implementation Result

## Changes

实际完成：

- 新增 `socketAccounting`：客户端每个 `response.create` 帧（`generate:false` 预热除外）在
  转发前建立 `transport=websocket` 请求记录、登记本轮账号尝试并把真实账号身份写入父记录；
  上游终结事件（`response.completed`／`response.incomplete`）到达后写入该轮
  input／output／缓存 Token，并按成功收尾。
- 首字耗时复用既有判定（首个含内容的上游帧），不再为空。
- 中继结束时的收尾按可证实原因分类：客户端在终结事件前断开 → `request_cancelled`（499）；
  上游中途断开 → `upstream_disconnected`（502）；握手期失败仍走既有状态码。
- 记账不可用（`accounting_unavailable`）时不把该轮送给上游，与 HTTP 透明转发一致；
  其余解析失败只丢统计，不改字节。
- `dialPassthroughSocket` 额外交回成功握手的尝试身份，`relaySocket` 增加两个记账钩子。

## Files Changed

- `internal/core/proxy/passthrough_socket_accounting.go`（新增）
- `internal/core/proxy/passthrough_socket.go`
- `internal/core/proxy/passthrough_socket_accounting_test.go`（新增）
- `internal/core/proxy/socket_handshake_test.go`（签名调整）
- `docs/product/GATEWAY.md`、`docs/product/ACTIVITY.md`

## Technical Notes

- 按轮记账而不是按连接：与旧协议入口 `websocket.go` 的每轮 trace 口径一致，
  也让额度估算可以按账号／成员／工作区累计到「请求级」样本。
- usage 只取终结事件（`ParseUsage(..., stream=true)` 对流式 responses 本就只认终结事件），
  避免把中间快照重复计入。
- 客户端帧先记账再转发：上游消费不能发生在没有记录的情况下；上游帧先转发再统计：
  统计失败不能影响已经送达客户端的字节。

---

# Verification

## Tests

- [x] Unit Tests（`internal/core/proxy`：整轮 usage＋账号绑定、两轮各自成条、
      客户端断开记取消、上游断开记失败、`generate:false` 预热不记账）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task verify
PASS（format:check / lint: go vet + staticcheck + eslint / typecheck: tsc /
Go 全包测试，含 internal/core/proxy 4.534s / 前端 128 文件 760 项测试 / Windows production 构建）
```

加锁改动后重跑：`go test ./internal/core/proxy/ ./internal/application/activity/` 通过，
`task verify` 再次 exit 0（前端 128 文件／760 项测试）。
未执行 `go test -race`：本机没有 gcc，`CGO_ENABLED=1` 无法构建 runtime/cgo；
两道中继方向的并发访问改由 `socketAccounting.mu` 串行化。

未执行：真实 Codex 客户端经透明入口使用 WebSocket 的实机复核（需要用户重启客户端并安装带本次修改的构建）。

---

# Documentation

- [x] `docs/product/GATEWAY.md`：透明 WebSocket 改为「不解码帧内容但旁路逐轮记账」，补记账规则条目。
- [x] `docs/product/ACTIVITY.md`：补「透明转发 WebSocket 按轮记录」。
- [x] `docs/tasks/active/TASK-20261024-015-...`：把已登记的风险条目标注为已由本任务处理。
- [x] `docs/plans/CURRENT.md`

---

# Deviations

原计划只在 `upstreamFrame` 里读取用量；实际补了 `EndAttempt` 与
`closeTurn` 的收尾路径，否则未收到终结事件的轮次会留下悬空的 pending 尝试明细。

---

# Git

## Commit

```text
e539de9 feat(gateway): 透明转发 WebSocket 按轮记录请求与用量
b2b215f fix(gateway): 透明 WebSocket 轮次状态加锁
```

最后一笔把轮次状态加锁：客户端帧与上游帧在各自 goroutine 中中继，`turn` 会被两侧同时访问。

---

# Result

透明转发的 WebSocket 现在与 HTTP 同构地记账：每轮一条 `transport=websocket` 请求、
一条真实账号尝试、终结事件带来的 input／output／缓存 Token 与首字耗时；
客户端中断记 `request_cancelled`，上游中断记 `upstream_disconnected`，预热轮不记账。
帧字节保持原样，只有记账不可用时才拒绝把该轮送给上游。

---

# Completed

2026-10-25 10:35
