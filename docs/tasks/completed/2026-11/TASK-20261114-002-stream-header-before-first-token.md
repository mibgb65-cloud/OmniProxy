# TASK-20261114-002: 流式响应在首个 token 前不发送响应头，Antigravity 桌面端 60 秒取消

## Status

Completed

## Created

2026-11-14 19:45（Asia/Singapore）

## Request

用户截图显示请求页连续 6 条 `gemini-4-hydrogen-xhigh` 记录都是「已取消 / 未报告」，追问「它老是被取消是什么情况？」并要求查看生成端日志。定位后用户回复「可以。你看看按你的方法处理这个问题」。

## Goal

- 流式请求在等待首个可见 token（含长时间思考）期间不再让下游连接完全静默。
- Antigravity 桌面端不再因 60 秒等不到响应头而放弃请求，也就不会再出现连续的「已取消」。
- 立即失败（非 200、首个事件即错误）仍按 HTTP 错误返回，保持既有账号轮换语义。

## Background

用户请求全部来自 Antigravity 桌面端（`client_id=antigravity`，`model=gemini-4-hydrogen-xhigh`，`reasoning_effort=xhigh`），上游是 Google 订阅账号 `mibgb65@gmail.com`。

生成端日志 `%APPDATA%\Antigravity\logs\language_server.log`：

```text
19:14:28.927  Making Gateway streaming request to: http://127.0.0.1:3000/v1/chat/completions
19:15:28.941  Run: attempt 1 failed (... net/http: timeout awaiting response headers), retrying in 4s
19:16:32.965  Run: attempt 2 failed (... net/http: timeout awaiting response headers), retrying in 5.7s
19:18:49.625  Run: attempt 4 failed (... ), retrying in 27.3s
19:22:02.721  Run: attempt 6 failed (... ), retrying in 1m55s
19:23:57.785  Making Gateway streaming request ...            ← 第 7 次 11 秒成功
```

每次失败都是 60.0 秒后的 `timeout awaiting response headers`：桌面端 HTTP 客户端把 60 秒作为**响应头**等待上限。

网关日志同一时刻的记录是 `status_code=499`、`code=request_cancelled`、`latency_ms=6000x`、`input_tokens=0`、`usage_state=unknown`。数据库 `request_attempts` 里这些请求的上游 `status_code=200`、`first_token_ms` 为 NULL，说明上游已经接受请求，但一个可见 token 都没有产出时客户端就断开了。

## Root Cause

1. `internal/core/proxy/response.go` 的 `copyStream` 只 `Set` 了 `Content-Type`，直到读到第一条上游 SSE 数据才 `w.Write`；Go 的 `net/http` 在第一次写入正文前不会发送响应头。
2. `internal/provider/antigravity/chat_response.go` 明确丢弃思考分片（`if part.Text != "" && !part.Thought`），`forward.go` 在 `len(delta) == 0` 时继续读上游、不产出任何字节。
3. `xhigh` + 9–11 万 token 上下文的思考时间偶发超过 60 秒，于是下游在客户端看来整段静默，触发其响应头超时并断开；网关按「客户端断开」记成 499 已取消，UI 显示「已取消 / 未报告」。

桌面端的重试链（4s → 5.7s → 10.9s → 27s → 45.7s → 1m55s）让**一次任务在请求页里显示成 6 条「已取消」**，放大了体感。

## Scope

- `internal/core/proxy`：流式响应在静默期提交响应头并按固定间隔发送 SSE 注释心跳。
- 新增针对响应头提前下发、心跳、立即失败语义的回归测试。
- 任务记录与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不改 Antigravity 桌面端（生成端）的超时配置，它不可配置。
- 不改上游请求形状、模型目录、路由、账号轮换策略。
- 不改 passthrough（透明转发）与 WebSocket 路径，它们本来就在开头写响应头。
- 不改 `request_cancelled` / 499 的记录语义（客户端真的断开时仍是取消）。

## Existing Behavior

上游在首个可见 token 前思考超过 60 秒时，下游连接 0 字节，客户端按响应头超时断开，网关记 499。

## Expected Behavior

- 上游静默超过 `streamHeaderGrace`（3 秒）后立即下发 `200` 与 SSE 响应头。
- 之后每 `streamHeartbeatInterval`（10 秒）发送一条 SSE 注释 `: keep-alive`，直到流结束。
- 首个事件在 3 秒内到达并且是错误时，仍返回原来的 HTTP 错误状态，账号轮换不受影响。
- 客户端仍能拿到完整的 `data:` 事件序列与 `[DONE]`。

---

# Analysis

## Related Modules

- `internal/core/proxy/response.go`（`copyStream`、`copyResponse`）
- `internal/core/proxy/handler.go`（`committed` 决定失败是写状态码还是中断连接）
- `internal/core/proxy/trace.go`（首 token 计时、`hasFirstContent`）

## Dependencies

无新增依赖；只用标准库 `time`、`bufio`。

## Risks

- 响应头一旦提交，之后的上游错误只能中断连接（与现有「已提交后」行为一致），不能改成 502；用 3 秒宽限把立即失败挡在提交之前。
- 心跳是 SSE 注释行，标准客户端会忽略；需要回归测试确认不影响首 token 计时与既有解析。
- 读取改为泵协程 + `select`，必须保证退出时协程能结束，不能泄漏。
- 主观参数（3 秒 / 10 秒）需要写进常量并注明理由。

---

# Implementation Plan

