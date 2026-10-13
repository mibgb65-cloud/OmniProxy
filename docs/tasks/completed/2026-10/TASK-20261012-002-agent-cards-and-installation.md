# TASK-20261012-002: 本地 Agent 卡片入口与客户端安装

> Retroactive record：本记录按用户要求补录。实现先于记录；此前错误地将“启动开发环境不需要做记录”沿用到后续开发。以下只记实际完成的工作，不补造事前计划或提交。

## Status

Completed

## Created / Completed

2026-10-12（补录日期；原实现的具体开始时间未记录）

## Request

- “不需要这个请求验证功能。”
- “本地 Agent 接入……把本地 agent 的图标先做上去，做成一块一块的那种，然后再点击进入到页面后再进行一键配置。”
- “不同的 agent 的安装可以做吗？”用户选择“集成到 OmniProxy”。

## Goal / Scope

移除网关页面的请求验证入口；将本地客户端展示为品牌卡片及独立详情页；提供 Windows 安装检测、安装和官方说明。

## Analysis

卡片导航与文件配置分开，避免点击卡片就修改客户端。CLI 与桌面应用分别检测；安装使用固定软件包和渠道，不接收前端命令、下载地址或路径。

## Implementation Plan

以下为已实施步骤的追溯，不代表事前创建过计划：移除请求验证 UI → 卡片与详情导航 → 固定 WinGet 安装后端 → 安装状态 UI → 回归验证。

## Implementation Result

- 请求验证 UI 已移除。
- 增加 Codex、Claude Code CLI、Claude Code Desktop 品牌卡片、详情导航、焦点恢复和选项卡状态保留。
- 安装检测支持 CLI 可执行文件、卸载注册表与 MSIX 元数据；安装和网关接入状态独立。
- 固定四个软件包，安装串行执行，不自动升级已有版本；安装完成后重新检测。
- 系统授权由 Windows 处理，错误只展示稳定代码，原始安装器输出不进入 UI。

## Files Changed

- `frontend/src/features/gateway/agent-client-tile.tsx`、`agent-client-detail.tsx`、`local-agents.tsx`
- `frontend/src/features/gateway/agent-installation-card.tsx`、`use-agent-installations.ts`
- `internal/infrastructure/agentinstall/`、`internal/application/agentinstall/`
- `internal/transport/wails/agentinstall.go`
- `frontend/public/agent-icons/`、相关服务和中英文词条

## Verification

- 卡片导航、安装 ID、重复点击、检测异常、失败提示等回归测试已通过。
- 后续完整 `task verify` 已验证这些实现；2026-10-12 最近一次已完成验证包含 64 个前端测试文件、340 项测试，并通过 Go 测试、格式、lint、类型检查和构建。
- 安装命令测试使用固定假运行器；没有通过自动测试安装用户客户端。
- 卡片和配置页面曾在原生开发窗口验收。请求验证文案已做源代码检索核对。

## Documentation

行为说明已写入 `docs/product/LOCAL_AGENTS.md`；本次补录同步 `docs/plans/CURRENT.md`。

## Risks / Deviations

当前安装面向 Windows 本机，WSL 与其他平台未覆盖。原先遗漏事前 Task Record，本文件为明确标注的补录。

## Git

Not committed by Agent. 工作区存在大量既有未提交、未跟踪实现，不能将无关实现混入本任务提交；未创建代码提交或伪造提交哈希。

## Result

已完成品牌卡片、详情配置页和客户端安装入口；安装进度见 TASK-20261012-003。
