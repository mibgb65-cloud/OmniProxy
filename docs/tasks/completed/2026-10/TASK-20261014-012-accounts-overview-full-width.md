# TASK-20261014-012: 优化账号总览宽屏布局

## Status

Completed

## Created

2026-10-14 20:19（Asia/Singapore）

## Request

用户反馈账号页面右侧空出一列，要求优化页面，并直接在 main 分支上向前开发。

## Goal

账号总览充分利用主内容宽度，提供商卡片自动换行并等宽填满整行，窄窗口不横向溢出。

## Scope

- 账号总览的卡片网格布局与必要验证。
- 任务记录和 CURRENT 状态同步。

## Analysis

总览与详情共用 account-card-grid，列宽上限为 28rem，达到上限后剩余空间不分配给卡片，导致宽屏右侧留白。详情布局来自 TASK-20261013-013，解决拉宽窗口时卡片宽度跳变，应继续保留。仅让总览采用独立的弹性网格，避免改变详情行为。

## Implementation Plan

1. 为总览增加独立网格规则，以 24rem 为常规最小列宽，列数按容器可用宽度计算，剩余宽度由 1fr 均分；保持现有卡片和交互。
2. 用真实浏览器验证宽屏、窄窗口、字号缩放、侧栏收展及末行对齐，复跑账号交互测试。
3. 运行 task verify，在 main 对应原生窗口核对，审查差异、提交并归档记录。

## Risks

Low：总览新增列时卡片会重新均分宽度，换取整行利用可用空间；详情页保持原有连续列宽。

## Implementation Result

- 账号总览增加 `account-card-grid-overview`，使用 `repeat(auto-fit, minmax(min(100%, 24rem), 1fr))`，根据实际内容宽度排列并均分剩余空间。
- 提供商数量不足一行时自动合并空轨道；末行卡片与前面各列保持对齐和等宽。
- 窄于常规列宽时按容器 100% 收缩；rem 列宽随全局字号缩放。
- 只修改总览网格，保留详情页的连续列宽、现有卡片高度、账号操作和数据加载；没有新增依赖或 JS 尺寸监听。

### Files Changed

- `frontend/src/features/accounts/accounts-view.tsx`
- `frontend/src/features/accounts/account-card-grid.css`
- `docs/plans/CURRENT.md`
- 本任务记录。

## Verification

- PASS：账号总览与详情既有交互测试，2 个文件 / 16 项测试通过。
- PASS：本机已有 Chromium 读取实际 CSS 和总览组件使用的类名。16 个场景覆盖 1 / 2 / 10 / 11 张卡片、16 / 22.4px 根字号、64 / 224px 侧栏；逐像素扫描 960–3840px 窗口和 144–700px 窄容器，验证首行填满、卡片等宽、末行对齐和无横向溢出。另验证详情列宽单调不减且上限仍为 448px；合计 58,465 次宽度检查通过。
- PASS：旧布局在 2200px 容器显示 4 列且右侧剩余 372px，复现原问题。
- PASS：按可执行文件路径识别 main 原生开发进程 PID 38200，HMR 已应用新布局。960 / 1920 / 2560px 窗口分别显示 1 / 4 / 5 列，卡片头部操作保持同行；右侧仅保留正常页边距。截图人工核对通过，验收后恢复原窗口尺寸和位置，未修改业务数据。
- PASS：修改文件 UTF-8 无 BOM / LF 和 `git diff --check`。
- PASS：`VITEST_MAX_WORKERS=4 task verify BIN_DIR=E:/go/new/wails3-v/OmniProxy-main/tmp/accounts-overview/quality-bin` 退出码 0。格式、Go vet / staticcheck、ESLint、TypeScript、全部 Go 测试、70 个前端测试文件 / 386 项测试及 Windows 生产构建通过。既有 jsdom scrollTo 与构建体积提示不影响质量门通过。
- 尺寸验证脚本、原生截图与报告保存于忽略目录 `tmp/accounts-overview/`，完整质量日志位于 `tmp/accounts-overview-verify.log`。

## Documentation

已同步 CURRENT 的已完成状态。产品范围、接口和技术栈无需更新。

## Deviations

None.

## Git

`6070f83471d2991c0f4963b4104bccd290111af1 fix(account): 优化账号总览宽屏布局`

工作目录 E:/go/new/wails3-v/OmniProxy-main，分支 main。仅提交本任务代码与文档，既有未跟踪任务记录保持原样，未推送远端。

## Result

账号总览充分利用整行宽度，2560px 原生窗口的十家提供商排成五列两行，右侧仅保留正常页边距；窄容器、字号缩放、末行对齐及详情页原有列宽已验证。完整质量门通过，已提交并归档。

## Completed

2026-10-14 20:27（Asia/Singapore）