1. 新增 `internal/core/proxy/stream.go`，把 `copyStream` 从 `response.go` 移出（文件体积上限）。
2. 读取改为泵协程写入 channel；主循环 `select` 上游行、宽限计时器、心跳计时器，所有写仍只在主协程。
3. 宽限到点或首帧到达时提交 `200` + 响应头并 Flush；之后静默期发 `: keep-alive`。
4. 保持既有逐行校验、`maxStreamBytes`、`[DONE]`／`response.completed` 终止与首 token 计时逻辑。
5. 新增测试：响应头先于首 token、静默期心跳、首个事件即失败时未提交。
6. `task verify`，填写结果，更新文档并提交。

---

# Progress

- 2026-11-14 19:45：完成定位与方案确认，开始实现。
- 2026-11-14 19:55：`stream.go` 与回归测试完成，代理包全量测试通过。
- 2026-11-14 20:05：全仓 Go 测试与前端测试通过；发现心跳可能切断未完成事件，补充跳过逻辑与测试。
- 2026-11-14 20:15：格式／lint／类型检查／测试／构建全部通过，完成记录。

---

# Implementation Result

## Changes

- `internal/core/proxy/stream.go`（新增）：`copyStream` 从 `response.go` 迁入并重写。读取放到泵协程写入 channel，主循环 `select` 上游行、`streamHeaderGrace`（3 秒）与 `streamHeartbeatInterval`（10 秒），所有写入仍只发生在调用协程。
- 上游静默 3 秒即提交 `200`、`Content-Type: text/event-stream` 并 Flush；之后静默期每 10 秒写一条 SSE 注释 `: keep-alive`。
- 心跳只在没有未完成事件时发送：注释行自带的空行会提前终结正在拼装的 SSE 事件。
- 首个事件在宽限期内到达且非法时，仍是 `upstream_stream_failed` 502 且不提交响应头，账号轮换语义不变。
- `internal/core/proxy/response.go`：移除 `copyStream` 与不再使用的 `strings` 导入。
- `internal/core/proxy/stream_headers_test.go`（新增）：4 个回归测试。

## Files Changed

- `internal/core/proxy/stream.go`
- `internal/core/proxy/response.go`
- `internal/core/proxy/stream_headers_test.go`
- `docs/tasks/…/TASK-20261114-002-stream-header-before-first-token.md`
- `docs/plans/CURRENT.md`

## Technical Notes

- 宽限期存在的理由：立即提交响应头会让「首个事件即错误」无法再返回 HTTP 状态码，只能中断连接，从而丢掉按错误码轮换账号的能力。3 秒覆盖即时拒绝，又远小于桌面端 60 秒的响应头时限。
- 中断后的行为与既有「已提交后的上游错误」一致：`handler.go` 记录错误码后 `panic(http.ErrAbortHandler)`，客户端明确看到未完成响应。
- 泵协程用 `stop` channel 与 channel 关闭保证退出；上游阻塞在读取时由提供方在 `Forward` 返回后关闭响应体解除。

---

# Verification

## Tests

- [x] Unit Tests：新增 4 项，覆盖响应头先于首 token、静默期心跳、心跳不切割事件、首个事件失败不提交。
- [x] Integration Tests：`httptest` 真实 HTTP 服务 + 客户端 `ResponseHeaderTimeout` 复现修复前后差异。
- [ ] E2E Tests：需要在 Antigravity 桌面端安装新构建后复测。

## Quality Checks

- [x] Format：`task format:check` exit 0（Prettier 全部通过，`gofmt -l` 无输出）
- [x] Lint：`task lint` exit 0（staticcheck／go vet 无输出，ESLint 仅 2 条既有 warning）
- [x] Type Check：`task typecheck` exit 0
- [x] Test：`task test` exit 0（Go 全包通过，前端 154 文件／935 项通过）
- [x] Build：`task build` exit 0（Windows production 构建）
- [x] Verify：见下方说明，逐项执行等价于 `task verify`

## Verification Result

```text
go test ./internal/core/proxy/ -count=1                → ok
go test . ./internal/... -count=1                      → 全部 ok（exit 0）
task format:check                                      → exit 0
task lint                                              → exit 0
task typecheck                                         → exit 0
task test                                              → exit 0（前端 154 文件／935 项）
task build                                             → exit 0
```

- 新增测试在修复前会失败：`TestStreamHeadersArriveBeforeFirstToken` 用 200 ms 响应头超时的客户端请求一个 400 ms 才出首帧的上游，旧实现必然超时。
- 单次调用 `task verify` 在本机沙箱内卡住 12 分钟无子进程（`.task` 源哈希扫描遇到体积很大的 `tmp/` 历史目录）；改为逐项执行同一组门禁命令，结果如上，均为 exit 0。

---

# Documentation

- [x] PRD：无需修改（不改变产品契约）
- [x] Architecture：无需修改
- [x] Tech Stack：无需修改
- [x] Coding Rules：无需修改
- [x] Testing：无需修改
- [x] Build：无需修改
- [x] ADR：无需新增（属既有流式转发实现修正）
- [x] CURRENT：已追加本轮修复摘要

---

# Deviations

- 原计划「宽限到点或首帧到达时提交响应头」，实现中首帧到达时由 `writeFrame` 隐式完成提交，效果相同，少一次写状态码。
- 增加原计划未写明的保护：心跳遇到未完成事件时跳过，避免注释行的空行切断事件。
- 未执行单次 `task verify`，原因见 Verification Result；五个门禁步骤分别执行且全部 exit 0。

---

# Git

## Commit

见完成记录（实现提交与归档提交）。

---

# Result

流式响应在上游思考超过 3 秒后立即下发 `200` 与 SSE 响应头，并在静默期每 10 秒发送 `: keep-alive` 注释；Antigravity 桌面端的 60 秒响应头超时不再触发，连续「已取消／未报告」的记录应消失。立即失败与账号轮换语义保持不变。

---

# Completed

2026-11-14 20:15（Asia/Singapore）

