# Product Requirements Document

## Overview

OmniProxy is a local-first desktop application that acts as an AI API gateway
and multi-account management tool. It runs on the user's machine, proxies AI
API requests through multiple provider accounts, and manages the local
configuration of AI coding clients.

The product exists because users of multiple AI providers and multiple
accounts need one local place to:

- unify access to several AI providers behind a single endpoint,
- distribute load across accounts of the same provider,
- survive quota exhaustion and account failures automatically,
- observe usage, quota and request history,
- configure local AI clients (Codex, Claude Code, Gemini CLI, …) safely.

## Goals

- Provide one local AI API gateway with model routing and account scheduling.
- Support multiple accounts per provider with automatic failover when a
  request fails or an account is rate-limited.
- Track quota, usage and request history per account and provider.
- Manage local client configurations (detect, read, back up, apply, restore,
  validate) without hand-editing config files.
- Run as a Windows desktop application with a system tray, single instance,
  autostart and automatic updates.
- Keep all data local: accounts, credentials, history and settings stay on the
  user's machine.

## Target Users

- Developers and power users with multiple AI provider subscriptions.
- Users of AI coding CLIs who need to switch providers/accounts frequently.
- Anyone who needs visibility into AI API usage and quota consumption.

## Core Use Cases

1. Point an AI client at the local OmniProxy gateway instead of a provider
   endpoint directly.
2. Add a provider account (API key or OAuth credential) once; OmniProxy
   schedules requests across eligible accounts.
3. Route a model name to the provider/account that should serve it.
4. Watch quota and usage; see which account served which request.
5. Detect and configure local AI clients (Codex, Claude Code, Gemini CLI,
   OpenCode) from the UI, with automatic backup and restore.
6. Control the gateway from the desktop UI or from external tools through the
   local Control API.

## Core Features

- Multi-provider access via a unified adapter architecture (OpenAI,
  Anthropic, Google, DeepSeek, Zhipu AI, Moonshot AI/Kimi, MiniMax, Xiaomi MiMo,
  StepFun, and OpenAI-compatible endpoints).
- Nine named providers are enabled by default and can be switched off; users
  can add enabled-by-default custom OpenAI-compatible endpoints.
- 提供商卡片使用与账号一致的电源图标按钮控制启停：绿色表示启用，灰色表示停用，悬浮提示说明下一步操作。
- Provider cards show a curated model preview and an unauthenticated endpoint
  reachability/response-time check; actual model access depends on an account.
- Multi-account management with OS-protected local credential storage; account
  metadata keeps only opaque references to secrets.
- API Key 账号支持单个添加和批量添加。单个账号名称选填，留空时按 Key 的前最多 8 个 Unicode 字符命名；批量每行一个 Key、每次 1–100 个，只使用自动名称。
- OpenAI 支持通过本机官方 Codex 组件登录 ChatGPT 订阅账号，提供浏览器／设备码授权、多账号隔离、身份与套餐、刷新、退出和重新登录。系统凭证库与官方加密命名空间保存授权，已支持订阅 Responses 转发；详见 `docs/product/SUBSCRIPTION.md`、`docs/product/LOCAL_AGENTS.md`。
- OpenAI 账号设置提供应用级 Codex WebSocket 开关，默认开启、持久保存，控制当前网关的握手和连接内后续请求，不改写 Codex CLI／桌面端的提供商身份、认证或会话，HTTP/SSE 保持可用。
- OpenAI 账号设置在入口旁局部展开，自动激活 Codex 额度窗口默认关闭，切换开关直接持久保存并生效；对未开始的 5h／周窗口发送原账号最小消息并复查，持久去重防止重复消耗，已激活窗口不发送。
- 自动名称有意展示 Key 前缀，短于 8 个字符时使用实际长度；名称不写入日志。删除账号必须经过应用内、随主题切换的确认弹窗。
- 提供商账号详情支持卡片／列表切换并保存展示偏好；展示真实密钥的脱敏片段，用户可主动复制密钥。新的密钥读取接口只返回脱敏文本，复制在桌面端直接写入系统剪贴板。
- 账号支持编辑名称与替换 API Key；编辑弹窗不回填当前完整密钥，Key 留空时保留原凭证，名称清空继续自动命名。
- Account scheduling: eligibility, health, priority, load, quota, cooldown,
  concurrency and retry.
