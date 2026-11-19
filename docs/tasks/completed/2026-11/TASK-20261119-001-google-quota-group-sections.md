# TASK-20261119-001: Google 账号页按额度分组分区

## Status

Completed

## Created

2026-11-19 09:52

## Request

用户提供两张 Antigravity Pro 账号卡截图，问两个号的区别。确认档位徽章和 `account_id`
都无法区分「自费 Pro」与「学生 Pro」后，用户提出：

> 把含有 Gemini 4 Argon 额度的 pro 账号放到一起，不含有的放到一起这样

经确认采用以下口径：

- 分组依据：通用「按账号实际返回的额度分组名集合」分区，不写死 `Gemini 4 Argon`；
- 生效范围：只 Google 账号页，卡片视图与列表视图都生效；
- 形态：常开，不加开关。

## Goal

Google 账号页把订阅账号按额度快照里的分组名集合分区显示：分组集合相同的账号相邻成组，
分组更多的账号（例如多出 `Gemini 4 Argon`）单独成区，额度尚未获取的账号归入末尾分区。

## Background

`Gemini 4 Argon` 不是账号字段，而是 Google 额度快照里的第三个 `groups[].name`（另外两个是
`Gemini Models`、`Claude and GPT models`）。档位徽章来自 `auth_plan`（`paidTier.id`），
无法区分付费来源，所以「是否含 Argon」只能从额度快照判断。当前账号页按后端返回顺序直接
铺开，含与不含该分组的账号混在一起。

## Scope

本任务预计包含：

- 新增纯函数：把账号按「额度分组名集合」聚合为分区，定义分区顺序与签名；
- 新增 hook：为 Google 订阅账号订阅额度查询，供页面分区使用（复用现有查询键，不额外发请求）；
- Google 账号页：卡片视图与列表视图按分区渲染，分区标题显示分组名与账号数；
- 额度分组名的展示映射抽成共享常量，`GoogleQuotaValues` 与分区标题共用；
- 中英文案；
- 前端单元与组件测试。

## Out of Scope

本任务明确不处理：

- 不按套餐档位（Pro／Ultra／Free）分区，不尝试区分自费 Pro 与学生 Pro；
- 不改动其他提供商的账号页、总览页、配额总览页；
- 不改后端、不加数据库迁移、不加用户可配置的排序偏好；
- 不改变现有额度查询时机、并发队列与快照留存策略。

## Existing Behavior

Google 账号页把 `listAccounts` 返回的账号按后端顺序渲染成卡片网格或列表表格；每张卡的额度
区域各自查询自己的额度快照。含 `Gemini 4 Argon` 的账号与不含的账号按创建时间混排。

## Expected Behavior

- 页面按额度分组集合分区：分组集合更大的分区在前，其次按分组名排序；`API Key` 账号单独一区
  并排在已知分组之后；额度未获取（未登录、尚未刷新、读取失败）的订阅账号排在最后；
- 分区标题显示该区的分组名（已知分组本地化）与账号数；
- 只有一个分区时不渲染分区标题，页面与现在一致；
- 分区不改变账号顺序、不改变每张卡的额度查询行为，也不因分区额外发起额度请求；
- 卡片视图每区一个卡片网格，列表视图每区一张表格。

---

# Analysis

## Related Modules

预计涉及：

- `frontend/src/features/google/`：额度分组纯函数、额度查询 hook、分组名展示映射；
- `frontend/src/features/accounts/provider-accounts-view.tsx`：页面分区渲染；
- `frontend/src/i18n/locales/zh.ts`、`en.ts`：分区文案。

## Dependencies

- `@tanstack/react-query` 的 `useQueries`（已在依赖中）；
- 现有 `googleSubscriptionQuotaOptions` 查询键与 `readCachedGoogleSubscriptionQuota` 本地快照。

没有：N/A

## Risks

- 首次进入页面时额度尚未返回，全部账号先落在「未知」分区，数据到达后重新分区造成一次布局
  跳动。缓解：未知分区固定放在最后、只有一个分区时不显示标题，且本地快照会先行填充；
