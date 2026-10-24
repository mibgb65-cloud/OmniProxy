# TASK-20261024-015: Codex 透明转发的成败判定、用量观察与 CLI／桌面归属

## Status

In Progress

## Created

2026-10-24 17:40（Asia/Singapore）

## Request

用户三点反馈（附 Codex 桌面端会话截图与 OmniProxy 请求列表截图）：

1. 会话持续提示「正在重新连接 5/5」，
   `unexpected status 502 Bad Gateway: upstream_unavailable, url: ws://127.0.0.1:3000/codex/v1/responses`。
2. 请求列表里 `gpt-6.1-sol` 的每一行都是「失败 / 未报告」，但用户确认回答实际已经收到。
3. 请求列表的 Agent 列只有一个「Codex CLI / Desktop」，CLI 与桌面应用没有分开。

## Goal

- 上游已经交付完整回答的请求不再被记为 `failed`；真正的失败保留可定位的稳定错误码。
- 透明转发的用量（输入／输出／缓存 Token）能够统计，`未报告` 不再是无条件结果。
- WebSocket 失败不再一律压成 `502 upstream_unavailable`，并向日志给出可定位的稳定代码。
- 请求归属能把 Codex CLI、桌面应用与 IDE 扩展区分开。

## Background

### 本机证据

请求库（`request_records`，数据目录 `…/OmniProxy/v2`）与网关日志显示，Codex 侧
（`client_id='codex'`、`protocol='responses'`、`transport='http'`、模型 `gpt-6.1-sol`）
的每一条记录都是：

```text
state=failed status_code=200 error_code=client_write_failed
input_tokens=0 output_tokens=0  （日志 usage_state=unknown）
"upstream_media_type":"missing"
latency_ms=23000…46000
```

- `client_write_failed` 来自 `internal/core/proxy/passthrough_send.go`：`copyPassthroughStream`
  返回任何错误都被判为「写客户端失败」，**没有区分是上游读流出错、客户端连接关闭，
  还是用量记账失败**；错误本身既没有日志也没有分类，属于静默丢弃。
- `upstream_media_type=missing` 说明这些上游响应没有 `Content-Type`。透明转发只在
  `mime.ParseMediaType` 得到 `text/event-stream`／`application/json` 时才挂用量观察器
  （`ObserveResponse`），因此整段 SSE 从未被解析 → Token 恒为 0、前端显示「未报告」。
- 有同一模型的成功记录（`code=''`）与失败记录交错出现，说明失败判定取决于连接结束方式，
  而不是回答是否交付。

### WebSocket

以 `ws://127.0.0.1:3000/codex/v1/responses`（固定前缀、1400 字节官方登录令牌）直连本机网关
可稳定复现：

```text
status=502 body={"error":{"code":"upstream_unavailable",...}}
```

而同一台机器直连 `wss://chatgpt.com/backend-api/codex/responses` 的握手是可达的
（无凭据返回 `401`，说明网络与 TLS 正常）。网关日志里这次握手**没有任何记录**：
`servePassthroughSocket` 既不建 Trace 也不写日志。

代码层面的原因在 `internal/core/proxy/passthrough_socket.go`：`dialPassthroughSocket`
把 `session.Next` 的失败（如 `503 route_account_unavailable`、`503 credential_unavailable`、
`429 account_pool_cooling`）折叠成「返回 nil + status 0」，调用方再把 status 0 一律写成
`502 upstream_unavailable`，客户端因此只能无限重连。

WebSocket 握手没有正文，`passthroughModel` 读不到模型 → `OpenPassthrough` 走
`openFallbackSession`，而该回退会话只取**第一个**已授权 openai 账号并 `fixed=true`：
第一个账号借不出令牌时整条 WS 通路直接失败，而带模型的 HTTP 路径有候选轮换，所以能成功。

### Agent 归属

`request_records.client_id` 来自接入 Key（`codex`），前端把 `agents.codex`
渲染成「Codex CLI / Desktop」。CLI、桌面应用与 IDE 扩展共用同一个前缀，只能靠客户端
自己声明的来源标识（`originator`／`User-Agent`）区分，目前完全没有使用。

## Scope

- `internal/core/proxy/passthrough_send.go`：区分流的出错侧，按「是否已收到协议终结事件」判定成败。
- `internal/core/proxy/passthrough_observe.go`（新增）：上游未声明内容类型时按前缀判定 SSE／JSON 并挂用量观察器。
- `internal/core/proxy/usage_reader.go`、`usage.go`、`trace.go`、`tracked_http.go`：
  终结事件标记、记账失败不中断下行流、诊断字段。
- `internal/core/proxy/passthrough_socket.go`：交回真实失败原因并写稳定代码日志。
- `internal/application/gateway/passthrough.go`、`passthrough_session.go`：回退会话按全部可用账号轮换，
  借不出令牌时给出 `credential_unavailable` 而不是「账号池冷却」。
