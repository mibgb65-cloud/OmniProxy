# TASK-20261025-002: 额度估算容忍少量用量未知记录

## Status

Completed

## Created

2026-10-25 09:53

## Request

> 这个 5h 和一周额度的估算在我们换了整体流程后是不是没有正常接入呢，你帮我看看呢

诊断结论：接线本身是通的，`—` 是估值被判定为不可信时的降级显示。其中「已用不足 5%」是按设计；
真正让本应出数的一周估算变成 `—` 的原因是 [quota_estimate.go](../../../internal/application/activity/quota_estimate.go)
对 `usage_complete=0` 的记录采取**整窗否决**：实测生产库里 mibgb65 一周窗口 192 条记录中只有 4 条
「HTTP 200 成功但没解析出 usage」，2% 的脏记录让 98% 的有效样本一起显示 `—`。

## Goal

窗口内有少量用量未知的记录时，仍按已确认样本给出参考金额（金额本来就带 `≈`），
只有未知占比超过上限或完全没有已知样本时才显示 `—`，并同步产品文档的判定规则。

## Scope

- `internal/application/activity/quota_estimate.go`：`windowCost` 把 `incomplete` 从布尔否决改为计数＋占比上限。
- `internal/application/activity/quota_estimate_test.go`：补齐容忍／超限／全未知三种边界。
- `docs/product/SUBSCRIPTION.md`：更新「美元额度参考估算」的降级条件。

## Out of Scope

- 不改变「已用不足 5% 不估算」「未知型号不回退价格」两条既有规则。
- 不追查那 4 条成功响应为何没有 usage（另开任务定位上游）。
- 不改前端展示（`≈` 前缀与 `—` 语义保持）。

## Existing Behavior

`windowCost` 中任一记录 `usage_complete=0` 或 `state=pending` 即置 `incomplete=true`，
`Status` 随后被判为 `missing_usage`，前端两列金额显示 `—`。

## Expected Behavior

- 未知用量记录数 ≤ 窗口记录数的 20%：`Status=ready`，金额按已知样本估算。
- 超过 20%，或没有任何已知用量样本（`UsedMax<=0`）：保持 `missing_usage` / `no_usage`，显示 `—`。
- 未知价格模型仍整窗否决（`unknown_price`）。

---

# Analysis

## Related Modules

- `internal/application/activity/quota_estimate.go`（唯一判定点）
- `internal/transport/wails/subscription_estimate.go`（RPC 出口，不需改动）
- `frontend/src/features/accounts/subscription-quota-usd.tsx`（消费 `status`，不需改动）

## Dependencies

N/A

## Risks

- 金额从「不显示」变成「偏低的估计」。缓解：金额始终带 `≈`，且 20% 上限把最坏偏差限定在可解释范围；
  文档同时写明这是本机样本的 API 等值粗估。

---

# Implementation Plan

1. `windowCost` 用 `incomplete` 计数替代布尔，`known = Requests - incomplete - unpriced`。
2. 新增常量 `quotaEstimateUnknownShareLimit = 0.2`，`incomplete` 超过占比上限才判 `missing_usage`。
3. 单测覆盖：少量未知仍 ready、超限 missing_usage、全部未知 no_usage。
4. 更新 `SUBSCRIPTION.md` 判定条件。

---

# Progress

- 根因定位：生产库 `subscription_quota_snapshots` + `request_records` 交叉验证，确认
  `missing_usage` 由 4 条 `usage_complete=0` 的成功记录触发，而非快照缺失或价格未知。

---

# Implementation Result

## Changes

实际完成：

- `windowCost` 把 `incomplete` 从布尔否决改为计数，并新增占比上限
  `quotaEstimateUnknownShareLimit = 0.2`：未知占比超过两成才判 `missing_usage`。
- 判定顺序调整为 `unknown_price` → `missing_usage`（超限）→ `no_usage` → `ready`，
  让「有流量但样本不可用」与「没有流量」区分开。
- 未知价格模型仍整窗否决；已用不足 5% 的门槛不变。
- 新增两个回归用例：5 条已知 + 1 条未知 → `ready` 且金额只由已知样本构成；
  3 条已知 + 2 条未知 → 仍 `missing_usage`。

## Files Changed

- `internal/application/activity/quota_estimate.go`
- `internal/application/activity/quota_estimate_test.go`
- `docs/product/SUBSCRIPTION.md`、`docs/product/PRD.md`

## Technical Notes

实测生产库定位：mibgb65 一周窗口 192 条记录里有 4 条 `usage_complete=0` 的成功记录
（HTTP 200、0 token、codex-desktop），旧逻辑因此把整窗判为 `missing_usage`。
未知记录只让金额偏低、不会凭空偏高，所以按已知样本估算比整窗 `—` 更有信息量；
未知**价格**不同——它可能让整窗都无法定价，因此保持否决。

---

# Verification

## Tests

- [x] Unit Tests（`go test ./internal/application/activity/`，含新增两个子用例）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task verify
PASS（format:check / lint: go vet + staticcheck + eslint / typecheck: tsc /
Go 全包测试 / 前端 128 文件 760 项测试 / Windows production 构建）
```

---

# Documentation

- [x] `docs/product/SUBSCRIPTION.md`：降级条件改为「超过两成记录缺用量」，补充零星缺用量的取舍。
- [x] `docs/product/PRD.md`：同步同一规则。
- [x] `docs/plans/CURRENT.md`

---

# Deviations

原计划只改 `windowCost` 的判定；实际同时调整了判定顺序（把 `no_usage` 放到超限判断之后），
否则「全部记录都缺用量」会从 `missing_usage` 退化成 `no_usage`，掩盖真实原因。

---

# Git

## Commit

```text
7279fa8 fix(quota): 少数缺用量记录不再否决整窗美元估算
```

---

# Result

零星「成功但没有 usage」的记录不再否决整窗：未知占比 ≤ 20% 时按已确认样本给出参考金额
（`≈` 前缀已表达估算性质），超过 20% 或没有已知样本时仍显示 `—`；未知价格模型保持整窗否决。
mibgb65 一周窗口的 4/192 类情况现在能正常出数。

---

# Completed

2026-10-25 10:35
待填写。