- 分区依赖额度查询结果，若把查询写在页面层可能与卡片内的查询重复请求。缓解：复用同一
  `queryKey` 与同一 `queryOptions`，由 react-query 去重。

---

# Implementation Plan

1. 在 `google-quota-model.ts` 增加额度分组名的展示映射常量，`GoogleQuotaValues` 改用它，
   避免两处各写一份已知分组名。
2. 新增 `google-quota-groups.ts`：从额度快照取分组名签名、把账号聚合成分区，定义分区顺序
   （按分组数降序 → 分组名 → API Key → 未知）。
3. 新增 `use-google-quota-sections.ts`：用 `useQueries` 复用 `googleSubscriptionQuotaOptions`
   订阅各订阅账号的额度，把结果交给纯函数分区。
4. `provider-accounts-view.tsx`：Google 提供商时按分区渲染卡片网格与列表表格，分区标题带
   分组名与账号数；只有一区时退回原样渲染。
5. 补充中英文案。
6. 测试：纯函数单测覆盖签名、排序、未知与 API Key 分区；组件测试覆盖卡片视图与列表视图的
   分区标题与分组归属；回归现有账号页测试。
7. `task verify`，更新 `docs/product/GOOGLE_SUBSCRIPTION.md` 与 `docs/plans/CURRENT.md`，
   提交。

---

# Progress

- 已确认分组只能来自额度快照 `groups[].name`，`auth_plan` 不含付费来源信息。
- 纯函数与 hook 完成：签名保持上游分组顺序（卡片内也按此顺序展示），分区键用排序后的分组名，
  避免同一集合因上游顺序不同被拆成两区。
- 组件测试暴露一处假设错误：订阅账号的卡片标题显示授权邮箱而非本地账号名，断言改为邮箱后通过。
- 首次 `task verify` 失败两次：一次是 `stepfun.test.tsx` 在全量并行下 5s 超时（单跑 5/5 通过），
  一次是末尾 `powershell Remove-item *.syso` 收尾返回 1（本项目已知偶发收尾失败，`task build`
  单独重跑 exit 0）。串行重跑 `task verify` 通过。

---

# Implementation Result

## Changes

实际完成：

- 新增 `quotaGroupSignature` / `buildQuotaGroupSections`：按额度分组名集合把账号聚合成分区，
  分区顺序为「分组更多的账号 → 分组名 → API Key 账号 → 额度未知」，账号在每个分区内保持后端
  返回顺序；
- 新增 `useGoogleQuotaGroupSections`：用 `useQueries` 复用卡片的 `googleSubscriptionQuotaOptions`
  查询键，只有 Google 提供商页启用，非 Google 页面不发起任何额度查询；
- 新增 `AccountQuotaGroupSection`：分区标题（分组名 · 分组名）＋账号数，未知分区附「额度尚未
  获取」说明；`aria-label` 使用同一标题，便于定位；
- Google 账号页在 `sections.length > 1` 时分卡片视图或列表视图分区渲染，并显示一行分组口径
  说明；只有一个分区时保持原来的平铺；
- 已知分组名（`Gemini Models`、`Claude and GPT models`）的本地化映射抽到
  `google-quota-model.googleQuotaGroupLabel`，卡片内分组标题与分区标题共用；
- 中英文案：`accounts.quotaGroup.apiKey`／`unknown`／`unknownHint`／`hint`。

## Files Changed

- `frontend/src/features/google/google-quota-groups.ts`（新增）
- `frontend/src/features/google/google-quota-groups.test.ts`（新增，7 项）
- `frontend/src/features/google/use-google-quota-group-sections.ts`（新增）
- `frontend/src/features/accounts/account-quota-group-section.tsx`（新增）
- `frontend/src/features/accounts/provider-accounts-view.tsx`
- `frontend/src/features/accounts/provider-accounts-view.test.tsx`（新增 3 项）
- `frontend/src/features/google/google-quota-model.ts`
- `frontend/src/features/google/google-quota-values.tsx`
- `frontend/src/i18n/locales/zh.ts`、`en.ts`
- `docs/product/GOOGLE_SUBSCRIPTION.md`

