# TASK-20261024-016: 自动额度激活的失败原因可见，WebSocket 开关对已建立连接生效

## Status

In Progress

## Created

2026-10-24 18:15（Asia/Singapore）

## Request

用户附「OpenAI 账号设置」截图（Codex WebSocket 开、自动激活 Codex 额度窗口开、界面停在
「正在检测并复查额度窗口…」），反馈这两个功能「好像有点问题」，要求检查。

## Goal

- 自动激活的每轮检测必须能说明**哪个账号、因为什么稳定错误码**失败，不再只写 `activation_failed`。
- 关闭「Codex WebSocket」开关后，已经建立的透明转发 WebSocket 连接不能再继续转发新的请求帧，
  与开关文案「关闭后拒绝 WebSocket 请求」一致。
- 不在本任务内猜测上游行为：目录为空等问题只如实报告，不伪造模型或跨账号借用目录。

## Background

### 自动激活：本机证据

- 日志每 5 分钟一条：
  `Codex quota activation checked checked=2 sent=0 activated=1 pending=0 failed=1 code=activation_failed`，
  从 17:31 起持续复现（`codex_settings.auto_quota_activation=1`）。
- `checked=2` 是两个 OpenAI 订阅登录：`acct_b0f36…`（mibgb65@gmail.com，成功侧 `activated`）与
  `acct_bf1bf…`（x4107034@gmail.com，失败侧）。两者 `auth_account_id` 相同
  （`7fac9d2b-0421-42af-9f47-4cd065cfea01`，同一 ChatGPT workspace，两个登录）。
- `account_models` 里 `acct_b0f36…` 有 7 行、`acct_bf1bf…` **0 行**，但
  `account_catalog_versions` 里两者都有版本行 —— 说明该账号的目录同步成功执行却得到空目录
  （`internal/application/modelcatalog/sync.go` 在成功路径上先 `DELETE FROM account_models`
  再逐条插入，0 个模型就留下「有版本、无模型」的状态）。
- 该账号的额度快照正常（`subscription_quota_snapshots` 有 5h/7d 窗口），所以失败不在额度读取。
- 失败原因在代码里被丢弃：
  `internal/application/quotaactivation/check.go` 的 `Check` 把任何账号错误统一写成
  `s.failed("activation_failed")`，既没有账号标识，也没有 `checkAccount` 内部的具体错误码
  （例如 `models_unavailable`、`quota_unavailable`、`prepare_failed`、`request_failed`、
  `record_failed`、`identity_changed`、`awaiting_quota`），日志同样只有汇总行。

### WebSocket 开关

- 开关本身生效：`proxy.Options.WebSocketAllowed` 由 `CodexSettingsService.WebSocketAllowed`
  每请求读取，关闭时握手返回 `403 websocket_disabled`；本机 `websocket_enabled=1`。
  上一任务的 502 是「握手阶段账号/拨号失败被折叠成 upstream_unavailable」，已修复。
- 残留缺口：透明转发的 WebSocket 在用 `relaySocket` 纯字节中继，建连后不再读开关；
  协议路径（`websocket.go`）每帧复核策略，透明转发没有。关掉开关后旧连接仍可继续发起新请求，
  与开关文案不符。

## Scope

- `internal/application/quotaactivation/types.go`：`failure()` 返回携带稳定错误码的类型，
  新增 `codeOf`。
- `internal/application/quotaactivation/check.go`：区分批级失败与账号级失败，账号级失败写具体
  错误码并输出 `Codex quota activation account failed`（含账号 id）。
- `internal/core/proxy/passthrough_socket.go`：`relaySocket` 增加每帧策略复核。
- 前端：激活提示显示具体原因（已知码映射到文案，未知码保持通用文案）。
- 相关测试、`docs/product/SUBSCRIPTION.md` 或 `docs/product/GATEWAY.md`、`docs/plans/CURRENT.md`。

## Out of Scope

- 不为空目录账号伪造或跨账号借用模型目录；不为自动激活增加新的发送目标。
- 不改变 5 分钟检测间隔、`wakeDebounce` 与「准备失败可安全重试」的既有语义。
- 不改变 WebSocket 开关默认值，不改客户端配置。

---

# Implementation Plan

1. `failure()` 改为 `*codeError`；`codeOf(err)` 递归提取第一个稳定码（兼容 `errors.Join`）。
2. `Check`：账号级错误调用 `accountFailed(item.ID, err)`，写入具体码 + WARN 日志；批级错误保留原样。
3. `relaySocket` 增加 `guard`，客户端帧转发前复核 `webSocketPolicy`；关闭即断开两端。
4. 前端把 `errorCode` 映射成原因文案。
5. 测试：具体码进状态与日志、`codeOf` 组合错误、开关关闭后中继断开、前端原因文案。
6. `task verify`。

