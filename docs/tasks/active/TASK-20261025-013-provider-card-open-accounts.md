# TASK-20261025-013: 提供商卡片可进入对应账号页面

## Status

Completed

## Created

2026-10-25 16:20

## Request

用户附「提供商」页面截图（内置提供商卡片网格）并提出：

> 提供商这个页面需要能够实现点击某个提供商卡片跳转到对应的提供商的账号页面。

## Goal

- 在「提供商」页面点击任一已配置的提供商卡片（含内置与自定义），进入该提供商自己的账号页面
  （`/accounts/<providerId>`）。
- 卡片内的启用开关、测试、模型预览、编辑、删除等既有操作保持原样，不被跳转拦截或误触。

## Background

现有「提供商」页（`features/providers/providers-view.tsx`）只做服务商配置与连通性测试，卡片内部
已有多个按钮；账号页面的路由 `/accounts_/$providerId`（`ProviderAccountsView`）已存在，但提供商页
没有任何入口指向它，用户必须先切到「账号」再点进同一提供商。

## Scope

- `ProviderCard` 增加可选的「打开账号页面」行为：整张卡片可点击，标题提供键盘可达的入口，悬停与
  焦点有明确反馈。
- 内置提供商目录（`ProviderCatalog`）与自定义提供商列表（`ProviderList`）透传该行为。
- 路由层用 TanStack Router 的 `navigate` 跳转到 `/accounts_/$providerId`。
- 中英文文案与相关测试。

## Out of Scope

- 不修改账号页面本身（`ProviderAccountsView`）的布局、查询或返回行为。
- 不新增路由、不改变 URL 结构。
- 不为账号页面增加「返回提供商」联动返回栈（仍回到「账号」列表，见 Deviations）。
- 不改变服务商启停、测试连通性、增删改的接口与语义。

## Existing Behavior

- 提供商卡片仅在其内部按钮上响应点击，点击卡片空白或标题没有任何动作。
- 进入某提供商的账号页面只能从侧栏「账号」→ 对应提供商卡片进入。

## Expected Behavior

- 点击卡片任意非按钮区域（标题、模型说明、地址、状态行等）跳转到该提供商的账号页面。
- 卡片内的按钮（启停、测试、模型预览、编辑、删除、MiMo 说明）仍只执行自身操作，不触发跳转。
- 标题是真实可聚焦控件，键盘 Tab + Enter 也能进入；焦点与悬停有可见反馈。
- 未配置的内置预设（无对应 provider 记录）仍不可点击。

---

# Analysis

## Related Modules

- `frontend/src/features/providers/{provider-card,provider-catalog,provider-list,providers-view}.tsx`
- `frontend/src/app/routes/providers.tsx`（路由层注入导航）
- `frontend/src/i18n/locales/{zh,en}.ts`

## Dependencies

现有 TanStack Router，无新增依赖。

## Risks

- 卡片整体可点击后误吞按钮事件：用 `closest("button, a, input, select, textarea")` 守卫，并让标题
  控件 `stopPropagation`。
- `features/providers` 的既有测试直接渲染组件、没有 RouterProvider：因此导航通过可选回调注入，
  路由层才使用 `useNavigate`，避免给所有测试套上路由器。

---

# Implementation Plan

1. `ProviderCard` 增加可选 `onOpen`，卡片容器加点击守卫与悬停／焦点样式，标题包成 button。
2. `ProviderCatalog`、`ProviderList` 透传 `onOpen`。
3. `ProvidersView` 增加可选 `onOpenProvider(id)` 并传给两张列表。
4. `routes/providers.tsx` 用 `useNavigate` 注入跳转。
5. 文案 `providers.openAccounts`（中英）。
6. 测试：卡片点击跳转、按钮不跳转、视图回调带上正确 provider id。

---

# Progress

- 已确认账号路由与 `ProviderAccountsView` 已存在，只需补入口；现有测试无路由器上下文，故用回调注入导航。

---

# Implementation Result

## Changes

实际完成：

