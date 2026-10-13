# TASK-20261009-006: 本地 Agent 一键配置与官方模型同步

## Status

Completed

## Created

2026-10-09

## Request

网关页面增加本地 Agent 一键配置，首期 Codex 与 Claude Code，使用 API Key 管理接入。必须保留原官方会话，不改变导致会话被按提供商隔离的配置标识。同步 ChatGPT 与 DeepSeek 官方模型，并在路由中显示。

## Scope

- 读取现有客户端配置、配置预览、受控应用、备份／恢复和接入 Key 管理。
- Codex 保留现有 model_provider 的值；不存在时保留默认身份，不创建新的提供商标识来替换。
- 不改写历史、会话索引、官方登录文件、历史目录或客户端数据根目录。
- 读取 ChatGPT 隔离授权和 DeepSeek API Key 的官方模型目录，并接到路由选择。

## Confirmed scope

- 用户确认同时支持 Codex／Claude Code 的 CLI 和桌面 Code 功能。
- 用户确认本次同时实现 Responses／Messages 转发，配置后可实际使用。

# Analysis

当前网关只支持 DeepSeek Chat Completions，Codex／Claude 配置卡片仅为预览；旧预览写入 model_provider="omniproxy"，与当前保护官方会话的明确要求冲突，需替换方案。

需查明官方客户端支持的同标识 API Key／端点覆盖方式、认证优先级与模型目录协议，不能凭配置字段存在就保证 CLI／Desktop 会话一致可用。

# Implementation Plan

1. 核对本机 CLI schema、官方配置／会话规则与两个模型查询接口。
2. 按确认范围实现模型同步和路由显示，标明已同步模型与实际转发能力的区别。
3. 实现配置管理和网关接入 Key，在测试目录验证备份、字段级变更与原标识保留；桌面采用官方独立配置入口，不自动切换工作空间。
4. 按确认范围接入协议，执行关键路径／完整质量检查，更新记录并恢复开发环境。

# Progress

- 已查明现有模板会修改提供商标识，且网关缺少两种 Agent 所需协议。
- 已请求首期客户端及协议范围，继续核对官方能力。
- 实机隔离授权 model/list 成功返回 6 个模型；没有写入默认 Codex 配置或授权。
- Claude Desktop 需独立第三方推理配置库；Codex 保留原提供商标识，实机验证后内置 openai 改用根地址覆盖，已有自定义提供商才合并同表字段。

# Implementation Result

- 网关增加 Codex CLI／Desktop、Claude Code CLI、Claude Desktop Code 配置，模型选择、预览、应用、DPAPI 备份恢复、Key 复制／启停。Claude Desktop 添加独立条目，保留 appliedId／deploymentMode，首次使用需用户选择。
- Codex 0.155.1 禁止覆盖内置 openai 表，已改用 openai_base_url 和独立 Key 路径；保留原 model_provider、认证文件和会话目录。
- 模型目录按账号、凭据引用与工作区隔离并持久化；官方 model/list 分页首请求不发送空 cursor，DeepSeek GET /models 固定域名、有界验证，路由检查所属账号及启用状态。
- 接入 HTTP Responses、Messages 及 Responses WebSocket，原生 ChatGPT 流式，跨协议文本、namespace／custom 工具及结果关联；非支持内容显式失败。
- 兼容官方 Windows 旧 keyring／新 codex_auth.age；按 GetFinalPathNameByHandle 匹配 Rust canonicalize 解析 junction。官方实机密文 work factor=20，设置固定上限。仅读取隔离 CodexAuth，不写明文。
- 新增 migration 007、薄 Wails 绑定、ADR-005 及产品文档；保留其他任务未提交改动。

# Verification

- 关键单测通过：提供商身份及其他配置保留、原文件精确恢复、外部变更阻断、Key 认证／停用、Desktop 原工作空间保留、age 命名空间／密码和损坏边界、模型失败缓存及凭据版本隔离、协议工具往返、未知内容拒绝、WebSocket 上下文和本地鉴权。
- 测试目录使用本机 Codex CLI 和桌面 app-server 跑通请求，后者保留 ChatGPT 身份及原认证文件，并按 openai 筛选找到旧会话。没有改写用户默认客户端配置。
- 用户已管理的账号真实同步：ChatGPT 6 个模型、DeepSeek 2 个模型；两个提供商的 Responses／Messages 四种真实请求均返回 200。没有使用重置卡。
- 最终完整 task verify 通过：格式、Go vet／staticcheck、ESLint、TypeScript、Go 测试、42 个前端文件共 220 项测试和 Windows 生产构建。构建保留已有大包提示，不影响成功。
- 实际组件浏览器检查 32 种亮／暗、100／140 字号、1920／960／600／360 宽度与 1000／600 高度场景，预览、Esc 关闭、布局和可达性通过；修正小窗大字号内容挤出弹窗的问题，标题／操作固定、内容区滚动。
- 原生 Wails 窗口验证三种配置按钮、全局弹窗和路由页已同步 8 个账号模型；真实默认客户端配置未改写。最终质量门再次通过，开发窗口、Vite 9245、Control 3891 和本地网关 3001 已恢复运行。

# Git

本工作树包含约 198 项前序任务未提交改动和新文件，本任务依赖其中的订阅／路由／日志服务及绑定。同一文件也含前序已授权实现，无法形成独立、可构建且不混入其他任务的提交。已检查 git diff 与敏感示例，保留改动而不混合提交。
