# TASK-20261116-006: 配额页显示 DuMate 积分与本机 Token 用量

## Status

Planned

## Created

2026-11-16 09:35（Asia/Singapore）

## Request

用户截图显示配额页上百度搭子 DuMate 账号（「一口鲜特卤」）标为「暂不支持」，
只显示本机 Token 用量（输入 295.99K／输出 2.47K／缓存 ≥146.18K），底部「尚无成功查询结果」。
用户指出「这个也可以加上余额显示吧，它的好像是积分啥的」。

要求：配额页把 DuMate 的官方积分接到卡片上，与已有本机 Token 用量一起显示。

## Goal

配额页对 DuMate 登录态账号同时展示：

1. 官方控制台读取的剩余积分、已用／总量与到期信息；
2. 本机留存的 Token 用量（输入／输出／缓存）。

不再出现「暂不支持」与「尚未接入」提示，并纳入「可监控账号」与刷新能力。

## Background

- DuMate 积分查询已实现：`internal/application/dumate.Points` 经
  `AppService.DuMatePoints` 暴露，账号页 `DuMatePoints` 组件已在使用（含签到）。
- 结果用 `State` 区分 `live`／`stale`（控制台暂时失败但有保留快照）／`unavailable`
  （无可用数据），失败不显示成零余额；快照只在本进程内存保留 1 小时。
- 配额页的资源归类（`quota-model.ts`）只认订阅额度与若干余额类型，DuMate 落到
  `unsupported` 分支，因此不查询积分、没有刷新入口、不计入统计。
- 积分是 DuMate 个人额度单位，不是货币余额，不折算成金额，也不与本机 Token 混算。

## Scope

- `frontend/src/features/dumate/use-dumate.ts`：导出 `duMatePointsOptions(accountId)`，
  让配额页与账号页共用同一查询键与缓存；账号页行为不变。
- `frontend/src/features/quota/quota-model.ts`：新增 `dumate_points` 资源类型与归类；
  快照增加 `dumate`；状态按 `live`／`stale`／`unavailable` 与剩余积分判定。
- `frontend/src/features/quota/use-quota-overview.ts`：为 DuMate 账号订阅积分查询。
- `frontend/src/features/quota/dumate-quota-resource.tsx`（新）：积分区块、错误／快照
  提示、本机 Token 用量与说明。
- `frontend/src/features/quota/quota-resource.tsx`：分派到新组件。
- `frontend/src/features/dashboard/use-dashboard-resources.ts`：首页只订阅积分缓存
  （不主动请求），纳入「已检查」计数。
- 测试：`quota-model.test.ts`、`quota-view.test.tsx`、`quota-test-helpers.tsx`。
- 文档：`docs/product/QUOTA.md`、`docs/product/DUMATE.md`、`docs/product/PRD.md`。

## Out of Scope

- 不在配额卡片提供签到按钮（签到仍在账号详情）。
- 不新增后端接口、不改积分查询协议、不把积分换算成金额或 Token。
- 不改 DuMate 导入、网关与路由。

## Impact

- 配额页：DuMate 账号从「暂不支持」变为可查询、可单卡／批量刷新，并计入
  「可监控账号」「待确认」「查询异常」统计；首页资源卡片显示其缓存状态。
- 无数据库迁移、无公共 API 变更；查询复用既有 `DuMatePoints` 绑定。

---

# Analysis

问题与 HowToFix 余额同源：配额页只按资源类型决定查询与渲染，后端能力已存在但前端
归类缺失。DuMate 的积分查询还带有 `live`／`stale`／`unavailable` 三态语义，因此不能
照搬「余额充足／不足」的简单判定：`stale` 必须显示为已保留的旧快照并标注待刷新，
`unavailable` 必须显示为查询失败，两者都不能被当成零积分或已耗尽。

积分是 DuMate 个人额度单位，与预付美元余额不是同一概念；本机 Token 用量又是第三个
来源，因此卡片按「官方积分 → 错误／快照提示 → 本机 Token 用量」分区，并继续使用
「本机用量不表示官方剩余额度或账单」的既有说明，不做任何换算或相加。

复用账号页已有的查询键与缓存，避免同一账号在配额页与账号页各查一次控制台；配额页
对停用账号仍做只读查询，与余额类资源的既有语义一致。

---

# Implementation Plan

1. 新建本 Task Record（Status: Planned）。
2. `use-dumate.ts`：抽出 `duMatePointsOptions`，`useDuMatePoints` 复用之。
3. `quota-model.ts`：`dumate_points` 类型、归类、状态与 `fetchedAt`。
4. `use-quota-overview.ts`：订阅 DuMate 积分查询并写入快照。
5. 新增 `dumate-quota-resource.tsx` 并在 `quota-resource.tsx` 分派。
6. `use-dashboard-resources.ts`：只读缓存订阅与检查计数。
7. 补充测试，运行 `task verify`。
8. 更新文档、回填结果并提交；随后构建 `v2.3.1-beta.10` 供本地更新。

