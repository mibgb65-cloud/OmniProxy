# TASK-20261118-007: 账号级拒绝的真实错误码与换号

## Status

In Progress

## Created

2026-11-18 14:10

## Request

用户报告（附请求页截图）：

> 我发现一个问题，就是这个在调用 gemini-4-carbon 时，有有权限的账号他没有调用，他去调用了没有权限的账号。你看看这个问题是什么情况

定位后用户确认修复范围：

- A：错误码真实化（把适配器的稳定码转成网关边界错误，不再兜底成 `credential_unavailable`）
- B：账号级拒绝换号 + 冷却
- C（路由口径：裸家族名优先走自家订阅）本次不做

## Goal

同池还有候选账号时，一个账号被上游明确拒绝（模型权益、登录失效）不再让整条请求失败：

- 请求记录与日志显示真实稳定码（`model_not_entitled` 等），不再一律 `credential_unavailable`；
- 该（账号，模型）被冷却，后续请求自动换到别的账号；
- 没有其它候选时把真实错误交还客户端，不伪装成账号池冷却。

## Background

本机生产数据（`C:\Users\mimanchi\AppData\Roaming\OmniProxy\v2\omniproxy.db`）里 2026-11-18 13:48 有三条 `gemini-4-carbon` 失败：

| 时间 | provider | 账号 | 状态 | 错误码 | 尝试数 |
| --- | --- | --- | --- | --- | --- |
| 13:48:45 | WorkBuddy AI | mibgb65@gmail.com | 503 | credential_unavailable | 1 |
| 13:48:51 | WorkBuddy AI | mibgb65@gmail.com | 503 | credential_unavailable | 1 |
| 13:48:59 | WorkBuddy AI | mibgb65@gmail.com | 503 | credential_unavailable | 1 |

同池的第二个 WorkBuddy 账号 `xiabuji@gmail.com` 也发布 `gemini-4-carbon`，但从未被尝试。

## Scope

- `workbuddyai` 增加边界错误翻译（`ProxyError`）并在网关 WorkBuddy 转发处使用；
- 网关失败分类增加“账号级拒绝”：`model_not_entitled` 与 401 类（登录失效／凭据被拒）；
- 命中后写（账号，模型）冷却并继续下一个候选；无后续候选时返回真实错误；
- 请求页错误码文案补上这些稳定码；
- ADR-015 增补一节说明该例外。

## Out of Scope

- 路由口径（裸 `gemini-4-carbon` + 档位 是否优先走 Google 订阅的 `-high` 变体）：需要产品决定，本次不改。
- 历史请求记录的错误码回填：无法还原上游真实响应。
- 服务商级拒绝（`channel_rejected` 11128、`user_agent_refused` 10085、`signature_invalid` 11159）不换号：换号会得到同样的拒绝。
- 本地凭据故障（`credential_unavailable`）不换号：ADR-015 明确“本地错误不重放”。

## Existing Behavior

- WorkBuddy 适配器把业务码拒绝返回成 `*workbuddyai.Error`；网关 `attemptFailure` 只识别 `*proxy.Error`，其它错误统一记成 `503 credential_unavailable`。
- 换号（`rejectedAccount`）只看上游“响应对象”的 401／402／429／额度耗尽；适配器以 error 返回的拒绝不参与分类，请求第一次尝试后即失败。
- 因此账号级拒绝既看不出真实原因，也不会换到同池其它账号，也不会写冷却。

## Expected Behavior

- 记录里出现 `model_not_entitled`（403）等真实码，而不是 `credential_unavailable`。
- 该账号在冷却期内不再被选中；同池其它账号接手。
- 只有一个候选时返回 403 `model_not_entitled`，不返回 429 `account_pool_cooling`。
- 失败请求的尝试明细里，被拒绝的那次显示真实状态与错误码。

---

# Analysis

## Root Cause

