# TASK-20261119-003: Google 订阅账号按额度窗口参与选号

## Status

Completed

## Created

2026-11-19 11:02

## Request

用户提供两张截图（`mibgb65@gmail.com` 的 Gemini 5h 额度 0%、请求记录连续 `失败／未报告`）后要求：

> 如果这个五小时额度没有了，你就不需要再去调用它了哇，否则，这个请求老是会有重复报错，
> 你优化一下这些账号商的账号轮换机制，先给我确认，确认后你再做。

确认后的口径（用户选定）：

- 预筛力度：**C3 按模型分组精确预筛**（`gemini-4-argon*` → Argon 组、`claude*`／`gpt*` → 第三方组、
  其余 `gemini*` → Gemini 组），只跳过该分组已耗尽的模型；
- 冷却范围：**只冷却出问题的那个模型**；
- 同池账号全部冷却时：**保持返回 429 `account_pool_cooling`**。

## Goal

Google 订阅账号的额度耗尽不再表现为「同一个号被反复调用、请求反复失败」：

- 已观察到的额度快照里，某模型所属分组任一窗口为 0% 且重置时间在未来时，路由直接跳过该账号该模型
  （不发送请求）；
- 上游返回 429 时，Google **订阅**账号按账号级拒绝处理：同一请求内换下一个同池账号，并冷却该账号该模型；
- API Key 形式的 Google 账号保持现状（共享项目额度，换 Key 无意义）；
- 无法判断分组或无法确定重置时间时 fail-open，不预筛，只依赖 429 反应式冷却。

## Background

用户本机数据库（`omniproxy.db` 只读查询）显示 6 条失败请求全部是：

```text
mibgb65@gmail.com | failed | 429 | rate_limited | attempt_index=1（没有第二次尝试）
```

同一时刻该账号快照：`Gemini Models` 的 `gemini-5h` = 0%（重置 `2026-11-19T03:04:37Z`），
`3p-*` 与 `argon-*` 仍为 100%；`google_settings.account_rotation_mode=balanced`，池内 9 个 Google 号。

根因三处：

1. 上游 429 没有 `quota will reset after …` 正文，适配器只能归成 `rate_limited`
   （`internal/provider/antigravity/upstream_error.go`）；
2. `rejectedAccount` 对 `kind == "google"` 的 429 明确不分类（`failover_policy.go` 的 `t.kind != "google"`
   分支），本意是普通 API Key 共享项目额度；
3. 结果 `rejected == nil`：不换号、不写冷却，429 直接返回客户端。`subscriptionQuotaWait`
   只覆盖 Codex 订阅，Google 订阅没有任何额度预筛。

ADR-015 正文「普通 Google／Antigravity 的 429 不轮换」针对的是项目／共享额度；Google **订阅**账号
额度按账号独立，与本意冲突，需要按账号类型区分。

## Scope

- `internal/provider/antigravity`：模型 → 官方额度分组的归类与「耗尽到何时」的纯函数；
- `internal/application/googlesubscription`：只读额度快照的门（含重置时间解析）＋ 429 后异步刷新一次快照；
- `internal/application/gateway`：预筛接入 Google 轮换与故障切换、429 账号级化、冷却时长来源；
- `internal/transport/wails`：接线额度门与刷新日志；
- 测试：纯函数、额度门、轮换／切换行为与既有 Codex 路径回归；
- 文档：ADR-015 补充、`docs/product/GOOGLE_SUBSCRIPTION.md`、`docs/product/QUOTA.md`、`docs/product/PRD.md`、
  `docs/plans/CURRENT.md`。

## Out of Scope

- 不改 Codex／WorkBuddy／其他提供商的轮换与冷却；
- 不新增数据库迁移（沿用 038 起的额度快照表）；
- 不做定时额度轮询，不主动探测账号；
- 不改额度展示、卡片布局与前端；
- 不区分自费 Pro／学生 Pro（档位与账单来源无关）。

## Existing Behavior

Google 订阅账号只看运行时冷却（`failover.wait` + `failoverState`），而 429 不写冷却，所以已耗尽的账号
在 `balanced` 轮换下继续被选中，每次失败都是单次尝试。

## Expected Behavior

- 快照显示某分组窗口 0% 且未到重置：该分组模型在 `queue`／`balanced` 下都不选这个账号；
- 首次 429（快照还没刷新时）：同一请求换下一个候选，成功后客户端看不到错误；该（账号，模型）
  冷却到已知重置时间，未知时 60 秒；
