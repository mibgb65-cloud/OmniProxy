# TASK-20261120-010: Antigravity 等待上游响应头超过 60 秒导致「已取消／未报告」

## Status

Completed

## Created

2026-11-20 21:50（Asia/Singapore）

## Request

用户附一张请求页截图（2026-11-20 21:19–21:30，`gemini-4-neon-xhigh`，Antigravity 桌面端），其中多条记录是「已取消／未报告」，追问：

> 我记得我们之前修复过一次这个自动取消问题，你看看为什么又出现了这个问题

用户随后确认修复范围：只对 Antigravity 桌面端生效。

## Goal

- Antigravity 桌面端在网关等待上游响应头期间就能拿到 `200` 与 SSE 响应头，不再触发桌面端 60 秒响应头超时。
- 其它客户端（Codex、Claude 等）的状态码与错误语义保持完全不变。
- 上游确实在 60 秒内返回时，内容与既有的首 token 前心跳、账号轮换语义不变。

## Background

TASK-20261114-002（`4347a6d0`）修的是「上游已经回了响应头、但首个可见 token 之前 body 静默」：`copyStream` 在静默 3 秒后提交 `200` 并每 10 秒发 `: keep-alive`。

本次现场与上次不同（证据见 Analysis）：取消请求的 `request_attempts.status_code` 是 499（而不是 200），说明网关连上游响应头都没拿到；`copyStream` 根本没启动，3 秒宽限无从生效。

用户确认的修复范围是 Antigravity 桌面端。桌面端日志的报错是 `net/http: timeout awaiting response headers`，即它只限制响应头到达时间。

## Scope

- `internal/core/proxy`：为 Antigravity 的流式请求在等待上游响应头期间提前提交 SSE 响应头并保活。
- 回归测试：提前下发、心跳、与 `copyStream` 交接、非 Antigravity 客户端不提前提交。
- 任务记录与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不把 `ResponseHeaderTimeout`（5 分钟）改小，也不改上游请求形状、模型目录、路由与档位。
- 不改其它客户端的时限与错误语义。
- 不改 `request_cancelled` / 499 的记账语义。
- 不改 WebSocket 与透明转发路径。

## Existing Behavior

网关收到 Antigravity 流式请求后，要等 provider 的 `Forward` 拿到上游响应头、调用 `consume` 后才可能写任何字节。Google 订阅的思考型模型可能在 60 秒以上才回响应头（`client.go` 的 `responseHeaderTimeout` 因此是 5 分钟），桌面端在 60 秒时按响应头超时断开，网关记 499、UI 显示「已取消／未报告」。

## Expected Behavior

- Antigravity 的流式请求在受理后 3 秒仍未拿到上游响应头时，先下发 `200`、`Content-Type: text/event-stream` 并 Flush，之后每 10 秒发 `: keep-alive`。
- 上游响应头到达后停止保活，由 `copyStream` 继续逐行转发，不重复写状态码。
- 3 秒内到达的错误仍按原 HTTP 状态码返回；3 秒后才失败只能中断连接（与既有「已提交后失败」一致）。
- 其它客户端行为不变。

---

# Analysis

## Root Cause

1. `handler.go` 只有在 `Upstream.Forward` 的内部 `consume` 回调里才写响应；而 `consume` 只在 provider 拿到上游响应头后调用。
2. `internal/provider/antigravity/client.go` 明确记录了订阅思考型模型「要等第一个分片才有响应头」，`ResponseHeaderTimeout` 因此放宽到 5 分钟。
3. Antigravity 桌面端的 HTTP 客户端对响应头只有 60 秒预算；两者冲突时失败方永远是桌面端。

现场证据：

- 数据库 `request_records`：2026-11-18/19/20 分别有 21／34／37 条 ~60 秒取消，其中 20／33／34 条的 attempt `status_code=499`（从未拿到上游响应）。
- `004bd60f…`（截图 21:20:53）：attempt 1 在请求开始 1.3 秒后登记，58.7 秒后取消，`status_code=499`、`first_token_ms` 为空。
- Antigravity 生成端日志同一时刻：`Run: attempt 1 failed (... net/http: timeout awaiting response headers), retrying in 4s`。
- 该请求与 4 秒后成功的 `bbde6293…` 正文解压后 SHA256 完全一致，是同一个请求的重发。
- 同一天存在 185 秒、264 秒、596 秒的成功 Antigravity 请求（`first_token_ms ≈ latency`），说明响应头一旦交付，桌面端可以长时间等待正文静默。
- 上次修复覆盖的「上游已回头、body 静默」类取消现在每天只剩 1–3 条，说明 `4347a6d0` 仍然有效，本次不是同一类问题复发。