- Model routing rules.
- 模型路由首版支持精确别名映射、具体提供商配置与可选固定账号；规则持久化并动态生效，提供不读取密钥的匹配预览，模型发现接口同步可解析别名。
- 路由页依据已有模型目录自动展示同名初步映射，无需逐条新增规则；同名模型可保存多个服务商的优先级顺序。精确规则优先，默认映射按顺序选择可用服务商，跳过不可用配置，其余候选沿用自动选择；预览／网关／客户端目录保持一致。本次覆盖现有 DeepSeek 和 ChatGPT/Codex 适配器。
- 网关支持 DeepSeek 与 ChatGPT / Codex 的文本／工具调用，可通过 Chat Completions、Responses、Messages 和 Codex WebSocket 接入；跨协议先收完整结果再输出 SSE，其他提供商及自动调度仍待接入。
- 网关页支持 Codex CLI／Desktop、Claude Code CLI 和 Claude Desktop Code 配置，独立接入 Key、预览、加密备份和恢复；保留原提供商标识与官方会话，Desktop 添加独立配置条目。路由页可同步 ChatGPT／DeepSeek 官方模型目录并按账号选择；详见 `docs/product/LOCAL_AGENTS.md`。
- 一键接入自动提供适用的模型／路由目录，无需手选单个模型；优先保留仍可用的原默认。目录记录官方已确认元数据并保留未知能力，Codex 按身份／核心版本适配本地或远程目录，Claude Desktop 提供完整显式列表。受管目录支持安全更新和恢复。
- Quota monitoring and usage statistics.
- DeepSeek API Key 账号支持官方余额查询，展示币种、总余额、赠送／充值余额与充足状态，提供手动刷新，其他提供商余额仍待接入；详见 `docs/product/BALANCE.md`。
- Codex 订阅账号详情可查询真实额度窗口、剩余百分比及重置时间；Plus／Team 普通套餐展示 5h 与周额度，其他套餐仅周额度。配额页集中展示跨账号订阅额度及 DeepSeek 余额、异常概况、搜索／筛选和受控刷新，与账号详情共享缓存；额度历史和推理权限验证仍待后续实现。详见 `docs/product/QUOTA.md`。
- Request history with success/failure classification.
- 仪表盘提供本地今日／7 天／30 天运行总览、请求／Token／成功率／完整耗时、结果趋势、待处理问题、最近请求及资源／客户端配置概况；资源默认读缓存，官方查询由用户主动触发。独立时间桶保留明细清理后的聚合，历史缺口明确标示。详见 `docs/product/DASHBOARD.md`。
- 已实现持久请求明细、输入／输出 Token 累计及每日／成员／模型汇总；删除账号与明细淘汰不减少累计。请求默认保存 10000、设置可调整，日志缓存最多 5000，request_id 双向关联。Team 同时按邮箱和 account_id 识别。详见 `docs/product/ACTIVITY.md`。
- Provider health detection.
- Client integration management for external AI tools.
- Local HTTP Control API for CLI/agent/script/MCP access.
- Windows desktop UI, system tray, single instance, autostart, auto-update.

## Non-Goals

Explicitly out of scope for the current stage:

- Cloud sync, multi-user accounts, or any server-side component.
- Mobile applications (iOS/Android build assets exist in the Wails template but
  are not product targets).
- A billing or payment system.
- Prompt management, conversation storage, or any LLM content itself.
  OmniProxy moves requests; it does not store their content beyond metadata.
- Support for provider-specific features beyond the unified adapter interface.
- macOS/Linux desktop support (planned after the Windows product is complete;
  the architecture keeps this possible).
- Replacing or wrapping the provider SDKs beyond request/response translation.

## Constraints

- Local-first: the gateway binds to loopback; no data leaves the machine except
  to the AI providers the user configured.
- 密钥字段保存在受保护凭证存储中，数据库使用不透明引用；用户要求的自动账号名称允许保存前最多 8 个字符。日志不得包含密钥或这些名称。
- Development and production environments must not share data directories.
- The Go core must remain independent of Wails and React so it stays testable
  and reusable (e.g. behind a future CLI).

## Success Criteria

- A configured client can complete an AI API request through the local gateway.
- When one account fails or is exhausted, requests are retried on another
  eligible account without user intervention.
- The user can see per-account quota, usage and recent request outcomes.
- Client configurations can be applied and restored from the UI without manual
  file editing.
- The app starts on boot, runs in the tray, and updates itself.

TBD: concrete performance targets, exact quota-refresh semantics per provider,
and the first released version number.