- `internal/core/agentendpoint/agentendpoint.go`、`internal/core/proxy/client_identity.go`：
  由 `originator`／`User-Agent` 的固定标记得到 `codex-cli`／`codex-desktop`／`codex-vscode`。
- 前端：`agent-clients.ts`、请求列表／详情、`agents-zh.ts`／`agents-en.ts` 增加三种运行形态的标题。
- 相关测试与 `docs/product/GATEWAY.md`、`docs/plans/CURRENT.md`。

## Out of Scope

- 不改变透明转发的路径、头部与正文语义，不新增客户端配置项。
- 不为 WebSocket 建立逐轮记账（连接级记账仍是后续任务，见 Remaining Risks）。
- 不猜测未知 `originator` 的客户端：无法识别时保持 `codex`。
- 不修改 Codex 身份、登录、`auth.json` 与会话文件。

---

# Implementation Plan

1. Trace 增加 `terminal`（已收到协议终结事件）、`streamSide`（出错侧）与 `accountingFailed` 标记。
2. `usageReader` 逐行解析时标记终结事件；记账失败只标记状态，不再让读流返回错误。
3. 新增 `ObservePassthroughResponse`：内容类型可用时沿用既有观察器，缺失时读一段前缀判定形态。
4. `passthrough_send.go`：`copyPassthroughStream` 返回出错侧；终结事件已到即判成功，
   客户端连接结束记为 `request_cancelled`，其余上游读错记为 `upstream_stream_failed`。
5. `passthrough_socket.go`：`dialPassthroughSocket` 返回错误本身，握手被拒仍是 `upstream_socket_rejected`，
   无响应的拨号失败仍是 `upstream_unavailable`，并写 `gateway websocket failed` 日志。
6. `gateway`：回退会话按全部可用账号建候选、`fixed=false`；`attemptFor` 借不出令牌时记 `credential_unavailable`。
7. `agentendpoint.Runtime` + `recordedClient`：按来源标识细化 Codex 归属，HTTP 与 WS 两条路径都使用。
8. 测试：流转发分类、内容类型缺失时的用量统计、WS 失败代码、回退候选、归属映射、请求列表标签。
9. `task verify`。

---

# Implementation Result

## 结束分类不再把已交付的回答记成失败

- `internal/core/proxy/passthrough_send.go`：`copyPassthroughStream` 改为返回 `passthroughCopy`
  （已写字节、出错侧 `client`／`upstream`、错误本身）。`passthroughOutcome` 只按可证实信号分类：
  - 已收到协议终结事件 → 成功（空错误码）；
  - 请求上下文结束或写客户端失败 → `request_cancelled`；
  - 其余上游读流错误 → `upstream_stream_failed`。
  出错侧写进日志字段 `stream_error_side`。`client_write_failed` 不再是透明转发的兜底码。
- `internal/core/proxy/usage_reader.go`：逐行观察时标记终结事件
  （`response.completed`／`response.incomplete`／`message_delta`／`[DONE]`，非流式正文视为终结）。
  记账失败只标记 `accounting_failed`；透明转发（非严格模式）不再因为一次入库失败而截断已交付的流，
  协议转换路径仍按原样把记账失败上抛。
- `internal/core/proxy/trace.go`、`tracked_http.go`：新增 `terminal`、`streamSide`、`accountingFailed`
  与 `diagnosticLogFields()`，日志只输出固定分类。

## 未声明 Content-Type 的流也能统计用量

- 新增 `internal/core/proxy/passthrough_observe.go`：`ObservePassthroughResponse`。
  内容类型可用时沿用既有观察器；缺失或不可解析时只读一段前缀（≤4 KiB）判定
  SSE／JSON／HTML，再把前缀与剩余正文用 `io.MultiReader` 接回去继续转发，字节不变。
  判定结果以 `ContentTypeInferred` + `verified`／`non_sse_json`／`non_sse_html` 记入既有日志字段。

## WebSocket 交回真实失败原因并换号

- `internal/core/proxy/passthrough_socket.go`：`dialPassthroughSocket` 改为返回错误本身。
  - 上游明确拒绝握手 → 原状态码 + `upstream_socket_rejected`（与之前一致）；
  - 拨号没有拿到响应 → `upstream_unavailable`；
  - 账号侧错误（`credential_unavailable`、`account_pool_cooling`、`route_*`）原样传给客户端，
    不再折叠成 502。
  新增 `logSocketFailure`：写 `gateway websocket failed` 与稳定代码、状态码（无 URL／账号／底层文本）。
- `internal/application/gateway/passthrough.go`：`passthroughFallbackTargets` 返回全部已授权订阅账号，
  `openFallbackSession` 用它们建候选且 `fixed=false`，第一个账号借不出凭据时按顺序换号。
