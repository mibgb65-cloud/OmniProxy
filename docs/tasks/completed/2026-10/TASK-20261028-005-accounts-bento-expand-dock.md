# TASK-20261028-005: 账号概览页 Bento 原位展开与底部切换 Dock

## Status

Completed

## Created

2026-10-28 11:40（Asia/Shanghai）

## Request

> 请将"账号概览页"与"提供商详情页"改造为 Bento Grid 卡片原位展开交互（In-place Expand + Bottom Switcher Dock）：
> 1. 点击任意提供商卡片（如 OpenAI）时，不发生路由硬跳转，而是将该卡片在网格顶部原位横向撑满（col-span-2），内部平滑展开为完整的双栏账号额度详情。
> 2. 未被点击的其余多个提供商卡片（Anthropic、Google、DeepSeek）利用布局动画平滑压缩并重排到底部，变成一排紧凑的迷你切换卡片（Dock），点击底部任意迷你卡片可直接在顶部无缝切换查看对应服务商详情。
> 3. 全程保持 600ms 的 FLIP / Layout 动画过渡。

## Goal

账号概览页从「卡片墙 + 跳转到独立详情页」改为单页内的原位展开：被点击的提供商在网格顶部横向撑满并展开
完整的账号／额度详情，其余提供商压缩成底部迷你 Dock 可一键切换；整个过程由 600ms FLIP 过渡衔接，
不产生路由跳转；深链接 `/accounts/<providerId>` 仍然可用。

## Background

- 概览页：`features/accounts/accounts-view.tsx` 用 `AccountProviderCard` 渲染卡片墙
  （`.account-card-grid-overview`，`auto-fit minmax(24rem, 1fr)`），卡片标题是指向
  `/accounts/$providerId` 的 `Link`。
- 详情页：`features/accounts/provider-accounts-view.tsx`，含提供商设置入口、账号展示方式切换、
  账号卡片／列表（`AccountDetailsCard` 内含额度／订阅额度）、增删改弹窗。
- 路由：`app/routes/accounts.tsx`、`app/routes/accounts_.$providerId.tsx`。
- 提供商页卡片点击会导航到 `/accounts/<providerId>`（`app/provider-navigation.test.tsx` 覆盖），该路径必须继续可用。
- 项目无 Framer Motion（Coding Rules 要求新依赖必须有当前具体用途），因此 FLIP 用原生
  `getBoundingClientRect` + Web Animations API 手写，零新依赖。

## Scope

- 新增 `features/accounts/use-flip-reflow.ts`：`capture()` 记录 `[data-flip-id]` 旧矩形，
  布局副作用里按新旧矩形做 translate + scale 补间（600ms，`cubic-bezier(0.22, 1, 0.36, 1)`）。
- 新增 `features/accounts/provider-dock-card.tsx`（迷你切换卡片）与 `bento-accounts.css`。
- `accounts-view.tsx`：改为 bento 布局；展开态渲染顶部面板 + 底部 Dock，收起态渲染卡片墙；
  展开／切换／收起都不改变路由。
- `account-provider-card.tsx`：标题 `Link` 改为按钮，整卡可点击（复用 providers 页的
  `INTERACTIVE` 守卫），新增 `onExpand`。
- `provider-accounts-view.tsx`：新增 `embedded` / `onCollapse`（面板内用 `h2`、无返回链接、
  收起按钮代替），非嵌入用法与既有测试不变。
- i18n：新增 `accounts.dock.label`（zh／en）。
- 测试：改写「卡片跳转详情页」为「原位展开且不改路由」，新增 Dock 切换与收起回归。
- 文档：`docs/product/APPEARANCE.md`、`docs/plans/CURRENT.md`。

## Out of Scope

- 不改后端接口、账号／额度数据契约与设置弹窗内容。
- 不改 `/providers` 页面与其跳转行为。
- 不做拖拽排序、不做多选展开（同一时刻只有一个提供商展开）。
- 不引入动画库。

## Existing Behavior

点击概览卡片标题会路由跳转到 `/accounts/<providerId>` 进入独立详情页，返回需要点击「返回账号」；
卡片之间没有布局动画。

## Expected Behavior

- 点击卡片（标题或卡片空白处）在页内展开该提供商，卡片原位横向撑满并展开完整账号／额度详情，
  地址栏不变。
- 其余提供商卡片以 600ms FLIP 过渡压缩重排到底部 Dock；点击 Dock 任意迷你卡在顶部切换，
  被切换走的详情面板同帧压缩回 Dock。
- 面板提供收起入口，收起时反向播放同一段 600ms 过渡回到卡片墙。
- 深链接 `/accounts/<providerId>` 仍进入独立详情页；系统「减少动画」偏好下不做 FLIP。

