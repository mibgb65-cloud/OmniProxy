# TASK-20261028-003: 侧栏导航选中态滑动药丸背景

## Status

Completed

## Created

2026-10-28 10:50（Asia/Shanghai）

## Request

> 请帮我优化当前项目中左侧导航栏（包含"仪表盘"、"账号"、"提供商"、"网关"、"路由"、"配额"、"用量"、"请求"、"日志"等菜单项）的选中状态切换动画。核心交互效果（滑动药丸背景 / Sliding Pill Indicator）：点击切换不同导航菜单项时，选中的圆角背景块不要生硬消失和出现，而是平滑垂直位移过渡到新选中项；时长 400–500ms，缓动 `cubic-bezier(0.22, 1, 0.36, 1)`；技术方案在方案 A（DOM Ref + CSS Transform，零依赖）与方案 B（Framer Motion `layoutId`）中按现有技术栈自选；层级上背景块 `z-index: 0`，图标文字包裹层 `position: relative; z-index: 10`；样式保持极简深色风格（8px 圆角、深灰背景、极细顶边高光、未选中项仅文字提亮）；首次挂载测量完成前不得产生从顶部滑下的误触发动画。

## Goal

侧栏九个导航项的选中背景变成一个独立、带阻尼缓动的滑动药丸：相邻项与跨项（如「仪表盘」→「日志」）都平滑位移，无淡出淡入；首次挂载、侧栏收展、字号变化、无匹配项路由下都不出现错位或误触发动画；键盘与无障碍语义不变。

## Background

- 导航渲染在 `frontend/src/components/common/app-sidebar.tsx` 的 `SidebarLink` 中，
  选中态由 TanStack Router `Link` 的 `activeProps={{ "data-active": "true" }}` 给出，
  样式为 `data-[active=true]:bg-sidebar-accent`，即每个按钮各自的背景块。
- 导航列表位于 `ScrollRegion`（可滚动视口 `p-2`）内部，因此背景块必须放在滚动内容里，
  随内容一起滚动，不能用窗口坐标定位。
- 项目未安装 Framer Motion（`frontend/package.json` 无该依赖），Coding Rules 要求新依赖需
  有当前具体用途，因此选择方案 A（DOM 测量 + CSS transform），零新依赖。

## Scope

- 新增 `frontend/src/components/common/sidebar-nav-indicator.tsx`（测量 + 渲染背景块）。
- `app-sidebar.tsx`：导航列表包一层相对定位容器并挂载背景块；选中项不再自绘背景，
  图标与文字包裹层加 `relative z-10`。
- `app-sidebar.css`：滑动药丸样式、450ms `cubic-bezier(0.22, 1, 0.36, 1)` 过渡、
  首次测量前不加过渡、`prefers-reduced-motion` 关闭。
- 单元／集成测试：选中项变化时背景块位移到目标项、`data-measured` 防闪烁标记、
  原有导航测试保持通过。
- 文档：`docs/product/APPEARANCE.md` 补充导航选中态说明。

## Out of Scope

- 不改路由结构、导航项顺序、图标、语言文案与折叠逻辑。
- 不改 `SidebarGatewayStatus`、设置入口与滚动条实现。
- 不引入 Framer Motion 或任何新依赖。

## Existing Behavior

点击导航时，旧项的 `bg-sidebar-accent` 立刻消失、新项立刻出现（同一帧内硬切），
没有任何位移过渡；hover 也是整块背景变色。

## Expected Behavior

- 选中背景块是一个独立的绝对定位元素，随选中项变化在纵向上平滑滑动（450ms，带阻尼缓动），
  跨项长距离移动同样丝滑，不淡出淡入。
- 背景块位于图标与文字之下（`z-index: 0`，内容层 `z-index: 10`），不拦截点击。
- 首次挂载、侧栏收展、全局字号缩放后位置与高度自动重新测量，不会从顶部 (0,0) 滑下来。
- 未选中项 hover 只提亮文字／图标，不覆盖滑动块。
- 系统「减少动画」偏好下不做位移动画，直接到位。

