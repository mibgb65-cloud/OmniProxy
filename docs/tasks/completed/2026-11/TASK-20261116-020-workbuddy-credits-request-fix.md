# TASK-20261116-020: 修复 WorkBuddy AI Credits 读取失败（请求方法错误）

## Status

Completed

## Created

2026-11-16 21:35（Asia/Singapore）

## Request

安装 `v2.3.1-beta.16` 后账号已成功创建（`mibgb65@gmail.com`，1 个账号，模型目录已同步 25 条），
但账号卡的「WorkBuddy AI Credits」显示 `—` 并报「操作未完成，请检查网络和登录状态后重试。」，
套餐显示「套餐未知」。

## Goal

让 Credits 三窗口在真实账号上正确显示；同时修掉「上游错误在界面上塌成通用文案」这一类问题。

## Background

- 账号已落库：`accounts.auth_method='workbuddy_login'`、`auth_state='authenticated'`、
  `credential_ref` 有值；`account_models` 已写入 **25 条**真实目录（`/v3/config` 正常）。
- 日志只有 `WorkBuddy AI model auto sync incomplete`（无具体码），Credits 失败完全没有记录。
- 用真实账号令牌直连上游复现：`GET /billing/meter/get-user-resource-summary` → **404**；
  按社区插件的形态 `POST` + `{}` + 计费接口自己的头集 → **200**，返回：
  `CreditGroups[plan_base 100 / plan_bonus 0 / addon 0 / benefit 0]`、`TotalCredits: "100"`、
  `SubscriptionPackageName: "Free"`、`IsPaidUser: false`、`Packages[1]`。

## Analysis

### 根因一：计费接口的请求方法错了

社区插件 `meterAsked()` 用 `POST` 且带 `{"Content-Type":"application/json"}` 与空对象正文，
头集只有 `Accept`／`User-Agent: WorkBuddy/<app>`／`Authorization`／`X-User-Id`／`X-Domain`／
`X-Product: SaaS`／`X-IDE-Type: WorkBuddy`（企业账号再加 `X-Enterprise-Id`／`X-Tenant-Id`）。
我们的 `ReadCredits` 发的是 `GET`（还带了对话用的 UA 与 Agent 头），网关直接 404。

### 根因二：`proxy.Error` 的文案没带可识别前缀

404 会被映射成 `proxy.Failure("upstream_unavailable", 404)`，其 `Error()` 文本是
`gateway: upstream_unavailable`；前端 `workbuddyError()` 只认 `workbuddy_ai: <code>` 与
`subscription: <code>` 前缀，于是返回 `operation_failed`，界面只能显示通用文案。
**任何走到 `proxy.Error` 的 WorkBuddy 失败（额度、目录、转发）都会这样**，不只是额度。

### 次生

- 套餐名只从登录身份取，账号的 `/login/account` 没有 plan 字段，所以一直「套餐未知」；
  计费接口的 `SubscriptionPackageName` 才是权威来源。
- 目录同步的 WARN 只写死 `workbuddy_auto_sync_failed`，看不出是目录失败还是前端目录刷新失败。

## Scope

- `internal/provider/workbuddyai/quota.go`：改为 `POST {}` + 计费头集；套餐名取
  `SubscriptionPackageName`。
- `frontend/src/services/workbuddy.ts`：错误码识别改为「按分隔符切词后匹配」，覆盖
  `gateway:` 等前缀。
- `internal/transport/wails/workbuddy_subscription.go`：额度绑定把边界错误归一成
  `workbuddy_ai: <code>`；目录同步 WARN 带真实错误码。
- 测试：计费请求形态契约测试（方法／头／正文）＋ 用真实响应形状的解析用例；
  前端 `gateway:` 前缀映射。
- 文档：`docs/product/WORKBUDDY.md` 的 Credits 段与错误码说明。

## Out of Scope

- 不新增 paid/free packages 明细调用（首版只用 summary）。
- 不改变「额度只展示、不参与调度」。
- 不做真实 Chat 调用（消耗积分）。

---

# Implementation Result

## Changes

- `internal/provider/workbuddyai/quota.go`：`ReadCredits` 改为 `POST` + 空 JSON 对象正文，
  使用新的 `applyBilling` 头集（照社区插件 `meterAsked()`）；套餐名取计费接口的
  `SubscriptionPackageName`（登录身份没有该字段）。
- `internal/provider/workbuddyai/client.go`：新增 `billingUserAgent` 与 `applyBilling`；
  计费请求不带对话用的 Agent／Stainless／X-Requested-With 头，企业账号才带企业头。
- `frontend/src/services/workbuddy.ts`：错误码识别改为切词匹配，覆盖 `gateway:` 前缀，
  不再把核心边界的失败吞成 `operation_failed`。