---

# Analysis

## Related Modules

- `features/accounts/{accounts-view,account-provider-card,provider-accounts-view,account-details-card}.tsx`
- `features/providers/{use-providers,provider-brand-icon,provider-labels}.tsx`
- `app/routes/accounts*.tsx`、`i18n/locales/{zh,en}.ts`
- 测试：`features/accounts/accounts-view.test.tsx`、`provider-accounts-view.test.tsx`、`app/provider-navigation.test.tsx`

## Dependencies

N/A（无新依赖；`getBoundingClientRect`、`Element.animate` 均为浏览器原生能力）。

## Risks

- **缩放失真**：卡片与面板尺寸差异大，FLIP 的 scale 会把文字压扁。→ 飞行期间给外层标记
  `data-flip-flying`，内部内容淡到 0.2，落位后淡回，视觉上是「面板移动 + 内容materialize」。
- **jsdom 无 Element.animate 且矩形恒为 0**：→ 缺能力检测 + 矩形为 0 时跳过，测试改为断言结构变化。
- **重复弹窗**：面板复用 `ProviderAccountsView`，概览与面板各有一套增删改弹窗；展开时卡片墙已卸载，
  同一时刻只有一套可交互。
- **深链接**：`/accounts/<providerId>` 保持渲染独立详情页，避免破坏提供商页跳转与既有测试。
- **焦点**：展开／收起会移除获得焦点的元素 → 展开后聚焦面板、收起后聚焦对应卡片按钮。

---

# Implementation Plan

1. 建立任务记录。
2. 写 FLIP hook（capture + play，含减少动画与 jsdom 兜底）。
3. 写 Dock 迷你卡片与 bento 布局 CSS。
4. 改造 `accounts-view.tsx`（展开状态、布局切换、焦点）、`account-provider-card.tsx`（按钮 + 整卡点击）、
   `provider-accounts-view.tsx`（`embedded`）。
5. 补 i18n 键与测试，运行前端质量门。
6. 更新文档、提交并归档。

---

# Progress

- 确认可复用 `ProviderAccountsView` 作为展开面板内容（加 `embedded` 开关），避免重写账号／额度详情，
  同时保住 `/accounts/<providerId>` 深链接与既有 386 行详情测试。
- 手写 FLIP（`getBoundingClientRect` + `Element.animate`）替代 Framer Motion，零新依赖。
- 发现缩放会把文字压扁，加入飞行期内容淡化（`data-flip-flying`）。

---

# Implementation Result

## Changes

- 新增 `use-flip-reflow.ts`：`capture()` 在状态更新前记录 `[data-flip-id]` 元素的矩形，
  布局副作用里按新旧矩形做 `translate + scale` 补间（600ms `cubic-bezier(0.22, 1, 0.36, 1)`）；
  变化可忽略时跳过，`prefers-reduced-motion` 下不播放，jsdom（无布局／无 `Element.animate`）自动跳过。
- 新增 `provider-dock-card.tsx`：底部迷你切换卡（品牌图标 + 名称 + 账号数 + 上箭头），
  `aria-label` 复用 `accounts.openDetails`。
- 新增 `bento-accounts.css`：`.bento-accounts` 网格（自适应列宽，展开面板与 Dock
  `grid-column: 1 / -1`，即两列时的 col-span-2）、`.bento-panel` 卡片外观、
  `.bento-dock` 自适应迷你卡行、飞行期内容淡化（0.12，120ms，双向过渡）。
- `accounts-view.tsx`：`expandedId` 本页状态；展开时渲染「顶部面板 + 底部 Dock」，收起时渲染卡片墙；
  点击卡片／Dock 都走 `showProvider()`（先 `captureFlip()` 再 setState），不触碰路由；
  展开后聚焦面板、收起后把焦点交回对应卡片。
- `account-provider-card.tsx`：标题 `Link` 改为 `h2 > button`（保留标题语义），新增 `onExpand`；
  整卡可点击并复用 `INTERACTIVE` 守卫，卡片内按钮（添加账号、开关、删除、编辑）不触发展开。
- `provider-accounts-view.tsx`：新增 `embedded` / `onCollapse`——面板内标题降为 `h2`、
  返回链接改为收起按钮；非嵌入用法（`/accounts/<providerId>` 独立页）完全不变。
- i18n：新增 `accounts.dock.label`（zh「切换提供商」／en "Switch provider"）。
- 测试：原「卡片跳转详情页」改写为「原位展开且地址不变」，新增「Dock 切换」与「收起回卡片墙」两项。

## Files Changed

