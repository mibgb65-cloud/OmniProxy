# TASK-20261016-006: 修复 Claude Code CLI 模型菜单与接入预览不一致

## Status

In Progress

## Created

2026-10-16（Asia/Singapore）

## Request

用户反馈 Claude Code CLI 的 `/model` 菜单只有重复的 DeepSeek 和内置 Fable，与 OmniProxy 一键配置预览中的 18 个模型／路由不一致。

## Goal

核对当前 CLI 支持的模型菜单配置，使一键配置实际发布预览中的模型列表，并保留原配置备份、恢复和外部修改保护。

## Analysis

- 当前 CLI 版本为 2.1.292；本地配置只有默认模型与 Opus／Sonnet／Haiku 三档映射，没有完整菜单。
- `buildPlan` 对 CLI 仅设置订阅压缩参数，`RefreshCatalog` 未覆盖 CLI；预览展示全部模型，但模式仍为 `client-default`。
- 以 Claude Code 官方模型／设置文档及本机 CLI 验证可用字段；避免把仅用于限制选择的 `availableModels` 误当作菜单注册。
- 官方模型文档明确支持 `modelPicker`；本机 2.1.292 内嵌校验与菜单实现确认 `options: [{model, label?, description?, behavesAs?}]` 和 `replaceBuiltInOptions`。采用显式列表替换内置档位，保留 Default 行，不虚构能力或改写用户已有的选择限制。
- 不更换上游账号、密钥、模型路由或默认模型，不发送计费推理请求。

## Implementation Plan

1. 核对官方 `modelPicker` 格式及本机版本行为，选择完整菜单配置方式。
2. 补齐 CLI 模型列表写入、受管目录刷新及必要的预览说明。
3. 增加应用／刷新／恢复和外部修改保护的回归测试，运行 `task verify`。
4. 在实际开发版应用并核对 CLI 菜单，记录结果和限制，审查并提交本任务变更。

## Implementation Result

- 修复 CLI 只写默认模型和档位映射、未注册完整菜单的根因：`modelPicker.options` 按预览顺序写入真实模型 ID 和标签，替换内置菜单，保留客户端 Default 行。
- `RefreshCatalog` 覆盖已受管 CLI，复用文件哈希核对、失败回退与加密恢复；不改用户已有选择限制，不伪造模型能力。
- 预览模式改为 `cli-model-list`，中英文显示 CLI 模型菜单并提示旧版更新方式；同步 LOCAL_AGENTS 产品说明。
- 原生开发窗口经 Wails watcher 自动重建为 PID 31028，在现有 Claude CLI 配置上完成「重新读取配置 → 配置预览 → 应用配置」。原配置由既有流程加密备份；默认模型保持 `deepseek-flash[1m]`。
- 实际配置含 18 个菜单模型，与原生预览一致。新开独立 Claude Code 2.1.292 会话，运行 `/model` 后实际显示 Atria、两款 DeepSeek、Kimi 和其余模型（菜单首屏提示另有 14 个，共 18 个模型加 Default）；没有重复档位或 Fable。取消菜单并正常 `/exit`，用户原 CLI 会话保持运行。

## Files Changed

- `internal/application/agentconfig/claude_picker.go`、`catalog_patch.go`、`plan.go`、`refresh.go` 及 CLI 菜单回归测试。
- `frontend/src/features/gateway/agent-catalog-preview.tsx`、对应测试及中英文翻译。
- `docs/product/LOCAL_AGENTS.md`、当前任务记录与 CURRENT 看板。

## Verification

- PASS：`go test ./internal/application/agentconfig -count=1`，覆盖列表应用／更新／删除、标签／顺序、非受管文件和外部编辑保护，以及原始菜单与其他设置精确恢复。
- PASS：前端预览／配置弹窗定向测试，2 个文件、5 项测试通过。
- PASS：`task verify BIN_DIR=tmp/claude-cli-picker-20261016/bin`，退出码 0。格式、Go vet／staticcheck／ESLint、TypeScript、Go 全部测试、85 个文件／450 项前端测试及生产构建均通过；构建输出隔离以保留正在运行的开发版。
- PASS：原生预览显示「CLI 模型菜单」「将提供 18 个模型／路由」，应用提示原配置已加密备份；本机实际 CLI 菜单验证通过。
- PASS：`git diff --check`。忽略目录 `tmp/claude-cli-picker-20261016/` 保存验证日志及仅含公开模型状态的验收报告。
- 未发送计费推理请求，网关保持停止；本次验收针对模型菜单，与上游调用是否成功分别判断。

## Remaining Risks

旧版 Claude Code 可能忽略 `modelPicker`，仍可通过完整模型 ID 切换或自行更新；组织托管菜单／已有模型限制仍按客户端优先级生效。完整质量门仅有既有的 Vite 资源体积和测试环境 scrollTo 提示，无失败。

## Deviations

使用官方显式菜单配置，未使用需要运行网关且可能筛选第三方模型的动态发现。未改变网关或上游路由。

## Git

实现及文档已核对，待提交；保留任务开始前五个无关未跟踪记录。

## Result

CLI 模型菜单已与 18 项接入预览对齐，并已在用户的实际配置上应用。现有 CLI 会话重启后读取新菜单。