1. **错误码层**：`internal/provider/workbuddyai/client.go` 把业务码 11158／11128／11159／10085 映射成 `workbuddyai.Failure(code)`（`*Error`），而网关 `attemptFailure` 只对 `*proxy.Error` 取真实码，其余走兜底 `503 credential_unavailable`。同一账号 10:54 还成功同步过模型目录（`account_models` 写入 `2026-11-18T02:54:46Z`），凭据本身可读；三次尝试耗时 0.9–2.7 秒，说明请求确实到达中转站并被拒绝——不是本地凭据故障。
2. **换号层**：`rejectedAccount` 依据 `proxy.Response` 状态码分类，而 WorkBuddy 的拒绝是以 error 返回的，`responseStatus` 为 0，`rejected` 始终为 nil，于是 `failover` 循环直接返回错误。ADR-015 的“明确上游 401 换号”在 error 形态下从未生效。
3. **冷却层**：没有 `retryReason` 就不会写 `failoverState`，失败账号永远是同池的第一个候选（WorkBuddy 不做平滑轮询），每条请求重复撞同一个账号。

## Related Modules

- `internal/provider/workbuddyai`（errors、client）
- `internal/application/gateway`（workbuddy、failover、failover_policy）
- `frontend/src/features/gateway/gateway-errors.ts` 与 i18n 文案

## Dependencies

N/A

## Risks

- 状态码变化会影响客户端对 4xx／5xx 的解读（业务拒绝从 503 变 400／401／403）。缓解：只为已分类的稳定码改状态，未知码仍 502。
- 401 类换号是新行为，可能把“所有凭据都失效”表现为多个尝试。缓解：无后续候选时返回真实错误，不返回池冷却。

---

# Implementation Plan

1. `workbuddyai.ProxyError`：`*Error` → `*proxy.Error`（保留稳定码，按码映射状态）；已是 `*proxy.Error` 时原样返回。
2. 网关 `forwardWorkBuddy` 统一用 `ProxyError` 收口返回错误。
3. `failover_policy.go` 增加 `accountRejection(err)`：`model_not_entitled` → 冷却 30 分钟；401 类（`upstream_authentication_failed`／`sign_in_expired`／`refresh_rejected`／`unauthorized`）→ 冷却 5 分钟。
4. `failover.go`：在既有额度拒绝分支后接入，命中且还有候选则换号并写冷却；没有候选则返回真实错误；循环耗尽时优先返回最后一次真实拒绝，而不是池冷却。
5. 前端 `gateway-errors.ts` 与 `gateway-zh.ts`／`gateway-en.ts` 增加对应文案。
6. 测试：`workbuddyai.ProxyError` 表驱动、`accountRejection` 分类、WorkBuddy 双账号集成（首账号 11158 → 换号成功，次数/尝试码断言）。
7. ADR-015 增补、CURRENT.md、任务记录。
8. `task verify`。

---

# Progress

- 2026-11-18 14:10 完成定位：错误码兜底 + error 形态不参与换号 + 无冷却。
- 2026-11-18 14:15 完成 A（`workbuddyai.ProxyError`）与 B（`accountRejection` + 冷却），新增 6 项测试。
- 2026-11-18 14:20 `task verify` EXIT=0（前端 159 文件／977 项测试）。

---

# Implementation Result

## Changes

1. **错误码真实化**：`workbuddyai.ProxyError` 把 `*Error` 翻译成 `*proxy.Error`（保留同一稳定码，状态按码归类：401 登录类、403 模型权益、400 渠道／签名／指纹、429 限流、503 本地状态、其余 502）；网关 `forwardWorkBuddy` 统一收口，记录与日志不再兜底成 `credential_unavailable`。
2. **账号级拒绝换号**：`accountRejection` 把以错误形式返回的账号级拒绝翻译成换号理由——`model_not_entitled` 冷却 30 分钟，401 类（`upstream_authentication_failed`／`sign_in_expired`／`refresh_rejected`／`unauthorized`）冷却 5 分钟；命中后按真实状态写尝试明细并继续下一个候选。
3. **无候选时返回真实错误**：没有后续候选时直接交还真实状态与错误码（403 `model_not_entitled`），不再伪装成 429 `account_pool_cooling`；全部候选都被账号级拒绝时返回最后一次真实错误。
4. **服务商级拒绝不换号**：`channel_rejected`（11128）、`signature_invalid`（11159）、`user_agent_refused`（10085）不进换号分类，`credential_unavailable` 等本地错误仍不重放（ADR-015）。
5. **界面文案**：`gateway-errors.ts` 增加 6 个稳定码映射与中英文案，请求详情的失败原因不再是「无法完成操作」。

