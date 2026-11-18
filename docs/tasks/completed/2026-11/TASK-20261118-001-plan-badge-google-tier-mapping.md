# TASK-20261118-001: 套餐徽章的空值与 Google 档位显示

## Status

Completed

## Created

2026-11-18 08:40

## Request

用户反馈（附三张账号卡截图）：

1. WorkBuddy 账号卡上方徽章显示「套餐未知」，但同一张卡的额度标题旁显示「体验版」/「Free」。
   用户决定：**如果没有返回 plan 字段，就显示「无套餐」**。
2. Google / Antigravity 账号卡的徽章直接显示 `g1-ultra-tier`、`g1-pro-tier` 这类原始标识。
   用户要求：**映射成 Ultra、Pro 这些可读名称，并区分颜色**。

## Goal

- 账号卡套餐徽章在账号没有套餐字段时显示「无套餐」（中／英一致），不再显示「套餐未知」。
- Antigravity / Google 订阅的已知档位标识显示为可读档位名（Ultra、Pro），不同档位有可区分的标签配色；
  未列出的标识仍原样显示并保持中性配色，不推断等级、价格或额度。

## Background

套餐徽章 `SubscriptionPlanBadge` 的数据来自 `account.authPlan`（登录身份写入 `accounts.auth_plan`），
与 WorkBuddy 额度卡标题旁取自计费接口 `SubscriptionPackageName` 的套餐名是两个不同来源
（见 TASK-20261116-020）。WorkBuddy 的 `/v2/plugin/login/account` 不返回 `plan` 字段，
所以徽章一直为空；Antigravity 的 `loadCodeAssist` 返回 `paidTier.id` / `currentTier.id`，
真实值为 `g1-ultra-tier`、`g1-pro-tier`、`free-tier` 这类带产品前缀与 `-tier` 后缀的标识。

## Scope

- `frontend/src/features/accounts/subscription-plan-badge.tsx`：空值文案、已知 Google 档位映射、Ultra 配色。
- i18n：`subscription.planUnknown` 改为 `subscription.planNone`（文案「无套餐」/「No plan」）。
- 前端测试：新增徽章测试，更新既有的「套餐未知」断言。
- 文档：`docs/product/SUBSCRIPTION.md`、`docs/product/GOOGLE_SUBSCRIPTION.md`、`docs/product/WORKBUDDY.md`、
  `docs/plans/CURRENT.md`。

## Out of Scope

- 不改变徽章的数据来源（仍读 `account.authPlan`），不把计费接口的套餐名回写 `auth_plan`。
- 不改变 WorkBuddy 额度卡标题旁套餐名的显示规则（未知仍不显示）。
- 不修改任何 Go 代码、数据库结构或上游请求。
- 不为未知档位猜测价格、额度或等级。

## Existing Behavior

- `plan` 为空 → 显示「套餐未知」（英文「Unknown plan」）。
- `plan` 有值 → 原样显示，例如 `g1-ultra-tier`、`plus`、`Free`；颜色按原始值小写匹配。

## Expected Behavior

- `plan` 为空或只有空白 → 显示「无套餐」（英文「No plan」），中性配色。
- `plan` 为 `g1-ultra-tier` → 「Ultra」，rose 配色；`g1-pro-tier` → 「Pro」，amber 配色；
  `g1-free-tier` / `free-tier` → 「Free」，slate 配色。
- 其它任何标识（如 `plus`、`team`、`Pro`、`体验版`、未知 `g2-*`）保持原文与既有配色规则。

---

# Analysis

## Related Modules

- 前端徽章组件与 i18n 词典（`frontend/src/i18n/locales/subscription-{zh,en}.ts`）。
- 使用徽章的两处：账号卡（`SubscriptionAccountStatus`）与配额概览卡（`QuotaAccountCard`）。
- 上游档位来源：`internal/provider/antigravity/discovery.go`（只读，不改）。

## Dependencies

无新增依赖。

## Risks

- 徽章是多个服务商共用的组件，映射只对明确的 Google 档位标识生效，避免把其它服务商的
  套餐名误改成品牌名。
- 文案键名变更只影响一个调用点，测试会覆盖中英文两套词典的键存在性。

---

# Implementation Plan

