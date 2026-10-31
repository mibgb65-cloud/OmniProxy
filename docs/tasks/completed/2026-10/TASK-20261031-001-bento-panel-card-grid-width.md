# TASK-20261031-001: Bento 展开面板账号卡片宽度失控

## Status

Completed

## Created

2026-10-31 11:49

## Request

> 这个卡片的全屏后的效果怎么这么奇怪，和我之前设计的不一样啊，你看看是什么情况，我要的是每行可以排列多个，并且卡片宽度不能超过一个值

用户提供 2560×1392 全屏截图：账号页原位展开的「OpenAI 账号」面板里，第一行仅 2 张卡片、每张约 1075px 宽，
第二行 1 张同样被拉宽，卡片内部（额度进度条、5h／一周数值、Token 用量三列）全部被摊开。

## Goal

Bento 展开面板与独立详情页使用同一套卡片网格规则：列宽随可用宽度连续增长并封顶，
宽屏只增加列数，不再把单张卡片拉成半个屏幕宽。

## Background

TASK-20261013-013 为账号详情页确立了 `repeat(auto-fill, min(100%, clamp(20rem, calc((100% - gap) / 2), 28rem)))`：
列宽独立于列数连续增长，新增列只改变排列，宽屏不再压窄或拉宽已有卡片。
TASK-20261028-005／007 引入 Bento 原位展开后，`bento-accounts.css` 又给面板内的网格写了固定两列，
把上面这套封顶逻辑整体覆盖掉了。

## Scope

- 让 `.bento-panel .account-card-grid` 沿用封顶列宽，只覆盖间距。
- 在 `account-card-grid.css` 里把最小／最大列宽抽成变量，避免同一表达式在两处复制。
- 增加针对这两个 CSS 文件的回归测试。

## Out of Scope

- 不改动账号概览页的提供商卡片网格（`.account-card-grid-overview`，本任务不动）。
- 不改卡片内部信息结构、额度与 Token 分区。
- 不重新打包安装器，不安装或替换用户本地已安装版本。

## Existing Behavior

展开面板的账号卡片被强制两列等分：窗口越宽单卡越宽，2560px 全屏时单卡约 1075px。

## Expected Behavior

展开面板同样按 28rem 封顶自动增列：2560px 全屏时单卡 448px、每行多张；窗口变窄时减少列数，
卡片宽度始终落在 20–28rem 区间内（容器更窄时退回单列）。

---

# Analysis

## Related Modules

- `frontend/src/features/accounts/account-card-grid.css`
- `frontend/src/features/accounts/bento-accounts.css`
- `frontend/src/features/accounts/provider-accounts-view.tsx`（`.account-card-grid` 使用方，无需改）
- `frontend/src/features/accounts/accounts-view.tsx`（Bento 容器，无需改）

## Dependencies

没有：N/A

## Risks

- 面板列数变多后，卡片在极窄面板里可能过窄：由既有 `@container (max-width: 44rem)` 单列规则兜住。
- 宽屏下最后一列右侧会留白：与详情页既定观感一致（TASK-20261014-012 只让概览页吃掉剩余宽度）。

---

# Implementation Plan

1. 在 `account-card-grid.css` 抽出 `--account-card-min-width` / `--account-card-max-width`，base 规则改用变量。
2. 在 `bento-accounts.css` 删除 `.bento-panel .account-card-grid` 的固定 `repeat(2, minmax(0, 1fr))`，
   只保留 `--account-card-gap: 1rem`。
3. 新增 `account-card-grid.test.ts`：断言 base 规则是 auto-fill + clamp 封顶，面板覆盖不写固定列数。
4. 前端 format／lint／typecheck／test 及 `task verify`。

---

# Progress

- 定位根因：`bento-accounts.css` 第 107–110 行的固定两列覆盖了 `account-card-grid.css` 的封顶列宽。
- 已按计划改 CSS 并补回归测试。

---

# Implementation Result

## Root Cause

`bento-accounts.css` 里 `.bento-panel .account-card-grid { grid-template-columns: repeat(2, minmax(0, 1fr)) }`
（TASK-20261028-007 引入）用固定两列等分覆盖了 `account-card-grid.css` 的封顶列宽：

- 第 5 行的 `repeat(auto-fill, min(100%, clamp(20rem, calc((100% - gap) / 2), 28rem)))` 被整体替换，`28rem` 上限失效；
- `1fr` 让两列均分可用宽度，2560×1392 全屏时面板内容宽 ≈2165px，单卡被拉到 ≈1075px；
- 卡片内部的三列 Token、额度进度条、5h／一周数值按比例摊开，看起来「被拉平」；
- 同一份文档里的 `@container (max-width: 44rem)` 又把面板网格覆盖成 `1fr`，同样是「无限宽」而不是封顶。

