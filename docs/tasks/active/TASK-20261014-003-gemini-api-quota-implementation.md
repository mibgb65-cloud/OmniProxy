# TASK-20261014-003: 重建最新开发环境并接入 Gemini

## Status

In Progress

## Created

2026-10-14（Asia/Singapore）

## Request

用户要求先重新构建最新版本开发环境，完成后开始 Gemini 接入。承接 TASK-20261014-002 的 API 调用、用量、限流和额度规划。

## Goal

先使实际开发窗口运行已合并 main 的最新代码，再完成 P1/P2：Gemini Chat 原生兼容转发、官方模型目录、用量、项目绑定与限流／支付观察，以及资源页面。保留已有提供商、GPT 长上下文和用户数据。

## Scope

- 基线 main `75ed46e`；独立 Gemini 分支及最新开发启动／窗口验收。
- API Key 回调、真实目录元数据、Chat JSON/SSE、路由和 usage。
- Google 项目绑定、共享范围的限流／支付错误观察、账号和配额页。
- 最小充分测试、完整质量门、文档、提交和原生开发环境更新。

## Boundaries

首版未确认官方剩余资源的字段保持未知；不猜余额、不抓私人 AI Studio RPC、不用多个同项目 Key 绕过限流。Cloud OAuth／BigQuery 和完整 Agent 签名桥接作为后续阶段，不伪装已经支持；真实授权条件不阻碍完成 P1/P2。

# Analysis

已建立 `codex/gemini-api-quota`，基线包含 Atria 和 GPT 修复。旧开发进程由托盘正常退出，保留生产版与用户数据；新开发环境使用该工作区的正常 task dev。

适配器仅原生转发 Chat，目录过滤非文本生成方法并保留单独输入／输出限额；P2 以项目绑定和经过校验的错误观察形成资源快照，不伪造官方余额。界面沿用现有样式和 i18n，使用 ui-ux-pro-max 的异步错误／TypeScript 建议；不进行页面风格重构。

# Implementation Plan

1. 安全退出旧开发进程，在最新 main 基线建立 Gemini 分支，执行正常 task dev 并核对原生窗口。
2. 核对官方端点和已有接口，完成适配器、目录／路由、项目资源服务和界面。
3. 契约／数据库／界面回归、必要真实最小验收、task verify。
4. 更新任务结果与文档、检查差异、提交归档，保持最新开发版可用。

# Implementation Result

1. 最新开发版已先启动并核对 Atria；旧开发窗口通过托盘正常退出，生产版与用户数据保留。当前开发工作区为 `E:\go\new\wails3-v\OmniProxy-main`，分支 `codex/gemini-api-quota`。
2. 新增 Google 适配器：官方兼容 Chat JSON／SSE，保留工具签名与扩展，流式请求补充 usage，使用受控 API Key、取消、有界响应和稳定错误。无静默重试／账号切换。
3. 官方兼容目录与分页原生生成模型取交集，处理官方实际返回的 `models/` 前缀，保存独立输入／输出限制；按真实选中账号校验显式输出上限。路由、别名、模型目录与网关资源检查已接入 Google。
4. 迁移 020 与项目资源服务保存用户声明的共享项目范围、脱敏错误观察和重试时间。凭证版本隔离旧绑定及观察；在途旧版本拒绝写入。每日重置按太平洋日历和夏令时，过期不推算恢复。
5. Google 账号详情、配额页新增资源卡片、项目绑定、只读本机刷新和官方入口；输入／输出限制分别显示。官方余额／剩余额度显示未知，token 用量复用本机统计。
6. Gemini 仅公布 Chat 能力，不混入 Codex／Claude 一键配置模型列表。未改变既有 DeepSeek／GPT 1M 与 Atria 能力。更新产品说明及自然生成 Wails 绑定。

# Verification

- `task verify`：format、Go vet／staticcheck、ESLint、TypeScript、Go 测试、70 个文件／371 项前端测试及 Windows 生产构建全部通过。补充目录同步／版本回归与定向 vet／staticcheck 通过。
- 契约测试：增量 SSE／取消、扩展和签名不丢失、鉴权头／重定向防护、分页／前缀／边界、错误脱敏与指标关联；数据库测试：共享项目去重、凭证轮换、在途版本、解绑与夏令时；网关测试：别名、JSON／SSE usage、输出上限、未支持协议拒绝、限流观察。
- 前端测试：一个共享资源组显示未知官方资源、过期记录不恢复、绑定带账号版本、错误保持弹窗且可重试。既有账号／配额页回归通过。
- 使用开发环境已有受保护 Google Key，同步 46 个生成条目；最小真实网关调用 `gemini-3.1-flash-lite` 返回 HTTP 200，输入 5／输出 2 tokens，`usage_complete=true` 且入库一致。没有输出明文 Key、上游消息或真实响应正文。
- `gemini-2.5-flash` 诊断请求收到 404；`gemini-3.8-flash` 遇到诊断超时／上游网络失败，未据此宣称可用。原生及兼容目录可见不保证全部模型可调用。
- 原生开发窗口：Gemini 配额模块、未知额度、本机刷新和项目弹窗已验收，取消弹窗，未猜测或保存用户项目 ID。证据在忽略目录 `tmp/gemini/`。
- `git diff --check` 通过；未引入依赖或修改技术栈。

# Git

基线 main `75ed46e`，实现位于 `codex/gemini-api-quota`。实现提交待记录；不自动推送或合入 main。

# Result

P1/P2 完成并通过真实小请求、原生界面和质量门验收。官方 Cloud OAuth／项目 IAM／Monitoring 和账单账号资源仍未接入；Gemini Messages／Responses／Interactions 及完整 Agent 桥接保留在 P3/P4。官方剩余资源保持未知，其他模型真实权限仍须各自确认。

官方依据：[OpenAI 兼容](https://ai.google.dev/gemini-api/docs/openai)、[模型目录](https://ai.google.dev/api/models)、[项目限流](https://ai.google.dev/gemini-api/docs/rate-limits)、[计费](https://ai.google.dev/gemini-api/docs/billing)。
