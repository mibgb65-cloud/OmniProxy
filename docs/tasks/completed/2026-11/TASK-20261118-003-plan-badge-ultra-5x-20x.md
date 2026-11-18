# TASK-20261118-003: 套餐徽章区分 Google AI Ultra 5x／20x

## Status

Completed

## Created

2026-11-18 10:30（Asia/Singapore）

## Request

用户截图反馈：Antigravity 账号卡上 `g1-ultra-lite-tier` 仍显示原始标识，而 `g1-ultra-tier`
已被映射成「Ultra」。用户说明：`g1-ultra-lite-tier` 是 Ultra 的 **5x** 档，`g1-ultra-tier`
是 **20x** 档，要求把两者区分开。

## Goal

两个 Ultra 档位都显示为可读名称并能区分：5x 与 20x 各自有标签和配色，不再出现原始标识。

## Background

TASK-20261118-001 只映射了 `g1-ultra-tier`／`g1-pro-tier`／`g1-free-tier`／`free-tier`。
Google 的 `paidTier.name` 对 5x 与 20x 都返回「Google AI Ultra」，只有 `paidTier.id` 不同：
`g1-ultra-lite-tier` = Ultra 5x（升级提示可升到更高 Ultra），`g1-ultra-tier` = Ultra 20x
（社区实测与 Google 方案命名一致）。因此只能按 id 区分。

## Scope

- `subscription-plan-badge.tsx`：`g1-ultra-tier` → `Ultra 20x`、`g1-ultra-lite-tier` → `Ultra 5x`；
  Ultra 20x 保持 rose 实底，Ultra 5x 用同色系描边样式区分。
- 徽章测试补充两个档位的名称与配色断言。
- `docs/product/SUBSCRIPTION.md`、`docs/product/GOOGLE_SUBSCRIPTION.md` 同步档位对照。

## Out of Scope

- 不新增 `g1-pro-plus-tier` 等未经用户确认的档位映射。
- 不改变徽章数据来源，不改 Go 代码、数据库或上游请求。
- 不重新构建安装包（如需发版另开构建任务）。

## Existing Behavior

`g1-ultra-tier` → 「Ultra」（rose）；`g1-ultra-lite-tier` 未在对照表 → 原样显示、中性配色。

## Expected Behavior

- `g1-ultra-tier` → 「Ultra 20x」，rose 实底。
- `g1-ultra-lite-tier` → 「Ultra 5x」，rose 描边（与 20x 同色系、不同填充）。
- `g1-pro-tier` → 「Pro」等既有映射不变。

---

# Implementation Plan

1. 对照表增加 `g1-ultra-lite-tier`，并把 `g1-ultra-tier` 标签改为 `Ultra 20x`。
2. 配色表增加 `ultra 5x`（描边），保留 `ultra` 实底。
3. 更新测试与产品文档。
4. `task verify`，提交并回填。

---

# Implementation Result

## Changes

- 档位对照表：`g1-ultra-tier` → `Ultra 20x`，新增 `g1-ultra-lite-tier` → `Ultra 5x`；
  `g1-pro-tier`／`g1-free-tier`／`free-tier` 不变。
- 配色：`Ultra 20x` 保持 rose 实底；`Ultra 5x` 用 rose 描边（透明底），同色系但可区分。
- 测试：新增 5x／20x 两个用例，断言名称与配色互不相同。
- 文档：`SUBSCRIPTION.md`、`GOOGLE_SUBSCRIPTION.md` 更新档位对照与「两档官方 name 相同」的说明。

## Files Changed

- `frontend/src/features/accounts/subscription-plan-badge.tsx`
- `frontend/src/features/accounts/subscription-plan-badge.test.tsx`
- `docs/product/SUBSCRIPTION.md`、`docs/product/GOOGLE_SUBSCRIPTION.md`

## Technical Notes

- 5x／20x 无法从 `paidTier.name` 区分（两者都返回「Google AI Ultra」），只能按 `paidTier.id`；
  这也是此前 `g1-ultra-lite-tier` 显示原始标识的原因。
- 未把倍数写进数据模型，只放在展示名里；官方若调整档位命名，只需改这张对照表。

---

# Verification

## Tests

- [x] 前端 vitest：徽章 8 项用例通过（含 5x／20x 区分）

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
lint          go vet / staticcheck 通过；eslint 0 errors（2 个既有 warnings）
typecheck     tsc --noEmit 通过
test          Go 全包通过；前端 159 files / 977 tests passed
build         wails3 生产构建通过
```

---

# Documentation

- [x] `docs/product/SUBSCRIPTION.md`
- [x] `docs/product/GOOGLE_SUBSCRIPTION.md`

---

# Deviations

None.

---

# Git

## Commit

```text
c8edf0ea feat(accounts): 区分 Google AI Ultra 5x 与 20x 套餐徽章
```

---

# Result

Antigravity 账号卡的两个 Ultra 档位已分开：`g1-ultra-tier` 显示「Ultra 20x」（rose 实底），
`g1-ultra-lite-tier` 显示「Ultra 5x」（rose 描边），不再出现原始标识；其它档位映射不变。

---

# Completed

2026-11-18 10:45（Asia/Singapore）
