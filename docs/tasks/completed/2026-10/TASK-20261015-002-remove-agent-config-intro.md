# TASK-20261015-002: 移除一键配置弹窗的两段说明

## Status

Completed

## Created

2026-10-15 11:31（Asia/Singapore）

## Request

一键配置页面不要显示以下内容：

- 通过独立的本地 API Key 接入网关，保留原有登录与会话。
- 自动提供可用模型与路由，无需在此选择单个模型；接入后在客户端中切换。

## Goal

所有客户端的一键配置弹窗移除上述两段通用说明及其占位，继续提供现有预览、重新读取和应用操作。

## Analysis

两段文字仅由共用 `AgentConfigDialog` 渲染，使用 `agents.description` 和 `agents.automaticModels` 翻译键。删除对应节点并清理中英文孤立键；将对话框的可访问描述改为保留的客户端重启提示，避免悬空 ARIA 引用。此变更不涉及配置业务或接口。

## Implementation Plan

1. 移除两段文字及中英文翻译键，将可访问描述关联到现有重启提示。
2. 运行相关现有交互测试和质量检查，核对原生开发窗口。
3. 记录验证结果、检查差异，仅提交本任务文件并归档。

## Implementation Result

- 共用弹窗移除两段通用说明及对应占位，中英文翻译中的两个孤立键同步删除。
- 现有重启提示使用 `Dialog.Description`，继续为对话框提供有效的可访问描述，不增加可见文案。
- 修改文件：`agent-config-dialog.tsx`、`agents-zh.ts`、`agents-en.ts`；没有修改接入配置业务或接口。
- 开发窗口已通过 Vite 热更新；原生 Claude Code Desktop 弹窗核对两段文字不存在，保留重启提示、预览、重新读取和关闭按钮；实际重新读取后仍显示成功提示。
- 截图与原生检查结果位于忽略目录 `tmp/remove-agent-config-intro/`。

## Verification

- PASS：原生开发进程 PID 25652，绝对路径属于当前 main 项目；通过 Windows UI Automation 验证删除内容及保留操作，并人工检查截图中未留空白占位。
- PASS：点击「重新读取配置」后仍显示「已重新读取当前配置，请重新预览后应用。」；未应用或改写客户端配置。
- PASS：格式检查、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试及 73 个前端测试文件／401 项测试。
- PASS：`VITEST_MAX_WORKERS=4 task verify BIN_DIR=E:/go/new/wails3-v/OmniProxy/tmp/remove-agent-config-intro/quality-bin` 完整质量门，退出码 0，包含生产前端与 Windows 桌面构建。日志保存在 `tmp/normal-desktop-dev-20261015/remove-intro-verify.log`；使用独立输出目录避免覆盖运行中的开发应用。
- PASS：`git diff --check`；生产构建生成的绑定没有额外差异。
- 不为简单文案删除新增重复实现的测试，复用现有配置交互测试。

## Documentation

已在 CURRENT 登记完成；产品配置行为未变，无需修改产品或接口文档。

## Git

`34f855faa06a6eac061e90d75e9d44a5c9cb3ea9 fix(gateway): 精简一键配置弹窗说明`

仅提交本任务代码与记录；四个既有未跟踪任务记录保持原样。

## Result

两段说明已从共用弹窗移除，开发窗口已更新；原生界面及完整质量门通过，代码已提交并归档本记录。构建后再次核对 main 原生窗口、开发版健康接口和 Vite 热更新，均正常。

## Deviations

None.

## Remaining Risks

无新增业务风险。既有 jsdom `scrollTo` 与 Vite 体积提示未影响验证结果。

## Completed

2026-10-15 11:37（Asia/Singapore）