- 429 后异步刷新该账号额度快照：同组其它模型随后也被跳过，界面同步显示真实剩余；
- 全部候选不可用：仍返回 429 `account_pool_cooling`（附最早恢复秒数）；
- API Key Google 账号、Codex 账号的既有行为不变。

---

# Analysis

## Related Modules

- `internal/provider/antigravity/quota*.go`（分组结构、快照校验）
- `internal/application/googlesubscription/{service,quota,quota_cache}.go`
- `internal/application/gateway/{failover,failover_policy,google_rotation,google_quota,service}.go`
- `internal/transport/wails/gateway.go`（接线）

## Dependencies

无新依赖、无迁移。

## Risks

- 分组归类依赖模型命名（`argon`／`claude`／`gpt`／`gemini`）：无法归类的模型不预筛，宁可多试一次；
- 快照可能过期：只在「重置时间仍在未来」时才跳过，过期的 0% 不会永久屏蔽账号；
- 预筛读库在网关热路径：单行主键查询（`account_id`），失败按不预筛处理，不影响请求正确性；
- 429 后异步刷新会增加一次官方额度查询（每账号 60 秒去重），不做全量轮询。

---

# Implementation Plan

1. `antigravity`：新增 `QuotaGroupForModel` 与 `QuotaExhaustedUntil`（纯函数 + 单测）。
2. `googlesubscription`：新增 `QuotaGate.BlockedUntil`（只读快照、按账号版本与登录态校验、fail-open）
   与 `Service.ObserveQuota`（429 后异步刷新，按账号去重，失败记稳定码日志）。
3. `gateway`：新增 `GoogleQuotaGate` 接口与 `googleQuotaWait`；接入 `rotateGoogleAccounts`、
   `forwardWithFailover`、passthrough 会话；新增 `googleRateLimited`，把 Google 订阅 429 变成账号级拒绝
   （冷却到快照重置时间，未知 60 秒），无后续候选时返回真实 429。
4. `transport/wails`：`SetGoogleQuotaGate(googlesubscription.NewQuotaGate(s.db))`，并给订阅服务装日志。
5. 测试：分组归类、耗尽时间、额度门、网关行为与边界。
6. `task verify`，更新 ADR-015 补充与产品文档，提交。

---

# Progress

- 2026-11-19 11:02 只读核对用户本机数据库，确认 429 / rate_limited / 单次尝试 与快照 0% 的对应关系。
- 2026-11-19 11:05 用户确认 C3 + 只冷却该模型 + 全冷却返回池冷却。
- 2026-11-19 11:12 实现完成：分组归类与耗尽时间纯函数、只读额度门、网关预筛与 429 账号级化、
  429 后异步刷新快照；新增 13 项测试，相关包 `go test` 全绿。
- 2026-11-19 11:16 发现两处文档与实现冲突（PRD「不作实时调度依据」、GOOGLE_SUBSCRIPTION「不参与调度」），
  已按新语义收窄：只有「已确认 0% 且未到重置」参与跳过，其余仍不推断。

---

# Implementation Result

## Changes

- `antigravity.QuotaGroupForModel`：按官方分组名把模型归入 Gemini 主组／第三方组／Argon 组，
  Argon 优先于 Gemini，无法归类返回空（不预筛）。
- `antigravity.QuotaExhaustedUntil`：组内任一窗口 0% 时取这些窗口里最晚的重置时间（都要等），
  比例按原始精度判断，重置已过期、缺重置或无法归类都不预筛。
- `googlesubscription.QuotaGate.BlockedUntil`：只读 `google_subscription_quota_snapshots`，要求账号
  `updated_at` 与登录态一致、JSON 可解析且通过快照校验；任何失败都按不预筛返回。
- `googlesubscription.Service.ObserveQuota`：429 后异步刷新一次额度（15 秒上限、每账号 60 秒去重、
  关闭后不再刷新），失败只记 `google quota refresh failed` 与稳定码。
- 网关：`GoogleQuotaGate` 接口与 `SetGoogleQuotaGate`；`googleQuotaWait` 接入 `rotateGoogleAccounts`、
  `forwardWithFailover` 与 passthrough 会话；`googleRateLimited` 把 Google 订阅 429 变成账号级拒绝
  （Retry-After → 快照重置时间 → 60 秒，上限 `maxQuotaBlockWindow`），没有后续候选时交还真实 429；
  429 后调用 `refreshGoogleQuota`。
