# TASK-20261016-002: 小米 MiMo 账号设置入口

## Status

Completed

## Created

2026-10-16（Asia/Singapore）

## Request

给小米 MiMo 账号页增加和 OpenAI 账号页一样的设置按钮。用户已确认：先加入口和空设置面板。

## Goal

在 MiMo 账号页头部、展示方式切换左侧加入同款设置按钮；点击在入口旁显示「小米 MiMo 账号设置」浮层和「暂无可配置项」。

## Analysis

OpenAI 使用 Radix 非模态 Popover，提供图标按钮／提示、标题／关闭、边缘避让与滚动，支持 Escape、外部点击和焦点恢复。将这段展示外壳提取为账号设置公共组件，OpenAI 保留原有内容，MiMo 仅传入空状态。此变更不需要后端设置、持久化或新依赖。工作区已有五份未跟踪任务记录，保持原样。

## Implementation Plan

1. 提取并复用现有设置浮层外壳，在 MiMo 页头接入按钮和空状态。
2. 补充中英文文案与 MiMo 使用文档。
3. 运行 `task verify`，在正常桌面开发窗口核对两种主题、窄窗口及开关／关闭交互，审查差异并归档。

## Implementation Result

- 在 MiMo 账号详情页头部、展示方式切换左侧加入与 OpenAI 一致的设置图标按钮和提示。
- 提取 `AccountSettingsPopover` 展示外壳供 OpenAI／MiMo 共用，保持 420px 宽、局部非模态浮层、边缘避让、关闭按钮及滚动行为，避免复制整段结构。
- MiMo 面板显示「小米 MiMo 账号设置」和「暂无可配置项」；补齐中英文文案，不新增后端接口、设置项、持久化或依赖。
- 正常桌面开发窗口通过热更新显示最新内容，验证后恢复原主题／窗口尺寸，并停留在 MiMo 设置面板。

## Verification

- PASS：`VITEST_MAX_WORKERS=4 task verify BIN_DIR=E:/go/new/wails3-v/OmniProxy/tmp/mimo-account-settings-20261016/quality-bin`，退出码 0；格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、80 个前端测试文件／439 项测试与 Windows 生产构建通过。验证产物输出到忽略目录，未替换运行中的开发程序。
- PASS：现有 OpenAI 设置测试继续覆盖局部浮层开关／关闭、焦点、原有设置保存及互锁；此处仅增加展示入口和空内容，未新增重复测试。
- PASS：按路径识别正常桌面开发应用 PID 8336，MiMo 入口位于展示切换左侧；再次点击、关闭按钮、Escape 和外部点击均关闭浮层，关闭按钮及 Escape 恢复入口焦点。
- PASS：MiMo 面板深浅主题及 960×600 最小窗口截图检查，标题和空状态可见，浮层未超出窗口；OpenAI 原有 Codex WebSocket／自动激活内容正常显示，未切换真实账号设置。
- PASS：`git diff --check`；已审查公共外壳提取及本次代码／文档差异。

原生校验脚本、截图、报告和质量日志位于忽略目录 `tmp/mimo-account-settings-20261016/`。临时脚本已修正既有辅助函数对文本控件及 Expand／Collapse 的处理，并处理离屏链接与窗口聚焦后完成验收；产品代码没有因此调整。既有 jsdom scrollTo、工具探测和打包体积提示未导致质量门失败。

## Documentation

已更新 MIMO 使用说明及 CURRENT 已完成概览。

## Git

`2b94ac028dce74e317a4da8678778848f52cf9d7 feat(accounts): 增加小米 MiMo 账号设置入口和空面板`

只提交本任务文件，保留五份既有未跟踪记录。任务归档与 CURRENT 更新另以文档提交保存。

## Result

MiMo 账号页已增加同款设置按钮及空设置浮层，开发窗口已显示；完整质量门与原生界面验收通过。

## Deviations

None.

## Remaining Risks

按用户确认，本次面板仅显示空状态，没有实际可配置项。

## Completed

2026-10-16（Asia/Singapore）
