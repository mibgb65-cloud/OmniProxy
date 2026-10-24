# TASK-20261024-022: Codex 透明转发的额度耗尽换号

## Status

Completed

## Created

2026-10-24 22:55

## Request

用户反馈两点：

1. OpenAI 订阅额度用完后，网关的自动切换没有成功。
2. v1（`E:\go\new\OmniProxy`）在登录账号额度用完后仍然可以继续任务，网关会自动换到新账号；
   v2 在登录账号额度用完后就无法发送新消息。

用户指定只读对照 v1 仓库，要求定位是哪个部分的不同导致的。

## Goal

让 OpenAI 订阅账号在执行前额度耗尽时，真实客户端入口（`/codex/v1/...` 透明转发）也能在
同一条请求内换号，并把该账号按真实重置时间标为额度阻断，使**下一条消息**不再先撞同一个
耗尽账号；不重放已经开始生成的请求。

## Background

当前 v2 的一键配置把 Codex 客户端写成 `openai_base_url = …/codex/v1`
（`internal/application/agentconfig/patch.go:17,85`），该前缀之后全部走透明转发
（`internal/core/proxy/entry.go:96-103`、`handler.go:27-34`）。

## Scope

- `internal/core/proxy`：透明转发的执行前额度拒绝识别与尝试结束分类。
- `internal/application/gateway`：透明转发会话的冷却／额度阻断落地。
- `internal/application/subscription`：按需（去抖）额度观察刷新。
- 上述三处的回归测试与相关文档。

## Out of Scope

- WebSocket 中继的换号重拨（本次只让额度阻断在后续请求生效）。
- 客户端（Codex Desktop 工作区）自身的发送前额度拦截。
- 新增数据库迁移、依赖、后台全量探测或额度历史。

---

# Analysis

## Related Modules

- `provider/chatgpt`：额度拒绝判定（`quota_rejection.go`、`response_type.go`）。
- `application/gateway`：`forwardWithFailover`、`passthroughSession`、`failoverState`。
- `application/subscription`：`ReadQuota`／`rememberQuota`／`QuotaBlockedUntil`。
- `core/proxy`：`servePassthrough`、`forwardPassthrough`、`PassthroughSession`。

## Root Cause

v1 的"自动换号"由三个部件合力完成；v2 的对应实现分别退化了：

1. **无条件额度巡检**：v1 `OmniProxyBackend/token_maintenance.go:490` 的 `healthMonitor`
   每 30 秒刷新最近使用账号额度、每分钟跑到期健康检查（`main.go:55-58`），结果直接写
   `StatusExhausted`（`internal/token/manager_usage.go:59-83`）；取号只从 `Active/Low`
   里选（`internal/token/manager_health.go:136-156`），所以下一条消息自动换号。
   v2 唯一的定时读额度在 `application/quotaactivation`（5 分钟，`types.go:25`），且被
   "自动激活 Codex 额度窗口"开关门控（`check.go:27-29`，默认关闭）。
2. **请求内换号判据**：v1 的 HTTP 路径按 `429/5xx` 重新取号（`internal/proxy/service_http.go:139-227`、
   `service_retry.go:96-118`），且 429 冷却取 `Retry-After`／`X-RateLimit-Reset`，缺省 5 分钟
   （`internal/proxy/retry.go:52-90`）。v2 的透明转发只看状态码
   （`core/proxy/passthrough.go:57-89`、`application/gateway/passthrough_session.go:167-199`），
   冷却固定 30 秒且不读 `Retry-After`（同文件 `:202-212`）。
3. **执行前额度拒绝识别只覆盖适配器路径**：`provider/chatgpt/response_type.go:49-53` →
   `failover_policy.go:21-28` → `agent.go:46`，只作用于 `/v1/responses`、`/v1/messages`、
   `/v1/chat/completions`；`/codex/v1` 透明入口不经过它，HTTP 200 包着的
   `usage_limit_reached` 既不换号也不写额度阻断。

叠加 v2 的主账号选择是静态的（`routing/preview.go:174` → `core/routing/account.go:74-85`，
创建时间最早者）与默认队列顺序（`account_rotation.go:43-45`），耗尽账号会被每条新消息
反复命中，表现为"再也发不出去"。

## Risks

