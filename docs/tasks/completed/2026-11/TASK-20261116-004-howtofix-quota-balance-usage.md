# TASK-20261116-004: 配额页同时显示 HowToFix 余额与本机 Token 用量

## Status

In Progress

## Created

2026-11-16 09:05（Asia/Singapore）

## Request

用户截图显示「配额」页上 4 个 HowToFix 账号全部标为「暂不支持」，卡片里只写
「此账号的额度或余额查询尚未接入，改为显示本机记录的已用 Token。」，Token 用量
输入／输出／缓存全为 0，底部写「尚无成功查询结果」；而「账号」页的余额已经能读到。

要求：把已经接入的余额接到配额页，和 Token 用量一起显示。

## Goal

配额页对 HowToFix 账号同时展示两件事：

1. 官方读取的预付余额（USD，剩余额度），并根据余额是否为正给出「额度／余额充足」或
   「额度耗尽／余额不足」状态；
2. 本机留存的 Token 用量（输入／输出／缓存），并注明本机用量不代表官方账单。

不再出现「暂不支持」与「尚未接入」提示。

## Background

- `internal/application/balance` 已按服务商类型分发，`howtofix` 分支
  （`readHowToFix`）通过 `GET /dashboard/billing/subscription` 读取预付余额，
  返回 `Info.HowToFix{currency, remaining}`，账号页 `AccountBalance` 已在使用。
- 配额页的资源类型判定在 `frontend/src/features/quota/quota-model.ts` 的
  `describeQuotaAccounts` 内，只把 `stepfun/payg`、`deepseek`、`kimi` 归为
  `balance` 类，`howtofix` 落到 `unsupported`，于是既不查询余额，也不参与
  「可监控账号」统计。
- `unsupported` 分支由 `quota-resource.tsx` 渲染，只展示本机 Token 用量。

## Scope

- `frontend/src/features/quota/quota-model.ts`：
  - `describeQuotaAccounts` 把 `howtofix` + API Key 归为 `balance` 类；
  - `quotaEntryStatus` 的 `balance` 分支处理 HowToFix：余额为空或不可解析为
    `unknown`，否则按 `available` 判定 `normal`／`exhausted`。
- `frontend/src/features/quota/quota-resource.tsx`：
  - `balance` 渲染链新增 `HowToFixBalanceValues`；
  - HowToFix 卡片在余额下方追加 `AccountUsagePanel` 与本机用量说明。
- 测试：
  - `frontend/src/features/quota/quota-model.test.ts`：类型归类与三种余额状态；
  - `frontend/src/features/quota/quota-view.test.tsx`：余额与 Token 用量同卡显示，
    且不再渲染「暂不支持」。
- 文档：`docs/product/HOWTOFIX.md`、`docs/plans/CURRENT.md`。

## Out of Scope

- 不改后端余额查询、不新增后端接口、不改 Wails 绑定（`Info.HowToFix` 已存在）。
- 不为其他未接入余额的服务商（River、Oxion、自定义 OpenAI 兼容等）增加额度查询。
- 不改「账号」页余额卡片与用量页统计口径。

## Impact

- 配额页：HowToFix 账号从「暂不支持」变为可查询、可单卡刷新、可批量刷新，并计入
  「可监控账号」「查询异常」等汇总。
- 首页资源卡片复用同一 `describeQuotaAccounts`，因此 HowToFix 余额也会进入首页
  「资源」统计与主动检查范围（逻辑一致，无额外改动）。
- 无数据模型、协议或 API 兼容性变化。

---

# Analysis

配额页能否显示资源只取决于 `describeQuotaAccounts` 给出的资源类型，余额读取本身
已经可用；因此问题属于前端归类缺失，而不是后端能力缺失。把 HowToFix 归入既有
`balance` 类型可以复用已有的查询、刷新、留存、状态汇总链路，避免为单一服务商
新增类型分支。

余额状态的判定必须区分「上游没返回金额」和「余额为 0」：前者是未知，后者才是耗尽，
否则会把查询不到数据误报成余额不足（与 `HowToFixBalanceValues` 的空值显示规则一致）。

Token 用量继续复用 `AccountUsagePanel`，它的数据来自本机请求记录，与官方余额是两个
不同来源，因此需要在同一卡片内用标题与说明区分，而不是拼成一个数字。

---

# Implementation Plan

1. 新建本 Task Record（Status: Planned）。
2. `quota-model.ts`：把 `howtofix` 加入 `balance` 归类；在 `quotaEntryStatus`
   的 `balance` 分支前置 HowToFix 判定。
