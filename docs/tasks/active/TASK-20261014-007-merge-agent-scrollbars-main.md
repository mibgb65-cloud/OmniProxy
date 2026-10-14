# TASK-20261014-007: 将一键配置滚动条优化合入 main

## Status

In Progress

## Created

2026-10-14（Asia/Singapore）

## Request

用户要求「合并到main」，承接一键配置弹窗及模型目录滚动条优化。

## Goal

将 `codex/deepseek-claude-context` 的滚动条实现和验收记录合入本地 `main`，保留主分支已有实现和未提交的其他任务内容。

## Analysis

源分支为 `dfc55c5`，仅 `b078db0` 和 `dfc55c5` 两个提交未进入主分支；`main` 为 `6548ca3`，包含已合并的 Atria、Gemini 及相关文档，工作区干净。主分支已使用 TASK-20261014-002 记录 Gemini 规划，因此滚动条原任务需要在合并时改为 TASK-20261014-006；本次合并记录使用 TASK-20261014-007。源工作区的既有未跟踪 Atria 规划不属于本任务，不纳入提交。

## Implementation Plan

1. 在已有 main worktree 执行保留历史的合并，检查并解决文档冲突，统一任务编号与 CURRENT 链接。
2. 核对两个滚动组件与已验证源分支一致，运行合并后的 `task verify`，生产构建输出至忽略目录以避开正在运行的开发二进制。
3. 审查差异、完成合并提交，将实际合并哈希和验证结果写回记录并归档。

## Implementation Result

- 在已有 `OmniProxy-main` worktree 执行 `git merge --no-ff --no-commit codex/deepseek-claude-context`。两个滚动组件无代码冲突，内容与已通过原生验收的源分支完全一致。
- LOCAL_AGENTS 自动保留双方描述，CURRENT 手动保留主分支所有任务并加入滚动条记录；没有改动主分支已有 Atria／Gemini 实现。
- 滚动条任务从原编号 TASK-20261014-002 改为 TASK-20261014-006，记录原编号与原因，同步 CURRENT 链接，避免与主分支 Gemini 规划冲突。
- 仅合并本任务提交和相关文档；源工作区既有未跟踪 Atria 规划保持不变。未删除分支、worktree 或改写历史。

## Verification

- PASS：两个滚动组件与源分支差异为空，未解决冲突列表为空，合并文件差异和 UTF-8 无 BOM 检查通过。
- PASS：任务编号没有重复，CURRENT 的所有相对任务链接有效。
- PASS：`VITEST_MAX_WORKERS=4 task verify BIN_DIR=E:/go/new/wails3-v/OmniProxy-main/tmp/merge-agent-scrollbars/quality-bin` 退出码 0。格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、70 个前端测试文件／384 项测试和 Windows 生产构建通过。
- 构建未产生额外的源码或绑定变更。日志位于 `tmp/merge-agent-scrollbars/verify.log`；构建输出使用独立目录，避开开发二进制。既有工具探测、jsdom scrollTo 和打包体积提示未影响质量门。

## Documentation

合并并保留 LOCAL_AGENTS 双方内容，更新 CURRENT 与滚动条任务编号。

## Deviations

任务看板存在内容冲突，已合并保留双方有效记录；为避免任务编号重复，将滚动条原记录重编号为 006。没有代码冲突或额外实现变更。

## Git

待提交。

## Result

待完成。