- 额度判定必须在写给客户端之前完成，不能把上游字节先交付再换号。
- 首事件预读会带来额外延迟；必须用很小上限并把预读字节原样接回。
- 新增额度刷新不能变成全量后台探测，也不能延长请求时延；只能对真实在用账号去抖刷新。

---

# Implementation Plan

1. 把额度拒绝判定上提到 `core/proxy`（`QuotaRejected`／`InspectableQuotaResponse`），
   `provider/chatgpt` 保留同名薄包装，避免两份实现。
2. `PassthroughSession.Done` 改为携带 `PassthroughOutcome`（状态码、`Retry-After`、
   `resets_at`），透明转发为此提供统一的结束分类。
3. 透明转发对 200/429 首事件做有界预读（4 KiB），命中执行前额度拒绝则不转发、
   按真实重置时间冷却并换号；全部候选都被拒绝时给客户端受控 429，明细保留真实状态。
4. `application/subscription` 增加按账号＋版本去抖的额度观察刷新（5 分钟），
   由透明转发的成功／额度拒绝回调触发，恢复 v1 "最近使用账号额度被持续观察"的效果。
5. 回归测试：透明转发的额度拒绝换号、`Retry-After` 冷却、去抖刷新。
6. 跑定向测试与质量门，同步 `FAILOVER.md`、`OVERVIEW.md`、`CURRENT.md`。

---

# Progress

- 完成 v1/v2 对照定位，确认根因是三处退化叠加（见 Analysis）。
- 用户确认按计划 1+2+3 实施。
- 实现阶段新增第四个根因：透明转发在发送前就把父记录绑定到首个账号，换号时
  `activity.Update` 因 subject 变化失败，客户端实际收到 503 `accounting_unavailable`。
  改为尝试级记账（`BeginAttempt`／`EndAttempt`），父记录只在被接受的响应上绑定。
- 定向与全量 Go 测试通过，`gofmt`／`go vet`／`staticcheck` 无输出。

---

# Implementation Result

## Changes

- 额度拒绝判定上提到 `internal/core/proxy/quota_rejection.go`（`QuotaRejected`、
  `InspectableQuotaResponse`、`firstQuotaPayload`），`provider/chatgpt` 保留同名薄包装，
  适配器与透明入口共用同一份证据。
- 透明转发新增 `inspectPassthroughQuota`：对 200／429 有界预读首个完整事件
  （最多 4 KiB，未命中按原字节经 `prefixBody` 接回），命中执行前额度拒绝即丢弃该响应、
  按真实 `resets_at` 冷却并换号；全部候选被拒绝时客户端收到受控 429
  `upstream_quota_exhausted`，请求明细保留上游真实状态码。
- `PassthroughSession` 契约补充 `PassthroughOutcome`（状态码、`Retry-After`、`resets_at`、
  `UpstreamQuotaRejected`）与 `Switchable()`；`Done` 由 `int` 改为携带结果，
  固定账号或关闭故障切换时登记额度阻断但仍不换号。
- 透明转发改为尝试级记账：`BeginAttempt`／`EndAttempt` 记录每次账号尝试，
  父记录只在被接受的响应上 `Update` 绑定账号，修掉换号即 `accounting_unavailable` 的缺陷。
- 冷却语义统一：`passthroughRejection` 优先采用上游合法 `Retry-After`（最大一天），
  缺失／非法时 429 仍为 30 秒；`quotaRetryReason` 按真实重置时间换算阻断窗口（上限 8 天），
  未知重置只短期跳过。`maxQuotaBlockWindow` 与适配器路径共用。
- 新增按需额度刷新：`application/subscription.ObserveQuota` 在请求结束后对
  「账号 + 凭据版本」去抖 5 分钟重新观察额度（有界缓存、应用退出取消），
  网关在透明转发与适配器路径成功后以及额度拒绝后调用；不新增定时器或全量后台探测。

## Files Changed

- `internal/core/proxy/quota_rejection.go`（新增）、`passthrough_quota.go`（新增）、
  `passthrough.go`、`passthrough_types.go`、`passthrough_socket.go` 与相邻测试。
- `internal/provider/chatgpt/quota_rejection.go`（改为委托）。
- `internal/application/gateway/passthrough_session.go`、`passthrough_rejection.go`（新增）、
  `subscription_quota.go`、`failover.go`、`failover_policy.go`。