- `internal/transport/wails/workbuddy_subscription.go`：额度绑定把跨层错误归一成
  `workbuddy_ai: <code>`；目录同步 WARN 带真实错误码。
- `frontend/src/features/accounts/workbuddy-credits.tsx`：额度卡标题旁显示套餐名（未知不显示）。
- 测试：计费请求形态契约测试（方法／正文／UA／头集／不泄漏对话头）与真实 Free 计划正文的
  解析用例；前端 `gateway:` 前缀映射。

## Files Changed

- `internal/provider/workbuddyai/quota.go`、`client.go`、`quota_test.go`
- `internal/transport/wails/workbuddy_subscription.go`
- `frontend/src/services/workbuddy.ts`、`workbuddy.test.ts`、
  `frontend/src/features/accounts/workbuddy-credits.tsx`
- `docs/product/WORKBUDDY.md`

## Technical Notes

- 真实账号核对（写修复前）：`GET /billing/meter/get-user-resource-summary` → `404`；
  `POST {}` + 官方头 → `200`：
  `CreditGroups[plan_base 100, plan_bonus 0, addon 0, benefit 0]`、`TotalCredits "100"`、
  `SubscriptionPackageName "Free"`、`IsPaidUser false`、`Packages[1]`。
- 解析只认「读到真实数字」：字符串数字（`"100"`）已支持；Free 计划只有 `plan_base` 有总量，
  因此只给一个合计窗口（插件同样语义）。`TotalCredits` 与窗口合计一致，仍不单独出窗口。
- 无数据库结构变化、无新增依赖。
---

# Verification

## 质量门（`task verify`，2026-11-16 21:40–21:52 Asia/Singapore）

```text
task verify → exit 0
```

| 阶段 | 结果 |
| --- | --- |
| format | gofmt 无输出（首次 verify 因新测试文件未格式化失败，`gofmt -w` 后通过）；prettier 通过 |
| lint | `go vet`／`staticcheck` 无告警；`eslint` 0 error（2 条既有 warning） |
| typecheck | `tsc --noEmit` exit 0 |
| test | Go 全包通过；前端 **157 files / 965 tests**（+1 错误码映射用例） |
| build | 绑定 + vite + Windows production 构建通过 |

## 真实上游核对（用户账号，不消耗积分）

```text
GET  /billing/meter/get-user-resource-summary   → 404
POST /billing/meter/get-user-resource-summary {} → 200
  CreditGroups: plan_base 100 / plan_bonus 0 / addon 0 / benefit 0
  TotalCredits "100"  SubscriptionPackageName "Free"  IsPaidUser false
```

会话令牌由本机 DPAPI 解出，只用于这次核对，未写入日志或任务记录。

## 未验证

- 安装 `v2.3.1-beta.17` 后应用内的额度显示（需用户执行）。
- 付费套餐（`plan_bonus`／`addon`／`benefit` 有值）与预付余额窗口的真实形状。
- 真实 Chat 与计费扣减（消耗积分）。

---

# Documentation

- [x] `docs/product/WORKBUDDY.md`（Credits 接口形态与真实返回、套餐名来源、三层错误码前缀）

---

# Deviations

- 计划里只写「修计费请求」，实际同时修了前端错误码识别：`proxy.Error` 的文本是
  `gateway: <code>`，只认 `workbuddy_ai:`／`subscription:` 的话，任何边界失败仍然会显示通用文案。
- 额外把套餐名显示在额度卡标题旁（计划只写到「取 SubscriptionPackageName」）：登录身份没有
  套餐字段，不显示的话账号卡会一直「套餐未知」。
- 未新增 paid/free packages 明细调用（首版仍只用 summary），与计划一致。

---

# Git

## Commit

bf809c3e fix(provider): 修复 WorkBuddy AI Credits 读取并识别边界错误码

---

# Result

真实账号（Free 计划，100 credits）上 Credits 读不出来的原因有两个：计费接口是
`POST` + 空对象正文 + 短 UA，我们发的是 `GET`，被网关 404；而 404 变成
`proxy.Error` 后文本是 `gateway: upstream_unavailable`，前端只认 `workbuddy_ai:`／
`subscription:` 前缀，于是显示成「操作未完成，请检查网络和登录状态」。

修复后：额度请求按官方客户端形态发送；解析以真实返回正文固定为单元测试；
套餐名取自 `SubscriptionPackageName` 并显示在额度卡；三层错误码都能被识别，
任何上游/边界失败都会给出对应文案而不是通用文案。验证需安装 TASK-20261116-021 的构建。

---

# Completed

2026-11-16 21:51（Asia/Singapore）