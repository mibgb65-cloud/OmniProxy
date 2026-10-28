# TASK-20261028-001: 主题切换圆形扩散（Circular Reveal）过渡动画

## Status

Completed

## Created

2026-10-28 09:45

## Request

在标题栏右上角的亮／暗主题切换按钮上实现「基于点击位置向四周圆形扩散」的主题切换过渡动画：

- 使用原生 `document.startViewTransition`（View Transitions API），不支持时优雅降级为普通状态切换。
- `onClick(event)` 捕获 `clientX/clientY`，半径取「点击点到屏幕四角最远距离」：
  `Math.hypot(Math.max(x, w - x), Math.max(y, h - y))`。
- 用 CSS `clip-path: circle(...)` 在 `::view-transition-new(root)` 上做扩散：
  `circle(0px at x y)` → `circle(radius at x y)`，时长 600–800ms，缓动 `cubic-bezier(0.4, 0, 0.2, 1)`。
- 主题状态更新在 `startViewTransition` 回调内完成。
- 封装为可复用 Hook（`useThemeTransition`）。
- 禁用 `::view-transition-old(root)` / `::view-transition-new(root)` 的默认淡入淡出
  （`animation: none; mix-blend-mode: normal;`），只保留 `clip-path` 扩散，让新主题在上层平滑展开。

## Goal

点击主题按钮时，新主题从按钮点击处向外扩散覆盖旧主题，动画约 700ms、丝滑不突兀；不支持该 API 的运行时（jsdom 测试环境、旧 WebView）与系统「减少动画」偏好下，主题仍立即切换，且不抛错、不留残余样式。

## Background

- 主题切换入口为 `frontend/src/components/common/app-titlebar.tsx` 的主题按钮，`onClick` 原本直接调用
  `useUIStore.toggleTheme()`。
- `frontend/src/app/theme-provider.tsx` 在 `useLayoutEffect` 中调用 `lib/theme.ts` 的 `applyTheme()`，
  负责切换 `document.documentElement` 的 `dark` 类与 `color-scheme`。
- Wails v3 的 WebView2 基于 Chromium，支持 View Transitions API。

## Scope

- 新增 `frontend/src/hooks/use-theme-transition.ts`（Hook）。
- `frontend/src/lib/theme.ts` 新增纯函数 `themeRevealRadius()`。
- 新增 `frontend/src/components/common/theme-reveal.css`（View Transition 伪元素样式与关键帧）。
- `app-titlebar.tsx` 主题按钮改用该 Hook。
- 单元测试：半径计算、无 API 时降级、有 API 时坐标与半径写入、减少动画偏好、打断与拒绝启动。
- 文档同步：`docs/product/APPEARANCE.md`、`docs/plans/CURRENT.md`。

## Out of Scope

- 不改主题色板、字体大小或语言切换动画。
- 不为路由切换或其他界面变化引入 View Transition。
- 不新增依赖。

## Existing Behavior

点击主题按钮，整个界面在同一帧内直接由亮变暗（或反向），没有任何过渡。

## Expected Behavior

- 支持的 WebView 中：点击位置为圆心，新主题以圆形扩散覆盖旧界面，约 700ms 完成，无淡入淡出。
- 键盘激活（`clientX/clientY` 为 0）时以按钮中心为圆心，不会从窗口左上角扫过。
- 不支持 API／系统要求减少动画时：同一帧内立即切换，行为与现状一致。

---

# Analysis

## Related Modules

- `frontend/src/components/common/app-titlebar.tsx`：主题按钮 `onClick`。
- `frontend/src/stores/ui-store.ts`：`theme` / `toggleTheme`（持久化）。
- `frontend/src/app/theme-provider.tsx`、`frontend/src/lib/theme.ts`：`applyTheme()`。
- `frontend/src/components/common/app-titlebar.test.tsx`：已有主题切换回归测试。

## Dependencies

N/A（使用浏览器原生 API，无新增依赖）。

## Risks

- **状态与快照不同步**：`startViewTransition` 的更新回调执行时 React 未必已提交渲染，
  新快照可能仍带旧图标。→ 在回调中先同步 `applyTheme(next)`，再用 `flushSync` 提交 Zustand 更新，
  保证 DOM 与 React 树在同一次回调内一致。
- **jsdom 无 `startViewTransition` / `matchMedia`**：→ 类型守卫 + 能力检测，缺失即走普通切换。
- **CSS 与 TS 各写一份半径**：半径通过 `:root` 自定义属性传入，动画时长与缓动只定义在 CSS。
- **连续点击**：新一次点击会打断上一段过渡并让它的 `finished` reject；若旧清理逻辑按属性名直接删除，
  会把新过渡的圆心抹掉。→ 用递增 id 只允许最新一次点击清理。

---

# Implementation Plan

1. 在 `lib/theme.ts` 实现 `themeRevealRadius()`；`hooks/use-theme-transition.ts` 实现
   `useThemeTransition()`。
