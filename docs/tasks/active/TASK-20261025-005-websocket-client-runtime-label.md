# TASK-20261025-005: WebSocket 请求补上 Codex 运行形态归属

## Status

In Progress

## Created

2026-10-25 11:15（Asia/Singapore）

## Request

> 我们昨天不是做了这个 Codex CLI / Desktop 的识别吗？怎么现在又失效了，那个代码没有合并到 main 分支吗？

用户附一张请求行截图：`2026-10-25 10:48:26 / gpt-6.1-sol / Codex CLI / Desktop / mibgb65@gmail.com / 进行中 / 未报告`。

## Goal

WebSocket 请求记录与 HTTP 一样把 Codex 归属细化成 `codex-cli`／`codex-desktop`／`codex-vscode`，
不再退回笼统的 `codex`（界面显示「Codex CLI / Desktop」）。

## Background

识别代码在 main 上，没有丢：

- `internal/core/proxy/client_identity.go` 的 `recordedClient` → `agentendpoint.Runtime`（TASK-20261024-015，提交 `e8f0756`）。
- HTTP 透明入口 `passthrough.go` 创建 trace 时调用 `recordedClient(access.client, r.Header)`，
  生产库里 205 条 `codex-desktop` 的 HTTP 记录表明它工作正常。

生产库实测（只读副本 + WAL）：

```text
1b44a94da7ed31561d46294cf565fd9d  2026-10-25T02:48:26Z  responses  websocket
client_id=codex  state=success  200  input=226461  output=2287  email=mibgb65@gmail.com
```

截图正是这一行（截图时 `state=pending`，所以各列为 `—`／`未报告`）。
所以不是"没合并"，是 **WebSocket 路径没有调用这套细化**：

- 透明入口（本次 Beta.38 新增的逐轮记账）把 `access.client`（前缀归属，值就是 `codex`）直接交给了 `beginTrace`。
- 旧协议入口 `websocket.go` 同样直接使用 `access.client`，从未细化过；历史 WS 记录也全是 `codex`。

`access.client` 不能直接改写：`reauthorize` 每帧用 `current.client != access.client` 判定
`client_mismatch`，把细化后的值写回会误判成 403。

## Scope

- `passthrough_socket.go`／`passthrough_socket_accounting.go`：连接级细化一次，交给逐轮记账。
- `websocket.go`：协议入口按同样规则细化（不改动 `access.client` 原始值）。
- 测试：透明 WS 的 `codex_cli_rs` → `codex-cli`、`codex desktop` → `codex-desktop`。

## Out of Scope

- 不改变 `agentendpoint.Runtime` 的标记表与未知来源回退 `codex` 的行为。
- 不回溯修正历史记录的 `client_id`。
- 顺带发现（不在本任务）：`state=pending` 的 WS 记录在界面显示「进行中 / 未报告」，属正常进行中状态。

---

# Implementation Plan

1. `newSocketAccounting` 改成接收已细化的 client 字符串，去掉未使用的 `socketAccess` 字段。
2. `servePassthroughSocket` 在握手后调用 `recordedClient(access.client, r.Header)`。
3. `serveResponsesSocket` 用局部变量 `client` 细化，`access.client` 保持原值供逐帧复核。
4. 单测断言 WS 记录的 `Client` 为 `codex-cli`／`codex-desktop`。

---

# Progress

- 已用生产库只读连接（含 WAL）确认截图行是本次新增的 WebSocket 逐轮记录，且 `client_id=codex`。

---

# Implementation Result

## Changes

- `newSocketAccounting` 改为接收**已细化**的客户端标识，去掉未使用的 `socketAccess` 字段；
  `servePassthroughSocket` 在握手后调用 `recordedClient(access.client, r.Header)`，
  逐轮 `beginTrace` 用细化后的值。
- `serveResponsesSocket`（协议入口）每帧复核后同样细化一次局部 `client` 变量；
  `access.client` 保持原始前缀／接入 Key 值，`reauthorize` 的 `client_mismatch` 判定不受影响。
- 新增回归用例：
  - `TestPassthroughSocketRecordsCodexRuntime`：透明 WS 的 `codex_cli_rs` → `codex-cli`、
    `Codex Desktop` → `codex-desktop`、未知来源 → `codex`。
  - `TestCodexWebSocketRecordsRuntimeLabel`：协议入口 WS 的两种来源标识。

## Files Changed

- `internal/core/proxy/passthrough_socket_accounting.go`
- `internal/core/proxy/passthrough_socket.go`
- `internal/core/proxy/websocket.go`
- `internal/core/proxy/passthrough_socket_accounting_test.go`
- `internal/core/proxy/websocket_prefix_auth_test.go`

## Technical Notes

根因是 Beta.38 新增的透明 WS 记账把 `access.client`（前缀归属，值恒为 `codex`）直接交给了
`beginTrace`，而识别逻辑只在 HTTP 入口调用过；旧协议 WS 入口同样从未细化（历史 2260 条 WS
记录全是 `codex`）。两处统一后，WS 与 HTTP 的归属口径一致。

---

# Verification

## Tests

- [x] Unit Tests（`internal/core/proxy`，含上述两个新用例）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task --force verify
第一次：exit 1 —— staticcheck SA4006（协议入口的 client 初值在循环内被重新赋值，判定为未使用）
修复后：exit 0（format:check / go vet + staticcheck + eslint / tsc / Go 全包测试 /
        前端 128 文件 760 项测试 / Windows production 构建）
```

未执行：实机 Codex WebSocket 会话的运行形态显示（需要安装带本次修复的构建后复测）。

---

# Documentation

- [x] `docs/releases/v2.0.0-beta.39.md`（随 TASK-20261025-006 发布说明记录行为变化）
- [x] `docs/plans/CURRENT.md`

---

# Deviations

原计划只在透明入口细化；实现时发现协议入口（`websocket.go`）同样从未细化，
一并修复，否则同一客户端换入口时归属仍会退回 `codex`。

---

# Git

## Commit

```text
dde6dc8 fix(gateway): WebSocket 请求按来源标识细化 Codex 运行形态
```

---

# Result

WebSocket 请求记录现在与 HTTP 一致地记为 `codex-cli`／`codex-desktop`／`codex-vscode`，
未知来源仍为 `codex`。截图里的 `client_id=codex` 情况（透明 WS 逐轮记录）不再出现。

---

# Completed

2026-10-25 11:30