---

# Implementation Result

## 自动激活的失败原因可见

- `internal/application/quotaactivation/types.go`：`failure()` 改为返回 `*codeError`（`Error()` 仍输出
  `activation: <code>`，既有断言不变），新增 `codeOf(err)` 递归读取组合错误（`errors.Join`）里的第一个稳定码。
- `internal/application/quotaactivation/check.go`：
  - 账号级失败改走新增的 `accountFailed(item.ID, err)`：`status.ErrorCode` 写具体码，
    并输出 `Codex quota activation account failed`（`code` + `account`，不含凭据与提示词）；
  - 批级失败保留原 `failed(code)`，不再与账号级混用 `activation_failed`。
- `internal/application/quotaactivation/account_check.go`：额度读取失败补 `failure("quota_unavailable")`
  稳定分类；发送前失败改为 `prepareFailure(sendErr)` —— 发送方给出的具体码（如 `models_unavailable`）
  原样保留，只有拿不到码时才退回 `prepare_failed`。
- `internal/application/quotaactivation/send_plan.go`：新增 `prepareFailure` 并附「为什么保留具体码」的说明。

## WebSocket 开关对已建立连接生效

- `internal/core/proxy/passthrough_socket.go`：`relaySocket` 增加 `guard func(context.Context) error`，
  在把每个客户端帧写上游之前复核；`servePassthroughSocket` 传入 `webSocketPolicy`
  （包装成 `error` 返回值，避免 typed-nil）。关闭开关只断开该 WS 连接，不影响普通 HTTP/SSE，
  与协议路径的逐帧复核一致。

## 前端：原因不再只有一个笼统句子

- 新增 `frontend/src/features/accounts/activation-reason.ts`：稳定码 → 文案键，未知码返回 `null`。
- 激活设置面板在检测结束且存在 `errorCode` 时显示具体原因；悬浮提示同样优先显示原因。
- `subscription-zh.ts`／`subscription-en.ts` 增加 8 条原因文案（含 `models_unavailable` 的
  「请先同步该账号模型」）。

## 文档

- `docs/product/SUBSCRIPTION.md`：补充失败原因的分类、日志字段与「未知码才用通用文案」。
- `docs/product/GATEWAY.md`：补充开关在握手与每帧两处复核，关闭后只断开该连接。
- `docs/plans/CURRENT.md`：登记本任务。

# Verification

```text
gofmt -l internal cmd main.go                     空
go vet . ./internal/...                           无输出
staticcheck ./internal/application/quotaactivation ./internal/core/proxy   无输出
go test ./internal/application/quotaactivation ./internal/core/proxy ./internal/transport/wails   PASS
task verify BIN_DIR=…/tmp/activation-outcome-20261024/quality-bin   退出码 0
```

`task verify` 覆盖格式检查（含 Prettier）、lint（go vet／staticcheck／ESLint）、类型检查、
全部 Go 测试、前端全部测试与 Windows production 构建；二进制只写到独立临时目录。

缺陷复现：

- WebSocket 每帧复核：临时删除 `relaySocket` 里的 guard 调用，
  `TestPassthroughSocketStopsFramesAfterPolicyDisabled` 失败（上游仍收到第二帧 `2`），恢复后通过。
- 错误码：`TestQuotaReadFailureIsReportedWithStableCode`、`TestPreparationFailureKeepsSenderCode`
  断言 `Status().ErrorCode` 为 `quota_unavailable`／`models_unavailable`，而不是 `activation_failed`。

前端：`codex-quota-activation-settings.test.tsx` 7 项通过（新增「检测失败时直接给出具体原因」）。

未执行：真实账号重新检测（需用户安装带本次修复的构建后观察）；未替用户同步 x4107034 账号的模型目录。

# Remaining Risks

- 具体原因现在能看到了，但**不会自动修复**：`acct_bf1bf…`（x4107034@gmail.com）的官方模型目录为空，
  自动激活没有可用模型。要么在账号页同步该账号模型，要么重新授权；同 workspace 的另一登录有 7 个模型，
  说明更可能是该登录的目录／权限问题，而不是工作区整体不可用。
- 「准备失败可安全重试」的既有语义未改：只要目录仍为空，每 5 分钟仍会重试一次并记为失败
  （现在会带具体码）。是否改为带退避的重试节奏需要单独的产品决定。
- 打开额度页面或读取额度会唤醒检测（30 秒限频），面板可能长时间显示「正在检测并复查额度窗口…」，
  这是限频唤醒的既有行为，不是卡死。
- 未提交：本次改动的透明转发实现仍是 TASK-20261024-008 未提交的工作，`docs/plans/CURRENT.md`、
  `docs/product/GATEWAY.md` 也混有该任务的未提交内容；按 Git 安全规则不夹带提交。
