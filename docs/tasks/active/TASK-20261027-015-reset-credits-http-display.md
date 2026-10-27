# TASK-20261027-015: 重置卡信息在额度改走 HTTP 后恢复读取

## Status

In Progress

## Created

2026-10-27

## Request

用户反馈：重置卡面板显示「官方暂未返回重置卡信息」，当前可用重置卡为 `—`，无法使用卡片，要求定位并修复。

## Goal

界面重新拿到官方可用重置卡数量与明细，面板与入口角标恢复正常；官方确实未返回时仍显示未知而不是零；额度窗口本身不受重置卡读取失败影响。

## Background

- TASK-20261025-007（提交 `5ce4bfcf`）把订阅额度显示读取改为官方 HTTP 额度接口 `/backend-api/wham/usage`，只有传输／协议失败或需要续期时才回退 stdio。
- 重置卡汇总此前只来自 stdio 的 `account/rateLimits/read.rateLimitResetCredits`。前端 `ResetCreditsPopover`／`ResetCreditsContent` 的数量、角标与明细全部取自同一次额度查询返回的 `resetCredits`。

## Scope

- 官方 HTTP 重置卡接口 `GET /backend-api/wham/rate-limit-reset-credits` 的读取与解析。
- 显示读取把重置卡汇总并入额度结果；失败降级为「暂未返回」。
- 相关回归测试、产品文档与任务记录。

## Out of Scope

- 兑换（使用重置卡）链路，继续走既有 stdio `account/rateLimitResetCredit/consume` 与持久化幂等记录。
- 额度窗口、激活、故障切换与留存快照的既有语义。
- 前端面板布局、文案与交互。

## Existing Behavior

额度显示读取只调用 `wham/usage`，返回的 `RateLimits` 没有 `ResetCredits`；`normalizeQuota` 因此给出 `resetCredits: null`，面板显示「官方暂未返回重置卡信息」，角标不显示，`使用下一张可用卡` 一直禁用。

## Expected Behavior

- 打开面板或刷新额度时读取官方重置卡汇总：数量以 `availableCount` 为准，明细来自 `credits` 数组。
- 重置卡接口失败（未授权、传输、协议）不否决已读到的额度窗口，只让重置卡信息保持未知。
- 后台观察与网关调度读取不额外请求重置卡接口。

---

# Analysis

## Related Modules

- `internal/infrastructure/codex`：官方 HTTP 客户端（`UsageClient`）与重置卡模型／校验。
- `internal/application/subscription`：`readQuotaHTTP`、`normalizeQuota`、`normalizeResetCredits`。
- 前端 `features/accounts`：仅消费 `quota.resetCredits`，无需改动。

## Dependencies

官方 `wham/rate-limit-reset-credits` 的响应形状沿用 v1（`OmniProxyBackend/internal/proxy/codex_reset_credits.go`）与 cc-switch（`services/subscription.rs`）的既有实现：顶层可选 `data` 包裹，`available_count`／`credits[]`，条目字段 `id`、`title`、`description`、`reset_type`、`status`、`granted_at`、`expires_at`，时间可能是秒、数字字符串或 RFC3339。

## Risks

- 官方可能不返回 `granted_at`：需要允许缺失，界面继续显示 `—`，不能因缺字段让整份汇总失效。
- 额外一次网络请求：只在显示读取时发起，与主额度请求并行，失败不影响额度。

## Root Cause

额度显示路径从 stdio 换成 HTTP 后，HTTP 客户端只解析 `rate_limit` 窗口，没有任何代码读取重置卡；而前端唯一的重置卡来源就是这次额度结果，于是恒定显示未知。把「重置卡只来自 stdio」的旧假设保留在 HTTP 快路径上，形成了事实上的功能回归。

## Fix

在官方 HTTP 客户端补上重置卡读取，并在显示读取中与额度并行取得、合并到同一次 `Quota` 结果。

## Regression Prevention

新增基础设施层解析／失败分类测试与订阅服务层的「显示读取带重置卡、失败不影响额度、非显示读取不请求」测试。

---

# Implementation Plan