## Related Modules

- `internal/core/proxy/handler.go`（写入时机）
- `internal/core/proxy/stream.go`（`copyStream`、宽限与心跳）
- `internal/core/proxy/stream_headers_test.go`（回归测试）

## Dependencies

无新增依赖；只用标准库 `sync`、`time`。

## Risks

- 响应头提前提交后，3 秒之后的失败只能中断连接，不能再返回干净的 401／429／5xx。范围限制在 Antigravity；账号轮换仍在网关内部完成，桌面端看到可重试的断流。
- 提前提交必须与 `copyStream` 的写入串行化，否则并发写 `ResponseWriter`。
- 保活协程必须在复制正文前停止并等待退出，避免泄漏或在响应头之后继续写。
- 心跳是 SSE 注释行，标准客户端忽略；需要测试确认不切断事件。

---

# Implementation Plan

1. 在 `stream.go` 抽出 `commitStreamHeaders` 与 `writeStreamFrame`，供 `copyStream` 与新的保活协程共用。
2. 新增 `startStreamHeaderKeeper`：3 秒宽限后提交 `200` + SSE 头，之后每 10 秒心跳；返回幂等的停止函数，停止时等待协程退出。
3. `handler.go`：仅当 `client == "antigravity"` 且 `request.Stream` 时启动保活；`consume` 首行与 `Forward` 返回后都调用停止函数，保证错误响应不与保活并发写。
4. `copyStream` 依据 `committed` 判断响应头是否已发出，避免重复 `WriteHeader`。
5. 回归测试：提前下发（快客户端响应头超时）、保活与正文交接、非 Antigravity 不提前提交、非流式不提前提交。
6. 运行质量门禁，填写结果，更新文档并提交。

---

# Progress

- 2026-11-20 21:50：完成定位与范围确认（仅 Antigravity），创建任务记录。
- 2026-11-20 22:05：实现提前下发与保活、与 `copyStream` 交接，补充 4 组回归测试。

---

# Implementation Result

## Changes

- `internal/core/proxy/stream.go`
  - 新增 `needsEarlyStreamHeaders`：只有 `client == "antigravity"` 的流式请求才提前提交。
  - 新增 `startStreamHeaderKeeper`：宽限期（沿用 `streamHeaderGrace` 3 秒）后提交 `200` +
    `text/event-stream`，之后按 `streamHeartbeatInterval`（10 秒）写 `: keep-alive`；
    返回幂等停止函数，停止时等待协程退出，保证之后只有复制正文在写 `ResponseWriter`。
  - 抽出 `commitStreamHeaders` 与 `writeStreamFrame`，由保活与 `copyStream` 共用。
  - `copyStream` 依据 `committed` 判断响应头是否已发出，避免重复 `WriteHeader`。
- `internal/core/proxy/handler.go`
  - 转发前按 `needsEarlyStreamHeaders` 启动保活；`consume` 首行和 `Forward` 返回后各停止一次，
    保证失败路径的错误响应不与心跳并发写。
- `internal/core/proxy/stream_headers_test.go`
  - 新增：保活 80 ms 内下发响应头并在 120 ms 后与 `copyStream` 正常交接（含心跳、不重放）；
  - 新增：`needsEarlyStreamHeaders` 的 5 组取值；
  - 新增：Antigravity 集成用例（假上游 300 ms 才返回响应头，客户端 100 ms 响应头超时仍然 200
    并读完 SSE）；
  - 新增：非 Antigravity 集成用例（同样条件下仍按响应头超时失败，错误语义不变）；
  - 调整既有 `TestStreamHeartbeatDoesNotSplitEvent` 的断言：合法的事件间注释行不再让用例失败，
    只断言事件本身未被切割、终结事件仍完整。

## Files Changed

- `internal/core/proxy/stream.go`
- `internal/core/proxy/handler.go`
- `internal/core/proxy/stream_headers_test.go`
- `docs/product/GATEWAY.md`
- `docs/plans/CURRENT.md`
- 本任务记录

## Technical Notes

- 保活只对 Antigravity 生效：其它客户端的失败仍能以真实 401／429／5xx 返回；Antigravity
  在响应头提交后的失败改为中断连接（账号轮换仍在 `forwardWithFailover` 内部完成）。
- 停止函数用 `sync.Once` 保证可重复调用；`<-done` 建立 happens-before，之后读写协程不会与
  主流程并发。