## Changes

- `account-card-grid.css`：把最小／最大列宽抽成 `--account-card-min-width`（20rem）与
  `--account-card-max-width`（28rem），base 规则改成引用变量，表达式只保留一份。
- `bento-accounts.css`：删除面板网格的固定两列，只保留 `--account-card-gap: 1rem`；
  新增 `@container (max-width: 41rem)` 兜底——面板窄到只放得下一列时，这一列长到 28rem 封顶宽度
  （`min(100%, var(--account-card-max-width))`），不会卡在 20rem、右侧留一大片空白；
  44rem 断点只保留 `.bento-accounts`，不再用 `1fr` 覆盖账号卡片网格。
- 新增 `account-card-grid.test.ts`（3 项）：断言 base 规则是 auto-fill + clamp 封顶、
  面板覆盖不写固定列数、面板里任何 `grid-template-columns` 覆盖都必须带上限。
- 新增 `src/test/node-fs.d.ts`：只为读源码文本声明 `readFileSync`，不引入 `@types/node`。

## Files Changed

- `frontend/src/features/accounts/account-card-grid.css`
- `frontend/src/features/accounts/bento-accounts.css`
- `frontend/src/features/accounts/account-card-grid.test.ts`（新增）
- `frontend/src/test/node-fs.d.ts`（新增）

## Technical Notes

隔离 Chromium（真实两份 CSS、`.accounts-page` 同款 `container-type: inline-size`）实测，
数值为容器宽度（内容盒 = 宽度 − 16px 内边距）：

```text
2165px → 4 列 × 448px   修复前：2 列 × ≈1075px
1400px → 3 列 × 448px
1000px → 2 列 × 448px
 760px → 2 列 × 364px
 700px → 2 列 × 334px
 660px → 1 列 × 448px（只放得下一列时用满上限）
 400px → 1 列 × 384px
```

单卡宽度在任何宽度下都不超过 28rem（448px），宽屏只增加列数。

---

# Verification

## Tests

- [x] Unit Tests：新增 `account-card-grid.test.ts`（3 项 CSS 规则断言）
- [ ] Integration Tests：N/A
- [ ] E2E Tests：N/A（原生 WebView2 观感待用户实机确认）

## Quality Checks

- [x] Format（`prettier --check`）
- [x] Lint（`eslint`，仅 `reset-credit-list.tsx` 两条既有 fast-refresh 警告）
- [x] Type Check（`tsc --noEmit`）
- [x] Test（146 文件／890 项前端测试，Go 测试通过）
- [x] Build（Windows amd64 production）
- [x] Verify

## Verification Result

```text
task verify
PASS（exit 0）
```

前端 146 文件／890 项测试通过（新增 3 项），`go vet`／`staticcheck`／Go 测试与
`windows:build:native` 通过。布局实测用隔离 Chromium 加载真实两份 CSS 与同样式的
`container-type: inline-size` 容器，见 Implementation Result 的 Technical Notes。

---

# Documentation

- [x] CURRENT（In Progress / Recently Completed）

其余文档无需更新：无产品需求、架构、技术栈或 ADR 变化。

---

# Deviations

计划里只打算「删掉固定列数、只留间距覆盖」。实现时补了一条
`@container (max-width: 41rem)` 兜底（两列至少需要 `2 × 20rem + 间距`）：面板窄到只放得下一列时，
`clamp` 会停在 20rem 最小列宽、右侧留出大片空白（实测 620px 容器 → 320px 卡片 + 300px 空白）。
兜底让这一列长到 28rem 封顶宽度，上限仍然生效，断言测试里也锁住了「任何列宽覆盖都必须带上限」。
另外删掉了 44rem 断点里对账号卡片网格的 `1fr` 覆盖，它同样是无界拉伸。

---

# Git

## Commit

```text
6c934061 fix(accounts): 展开面板账号卡片恢复列宽封顶并按可用宽度增列
```

包含两份 CSS、新增断言测试与 `node-fs.d.ts`、本任务记录及 CURRENT 回填。未推送、未打标签。

---

# Result

展开面板的账号卡片恢复为「列宽封顶 + 按可用宽度自动增列」：2560×1392 全屏时单卡 448px、
每行 4 张（同一容器宽度修复前是 2 张 × ≈1075px），1400px 3 张、1000px 2 张，
单卡宽度在任何窗口宽度下都不超过 28rem。改动只在两份 CSS 加 3 项断言测试，
组件与数据链路未变；运行中的安装版需要重新构建后才会看到效果。

---

# Completed

2026-10-31 12:14