1. 徽章组件：抽出已知档位对照表（标识 → 展示名），空值走 `subscription.planNone`。
2. 配色表补 `ultra`（rose），沿用既有 `pro`（amber）与 `free`（slate）。
3. i18n 改名并更新文案；更新既有测试断言。
4. 新增 `subscription-plan-badge.test.tsx`：空值、空白、映射、未知标识、配色区分。
5. 更新产品文档与 CURRENT。
6. `task verify`，提交，回填 commit。

---

# Progress

- 2026-11-18 08:40：定位到徽章读 `account.authPlan`，Google 档位原样透传；确认只改前端展示层即可，无需迁移。
- 2026-11-18 09:00：`task verify` EXIT=0（前端 159 文件／976 项测试全部通过，Go 全部包通过，生产构建产出 `bin/omniproxy.exe`）。

---

# Implementation Result

## Changes

实际完成：

- 套餐徽章：`plan` 为空或只有空白时显示「无套餐」（英文「No plan」），文案键由
  `subscription.planUnknown` 改名为 `subscription.planNone`。
- 新增已知 Google 档位对照：`g1-ultra-tier` → Ultra、`g1-pro-tier` → Pro、
  `g1-free-tier` / `free-tier` → Free；未列出的标识仍原样显示。
- 新增 `ultra` 配色（rose），沿用 `pro`（amber）与 `free`（slate），Ultra 与 Pro 视觉可区分。
- 新增徽章组件测试 7 项；更新既有「套餐未知」断言为「无套餐」。
- 同步 `SUBSCRIPTION.md`、`GOOGLE_SUBSCRIPTION.md`、`WORKBUDDY.md` 与 `CURRENT.md`。

## Files Changed

主要修改文件：

- `frontend/src/features/accounts/subscription-plan-badge.tsx`（空值文案 + 档位映射 + ultra 配色）
- `frontend/src/features/accounts/subscription-plan-badge.test.tsx`（新增）
- `frontend/src/features/accounts/subscription-login.test.tsx`（断言文案）
- `frontend/src/i18n/locales/subscription-zh.ts`、`subscription-en.ts`（键名与文案）
- `docs/product/SUBSCRIPTION.md`、`docs/product/GOOGLE_SUBSCRIPTION.md`、`docs/product/WORKBUDDY.md`
- `docs/plans/CURRENT.md`

## Technical Notes

- 映射只发生在展示层：数据库里的 `auth_plan` 仍保存上游原始标识（如 `g1-ultra-tier`），
  不改数据、不加迁移，也不影响按 `authPlan` 判断 Codex 五小时窗口等既有逻辑。
- 徽章数据来源未变（仍读 `account.authPlan`），WorkBuddy 账号因此显示「无套餐」；
  额度卡标题旁仍显示计费接口的 `SubscriptionPackageName`，两者不互相回写。
- 不上游未返回的档位不做推断：`legacy-tier` 等未列入对照表的标识保留原文与中性配色。

---

# Verification

## Tests

- [x] Unit Tests（前端 vitest：新增 7 项徽章测试，全部通过）

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
PASS (EXIT=0)

format:check  prettier 全部通过
lint          go vet / staticcheck 通过；eslint 0 errors（2 个既有 react-refresh warnings，位于
              reset-credit-list.tsx，与本次改动无关）
typecheck     tsc --noEmit 通过
test          go test . ./internal/... 全部 ok；前端 159 files / 976 tests passed
build         wails3 生产构建通过，产出 bin/omniproxy.exe
```

---

# Documentation

- [x] `docs/product/SUBSCRIPTION.md`
- [x] `docs/product/GOOGLE_SUBSCRIPTION.md`
- [x] `docs/product/WORKBUDDY.md`
- [x] `docs/plans/CURRENT.md`

---

# Deviations

None.

---

# Git

## Commit

```text
2b995410 feat(accounts): 套餐徽章空值显示无套餐并映射 Google 档位名
```

---

# Result

账号卡套餐徽章在账号没有套餐字段时显示「无套餐」；Antigravity 的已知档位标识显示为
Ultra／Pro／Free 并分别配色，其它标识仍原样显示。WorkBuddy 徽章与额度卡套餐名仍是两个
不同来源，本次未合并（如需要可另开任务统一来源）。

---

# Completed

2026-11-18 09:05