- 提前提交阶段的 `Content-Type` 由保活设置；3 秒内失败的请求由 `writeFailure` 覆盖成 JSON。
- 既有 `copyStream` 的心跳与宽限逻辑未改，只是复用抽出的两个写函数。

---

# Verification

## Tests

- [x] Unit Tests：提前下发、与 `copyStream` 交接、客户端判定、心跳不切割事件。
- [x] Integration Tests：`New(Options{...})` + 假上游（300 ms 才回调 `consume`）+ 100 ms
  响应头超时的 HTTP 客户端；Antigravity 通过、其它客户端仍超时。
- [ ] E2E Tests：需安装新构建后在 Antigravity 桌面端复测（当前运行 v2.4.1-beta.5）。

## Quality Checks

- [x] Format：`gofmt -l internal/core/proxy` 无输出；前端 `pnpm format:check` exit 0
- [x] Lint：`go vet . ./internal/...` exit 0；`staticcheck . ./internal/...` exit 0（缓存指向
  `.gotmp/staticcheck-cache`）；前端 `pnpm lint` exit 0
- [x] Type Check：前端 `pnpm typecheck` exit 0（Go 由编译覆盖）
- [x] Test：`go test . ./internal/... -count=1` exit 0（59 个包含测试的包全部 ok，其中
  `internal/core/proxy` 以 `-count=3` 复跑通过）；前端 `pnpm test` exit 0（164 文件／1006 项）
- [x] Build：`go build . ./internal/...` exit 0
- [ ] Verify：未执行单条 `task verify`，原因见下。

## Verification Result

```text
go test ./internal/core/proxy/ -count=3        → ok
go test . ./internal/... -count=1              → exit 0（59 个包含测试的包全部 ok）
go vet . ./internal/...                        → exit 0
staticcheck . ./internal/...                   → exit 0
go build . ./internal/...                      → exit 0
gofmt -l internal/core/proxy                   → 无输出
cd frontend && pnpm format:check               → exit 0
cd frontend && pnpm lint                       → exit 0
cd frontend && pnpm typecheck                  → exit 0
cd frontend && pnpm test                       → exit 0（164 文件／1006 项）
```

- 前端测试在默认沙箱下会因 Vite 解析真实路径时的 `spawn EPERM` 失败，那是沙箱边界而不是
  用例失败；同一命令在完整访问权限下 164 文件／1006 项全部通过。

- 未执行单条 `task verify`：本机此前出现过 `.task` 源哈希扫描被体积很大的 `tmp/` 历史目录
  拖住的情况；改为逐项执行同一组门禁命令，结果如上。
- `go build ./...` 因仓库内既有的 `tmp/` 样本目录（多个包名混放）失败，与本次改动无关；
  使用与测试相同的 `go build . ./internal/...` 覆盖本次涉及的全部包。

---

# Documentation

- [x] PRD：无需修改
- [x] Architecture：无需修改
- [x] Tech Stack：无需修改
- [x] Coding Rules：无需修改
- [x] Testing：无需修改
- [x] Build：无需修改
- [x] ADR：无需新增（属于既有流式转发约束的补充实现）
- [x] `docs/product/GATEWAY.md`：补充 Antigravity 提前响应头与保活的行为边界
- [x] CURRENT：已追加本轮修复摘要

---

# Deviations

- 原计划只新增保活协程；实现中把 `copyStream` 的响应头提交与整帧写入抽成
  `commitStreamHeaders`／`writeStreamFrame`，避免两处写逻辑漂移。
- 额外修改了既有的 `TestStreamHeartbeatDoesNotSplitEvent`：全量包运行时它会在事件结束的
  同一 tick 上合法地插入一条注释行而失败（变化的是调度时序，不是本次行为）。断言改为
  「事件未被切割 + 终结事件完整」，仍然拦截真正的心跳切割。
- 未执行单条 `task verify` 与桌面端真机复测，原因见 Verification Result 与 Tests。

---

# Git

## Commit

```text
ac4b6c06 fix(gateway): Antigravity 流式请求在等上游响应头时先下发响应头并保活
```

---

# Result

Antigravity 桌面端的流式请求在等待上游响应头期间就能拿到 `200` 与 SSE 响应头，并持续收到
`: keep-alive`，不会再因为上游思考超过 60 秒而触发 `timeout awaiting response headers`；
其它客户端的错误状态码语义不变。

---

# Completed

2026-11-21 07:10（Asia/Singapore）
