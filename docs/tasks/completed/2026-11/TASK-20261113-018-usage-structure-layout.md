# TASK-20261113-018: 用量页结构卡片改为两栏并拉齐高度

## Status

Completed

## Created

2026-11-13 15:05（Asia/Singapore）

## Request

> 这个的效果感觉有点一般啊，要不然你做个滚动条在中间这个，或者左右两边再上下拉高一点？

截图：1440 级别宽度下，「Token 构成」「模型用量占比」「提供商用量占比」三列高度差异很大，
模型卡约 700px，构成卡与提供商卡下方各留出大片空白。

## Goal

- 结构区不再出现三列高低不齐的空白：构成与提供商在左列上下堆叠，模型占比在右列跨两行。
- 模型列表在高度富余时均匀铺开，不再只挤在卡片顶部。
- 保持现有数据口径、卡片内容与可访问名称；小屏单列堆叠顺序与宽屏阅读顺序一致（构成 → 提供商 → 模型）。

## Background

- 上一版使用 `xl:grid-cols-3 items-start`：三张卡各自按内容高度排列，
  模型卡有 6 个模型 + 「其他」，提供商通常只有 2–3 个，导致左右两列大量留白。
- 模型卡是信息量最大的一张，压缩成滚动条会隐藏一半 Top 6 模型；
  把提供商移到构成卡下方后，两列总高度接近，不需要隐藏内容。

## Scope

- `usage-view.tsx` 的结构区网格与卡片定位。
- `usage-charts.tsx` 的卡片内容改为可纵向伸展。
- `usage-composition.tsx` 的环形图区域在卡片被拉高时垂直居中。
- `usage-share.tsx` 的模型／提供商卡支持传入布局类名，模型列表在富余高度内均匀分布。

## Out of Scope

- 不改任何统计口径、接口、文案与数据卡片数量。
- 不改「模型每日用量」「时段」「质量」「账号排行」「明细表」的布局。
- 不引入图表库或滚动条方案。

## Existing Behavior

三列等高网格，内容高度决定卡片高度，`items-start` 让左右两列在模型卡下方留白。

## Expected Behavior

- 宽屏：左列为「Token 构成」+「提供商用量占比」，右列为跨两行的「模型用量占比」，
  三张卡在视觉上拼成一个整齐的矩形。
- 窄屏：按构成 → 提供商 → 模型顺序单列堆叠。DOM 顺序与宽屏视觉顺序一致，
  读屏与键盘顺序不会因为网格定位而跳跃。
- 模型列表条目在卡片被拉高时均匀分布，不出现顶部拥挤、底部空白。

---

# Analysis

## Root Cause

三列网格把「条目数差异很大」的三张卡当作等宽独立卡片排布，
行高只由最高的模型卡决定，另外两张卡没有可伸展的内容。

## Related Modules

- `frontend/src/features/activity/usage-view.tsx`
- `frontend/src/features/activity/usage-charts.tsx`
- `frontend/src/features/activity/usage-composition.tsx`
- `frontend/src/features/activity/usage-share.tsx`

## Risks

- `row-span` 与 `items-stretch` 组合下，模型卡变高时左列两张卡可能被拉伸；
  构成卡已改为居中，提供商卡列表仍保持顶部对齐并限高滚动。
- 小屏顺序必须保持不变，避免读屏与键盘顺序变化。

---

# Implementation Plan

1. 结构区改为 `xl:grid-cols-2`，模型卡 `xl:col-start-2 xl:row-span-2`，
   提供商卡 `xl:col-start-1 xl:row-start-2`；DOM 顺序改为构成 → 提供商 → 模型，
   用显式的行列起点定位，不改宽屏视觉顺序。
2. 卡片内容容器允许纵向伸展；构成卡环形区在富余高度中居中。
3. 模型列表使用 `flex-1 justify-between`，让条目在拉高时均匀分布。
4. 跑相关测试、lint、typecheck 与 `task verify`，用临时预览页核对 1100／1280／1440／2073 宽度后删除。

---

# Progress

- 2026-11-13 15:05：创建记录。
- 2026-11-13 15:2x：改为两栏网格、卡片内容可伸展、模型列表均匀分布；
  临时预览页核对 1100／1280／1440／2073 宽度，深色与浅色各一份。
- 2026-11-13 15:3x：按「DOM 顺序对齐宽屏视觉顺序」把提供商卡挪到模型卡之前，
  格式化、lint、typecheck 与 activity 相关测试通过，删除临时预览页。

---

# Implementation Result

- `usage-view.tsx`：结构区改为 `grid min-w-0 gap-4 xl:grid-cols-2`（去掉原来的
  `items-start` 与三列），构成卡留在第 1 行第 1 列，提供商卡
  `xl:col-start-1 xl:row-start-2`，模型卡 `xl:col-start-2 xl:row-span-2 xl:row-start-1`。
  DOM 顺序改为构成 → 提供商 → 模型，使窄屏堆叠顺序与宽屏从左到右、从上到下的阅读顺序一致；
  宽屏位置由显式行列起点决定，与 DOM 顺序解耦。
- `usage-charts.tsx`：`UsageCard` 的 `CardContent` 增加
  `flex min-h-0 min-w-0 flex-1 flex-col`，让每张卡的内容区都能吃掉网格拉伸出的多余高度；
  这是「拉高左右两边」而不引入滚动条的关键。
- `usage-composition.tsx`：环形图与图例的容器加 `flex-1`，卡片被拉高时环形区垂直居中，
  不再顶在卡片上沿。
- `usage-share.tsx`：`ModelShareCard`／`ProviderShareCard` 接受可选 `className` 并透传给
  `UsageCard`；模型列表由 `space-y-3` 改为 `flex flex-1 flex-col justify-between gap-3`，
  富余高度平均分给条目之间而不是堆在底部。提供商列表仍是 `max-h-96` 卡片内滚动，
  因为提供商数量理论上是开放的，模型列表固定为 Top 6 + 其他。
- 未改动任何统计口径、接口、文案与卡片数量；「模型每日用量」「时段」「质量」
  「账号排行」「按提供商与模型」布局保持原样。
- 选型说明：用户给的两个方向是「中间卡加滚动条」或「两边卡片拉高」。前者会隐藏一半
  Top 6 模型且引入第二处滚动条，后者用两栏重排就能吃掉全部空白，因此采用后者，
  滚动条只保留在提供商卡这一处既有实现上。

---

# Verification

## Tests

- [x] Unit Tests

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

- `pnpm exec vitest run src/features/activity`：9 个文件 / 45 个用例通过。
- 视觉核对（临时 `preview.html` + 固定数据，浅色与深色各一份，核对后已删除）：
  - 1100px：单列，构成 → 提供商 → 模型，卡片无纵向空白。
  - 1280px：两栏成立，左列两张卡总高与右列模型卡基本齐平，模型条目均匀分布。
  - 1440px：同上，三张卡拼成整齐矩形。
  - 2073px（贴近用户实际视口）：深浅主题都对齐，没有横向溢出，也没有把条目挤到卡片顶部。
- `task verify`（2026-11-13 15:17）：`VERIFY_EXIT=0`，prettier `All matched files use Prettier code style!`、
  `go vet`／`staticcheck`／`eslint .` 无输出、Go 全包测试 `ok`、
  前端 `Test Files 151 passed (151)`／`Tests 924 passed (924)`、`tsc && vite build` 与
  `go build -tags production` 成功（仅保留既有的 chunk >500kB 提示）。

---

# Git

- Commit: `d8d673d9`（feat(activity): 用量页结构卡片改为两栏并拉齐高度）
