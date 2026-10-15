# TASK-20261015-009: 禁止侧栏导航意外拖出链接预览

## Status

Completed

## Created

2026-10-15（Asia/Singapore）

## Request

用户反馈左侧导航选项在无意识拖动时出现带页面标题和 `wails.localhost` 地址的灰色预览，提供了「提供商」截图。

## Analysis

侧栏使用 TanStack Router 的真实链接，未关闭 HTML 链接的默认拖拽；按下并移动后，WebView2 会启动浏览器链接拖放，生成标题／URL 拖影。这不是应用提示组件。导航文字也未禁止选中，可能加重误触。

## Goal

侧栏作为桌面应用导航，拖动时不产生链接／选中文本拖影，点击和键盘导航、收起提示保持可用。

## Implementation Plan

1. 仅在侧栏关闭链接及品牌图片的浏览器拖拽、禁止文字选中，并拦截导航拖放事件；不阻止普通点击或按键。
2. 增加展开／收起下的回归验证，运行完整质量门；检查原生拖动效果。
3. 同步任务和状态文档，审阅差异、提交及归档。

## Implementation Result

- 侧栏链接显式设置 `draggable=false`，并取消 `dragstart` 默认行为，覆盖从链接文字或内部图标触发拖放的情况。
- 导航链接使用 `select-none`，避免无意识拖动选中文字；品牌图片也关闭默认图片拖放。没有对正文、网关信息或整个应用设置全局禁止选择／拖放。
- 保留真实链接和原有 Tooltip、折叠、点击、键盘行为；未使用 `mousedown`／`pointerdown` 拦截，避免影响正常激活。

## Verification

- 针对性测试 PASS：`app-sidebar.test.tsx` 共 5 项；新增展开／收起两个回归场景，验证 11 个链接均取消拖拽、品牌图片不可拖动，以及鼠标点击进入提供商页、聚焦后 Enter 进入账号页仍有效。原有语言、收展和无障碍标签测试保持通过。
- 完整质量门 PASS：`task verify BIN_DIR=tmp/sidebar-link-drag-20261015/verify-bin`，格式、Go vet／staticcheck／ESLint、类型检查、Go 全量测试、79 个前端测试文件／433 项测试及 Windows 生产构建全部通过。无新增依赖、无锁文件变化；既有工具探测、jsdom scrollTo 和 Vite 大包提示不阻断。
- 原生验证 PASS：独立 test 数据、配置和 WebView 存储，960×600 窗口，实际按住鼠标拖动「提供商」链接，覆盖展开／收起侧栏；已查看拖动期间的屏幕截图，未出现用户截图中的标题／URL 灰色拖影，随后正常展开并切换到账号页。测试窗口已正常退出，没有修改用户配置或发送模型请求。证据位于忽略目录 `tmp/sidebar-link-drag-20261015/`。
- `git diff --check`：PASS。

## Documentation

只需更新 CURRENT 和任务记录；无产品能力、架构、配置或接口变更。

## Git

`8461b1a588a416c07332862c51e5ca36b1f8bf4f fix(frontend): 禁止侧栏导航意外拖出链接预览`

随后补交任务归档和 CURRENT 完成状态。原有四份无关未跟踪任务文档保持原样，未纳入提交。

## Result

已修复侧栏误拖触发 WebView2 原生链接拖放预览的问题，正常导航保持可用；完整质量门与原生拖动验证通过。

## Completed

2026-10-15 18:05（Asia/Singapore）