- `internal/application/gateway/passthrough_session.go`：`attemptFor` 借不出令牌时记
  `credentialFailed`；没有可用候选时给出 `credential_unavailable`（503），不再伪装成
  `account_pool_cooling`（429）；固定账号分支同样区分凭据失败。

## 请求归属细分 Codex 运行形态

- `internal/core/agentendpoint/agentendpoint.go`：新增 `Runtime(client, originator, userAgent)`，
  按 `originator` 优先、`User-Agent` 次之匹配固定标记，得到 `codex-cli`／`codex-desktop`／
  `codex-vscode`；未知来源保持 `codex`，其他客户端原样返回。
- `internal/core/proxy/client_identity.go`：新增 `recordedClient`，透明转发建 Trace 时使用；
  原始头部不入库，只落映射后的固定标识。
- 前端：`agent-clients.ts` 新增 `requestAgentTitleKey`；请求列表与请求详情改用它；
  `agents-zh.ts`／`agents-en.ts` 增加三个标题键（Codex CLI／Codex Desktop／Codex（VS Code 扩展））。

## 文档

- `docs/product/GATEWAY.md`：补 WebSocket 取号与失败码、未声明内容类型的观察、
  结束分类与归属细化说明。
- `docs/plans/CURRENT.md`：登记本任务。

# Verification

```text
gofmt -l internal cmd main.go                     空
go vet . ./internal/...                           无输出
staticcheck ./internal/core/proxy ./internal/core/agentendpoint ./internal/application/gateway   无输出
go test . ./internal/...                          PASS（全包）
task verify BIN_DIR=…/tmp/codex-outcome-20261024/quality-bin   退出码 0
```

`task verify` 覆盖格式检查、lint（go vet／staticcheck／ESLint）、类型检查、全部 Go 测试、
126 个前端测试文件 752 项测试与 Windows production 构建；二进制只写到独立临时目录，
未覆盖正在使用的 `bin/`。

缺陷复现（临时回退）：把 `ObservePassthroughResponse` 换回 `ObserveResponse`、
让 `passthroughOutcome` 恢复 `client_write_failed` 后，
`TestPassthroughDeliveredStreamCountsAsSuccess` 立即失败，事件与用户截图完全一致：

```text
State:failed ErrorCode:client_write_failed StatusCode:200
Usage:{Input:0 Output:0 InputKnown:false OutputKnown:false}
upstream_media_type=missing
```

恢复修复后该用例通过，用量为 `input 12 / output 4`。

本机实测（修复前）：`ws://127.0.0.1:3000/codex/v1/responses`（固定前缀、1400 字节官方登录令牌）
返回 `502 {"error":{"code":"upstream_unavailable"}}`，网关日志无任何对应记录；
同一机器直连 `wss://chatgpt.com/backend-api/codex/responses` 返回 `401`，说明上游可达。

前端：`tsc --noEmit` 通过；`request-agent.test.tsx` 3 项通过（含新增运行形态用例）；
改动文件的 ESLint 与 Prettier 检查通过。

未执行：真实 Codex CLI／Desktop 会话复测（需用户重启客户端并重新安装带本次修复的构建）；
安装包构建与发布。

# Remaining Risks

- 提交边界：透明转发实现（`passthrough*.go`、`gateway/passthrough*.go`）与 TASK-20261024-008 的工作互锁，无法单独编译；已按可构建边界与 008 一并提交为 `94985cb`，文档随 `90681e0`。
- 透明转发的 WebSocket 仍不建请求记录：连接成不成功只体现在客户端提示与
  `gateway websocket failed` 日志里，请求列表看不到这类失败。逐轮（每个 `response.create`）
  记账需要先确定连接级与轮级的归属语义，留作后续任务。
- `codex-desktop`／`codex-vscode` 的识别依据是客户端自报的 `originator`／`User-Agent`；
  上游若改变这些标记，归属会退回 `codex`，不会错误归属到别的客户端。
- 未声明 `Content-Type` 时按前缀判定，极少数以 `data:` 开头的非 SSE 正文会被当成流观察；
  只影响用量统计，转发字节不变。
- `request_cancelled` 记录为「取消」状态而非「失败」，前提是上游没有交付终结事件；
  如果客户端在上游终结事件之前断开，这确实是未完成的回答。

# Git

## Commit

```text
94985cb feat(gateway): Codex 透明转发、网关自有登录与 WebSocket 握手预算
e8f0756 feat(frontend): 激活失败原因、Codex 运行时标签与请求页字段
90681e0 docs: 同步 Codex 透明转发决策、产品文档与 2026-10 任务记录
```

后端随 94985cb、前端随 e8f0756 落库；记录随 90681e0 归档。

---
