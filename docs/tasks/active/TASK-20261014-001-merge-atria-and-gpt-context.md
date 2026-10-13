# TASK-20261014-001: 合并 Atria 与 GPT 长上下文到 main

## Status

In Progress

## Created

2026-10-14（Asia/Singapore）

## Request

用户要求将独立分支 `codex/atria-provider` 合入主 `main`，完成 Atria 接入；随后明确同时合入刚完成的 GPT 长上下文分支。

## Goal

主分支同时包含 Atria 原生三协议接入和已验证 GPT 长上下文／GPT-6.1 Sol，完成冲突处理、回归、质量门及本地合并提交。

## Scope

- `codex/atria-provider`：`ef4eb36`，实现 `015cd6a`。
- `codex/deepseek-claude-context`：`153e37a`，GPT 实现 `cbd7176`。
- 独立 main 工作区合并、最小冲突修复、必要绑定生成、验证和文档更新。

## Out of Scope

推送远端、覆盖未提交文件、改写共享历史、重做提供商实现或使用真实账号做额外推理。

# Analysis

当前 main 为 `7bf2772`；两分支共同祖先为 DeepSeek 已完成修复后的 `a00ed05`。原工作区有一份未提交的旧 Atria 规划记录，保留不动。Atria 工作区干净，已记录 369 项前端测试与质量门通过；无有效 Atria Key，真实上游与客户端专有协议验证仍有限制。

GPT 分支改动目录授权版本、CLI 逐模型压缩、Desktop 路径和默认保留；Atria 扩展目录预设来源、路由与提供商。合并时需保留两侧行为，并检查 CURRENT 的分支说明与目录文档。

# Implementation Plan

1. 创建独立 main 工作区，带入本记录，保留原工作区。
2. 快进纳入已完成 GPT 分支，再合并 Atria，解决实际冲突。
3. 检查 Atria 迁移／协议／路由及 GPT 确认能力回归，运行完整 `task verify`。
4. 同步文档，检查差异，提交合并并记录哈希、归档。

# Implementation Result

已在 `E:/go/new/wails3-v/OmniProxy-main` 的 main 工作区快进纳入 GPT 分支，并合并 Atria。仅 CURRENT 状态看板出现内容冲突；保留两侧任务结果，移除独立分支时态说明。目录代码自动合并后保留了 Atria 预设来源与 GPT 授权版本确认能力。

保留 Atria 019 升级迁移、内置卡片、原生 Chat／Responses／Messages HTTP/SSE、默认／显式路由、工具与用量统计；同时保留 GPT-6.1 Sol、账号授权版本确认、1M 选项、逐模型压缩、DeepSeek 1M、Desktop 默认与路径保护。没有改动原工作区未提交记录或真实账号／客户端配置。

# Verification

- 已安装锁定版本依赖并先完成正常前端构建，准备全新工作区所需的 Go 嵌入资源。
- 定向 Go 回归 PASS：Atria 适配器、模型目录、路由、网关、Agent 配置与 Core 目录。
- 合并后 `task verify` PASS：格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、69 个前端文件／369 项测试、Windows 生产构建。
- 自动绑定重新生成后无额外未提交差异；`git diff --check` 和暂存差异检查 PASS，无未解决冲突。
- 证据：`tmp/merge-atria-gpt/verify.log`（忽略目录，不提交）。未发送真实推理，没有有效 Atria Key，真实上游和客户端专有扩展仍待实测。

# Git

待填写。

# Result

两个授权分支的实现已整合并通过质量门，等待本地合并提交。main 生产二进制已生成，真实 Atria 调用限制保持如实记录。