## Files Changed

- `internal/provider/workbuddyai/errors.go`、`internal/provider/workbuddyai/proxy_error_test.go`
- `internal/application/gateway/workbuddy.go`
- `internal/application/gateway/failover_policy.go`
- `internal/application/gateway/failover.go`
- `internal/application/gateway/failover_entitlement_test.go`
- `frontend/src/features/gateway/gateway-errors.ts`、`frontend/src/i18n/locales/gateway-zh.ts`、`frontend/src/i18n/locales/gateway-en.ts`
- `docs/decisions/ADR-015-account-failover.md`、`docs/product/FAILOVER.md`、`docs/product/WORKBUDDY.md`
- `docs/plans/CURRENT.md`

## Technical Notes

- 换号分类只看核心边界错误，因此适配器必须先翻译；`accountRejection` 对未翻译的 `*workbuddyai.Error` 返回 nil，测试里专门锁住这一点。
- 冷却复用既有 `failoverState` 的（账号，版本，模型）键：模型权益按模型冷却，凭据版本变化（重新登录）后立即失效。
- 冷却只在确实换号时写入；单候选场景不写冷却，避免之后整池返回 429 掩盖权限问题。
- 11128 被适配器自身重试一次（系统提示词下移），因此该场景上游调用数是 2、网关尝试数仍是 1；测试按这个口径断言，防止把适配器重试误判成换号。

---

# Verification

## Tests

- [x] Unit Tests：`workbuddyai` 状态映射与透传（2 项，10 个子用例）
- [x] Integration Tests：网关 WorkBuddy 双账号换号、单候选真实错误、服务商级拒绝不换号、分类表（4 项）
- [ ] E2E Tests：N/A（真实 WorkBuddy 账号权限差异无法在测试中复现）

## Quality Checks

- [x] Format：`gofmt -l` 无输出；`prettier --check` 通过
- [x] Lint：`task verify` 内的 ESLint 通过（仅既有 2 条 warning）
- [x] Type Check：`tsc --noEmit` 通过
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task verify
EXIT=0
Go：internal/... 全部通过（gateway 56s、provider/* 全部 ok）
Frontend：159 files / 977 tests passed
wails3 production build + vite build 通过
```

---

# Documentation

- [x] ADR：ADR-015 增补「以错误返回的账号级拒绝」
- [x] CURRENT
- [x] 产品文档：`docs/product/FAILOVER.md`、`docs/product/WORKBUDDY.md`
- [ ] PRD / Architecture / Tech Stack / Coding Rules / Testing / Build：No documentation changes required.

---

# Deviations

- 原计划只处理 `model_not_entitled`；实际把 401 类一并纳入同一条账号级通道，因为 ADR-015 原本就写明“明确上游 401 可换号”，而 error 形态下这条从未生效。
- 原计划未包含界面文案；实现时补了 6 个映射，否则新错误码在请求详情里仍显示成通用失败。
- 测试中发现 11128 存在适配器自身的重试，断言改为“网关尝试数 1、上游调用数 2”，避免把适配器重试误判成换号。

---

# Git

## Commit

```text
Planned
```

---

# Result

`gemini-4-carbon` 这类“同池有账号没权限”的失败不再静默：错误码真实可见，同池其它账号会被自动尝试，没有权益的账号按（账号，模型）冷却，单候选时保留真实 403 而不是伪装成限流。路由口径（裸家族名是否优先走 Google 订阅的档位变体）仍待单独决定。

