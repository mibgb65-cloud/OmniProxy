# TASK-20261015-005: 优化 ChatGPT 订阅登录弹窗

## Status

Completed

## Created

2026-10-15（Asia/Singapore）

## Request

用户提供 ChatGPT 订阅账号登录弹窗截图，认为 UI 不美观，要求重新优化。

## Goal

改善标题、授权方式选择、状态反馈和操作区层级，适配深浅主题与小窗口，纠正过时文案，同时保持已有授权生命周期。

## Scope

订阅登录弹窗、授权方式卡片、等待／设备码／完成／错误展示、中英文文案、必要交互回归和原生视觉检查。

## Out of Scope

不改后端认证、凭证存储、网关或账号授权规则；不触发用户真实登录、取消或退出操作。

# Analysis

现有两种授权方式使用紧凑浏览器原生单选控件，说明和按钮同一区块；状态组件缺少明确视觉层级。`subscription.scope` 仍称转发待接入，与当前代码不一致。复用 Radix Dialog、已有品牌图标、Lucide 和语义主题色，不引入依赖。

# Implementation Plan

1. 对齐现有组件和授权状态，使用整块单选卡片、分区布局及明确操作反馈。
2. 调整等待／设备码／完成／错误状态和相关中英文说明，保持关闭恢复及显式取消语义。
3. 回归关键交互，运行 task verify，核对实际开发窗口深浅主题和最小窗口。
4. 更新任务／看板，审查差异，提交归档。

# Implementation Result

- 登录窗口改为品牌标题、内容区和固定操作区；宽度 560 px，正文独立滚动，统一圆角、间距和语义主题色。
- 抽出授权方式卡片，保留原生单选及键盘行为；整块可点击，选中／焦点可区分，说明与无障碍描述关联。
- 等待／设备码／完成／错误改为独立状态面板，强调一次性设备码和官方页面入口，保留关闭恢复及显式取消语义。
- 中英文文案同步精简，移除旧 subscription.scope，保留本机凭证存储提示。未修改后端和授权 Hook，不新增依赖。

# Verification

- PASS：针对 7 项前端测试，包含原有浏览器／设备码登录、关闭恢复、取消、完成及错误脱敏，新增整卡点击、键盘选择和无障碍说明；标题、存储提示与过时文案回归检查。
- PASS：项目原生开发窗口 PID 3140、路径 `E:/go/new/wails3-v/OmniProxy/bin/omniproxy.exe`，检查新增／重新登录、浏览器／设备码选择、深浅主题和 960×600 最小窗口；已逐张查看截图，弹窗和主按钮均未超出边界。恢复原主题和窗口尺寸，没有开始真实授权或取消现有授权。证据在忽略目录 `tmp/subscription-login-ui-20261015/`。
- PASS：`task verify BIN_DIR=tmp/subscription-login-ui-20261015/verify-bin`；格式、Go vet／staticcheck／ESLint、TypeScript、全部 Go 测试、75 个前端测试文件／408 项测试和生产构建通过。无新增 Lint 警告；既有 uname／tail 缺失、jsdom scrollTo 和构建大包提示不影响结果。
- PASS：`git diff --check`、UTF-8 无 BOM／LF、变更范围与无依赖变更检查；提交前仅暂存本任务文件，保留四份既有未跟踪任务记录。

# Documentation

同步 `product/SUBSCRIPTION.md` 的界面说明和 CURRENT；产品能力与架构边界不变。

# Remaining Risks

本轮没有触发真实浏览器／设备码授权；授权流程沿用现有后端与 Hook，等待／完成／错误等状态使用原有服务 Mock 回归。实际原生窗口仅检查安全的初始状态和授权方式选择。

# Git

`4086f19` feat(accounts): 优化 ChatGPT 订阅登录弹窗。仅提交本任务 10 个文件，四份既有未跟踪任务记录保持原状，未推送。

# Deviations

None。按已有技术栈与组件实现，无后端和授权生命周期改动。

# Result

界面优化、必要交互回归、完整质量门及实际开发窗口视觉验收完成。登录与重新登录均使用新布局，窗口尺寸和主题已恢复。

# Completed

2026-10-15（Asia/Singapore）