---

# Implementation Result

## Changes

- `use-dumate.ts`：抽出并导出 `duMatePointsOptions(accountId)`，账号页与配额页共用同一
  查询键、缓存与失败策略；`useDuMatePoints` 行为保持不变。
- `quota-model.ts`：新增 `dumate_points` 资源类型与归类（`kind === "dumate"` 且
  `authMethod === "dumate_cookie"`）；快照增加 `dumate`；状态按
  `live`（剩余积分 >0 充足／0 耗尽）、`stale`（待刷新）、`unavailable`（查询失败）、
  未登录（需要登录）分别判定；`fetchedAt` 纳入积分查询时间。
- `use-quota-overview.ts`：为 DuMate 账号订阅积分查询（未登录不发起），停用查询不计为
  「正在读取」，避免空白卡片长期显示占位。
- `dumate-quota-resource.tsx`（新）：积分区块（可用积分、已用／总量、到期信息）、
  错误／保留快照／需要登录提示，以及本机 Token 用量与说明。
- `quota-resource.tsx`：`dumate_points` 分派到新组件。
- `use-dashboard-resources.ts`：首页只订阅积分缓存（不主动请求），并纳入「已检查」计数。
- 测试：资源归类与三态状态、页面级「积分与本机用量同卡」、登录态失效不查询。

## Files Changed

- `frontend/src/features/dumate/use-dumate.ts`
- `frontend/src/features/quota/quota-model.ts`
- `frontend/src/features/quota/quota-model.test.ts`
- `frontend/src/features/quota/use-quota-overview.ts`
- `frontend/src/features/quota/quota-resource.tsx`
- `frontend/src/features/quota/dumate-quota-resource.tsx`（新）
- `frontend/src/features/quota/quota-test-helpers.tsx`
- `frontend/src/features/quota/quota-view.test.tsx`
- `frontend/src/features/quota/quota-cache.test.tsx`
- `frontend/src/features/dashboard/use-dashboard-resources.ts`
- `docs/product/DUMATE.md`
- `docs/product/QUOTA.md`
- `docs/product/PRD.md`

## Technical Notes

- 积分三态是后端既有语义，配额页不重新推断：`stale` 只降级为「待刷新」并保留数值，
  `unavailable` 记为查询失败；两者都不进入「已耗尽」统计，避免把控制台故障报成没额度。
- 登录态失效（`authState !== "authenticated"`）时不发起控制台请求，直接显示需要登录，
  与账号页 `DuMateAccountStatus` 的判定一致。
- 积分不折算金额或 Token，本机 Token 用量单独展示并保留「不是官方账单」说明。
- 首页复用同一查询键但 `enabled: false`，挂载不产生控制台请求；「检查」按钮可刷新。

---

# Verification

## Tests

- [x] `pnpm exec vitest run src/features/quota src/features/dashboard src/features/dumate`：
  10 文件 / 54 项通过
- [x] 新增 DuMate 定向用例（`-t "DuMate"`）：3 项通过
- [x] 全量前端测试 155 文件 / 948 项通过（原 945 项 + 新增 3 项）
- [x] 全部 Go 包测试、Windows production 构建随 `task verify` 通过
- [ ] 原生窗口视觉验收未执行（需用户安装新构建后确认）

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
Tests       948 passed (948)
```

---

# Documentation

- `docs/product/DUMATE.md`：积分与签到章节补充配额页展示规则。
- `docs/product/QUOTA.md`：开头、页面规则、查询与缓存，以及新增「DuMate 积分」小节。
- `docs/product/PRD.md`：配额页汇总范围加入 DuMate 积分。
- `docs/plans/CURRENT.md`：登记本任务结果。

---

# Git

## Commit

```text
待回填
```

未推送、无标签、无历史改写。

---

# Result

配额页不再把 DuMate 登录态账号标为「暂不支持」：卡片展示官方控制台返回的可用积分、
已用／总量与到期信息，下方同时展示该账号本机 Token 用量并注明不是官方账单；实时成功按
剩余积分判定充足／不足，`stale` 标为待刷新、`unavailable` 标为查询失败，登录态失效显示
需要登录且不请求控制台。积分查询与账号页共用缓存，纳入单卡／批量刷新与「可监控账号」
统计，首页资源卡片显示其缓存状态。

---

# Completed

2026-11-16 09:50（Asia/Singapore）