## Technical Notes

- 分区键用「排序后的分组名集合」，签名本身保持上游顺序：同一集合不会因上游顺序不同被拆区，
  卡片与分区标题仍按上游顺序展示（`Gemini 模型 · 第三方模型 · Gemini 4 Argon`）。
- 分区只读分组名，不读比例与重置时间，因此额度用掉多少不会让账号换区。
- 分组名之外的标识保留原文，不推断含义；分区不读 `auth_plan`，与套餐档位无关。
- 「未获取」单独成区并给出原因，避免把「还没读到」误当成「该账号没有这组额度」。

---

# Verification

## Tests

- [x] Unit Tests — `google-quota-groups.test.ts` 7 项（签名、去重、上游顺序、额外分组分居、
  顺序无关合并、分区内顺序、API Key／未知排序、空输入）
- [x] Component Tests — `provider-accounts-view.test.tsx` 3 项（卡片与列表视图分区、单一分区不
  显示标题、未知分区说明），并回归该文件既有 17 项
- [x] E2E Tests — 不适用：桌面窗口实机目视检查未执行

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
EXIT=0
format:check  All matched files use Prettier code style
lint          go vet / staticcheck / eslint 通过（仅 2 条既有 react-refresh 警告，非本次文件）
typecheck     tsc --noEmit 通过
go test       . ./internal/... 全绿
frontend test 163 files / 1000 tests passed
build         windows:build:native → bin/omniproxy.exe 24,785,408 bytes
```

过程中两次失败均为非本次改动引起、且已单独复现排除：

1. `src/features/accounts/stepfun.test.tsx > uses one platform and billing selection for the
   entire batch`：全量并行下 5046ms 超过 5s 上限，单跑 5/5 通过（1642ms）。
2. `windows:build:native` 末尾 `powershell Remove-item *.syso` 返回 1：本项目多次记录的偶发收尾
   失败，`task build` 单独重跑 exit 0 且产出 `bin/omniproxy.exe`；随后串行重跑 `task verify`
   EXIT=0。

---

# Documentation

- [ ] PRD — 未改：不改变产品定义，账号页交互细节记在 `docs/product/GOOGLE_SUBSCRIPTION.md`
- [ ] Architecture — 无变化
- [ ] Tech Stack — 无变化
- [ ] Coding Rules — 无变化
- [ ] Testing — 无变化
- [ ] Build — 无变化
- [ ] ADR — 无新增长期架构决策
- [x] CURRENT — 已更新

`docs/product/GOOGLE_SUBSCRIPTION.md` 已补充账号页分区口径、「未获取」不代表缺少该分组、
分区复用卡片查询与本地快照且不额外请求上游。

---

# Deviations

- 原计划未提分区口径说明行；实际在出现多个分区时增加一行 `accounts.quotaGroup.hint`，
  否则读者无法判断分区依据。
- 组件测试最初按本地账号名断言卡片标题，实际订阅账号标题是授权邮箱（`accountDisplayName`），
  断言改为邮箱；实现未改。
- 计划中的 `use-google-quota-sections.ts` 实际命名为 `use-google-quota-group-sections.ts`，
  与纯函数模块名对齐。

---

# Git

## Commit

```text
da458b60 feat(accounts): Google 账号页按额度分组分区
```

文档（本任务记录与 `docs/plans/CURRENT.md`）另行提交。

---

# Result

Google 账号页现在按官方额度分组集合分区：含 `Gemini 4 Argon` 等额外分组的账号自成一区并排在
前面，分组相同的账号相邻，API Key 账号与额度未知的账号排在最后；分区块标题给出分组名与账号数，
只有一个分区时不显示标题。分区复用卡片已有的额度查询与本地快照，不额外请求上游，卡片视图与
列表视图都生效。`task verify` EXIT=0（前端 163 文件／1000 项测试）。

---

# Completed

2026-11-19 10:30
