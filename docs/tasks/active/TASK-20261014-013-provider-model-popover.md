# TASK-20261014-013: 提供商模型浮层与小米旧模型清理

## Status

In Progress

## Created

2026-10-14（Asia/Singapore）

## Request

小米不再提供 `mimo-v2.5-pro` 和 `mimo-v2.5`；模型展开撑高卡片，导致同一行左侧卡片出现多余留白。将模型列表改成悬浮展示，保持卡片高度。

## Goal

移除小米内置目录中的两个旧模型，以不参与卡片布局的浮层展示模型列表，保留鼠标与键盘访问。

## Scope

- 内置提供商模型目录及相关回归测试。
- 提供商卡片的模型列表交互及相关回归测试。
- 必要文档和质量验证。

## Analysis

当前聊天工作区是旧功能分支，实际开发窗口来自 `E:/go/new/wails3-v/OmniProxy-main`。用户明确要求同步到 main 并让当前窗口生效，因此将尚未实施的任务记录移至 main，使用该工作区下一个任务编号 013。

根因：`ModelPreview` 将列表插入卡片正常文档流，网格默认拉伸同一行卡片。采用现有 Radix Popover 的 Portal、碰撞避让与非模态交互，不引入依赖；列表浮层不参与卡片布局。小米模型统一来自 `internal/provider/mimo/catalog.go`，移除旧 ID 同步影响提供商预览、官方预设、模型发现过滤与默认路由。保留用户已有精确规则及账号配置。

风险：已保存的旧模型规则仍保留，但执行时会按不支持模型返回错误。不得自动删除用户规则或重写客户端配置。

## Implementation Plan

1. 定位小米模型目录与卡片展开根因，选择现有浮层能力。
2. 清理旧模型并实现不改变卡片高度的模型浮层。
3. 验证目录与浮层交互，运行 `task verify`。
4. 复核 diff，填写实际结果并归档、提交本任务文件。

## Implementation Result

- 提取 `ModelPreview`，沿用已有 Radix Popover 的非模态浮层与 Portal，保持两款模型摘要和剩余数量；完整列表不参与卡片布局。
- 浮层使用深浅主题语义色、窗口边缘避让、可换行标签和已有 `ScrollRegion`；支持 Enter 打开、入口／关闭按钮／Esc／外部点击关闭和焦点恢复。
- 小米统一模型来源移除 `mimo-v2.5-pro`、`mimo-v2.5`，仅保留三款 V2.6 模型；预设、发现过滤、支持检查和默认路由同步使用新列表。
- 增加模型目录与历史发现结果过滤回归测试，以及浮层挂载、键盘和外部点击回归测试；自定义接口无模型时继续显示原说明。
- 主工作区原有正常桌面 `task dev` 自动重建 Go 后端；实际原生进程 41916，路径为 `E:/go/new/wails3-v/OmniProxy-main/bin/omniproxy.exe`，已核对新目录及新 UI 生效。

主要文件：`frontend/src/features/providers/model-preview.tsx`、`model-preview.test.tsx`、`provider-card.tsx`、双语词典，以及 `internal/provider/mimo/catalog.go` 和相关测试。

## Verification

- PASS：Go 定向测试 `go test ./internal/provider/mimo ./internal/application/provider ./internal/application/modelcatalog ./internal/application/routing`。
- PASS：浮层与提供商页面定向测试 14 项；品牌图标测试随完整质量门通过。
- PASS：`task verify BIN_DIR=tmp/provider-model-popover/quality-bin`，退出码 0；格式、Go vet／staticcheck、ESLint、TypeScript、Go 测试、71 个前端测试文件／392 项测试及生产构建全部通过。构建输出置于忽略目录，保留正在运行的开发程序。
- PASS：最终浮层文件补充 Prettier 检查；`git diff --check`。
- PASS：main 原生窗口四种组合（深／浅主题 × 960／1717 px），逐一对比所有提供商「测试」按钮矩形和相邻 MiniMax 标题位置，展开前后完全一致；核对三个 V2.6 模型、无两个旧模型、窗口边界及 Esc 关闭。
- 截图、UI Automation 报告和验证日志：忽略目录 `tmp/provider-model-popover/`。验证脚本恢复窗口大小和原主题，不调用上游推理或变更账号凭据。
- 首次浮层测试发现额外的 `hideWhenDetached` 在不计算几何尺寸的 jsdom 中隐藏面板；移除此非必要选项，沿用仓库现有浮层的碰撞处理，最终全部通过。
- 既有 Vite 大包体提示仍存在，不影响构建；本任务不扩展为打包优化。

## Documentation

已同步 PRD 的浮层行为、MiMo 的三模型支持范围和 CURRENT；无架构、技术栈或依赖变更。

## Remaining Risks

已保存的 V2.5 精确规则需切换为仍支持的模型；历史账号模型同步快照可能保留旧条目，可重新执行模型同步更新。本任务不自动删除用户配置或重写客户端文件。

## Deviations

用户确认直接交付到 main 后，将未实施的记录从旧功能分支工作区迁至主工作区；没有反向合并旧代码，也没有变更旧分支文件。

## Git

待本次实现提交后写入提交哈希并归档；main 原有未跟踪的 TASK-20261014-010 运行记录不纳入本任务提交。

## Result

小米预设仅保留三个 V2.6 模型；完整模型列表在局部浮层展示，原生窗口确认当前及相邻卡片高度稳定，完整质量门通过。
