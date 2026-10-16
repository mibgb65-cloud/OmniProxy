# TASK-20261017-005: 合并 Claude 客户端入口展示

## Status

Completed

## Created

2026-10-17（Asia/Singapore）

## Request

用户希望将本地 Agent 接入页的 Claude Code CLI 与 Claude Code Desktop 放在一起显示。

## Goal

将两个入口合并到同一张 Claude Code 卡片，分别展示安装／接入状态并保留各自配置入口。

# Analysis

现有目录按三个客户端 ID 各渲染一张卡片。只调整入口的视觉归组，保留已有详情路由和
返回焦点标记，避免改变独立客户端配置、密钥及安装逻辑。

# Implementation Plan

1. 合并 Claude 目录卡片，复用接入状态和安装摘要，补齐中英文文案。
2. 验证两项独立状态、导航及返回焦点，运行质量门并检查开发窗口。
3. 同步说明、提交并归档任务。

# Implementation Result

- 本地 Agent 目录调整为 Codex、Claude Code 两张卡片；Claude 内以紧凑入口显示 CLI／Desktop。
- 复用已有客户端卡片的紧凑样式与安装摘要，每项保留独立安装／接入状态、详情链接和返回焦点。
- 增加中英文分组文案，同步本地 Agent 接入说明；没有后端、配置文件或凭据逻辑变化。

# Verification

- 定向 `local-agents`、`agent-installation` 两文件共 18 项测试通过，覆盖归组、两项独立状态、
  CLI／Desktop 键盘导航及返回焦点、既有安装流程。
- 实际项目原生开发窗口确认归组及两个详情入口，深浅主题和 960×600 截图检查通过；
  仅进入／返回详情，未应用配置或发起推理，窗口尺寸及主题已恢复。
- `task verify BIN_DIR=tmp/claude-client-group-20261017/verify-bin`：PASS，退出码 0；格式、
  静态检查、类型检查、全部 Go 测试、88 文件／479 项前端测试及 Windows 生产构建通过。
- `git diff --check` 通过；保留已有五份未跟踪任务记录。
- 验证证据位于忽略目录 `tmp/claude-client-group-20261017/`。

# Documentation

更新 `docs/product/LOCAL_AGENTS.md` 和 CURRENT；无需修改架构或公共接口。

# Deviations

无。

# Remaining Risks

窄窗口中的卡片随内容增高，沿用页面滚动；两个客户端仍使用各自已有配置，合并仅改变展示。

# Git

实现提交：`fc1fa0186f8fe0635f184bd8dd37456300ea02ad feat(frontend): 合并 Claude CLI 与 Desktop 入口展示`。

归档记录和 CURRENT 由后续文档提交保存。

# Result

CLI 与 Desktop 已在同一 Claude Code 卡片显示，两个配置入口和独立状态均保留。

# Completed

2026-10-17（Asia/Singapore）
