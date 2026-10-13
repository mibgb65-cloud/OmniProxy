# TASK-20261012-001: 修正 Claude Code Desktop 显示名称

## Status

Completed

## Created / Completed

2026-10-12

## Request / Goal

将截图中的“Claude Desktop · Code”改为“Claude Code Desktop”，中英文界面保持一致。

## Implementation Result

- 更新 `frontend/src/i18n/locales/agents-zh.ts` 与 `agents-en.ts` 的 `agents.claude-desktop` 名称。
- 同步 `local-agents.test.tsx` 和 `gateway-agent-configuration.test.tsx` 的现有名称期望。
- 保留客户端标识和接入逻辑，未修改其他产品文案。

## Verification

- 本地 Agent 页面、网关配置页面及国际化三个现有测试文件共 23 项测试通过。
- 四份修改文件的 Prettier 检查通过，逐处确认旧名称已替换。
- 测试环境有既有的 jsdom `window.scrollTo` 未实现提示，不影响测试结果。
- 仅调整字符串及现有测试期望，按根目录 AGENTS.md 对简单变更的约定，未运行针对显著变更的完整 `task verify`。

## Git

Not committed by Agent. 四份前端文件在本次修改前均为未跟踪文件；为保留用户原有实现及其他工作区变更，本任务不创建包含原有代码的提交。

## Result

本地 Agent 卡片、配置页标题及相关入口的中英文显示名称已统一为“Claude Code Desktop”。