- `internal/application/subscription/quota_refresh.go`（新增）、`service.go`、`lifecycle.go`。
- 回归测试：`core/proxy/passthrough_quota_test.go`、
  `application/gateway/passthrough_quota_failover_test.go`、
  `application/subscription/observe_quota_test.go`。

## Technical Notes

- 预读上限取 4 KiB：额度拒绝是几十字节的 JSON 或一条短 SSE 事件；更长的首事件
  （工具回显可达数百 KiB）不可能再被识别，必须尽早放行以免拖慢首字延迟。
- 透明转发不按上游声明的媒体类型过滤：官方可能用 200 或 429 包装同一拒绝，媒体类型
  也不稳定（实测会落成 `text/plain`）；判定强度来自正文证据本身。
- 额度阻断写入 `QuotaBlockedUntil`（账号 + 凭据版本），与界面／自动激活读取同一份观察；
  凭据版本变化后立即重新评估，不改用户启用开关。
- `passthrough_session.go` 超过 200 行后把拒绝分类拆到 `passthrough_rejection.go`。

---

# Verification

## Tests

- [x] Unit Tests：`go test . ./internal/... -count=1` 全部包 PASS（退出码 0）。
- [x] Regression：新增透明转发额度换号（HTTP 200 包装、整池耗尽、固定账号不换号、
  8 KiB 大首事件原字节交付）、`Retry-After`／重置时间换算、按需刷新去抖与退出停止、
  端到端 `/codex/v1/responses` 换号并在下一条消息跳过耗尽账号。

## Quality Checks

- [x] Format：`gofmt -l main.go internal` 无输出。
- [x] Lint：`go vet . ./internal/...` 与 `staticcheck . ./internal/...` 无输出。
- [x] Type Check：Go 编译通过（`go build ./internal/... ./cmd/... .`）。
- [x] Test：全量 Go 测试 PASS。
- [ ] Build / Verify：未执行完整 `task verify`（会重建前端资源与 Windows 安装产物）；
      本次改动只在 Go 后端，已用等价 Go 门禁覆盖。前端未改动。

## Verification Result

```text
go test . ./internal/... -count=1     PASS（exit 0）
go vet . ./internal/...               no output
staticcheck . ./internal/...          no output
gofmt -l main.go internal             no output
```

定向用例：

```text
go test ./internal/core/proxy -run Passthrough            PASS
go test ./internal/application/gateway -run Passthrough   PASS
go test ./internal/application/subscription -run ObserveQuota PASS
```

---

# Documentation

- [x] `docs/product/FAILOVER.md`：额度例外覆盖 `/codex/v1` 透明入口、按需额度刷新、
  冷却统一遵循 `Retry-After`。
- [x] `docs/architecture/OVERVIEW.md`：新增 2026-10-24 条目。
- [x] `docs/plans/CURRENT.md`：登记本任务。
- PRD 无需改动（既有条目已声明「耗尽后自动跳过并使用同池其他账号」）。

---

# Deviations

- 原计划只用固定预读判定后换号；实现中发现父记录提前绑定账号会让换号本身失败
  （503 `accounting_unavailable`），因此补做透明转发的尝试级记账。该改动同时让请求
  明细能看到被拒绝的那次尝试，与 FAILOVER 的记账约定一致。
- 原计划 3 打算做「无条件巡检」；按 FAILOVER 的既有边界改为「请求结束后按需去抖刷新」，
  效果等价（下一条消息不再先撞耗尽账号），但不新增定时器或对闲置账号的探测。
- WebSocket 中继的额度拒绝换号重拨仍未实现（本次让阻断在后续请求生效），保留为后续工作。

---

# Git

## Commit

```text
1173f20 fix(gateway): 透明转发在执行前额度拒绝时换号并阻断账号
8090b1c test(gateway): 断言透明转发换号保留两次尝试明细
```

---

# Result

Codex 透明入口（客户端实际使用的 `/codex/v1`）现在会在 HTTP 200／429 包装的执行前额度
拒绝时换号，并按真实重置时间把该账号标为额度阻断，下一条消息直接使用同池其他账号；
429 冷却与重置窗口取上游真实边界，成功请求后按需刷新额度观察。全量 Go 测试与
format／vet／staticcheck 通过。

---

# Completed

2026-10-24 23:35
