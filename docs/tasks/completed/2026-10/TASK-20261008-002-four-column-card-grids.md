# TASK-20261008-002: 大屏账号与提供商卡片最多四列

## Status

Completed

## Created

2026-10-08 10:17（Asia/Singapore）

## Request

这个账号和提供商一行在大屏的时候可以最多显示4个卡片比较好一些。

## Goal

账号与提供商相关卡片在宽屏每行最多显示四张，较窄窗口继续自适应排列。

## Analysis

当前内置／自定义提供商最多三列；账号总览四列断点为 3000px，提供商账号详情最多三列。卡片已有窄宽度截断、换行和操作区布局，采用现有 Tailwind 断点进行最小调整：1280px 起三列，1536px 起四列，最大列数固定为四；保留各页面原有的两列断点。

## Implementation Plan

1. 为内置提供商、自定义提供商、账号总览及提供商账号详情使用 `xl:grid-cols-3 2xl:grid-cols-4`。
2. 移除账号网格原有 1800px／3000px 的特殊断点，保留小窗口布局和卡片／列表切换。
3. 检查实际布局、现有组件测试及 `task verify`，同步任务记录与看板，恢复开发环境。

## Implementation Result

- 内置提供商、自定义提供商、账号总览和提供商账号详情均在 1280px 起使用三列、1536px 起使用四列；更宽窗口继续保持最多四列。
- 移除账号原有的 1800px／3000px 特殊断点，各页面原有单列、两列布局和账号列表视图保持可用。
- 仅修改四处网格类名：`provider-catalog.tsx`、`provider-list.tsx`、`accounts-view.tsx`、`provider-accounts-view.tsx`，未增加依赖。
- 检查后恢复 `task dev`，后台任务 PID 为 28384，桌面进程 PID 为 13324。

## Verification

- 隔离浏览器使用模拟数据渲染真实组件，验证四类网格在 960、1200、1280、1535、1536、1920、2560、3840px 的布局，共 32 组检查通过，并覆盖亮暗主题。
- 每组检查均确认实际 CSS 网格列数与首行卡片数量一致，卡片未超出网格，文档无横向溢出；1536px 起最多四列。
- 账号详情切换列表视图正常，八行模拟账号完整展示；2560px 提供商布局截图人工核对通过。
- `task verify` PASS：格式、lint、类型检查、全部 Go 测试、25 个前端测试文件共 139 项测试和 Windows 生产构建通过。
- 临时检查脚本、结果及截图位于 `C:/Users/mimanchi/AppData/Local/Temp/omniproxy-card-grid-qa-de617b12/`。
- 开发环境恢复后检查 Vite 首页、Control API 健康接口及桌面窗口，Git 变更核对和 `git diff --check` 通过。

## Documentation

已将本任务加入 `docs/plans/CURRENT.md`。产品、架构和构建文档无需更新。

## Deviations

None. 临时浏览器检查适配 Vite 优化依赖的默认导出和版本 URL，未因此修改产品代码。

## Remaining Risks

检查覆盖 960px–3840px 窗口；响应式断点按 CSS 视口宽度判断，系统显示缩放会影响实际可用视口。

## Git

`0835aa9f452aa35d3f765ae5978766b35a3c246f fix(frontend): 账号与提供商大屏卡片最多显示四列`

仅提交本任务文件和看板新增条目，保留其他任务的未提交修改。

## Result

账号和提供商相关卡片在大屏每行最多四张，较窄窗口自适应排列，完整检查通过，开发环境已恢复。

## Completed

2026-10-08 10:27（Asia/Singapore）