- `frontend/src/features/accounts/use-flip-reflow.ts`（新增）
- `frontend/src/features/accounts/provider-dock-card.tsx`（新增）
- `frontend/src/features/accounts/bento-accounts.css`（新增）
- `frontend/src/features/accounts/accounts-view.tsx`
- `frontend/src/features/accounts/account-provider-card.tsx`
- `frontend/src/features/accounts/provider-accounts-view.tsx`
- `frontend/src/features/accounts/accounts-view.test.tsx`
- `frontend/src/i18n/locales/{zh,en}.ts`
- `docs/product/APPEARANCE.md`、`docs/plans/CURRENT.md`

## Technical Notes

- 展开／切换／收起共用一条 FLIP 路径：三个动作都只是「同一批 `data-flip-id` 元素换了位置和尺寸」，
  因此不需要为每种情况单独写动画分支；被展开的提供商同时拥有卡片矩形与面板矩形，
  由同一 id 对接，形成「卡片长成面板」的连续观感。
- 面板与 Dock 都横跨整行，被展开者永远在网格顶部；同一时刻只有一个提供商展开。
- 面板内容复用 `ProviderAccountsView`，账号／额度／设置入口／弹窗行为与独立详情页一致；
  展开时卡片墙已卸载，同一时刻只有一套可交互的增删改弹窗。
- 焦点管理：展开聚焦面板（`role="region"` + `tabIndex={-1}`），收起聚焦回该提供商卡片的按钮。

---

# Verification

## Tests

- [x] Unit Tests（`accounts-view.test.tsx` 9 项，含新增 3 项）
- [x] Integration Tests（`provider-accounts-view.test.tsx` 23 项、`app/provider-navigation.test.tsx` 保持通过）
- [ ] E2E Tests：N/A

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
PASS（exit 0）
gofmt --check / prettier --check：通过
go vet / staticcheck / eslint：通过
tsc --noEmit：通过
go test . ./internal/...：通过
frontend vitest：135 文件 / 826 项通过（原 135／824，净增 2 项）
Windows production 构建：通过；产物 CSS 含 .bento-accounts / .bento-panel-slot /
  .bento-dock / .bento-flip[data-flip-flying] / .bento-dock-card
```

未执行：真实 Wails 窗口内的展开／压缩／切换观感（600ms 节奏、缩放期间内容淡化、
宽窗多列与窄窗单列、140% 字号）人工验收；FLIP 在 jsdom 中按设计跳过，测试只覆盖结构与状态。

---

# Documentation

- [x] APPEARANCE（Bento 原位展开与 Dock 段落）
- [x] CURRENT（In Progress 与 Recently Completed）
- [ ] PRD／Architecture／Tech Stack／Coding Rules／Testing／Build／ADR（无后端与数据契约变化）

---

# Deviations

1. **面板内容复用**：原计划把详情页内容搬进新面板组件；实际给 `ProviderAccountsView` 加
   `embedded` 开关后直接复用，减少重复代码，并保留 `/accounts/<providerId>` 独立详情页与既有测试。
2. **col-span-2**：实现为 `grid-column: 1 / -1`。卡片墙保留原有自适应列宽（不改成固定两列以免
   窄窗挤压、宽窗空位），两列时即为 col-span-2，多列时同样横向撑满。
3. **标题可点击元素**：原来的 `<Link>` 换成 `h2 > button`（button 不能合法包含 `h2`／`div`），
   既保住「标题」语义与既有 `getByRole("heading")` 断言，又不再产生导航。
4. **补充内容淡化**：需求未提，但 FLIP 的 scale 会把文字压扁，因此飞行期把内部内容淡到 0.12，
   落位后淡回。
5. **测试观察**：全量并行跑测试时曾出现一次与本任务无关的 `stepfun.test.tsx` 失败
   （实际额度面板断言超时），单独重跑 3 次与再次全量均通过，属既有偶发，未改其代码。

---

# Git

## Commit

```text
34953623 feat(accounts): 概览页改为 Bento 原位展开与底部切换 Dock
```

---

# Result

账号概览页现在是单页 Bento 交互：点击任意提供商卡片在原位撑满展开完整账号／额度详情（地址不变），
其余提供商以 600ms FLIP 压缩成底部 Dock，点 Dock 可在顶部无缝切换，面板内「返回账号」收起回卡片墙；
展开、切换、收起走同一条 600ms `cubic-bezier(0.22, 1, 0.36, 1)` 过渡，飞行期内容淡化避免缩放失真。
`task verify` 全量通过（135 文件／826 项前端测试 + Windows production 构建），真实窗口观感待用户验收。

---

# Completed

2026-10-28 15:50