- 接线：`SetGoogleQuotaGate(googlesubscription.NewQuotaGate(s.db))`，订阅服务 `SetLogger(s.log)`。

## Files Changed

- `internal/provider/antigravity/quota_gate.go`、`quota_gate_test.go`（新增）
- `internal/application/googlesubscription/quota_gate.go`、`quota_gate_test.go`、`quota_observe.go`（新增）
- `internal/application/googlesubscription/service.go`
- `internal/application/gateway/google_quota.go`、`google_quota_test.go`（新增）
- `internal/application/gateway/{failover,google_rotation,passthrough_session,service}.go`
- `internal/transport/wails/gateway.go`
- `docs/decisions/ADR-015-account-failover.md`（2026-11-19 补充）
- `docs/product/{GOOGLE_SUBSCRIPTION,QUOTA,PRD}.md`

## Technical Notes

- 预筛只读已保存快照，不在推理请求前发起官方查询；429 后的一次刷新按账号去重，避免每次失败都打官方接口。
- 冷却粒度是（账号，模型）：同组其它模型靠快照预筛跳过，不会被一次 429 连带整号停用
  （用户选定的口径）。
- 全部候选都因预筛被跳过时返回 `account_pool_cooling` 与最早恢复秒数；全部候选都被 429 拒绝时
  仍按既有语义交还最后一次真实错误。
- Argon 是 Gemini 家族里的独立额度池：`gemini-4-argon*` 只认 Argon 组，账号没有该组时不预筛，
  不退回 Gemini 主组以免按错的池跳过。

---

# Verification

## Tests

- [x] Unit Tests：`antigravity` 5 项（分组归类、Argon 隔离、多窗口取最晚、过期／缺重置／未知模型不预筛）
- [x] Unit Tests：`googlesubscription` 3 项（按分组阻断、账号版本／登录态、不可信快照 fail-open）
- [x] Integration Tests：`gateway` 5 项（429 换号并冷却、预筛跳过、全冷却返回池冷却、只有订阅账号参与、
     预筛范围与上限），驱动真实网关端口与两个同池账号
- [x] Regression：`gateway`／`googlesubscription`／`antigravity`／`transport/wails` 全包通过
- [ ] E2E：桌面窗口与真实账号未执行（不消耗用户额度，由用户安装后观察）

## Quality Checks

- [x] Format（gofmt）
- [x] Lint（go vet、staticcheck、ESLint）
- [x] Type Check（tsc --noEmit）
- [x] Test（Go 全包；前端 163 文件／1000 项未变）
- [x] Build（Windows 生产构建）
- [x] Verify

## Verification Result

```text
task verify
EXIT=0
go test . ./internal/... 全绿（含新增 13 项）
frontend 163 files / 1000 tests passed（本次未改前端）
windows:build:native → bin/omniproxy.exe 产出成功
```

---

# Documentation

- [x] PRD：Google 额度条目改为「只把已确认 0% 且未到重置用于选号跳过」
- [x] ADR：ADR-015 增加 2026-11-19 补充（Google 订阅 429 账号级化与额度预筛）
- [x] `docs/product/GOOGLE_SUBSCRIPTION.md`：新增「额度参与选号」条目并收窄留存值说明
- [x] `docs/product/QUOTA.md`：当前边界改为「只用于选号跳过，不做跨账号再分配」
- [x] CURRENT

---

# Deviations

- 计划里预筛接入点写「passthrough 会话可选」；实际一并接入（同一门、同一语义，未增加新行为）。
- 额度门最初打算做内存缓存，实测单行主键查询开销可忽略，改为不加缓存，避免引入过期窗口。
- 计划写「429 后刷新一次快照」，实现里加了每账号 60 秒去重与关闭检查，避免并发登录或退出后继续刷新。
- 文档中出现两处与实现冲突的旧表述（PRD、GOOGLE_SUBSCRIPTION），按新语义收窄而非删除，
  保留「不推断权限、不折算请求次数」的原边界。

---

# Git

## Commit

```text
dca88996 feat(gateway): Google 订阅按额度窗口跳过与换号
```

---

# Result

Google 订阅账号的额度终于参与选号：快照里该模型所属分组任一窗口为 0% 且未到重置时不再发送请求；
上游 429 按账号级处理，同一请求内换下一个同池账号并把该（账号，模型）冷却到真实重置时间，429 后
异步刷新快照让同组其它模型与界面同步。API Key 账号与 Codex 路径不变，无法判断时不预筛。
`task verify` EXIT=0。

---

# Completed

2026-11-19 11:34