---

# Analysis

## Related Modules

- `components/common/app-sidebar.tsx`、`app-sidebar.css`
- `components/common/sidebar-label.tsx`（收展动画，不受影响）
- `components/common/scroll-region.tsx`（导航滚动容器，背景块需在其内容层内）
- `stores/ui-store.ts`（`sidebarCollapsed`、`fontScale`）
- `components/common/app-sidebar.test.tsx`（现有导航回归测试）

## Dependencies

N/A（无新依赖；使用 `useRouterState`、`useLayoutEffect`、`ResizeObserver`）。

## Risks

- **路由变化时机**：TanStack Router 在同一批次里更新各 `Link` 的 `data-active`，
  背景块组件通过 `useRouterState` 订阅路径，布局副作用在 DOM 提交之后运行，因此读到的是新状态。
- **字号缩放**：行高是 rem 单位（`h-9`），`fontScale` 改变会改变像素高度；用 `ResizeObserver`
  监听列表容器重新测量。
- **jsdom**：`offsetTop`／`offsetHeight` 恒为 0，测试需按元素下标打桩后断言 transform。
- **滚动**：背景块在滚动内容里，随内容滚动，无需滚动监听。

---

# Implementation Plan

1. 建立任务记录。
2. 新增 `sidebar-nav-indicator.tsx`：订阅路径、测量当前 `[data-active]` 项的
   `offsetTop`／`offsetHeight`、首帧后再开启过渡、`ResizeObserver` 重新测量。
3. `app-sidebar.tsx`：相对定位容器 + 背景块；链接去掉自带背景，包裹层 `relative z-10`。
4. `app-sidebar.css`：药丸视觉与过渡曲线、防闪烁标记、减少动画偏好。
5. 测试与文档，运行前端质量门。

---

# Progress

- 确认项目未安装 Framer Motion，`frontend/package.json` 无该依赖 → 采用方案 A（DOM 测量 + CSS transform）。
- 先按「订阅路由状态 + `useLayoutEffect` 测量」实现，但在集成测试中发现两个时序问题：
  兄弟节点在提交阶段的 ref 附件晚于自身布局副作用、以及路由首次匹配完成不一定改变 `pathname`。
  改为「回调 ref 以状态持有列表元素 + `MutationObserver` 观察 `data-active`」，不再依赖路由内部时序。

---

# Implementation Result

## Changes

- 新增 `SidebarNavIndicator`：读当前选中项（`[data-active="true"]`）的真实 `offsetTop`／`offsetHeight`，
  以 `transform: translateY()` + `height` 渲染独立背景块；`data-active` 变化由 `MutationObserver`
  捕获，容器尺寸变化（字号缩放、侧栏收展）由 `ResizeObserver` 重新测量；
  首次测量结果显示完成后的下一帧才加上 `data-measured="true"`，仅供过渡生效。
- `app-sidebar.tsx`：导航列表外加相对定位容器并挂载背景块；链接不再自绘选中／悬停背景，
  改为文字提亮 + `transition-colors`；图标与文字包在 `relative z-10` 的包裹层里。
- `app-sidebar.css`：背景块绝对定位、`z-index: 0`、`pointer-events: none`、
  `border-radius: var(--radius-md)`（8px）、`background-color: var(--sidebar-accent)`
  （深色主题即 `#262626` 级别的 oklch(0.269 0 0)）、`inset 0 1px 0` 顶边高光；
  过渡为 `transform` 与 `height` 各 450ms、`cubic-bezier(0.22, 1, 0.36, 1)`，
  仅在 `[data-measured="true"]` 时生效，`prefers-reduced-motion` 下关闭。