- `ProviderCard` 新增可选 `onOpen`：提供后整张卡片可点击进入账号页面，卡片内的按钮、链接、输入等
  交互元素用 `closest` 守卫排除；标题同时变为可聚焦按钮（`aria-label` 为「查看<名称>的账号」），
  键盘 Tab + Enter 也能进入。卡片增加 `cursor-pointer`、悬停边框与 `focus-within` 焦点环。
  未提供 `onOpen` 时卡片保持原来的只读展示。
- `ProviderCatalog`、`ProviderList`、`ProvidersView` 逐层透传（`ProvidersView` 新增可选
  `onOpenProvider(providerId)`），内置与自定义提供商卡片都生效。
- `app/routes/providers.tsx` 用 `useNavigate` 注入跳转：`/accounts/$providerId`。
- 中英文新增 `providers.openAccounts` 文案；PRD 增加一条交互说明。

## Files Changed

- `frontend/src/features/providers/{provider-card,provider-catalog,provider-list,providers-view}.tsx`
- `frontend/src/app/routes/providers.tsx`
- `frontend/src/i18n/locales/{zh,en}.ts`
- 测试：`frontend/src/features/providers/{provider-card,providers-view}.test.tsx`、
  新增 `frontend/src/app/provider-navigation.test.tsx`
- 文档：`docs/product/PRD.md`、`docs/plans/CURRENT.md`

## Technical Notes

- 导航通过回调注入而不是在 feature 内直接用 `Link`／`useNavigate`：`features/providers` 的既有测试
  直接渲染组件、没有 RouterProvider，回调注入避免给全部测试套路由器，同时把路由知识留在路由层。
- 卡片点击用 `event.defaultPrevented` 与 `closest("button, a, input, select, textarea, [role='dialog']")`
  双重守卫，弹层、按钮和输入不会被整卡点击吞掉。

---

# Verification

## Tests

- [x] Unit Tests
- [x] Integration Tests
- [ ] E2E Tests（本仓库无 E2E 框架；用路由级集成测试替代）

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
exit=0
```

- PASS：`provider-card.test.tsx` 新增 3 项——整卡与标题都能触发入口、启停与测试按钮不触发入口、
  未提供入口时保持只读卡片。
- PASS：`providers-view.test.tsx` 新增 1 项——点击内置与自定义卡片分别回调正确 provider id，
  启停按钮不触发跳转；既有 11 项视图测试全绿。
- PASS：新增 `app/provider-navigation.test.tsx`，用真实路由层组件 + 内存路由验证点击卡片后路径为
  `/accounts/builtin_openai` 且目标页面收到对应 `providerId`。
- PASS：前端 131 文件／792 项测试、`pnpm lint`、`pnpm typecheck`、`prettier --check` 全部通过；
  Go 全部包与 `go vet`／`staticcheck` 无回归；Windows production 构建通过。
- 未执行：真实窗口中的视觉与手感验收（悬停高亮、焦点环、卡片整卡点击区域），以及自定义提供商账号页
  的真实数据展示。

---

# Documentation

- [x] PRD（提供商卡片新增进入账号页面的交互说明）
- [ ] Architecture（无新组件边界，未改）
- [ ] Tech Stack（无变化）
- [ ] Coding Rules（无变化）
- [ ] Testing（无变化）
- [ ] Build（无变化）
- [ ] ADR（无架构决策变更）
- [x] CURRENT

---

# Deviations

1. 账号页面的返回链接仍指向「账号」列表而不是「提供商」页（`ProviderAccountsView` 既有行为）。
   跟随来源返回需要额外导航状态，超出本次范围，未实现。
2. 标题用 `<button>` 而不是 `<Link>`：`features/providers` 的测试没有 RouterProvider，用真实链接
   需要给所有相关测试套路由器；按钮 + 回调同样支持键盘操作，路由知识保留在路由层。
3. 未给未配置的内置预设（无 provider 记录的破坏态）加点击入口，保持原有的错误提示卡片。

---

# Git

## Commit

```text
Not committed yet.
```

---

# Result

提供商页的每张卡片（内置与自定义）现在都能点击进入该提供商自己的账号页面，标题同时是键盘可达入口；
卡片内的启停、测试、模型预览、编辑、删除仍只执行自身操作。`task verify` exit 0
（131 文件／792 项前端测试，含新增的路由级跳转验证）。
