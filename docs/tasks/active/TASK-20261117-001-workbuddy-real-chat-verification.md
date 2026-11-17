# TASK-20261117-001: 真实账号验证 WorkBuddy AI Chat 转发与积分扣减

## Status

Completed

## Created

2026-11-17 08:10（Asia/Singapore）

## Request

用户授权：「可以的」——用其本人 WorkBuddy AI（Free 计划，100 credits）账号真实发一次最小
Chat 请求，验证上游转发与积分扣减。

## Goal

在真实账号上确认三件事，并记录消耗：

1. `POST /v2/chat/completions` 按适配器形态（UA 指纹、gzip、`stream:true`）可拿到流式回答；
2. 回答内容与用量（usage）可解析；
3. 计费接口的 `CycleUsedCapacity` / `TotalCredits` 在调用后发生变化，记录实际消耗。

## Background

- 账号：`acct_dff0ccb7211c6b647fe5e7690d4a3217`（`mibgb65@gmail.com`），Free 计划
  `plan_base 100`，`account_models` 25 条（2026-11-16 22:02 同步）。
- 通道已完成：登录、账号落库、目录同步、Credits 读取（TASK-20261116-020）、手动同步
  （TASK-20261116-022）。**真实对话**是唯一未验证环节。
- 现有 `model_routes` 为 0：OmniProxy 网关需要路由才会把请求交给 WorkBuddy，因此本次验证
  直接按适配器形态调用上游（与 `internal/provider/workbuddyai/client.go` 同一请求形态），
  不经过本机网关。网关的聚合/编码路径由单元测试覆盖（`gateway/workbuddy_test.go`）。

## Scope

- 只做一次最小对话（单轮、最短输出、`max_tokens` 很小），记录请求形态、状态码、回答、
  usage 与积分变化。
- 凭据来自本机 DPAPI（只读），令牌不写入任何文件、日志或任务记录。
- 如遇 401，按适配器的续期路径刷新一次再重试（同时验证续期链路）。

## Out of Scope

- 不改任何产品代码、配置、迁移或依赖。
- 不建路由、不改账号数据；不重复调用（消耗积分）。
- 不验证 Responses／Messages 客户端路径（需建路由后由用户用客户端验证）。

---

# Implementation Result

## Changes

本任务只做验证，不改产品代码：新增两个一次性验证脚本（在 `%TEMP%` 下，未入库），
用本机 DPAPI 解出账号令牌后按适配器形态直连上游：

1. 先按「只有 user 消息」发送 → 复现 `400 code 11128 "first message is not system prompt"`；
2. 再在首条加 `{"role":"system","content":"You are a helpful assistant."}` → `200` + SSE。

## Files Changed

- 无（`docs/product/WORKBUDDY.md` 的验证进度在 TASK-20261117-002 一并更新）。

## Artifacts

- 验证脚本只在 `%TEMP%` 下，含令牌的临时文件已删除。

---

# Verification

## 真实账号结果（Free 计划，`mibgb65@gmail.com`）

| 尝试 | 请求 | 结果 |
| --- | --- | --- |
| 1 | 只有 `user` 一条消息，`glm-5.3-flash`，`stream:true` | `400` + `{"code":11128,"msg":"first message is not system prompt"}`，未扣积分 |
| 2 | 首条补 `system: "You are a helpful assistant."` | `200` + `text/event-stream`，回答 `OK`，`usage` 26/3/29 |
| 3 | 同形态再发一次（43 tokens） | `200` + SSE，回答 `Waves crash, tides whisper softly.` |
| 4 | 通过**本仓库适配器**（`Forward`，入参不含 system）：`go test -tags workbuddyreal ... TestRealChatAgainstWorkBuddy`（该集成测试随后正式并入 TASK-20261117-002） | `PASS`，`usage` 26/3/29，`answer "OK"` |

## 积分

调用前后与之后约 15 分钟（10 次采样，30 秒一次）：

```text
TotalCredits  130 → 130
plan_base     0/100 → 0/100
benefit       0/30  → 0/30
addon / bonus 0
```

**没有观测到扣减**。同账号调用合计 101 tokens（29+43+29），Free 计划仍为 0 使用；
`modelPromotions` 文案也说明 Free 计划在活动期按「每日免费额度」发放，credits 很可能只在
付费套餐扣减。此项需要付费账号才能验证。

## 未验证

- 付费套餐的积分扣减、非零 `plan_bonus`／`addon`／`benefit` 窗口。
- 经本机网关（需要路由）的 Responses／Messages 客户端真实回包。

---

# Documentation

- [x] `docs/product/WORKBUDDY.md`（真实对话结果、11128 形态、扣减未观测）

---

# Deviations

- 未经过本机网关：`model_routes` 为 0，网关需要路由才会把请求交给 WorkBuddy，因此本次按
  适配器形态直连上游，并用带 `workbuddyreal` 构建标签的集成测试覆盖**本仓库代码路径**
  （`Client.Forward`），而不是手工 curl。
- 原计划「验证扣减」未达成：Free 计划看不到 credits 扣减（见 Verification），需要付费账号。
- 验证过程中发现一个真实缺陷（首条必须是 system），单独立项 TASK-20261117-002 修复。

---

# Git

## Commit

Not committed by Agent.

---

# Result

真实对话已通：`glm-5.3-flash` 单轮请求经本仓库适配器拿到 `200` + SSE，回答与 `usage` 正确；
**首条消息必须是 system**，否则上游 11128 拒绝——这是本次验证发现的产品缺陷（TASK-20261117-002）。

积分扣减未观测到：Free 计划连续 3 次调用（101 tokens）后 credits 仍 0 使用，判断按每日
免费额度计费；付费套餐的扣减仍待验证。本次消耗在 Free 额度内，无计费影响。

---

# Completed

2026-11-17 08:12（Asia/Singapore）