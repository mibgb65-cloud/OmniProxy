# TASK-20261018-001: 优化时间范围选中卡

## Status

Completed

## Created

2026-10-18

## Request

用户反馈「今日／近 7 天／近 30 天」没有显示当前选中的时间，要求优化选中卡。

## Goal

让当前时间范围始终清晰可辨，保持现有界面风格，并兼顾深浅主题与键盘操作。

## Scope

检查并调整仪表盘、网关总览的同类时间范围选择器及必要的验证。

## Analysis

- 网关选中按钮使用 `secondary`，其背景与容器 `muted` 在深浅主题均为相同颜色，无法辨认当前时间范围。
- 仪表盘使用背景色覆盖按钮，两个入口选中态不一致；现有 `aria-pressed` 和统计切换逻辑正常。
- 对比保留浅色卡片与采用主题主色卡片两种方案，选择现有 `default` 按钮配色：无需新增颜色、样式系统或依赖，深浅主题均有足够反差；未选中项使用次要文字，卡片保留间距和阴影。

## Implementation Plan

1. 定位选中态不明显的原因，复用现有主题令牌实现清晰的选中卡。
2. 验证时间切换、深浅主题与键盘焦点，执行项目质量门。
3. 更新任务记录和当前看板，检查并提交本任务变更。

## Implementation Result

- 网关与仪表盘的时间选择按钮统一使用现有主题主色显示当前项，深色主题为浅底深字、浅色主题为深底浅字。
- 增加卡片间距和选中阴影，未选中项使用次要文字；网关补齐容器换行和最大宽度约束。
- 保留已有原生按钮、`aria-pressed`、焦点样式和统计查询逻辑；无新增依赖。

## Verification

- 定向测试：`pnpm exec vitest run src/features/dashboard/dashboard-view.test.tsx src/features/gateway/gateway-overview.test.tsx`，2 个文件共 13 项 PASS。
- 项目路径对应的正常开发原生窗口：网关／仪表盘 × 深浅主题 × 今日／7 天／30 天共 12 项切换检查，每次恰有一个按钮选中；960×600 两页显示和可访问焦点检查通过，共 14 项。已查看实际选中卡截图，主题、窗口尺寸和网关原时段在检查后恢复。
- 原生检查使用 UI Automation 切换与聚焦。全局物理 Tab 注入未进入应用，未将其计为键盘端到端通过；保留既有原生按钮键盘行为和焦点样式。
- 完整质量门 PASS（退出码 0）：`task --force verify BIN_DIR=tmp/time-range-selection-20261018/verify-bin`，格式、Go vet／staticcheck／ESLint、类型、全部 Go 测试、100 个文件 559 项前端测试、前端生产包和 Windows 生产构建均通过。已有 jsdom scrollTo 提示和 bundle 大小提醒未影响结果。
- 日志、原生检查脚本／结果与选中卡截图位于 `tmp/time-range-selection-20261018/`；构建使用独立输出目录。`git diff --check` 通过，无关用户任务文件保持原样。

## Documentation

已更新 CURRENT；无产品、架构或接口变更。

## Deviations

无。

## Git

实现提交：`c2b317e2257fc0e50687b2df2befdc0b8b8b26ee` — `fix(frontend): 增强时间范围选择器的选中态`。任务记录随后归档，未推送。

## Result

两个时间选择器的当前范围已清晰显示，正常开发窗口已生效，深浅主题、窄窗口和完整质量门验证通过。

## Completed

2026-10-18（Asia/Singapore）
