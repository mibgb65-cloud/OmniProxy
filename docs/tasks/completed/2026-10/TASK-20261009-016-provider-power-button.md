# TASK-20261009-016: 提供商电源启停按钮

## Status

Completed

## Created

2026-10-09

## Request

用户反馈提供商页面的内置提供商滑动开关不好看，要求参考图二的账号电源按钮修改。

## Goal

提供商启停使用与账号一致的紧凑电源按钮，清晰显示启用／停用状态，并保留原有操作行为。

## Analysis

- ProviderCard 由内置与自定义提供商共用，当前通过 ProviderSwitch 呈现蓝色滑动开关。
- 图二已有 AccountPowerButton 实现：32 像素圆角按钮、绿色 Power／灰色 PowerOff、Tooltip、aria-pressed 和忙碌禁用。
- ProviderSwitch 也用于模型路由；直接修改它会影响当前请求以外的页面。采用公共电源按钮呈现组件，由账号与提供商各自提供翻译和操作回调，保留路由开关。
- 工作区已有大量未提交实现，包括账号电源按钮与 Tooltip；保留已有改动，避免混合提交依赖先前任务的代码。

## Implementation Plan

1. 复用账号现有电源按钮呈现，接入共享的提供商卡片，并补齐中英文操作提示。
2. 适配现有提供商页面测试，验证启用／停用、键盘提示、请求期间禁止重复操作及服务调用。
3. 运行 task verify，在项目原生开发窗口核对视觉效果，更新任务记录与 CURRENT。

## Implementation Result

- 从现有账号按钮提取公共 PowerButton 呈现组件，保留原有 32×32 像素、圆角、绿色 Power／灰色 PowerOff、深浅主题、Tooltip、aria-pressed、焦点和禁用状态。
- ProviderPowerButton 与 AccountPowerButton 分别提供业务翻译和回调；内置／自定义 ProviderCard 使用电源按钮，模型路由仍使用原有 ProviderSwitch。没有新增依赖或修改后端启停行为。
- 新增中英文启用／停用提供商提示；适配原有卡片页面测试，扩展停用后重新启用及异步请求期间禁止重复操作的行为验证。
- 同步 PRD 和 CURRENT；开发窗口通过现有 Vite 热更新显示新按钮，保留原亮色主题、账号数据、全部提供商状态与网关运行状态。

主要文件：`components/common/power-button.tsx`、`features/providers/provider-power-button.tsx`、`features/providers/provider-card.tsx`、`features/providers/providers-view.test.tsx`、`features/accounts/account-power-button.tsx`、中英文语言文件。

## Verification

- `task verify BIN_DIR=C:/Users/mimanchi/AppData/Local/Temp/omniproxy-provider-power-qa-20261009/build` 通过：格式、Go vet、staticcheck、ESLint、TypeScript、全部 Go 测试、51 个前端文件／258 项测试、绑定生成、前端生产构建及 Windows 生产构建。
- 现有账号电源按钮测试通过公共组件验证键盘空格停用、键盘聚焦提示、停用状态点击启用和忙碌禁用；提供商页面测试验证状态持久化与原有服务参数，并确认请求期间不会重复调用。
- 按进程路径识别实际项目开发窗口（PID 57484），UIAutomation 确认 9 个提供商控件均为 32×32 像素按钮，TogglePattern 状态均为 On；没有为了验证视觉而修改真实提供商启停状态。
- 实际窗口亮色／暗色截图人工核对通过，按钮样式与参考账号一致，无卡片溢出；已恢复原亮色主题。截图与原生验证 JSON 位于临时目录 `omniproxy-provider-power-qa-20261009`。亮色截图采用 PrintWindow，避免前台遮挡影响截图。
- `git diff --check` 通过。质量构建输出到临时目录，避免覆盖运行中的开发二进制；Vite 9245 与开发 Control API 3891 保持监听。
- 生产构建仍有既有资源块大于 500 kB 的提示，构建成功；本任务没有扩展资源拆分范围。

## Deviations

None.

## Git

未提交。公共按钮依赖先前任务尚未提交的 Tooltip／账号按钮实现，提供商卡片、语言文件及文档也已有其他任务的未提交修改；按 Git 安全规则保留原有改动，避免混合提交或产生依赖未提交实现的孤立提交。

## Result

提供商卡片已按图二改为与账号一致的电源启停按钮；质量检查和原生窗口深浅主题验证通过，当前开发窗口已显示更新。

## Completed

2026-10-09
