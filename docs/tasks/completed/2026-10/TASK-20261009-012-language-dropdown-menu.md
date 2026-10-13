# TASK-20261009-012: 语言按钮展开选择菜单

## Status

Completed

## Created

2026-10-09

## Request

语言的具体切换值改为点击语言按钮后悬浮出现再选择，外部仅保留语言按钮。

## Goal

标题栏只显示语言按钮，点击后弹出中文／English 菜单，选中后关闭并保存偏好。

## Analysis

现有组件直接展示两个分段选项。仓库已有 radix-ui，可用 DropdownMenu 实现浮层、单选标记、键盘操作、边缘避让和焦点恢复，无需新增依赖。菜单使用 Portal 避免被标题栏或窗口布局裁切，选择值经过已有 isLanguage 校验。

## Implementation Plan

1. 替换为语言按钮与主题一致的单选浮层菜单，保留位置与持久化。
2. 迁移语言及标题栏验证，检查选择、收起侧栏、键盘／取消和窗口拖动。
3. 运行质量门禁及布局验证，更新文档与任务记录。

## Implementation Result

标题栏仅显示语言图标与“语言”按钮，点击后通过 Radix DropdownMenu Portal 展示中文／English 单选浮层，勾选标记当前语言。菜单沿用现有下拉主题、圆角、阴影和减少动画规则；选择值通过 isLanguage 校验后更新既有持久化偏好，选中后自动关闭并恢复焦点，支持键盘、Escape 和点击外部取消。

主要变更：`language-switcher.tsx`、既有语言／标题栏测试，以及外观文档。无新依赖或后端 API。

## Verification

- 语言及标题栏的 11 项针对性测试通过：默认隐藏选项、选中标记、选择与偏好保存、菜单关闭和焦点恢复、侧栏收起、键盘取消及窗口操作。
- 完整 `task verify BIN_DIR=<临时 QA 目录>/quality-bin` 通过：格式、Go 静态检查、ESLint、TypeScript、Go 测试、47 个前端文件共 241 项测试和 Windows 生产构建。保留已有构建包体与 Task 环境探测提示。
- 真实浏览器验证 12 种中英文／亮暗主题／80、100、140 字号组合：浮层位于窗口内且无溢出，勾选状态正确；方向键及 Enter 选择、Escape 恢复焦点、外部点击取消、持久化均通过，无页面错误。
- 原生 Wails UIAutomation 通过语言按钮键盘打开菜单，确认选项可见并截图，随后 Escape 关闭，未修改用户语言。开发窗口已通过 HMR 更新。截图与验证 JSON 位于临时目录 `omniproxy-language-menu-qa-20261009`。
- `git diff --check` 通过。

## Documentation

已同步 `docs/product/APPEARANCE.md` 和 `docs/plans/CURRENT.md`。

## Deviations

None.

## Git

保留前序未提交改动，不混合提交。

## Result

已实现点击语言按钮后再选择的交互，标题栏不再常驻语言值，验证通过。

## Completed

2026-10-09