2. `components/common/theme-reveal.css`：禁用默认淡入淡出，`::view-transition-new(root)` 上跑
   `clip-path` 关键帧（700ms，`cubic-bezier(0.4, 0, 0.2, 1)`），并在 `prefers-reduced-motion` 下关闭。
3. `app-titlebar.tsx`：`onClick` 改为传入事件对象调用 Hook。
4. 新增 `hooks/use-theme-transition.test.tsx` 与 `lib/theme.test.ts`。
5. 运行 `task verify`，回填 Task Record 与文档。

---

# Progress

- 确认主题状态由 Zustand 持有、`applyTheme()` 在 `ThemeProvider` 的 `useLayoutEffect` 里执行，
  因此过渡回调必须自行保证「快照拍摄时 DOM 已是新主题」。
- 实现 Hook 与 CSS，接入标题栏；补齐测试后发现连续点击会让旧过渡的清理抹掉新过渡的自定义属性，
  增加递增 id 保护与对应回归测试。
- 把半径函数从 Hook 移到 `lib/theme.ts`，避免单文件超过 200 行约定。

---

# Implementation Result

## Changes

- 新增 `useThemeTransition()`：读取当前主题算出下一主题，把点击坐标换算成圆心，写入
  `--theme-reveal-x / --theme-reveal-y / --theme-reveal-radius`，然后在
  `document.startViewTransition` 回调里 `applyTheme(next)` + `flushSync(toggleTheme)` 提交新主题。
- 无 `document.startViewTransition`、系统要求减少动画、或浏览器拒绝启动过渡时，直接提交主题切换。
- 键盘激活时用按钮中心作圆心；连续点击只由最新一次清理自定义属性，`finished` 结束或 reject 后清理。
- `theme-reveal.css` 关闭 `::view-transition-old/new(root)` 的默认动画与混合模式，
  只在 `::view-transition-new(root)` 上跑 700ms `clip-path` 扩散，`prefers-reduced-motion` 下不再播放。
- 标题栏主题按钮改为 `onClick={toggleThemeWithReveal}`，其余窗口按钮行为不变。

## Files Changed

- `frontend/src/hooks/use-theme-transition.ts`（新增）
- `frontend/src/components/common/theme-reveal.css`（新增）
- `frontend/src/components/common/app-titlebar.tsx`
- `frontend/src/lib/theme.ts`
- `frontend/src/hooks/use-theme-transition.test.tsx`（新增）
- `frontend/src/lib/theme.test.ts`（新增）
- `docs/product/APPEARANCE.md`
- `docs/plans/CURRENT.md`

## Technical Notes

- 更新回调里先 `applyTheme(next)` 再 `flushSync(toggleTheme)`：新快照在回调返回时拍摄，
  早于 React 常规渲染，两步保证快照既换了色板又换了按钮图标。
- 半径与圆心走 `:root` 自定义属性，`::view-transition-new(root)` 继承得到，动画时长与缓动只写一次。
- 过渡伪元素样式随标题栏模块导入，模块始终挂载，不需要额外的全局注册。

---

# Verification

## Tests

- [x] Unit Tests（`use-theme-transition.test.tsx` 7 项、`theme.test.ts` 1 项）
- [ ] Integration Tests：N/A
- [ ] E2E Tests：N/A（真实窗口动画观感需人工验收）

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
frontend vitest：134 文件 / 819 项通过
Windows production 构建（wails3 generate bindings + vite build + go build）：通过
```

人工验收未做：本任务没有启动真实 Wails 窗口观察扩散动画，观感与不同缩放比例下的表现留给用户实机确认。

---

# Documentation

- [x] PRD：无需改动（`docs/product/APPEARANCE.md` 已补充主题切换动画段落）
- [x] Architecture：N/A
- [x] Tech Stack：N/A（使用浏览器原生 API，无新增依赖）
- [x] Coding Rules：N/A
- [x] Testing：N/A
- [x] Build：N/A
- [x] ADR：N/A（不涉及长期架构取舍）
- [x] CURRENT：已加入 In Progress／本地状态看板

---

# Deviations

1. **半径函数位置**：原计划放在 Hook 内；实际移到 `lib/theme.ts` 并单独测试，
   原因是 Hook 测试文件加用例后超过 200 行约定，纯几何计算也更适合与主题工具同处。
2. **键盘激活**：需求只提到 `clientX/clientY`，实际补充「坐标为 0 时用按钮中心」，
   否则键盘操作会让圆形从窗口左上角扫过。
3. **连续点击保护**：需求未涉及，实现时发现旧过渡的清理会抹掉新过渡的圆心，
   增加递增 id 与回归测试。

---

# Git

## Commit

```text
<filled after commit>  feat(theme): 主题切换改为基于点击位置的圆形扩散过渡
```

---

# Result

标题栏亮暗模式按钮现在以点击位置为圆心做 700ms 圆形扩散切换主题，默认交叉淡入淡出被关闭；
不支持 View Transitions、系统要求减少动画或浏览器拒绝启动过渡时仍立即完成切换。
单元测试与 `task verify` 全量通过，真实窗口观感待人工验收。

---

# Completed

2026-10-28 10:15