1. `internal/infrastructure/codex`：新增 `ReadResetCredits`（固定路径、与 `Read` 相同的头与失败分类），宽松解析 `available_count`／`credits`／时间字段；`validResetCredits` 允许缺失 `grantedAt`。
2. `internal/application/subscription`：`UsageSource` 增加 `ReadResetCredits`；`readQuotaHTTP` 在显示读取时并行取重置卡并合并，失败只记稳定码告警；`normalizeResetCredits` 处理缺失 `grantedAt`。
3. 测试：`internal/infrastructure/codex` 解析与状态分类；`internal/application/subscription` 合并／降级／非显示读取不请求。
4. 文档：`docs/product/SUBSCRIPTION.md` 额度与重置卡条目；`docs/plans/CURRENT.md` 概览。
5. 质量门：gofmt、`go vet`、`go test`、前端 typecheck／test，条件允许时 `task verify`。

---

# Progress

- 定位根因：HTTP 快路径只解析窗口，重置卡来源消失。
- 完成解析、服务层合并、回归测试与文档。

---

# Implementation Result

## Changes

- 官方 HTTP 客户端新增 `ReadResetCredits`：由额度地址派生 `.../rate-limit-reset-credits`，复用同一传输、超时与失败分类（401／403 → 未授权，其他非 200 与传输失败 → 瞬时，形状不符 → 协议）。
- 解析支持 `data` 包裹、`available_count`／`availableCount`、`credits`（`null` 表示仅数量）、条目 `reset_type`／`type`、`status`／`state`、`granted_at`／`created_at`、`expires_at` 及分数／数字字符串／RFC3339 时间；未知条目丢弃，其余仍展示。
- `validResetCredits` 允许缺失获得时间；`normalizeResetCredits` 缺失时不伪造时间。
- `readQuotaHTTP` 在显示读取时并行读取重置卡并合并进 `Quota.ResetCredits`；失败只记录 `subscription reset credits not read` 及稳定码；后台／网关观察读取不额外请求。

## Files Changed

- `internal/infrastructure/codex/usage_reset_credits.go`（新增）
- `internal/infrastructure/codex/usage_reset_credits_test.go`（新增）
- `internal/infrastructure/codex/reset_credits.go`
- `internal/application/subscription/quota_http.go`
- `internal/application/subscription/reset_models.go`
- `internal/application/subscription/quota_http_test.go`
- `docs/product/SUBSCRIPTION.md`
- `docs/plans/CURRENT.md`

## Technical Notes

重置卡明细只在显示读取时获取：网关调度、自动激活与后台额度观察不使用重置卡，避免每 5 分钟的观察多一次官方请求。兑换仍走 stdio，指定的卡片在使用前按官方 stdio 汇总复核。

---

# Verification

## Tests

- [x] Unit Tests（`internal/infrastructure/codex`、`internal/application/subscription`）
- [ ] Integration Tests
- [ ] E2E Tests

## Quality Checks

- [x] Format（`gofmt -l` 无差异、`pnpm format:check` 通过）
- [x] Lint（`go vet . ./internal/...`、`staticcheck`、`pnpm lint` 通过）
- [x] Type Check（`pnpm typecheck` 通过）
- [x] Test（Go 全量包、前端 132 文件／811 项）
- [x] Build（`pnpm run build` + `go build -tags production -trimpath -ldflags="-w -s"` 通过）
- [ ] Verify（未整体调用 `task verify`；上述五步分别执行）

## Verification Result

```text
gofmt -l internal/infrastructure/codex internal/application/subscription   （无输出）
go vet . ./internal/...                                                    exit 0
staticcheck . ./internal/...                                               exit 0
go test . ./internal/... -count=1                                          PASS
pnpm format:check / lint / typecheck                                       exit 0
pnpm test                                                                  132 files / 811 tests PASS
pnpm run build + go build -tags production -trimpath -ldflags="-w -s"       exit 0（23,088,128 bytes）
```

新增测试：

- `internal/infrastructure/codex`：官方汇总映射（标题／说明／类型／状态／数字与 RFC3339 时间、缺失字段保持未知）、仅数量与空明细、`data` 包裹、失败分类（403／429／5xx／截断／缺字段／无总数／负数）、目标地址派生与缺身份不发请求。
- `internal/application/subscription`：显示读取带重置卡（一次窗口请求＋一次重置卡请求、不启动官方组件）、后台观察不请求重置卡、重置卡失败仍返回窗口并记录稳定码。

未在真实账号上调用官方重置卡接口验收；真实接口字段仍需一次实机刷新确认。

---

# Documentation

- [x] 产品文档 `docs/product/SUBSCRIPTION.md`

```text
No ADR / architecture / tech stack changes required.
```

---

# Deviations

None.

---

# Git

## Commit

```text
Not committed by Agent.
```

---

# Result

PENDING

---

# Completed

PENDING