3. `quota-resource.tsx`：`balance` 渲染链插入 `HowToFixBalanceValues`；HowToFix
   卡片追加 `AccountUsagePanel` 与 `quota.unsupportedUsageNote` 说明。
4. 补充 `quota-model.test.ts`、`quota-view.test.tsx` 用例。
5. `task verify`。
6. 更新 `docs/product/HOWTOFIX.md`、`docs/plans/CURRENT.md`，回填结果并提交。

---

# Implementation Result

## Changes

- `quota-model.ts`：`describeQuotaAccounts` 把 `howtofix` + API Key 归入既有 `balance`
  类型（与 DeepSeek／Kimi 普通余额同一链路）；`quotaEntryStatus` 的 `balance` 分支优先
  处理 HowToFix：金额为空或不可解析 → `unknown`，否则按后端 `available` 判
  `normal`／`exhausted`。
- `quota-resource.tsx`：`balance` 渲染链新增 `HowToFixBalanceValues`；HowToFix 卡片在
  余额与错误／提示之后追加 `AccountUsagePanel` 与本机用量说明，余额与 Token 用量分区展示。
- `quota-model.test.ts`：新增 HowToFix 归类与四种状态（正常／耗尽／未知／读取中）断言。
- `quota-view.test.tsx`：新增“余额与本机 Token 用量同卡、不再出现暂不支持”的页面级断言。
- 文档：`HOWTOFIX.md`、`BALANCE.md`、`QUOTA.md`、`PRD.md` 补充配额页行为。

## Files Changed

- `frontend/src/features/quota/quota-model.ts`
- `frontend/src/features/quota/quota-model.test.ts`
- `frontend/src/features/quota/quota-resource.tsx`
- `frontend/src/features/quota/quota-view.test.tsx`
- `docs/product/HOWTOFIX.md`
- `docs/product/BALANCE.md`
- `docs/product/QUOTA.md`
- `docs/product/PRD.md`

## Technical Notes

- 复用 `balance` 类型而不是新增类型：`use-quota-overview`、`use-dashboard-resources`、
  `quotaSummary` 与刷新队列都按 `balance` 订阅余额查询，因此配额页、首页资源卡片与
  批量刷新无需额外分支就能覆盖 HowToFix；区分渲染只靠 `entry.balance.howtofix`。
- 状态判定不使用前端浮点比较余额：`available` 由后端 `big.Rat` 计算 `remaining > 0`
  得出；前端只在金额缺失或不可解析时降级为 `unknown`，避免把缺数据误报成余额耗尽。
- Token 用量复用 `AccountUsagePanel`（`readAccountTokenUsage`），与官方余额是两个来源，
  因此同一卡片内分区并保留“本机用量不是官方账单”的既有说明，不合并成单一数字。

---

# Verification

## Tests

- [x] `pnpm exec vitest run src/features/quota`：3 文件 / 20 项通过
- [x] 新增两项用例定向通过（`-t "HowToFix"`：2 passed）
- [x] 全量前端测试 155 文件 / 945 项通过（原 943 项 + 新增 2 项）
- [x] 全部 Go 包测试、Windows production 构建随 `task verify` 通过
- [ ] 原生窗口视觉验收未执行（需用户在本机安装新构建后确认）

## Quality Checks

- [x] Format（`pnpm exec prettier --write` 修正一处测试文件格式）
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task verify
EXIT=0
Test Files  155 passed (155)
Tests       945 passed (945)
```

---

# Documentation

- `docs/product/HOWTOFIX.md`：余额章节补充配额页展示规则。
- `docs/product/BALANCE.md`：余额查询支持范围加入 HowToFix。
- `docs/product/QUOTA.md`：开头、页面规则、缓存复用与新增「HowToFix 余额」小节。
- `docs/product/PRD.md`：HowToFix 条目、余额查询范围与配额页汇总范围。
- `docs/plans/CURRENT.md`：登记本任务结果。

---

# Git

## Commit

```text
0a38226d fix(quota): 配额页显示 HowToFix 余额与本机 Token 用量
```

未推送、无标签、无历史改写。

---

# Result

配额页不再把 HowToFix 账号标为「暂不支持」：卡片上半部分直接显示官方预付美元剩余额度
（充足／不足状态来自后端 `available`，缺数据保持待确认），下半部分显示该账号本机记录的
输入／输出／缓存 Token 用量，并注明本机用量不是官方账单；余额查询纳入单卡刷新、
「刷新当前列表」与「可监控账号」统计，首页资源卡片通过同一归类自动覆盖。

---

# Completed

2026-11-16 09:15（Asia/Singapore）