- 测试：新增 `sidebar-nav-indicator.test.tsx`（3 项：定位与移动复用同一元素、无匹配项时不渲染、
  首帧不带过渡），`app-sidebar.test.tsx` 增加 2 项（导航切换时背景块位移到新项、收展后位置不变）；
  新增 `src/test/nav-row-metrics.ts` 为 jsdom 提供确定的行高与偏移。

## Files Changed

- `frontend/src/components/common/sidebar-nav-indicator.tsx`（新增）
- `frontend/src/components/common/app-sidebar.tsx`
- `frontend/src/components/common/app-sidebar.css`
- `frontend/src/components/common/sidebar-nav-indicator.test.tsx`（新增）
- `frontend/src/components/common/app-sidebar.test.tsx`
- `frontend/src/test/nav-row-metrics.ts`（新增）
- `docs/product/APPEARANCE.md`
- `docs/plans/CURRENT.md`

## Technical Notes

- 选中判定不复制路由匹配规则：直接观察路由写在链接上的 `data-active`，
  因此嵌套路由（如 `/accounts/<providerId>` 仍高亮「账号」）沿用既有行为。
- 背景块宽度用 `inset-inline: 0` 跟随列表，侧栏收展时不额外补偿，也不会出现横向跳动。
- 测量后若 `top`／`height` 未变则复用旧对象，避免 `ResizeObserver` 与状态更新互相触发。
- `react-hooks/set-state-in-effect` 对「提交后读布局再回写状态」报错，按 ESLint 允许的形式加了
  一条带理由的定向豁免；该写法不产生渲染期副作用，也不构成循环渲染。

---

# Verification

## Tests

- [x] Unit Tests（`sidebar-nav-indicator.test.tsx` 3 项）
- [x] Integration Tests（`app-sidebar.test.tsx` 共 8 项，含新增 2 项）
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
frontend vitest：135 文件 / 824 项通过（原 134／819，新增 1 文件 5 项）
Windows production 构建：通过；产物 CSS 含
  --sidebar-indicator-duration:.45s、cubic-bezier(.22, 1, .36, 1)、
  .sidebar-nav-indicator[data-measured=true] 的 transform/height 过渡
```

未执行：真实 Wails 窗口内的滑动观感（鼠标点击、键盘 Tab 切换、140% 字号、收起态）人工验收。

---

# Documentation

- [x] APPEARANCE（侧栏导航选中态与滑动背景段落）
- [x] CURRENT（In Progress 与 Recently Completed）
- [ ] PRD／Architecture／Tech Stack／Coding Rules／Testing／Build／ADR（无范围、架构与流程变化）

---

# Deviations

1. **选中判定来源**：计划用 `useRouterState` 订阅路径；实际改为 `MutationObserver` 观察
   `data-active`。原因：首次路由匹配完成不一定改变 `pathname`，且提交阶段兄弟节点 ref 的
   附件顺序会让首次测量读到空列表；直接观察 DOM 真值后不再依赖路由内部时序，也不必复制匹配规则。
2. **列表元素传递方式**：计划用 `useRef` 引用对象；实际改为回调 ref + 状态持有元素，
   保证元素真正挂载后才测量。
3. **ESLint 豁免**：`react-hooks/set-state-in-effect` 与「布局测量」天然冲突，加一条定向豁免并写明理由。

---

# Git

## Commit

```text
<filled after commit>
```

---

# Result

侧栏九个导航项的选中背景变成独立的滑动药丸：相邻与跨项点击都以 450ms
`cubic-bezier(0.22, 1, 0.36, 1)` 阻尼缓动纵向滑动，不再就地硬切；背景块位于内容之下、
不拦截点击，未选中项悬停只提亮文字；首次挂载、侧栏收展与字号变化后自动重新测量，
不会从顶部滑下；系统减少动画偏好下直接到位。`task verify` 全量通过，真实窗口观感待用户验收。

---

# Completed

2026-10-28 11:10

