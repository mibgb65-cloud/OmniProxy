# HowToFix

HowToFix（<https://howtofix.id>，运营方 NetroIndonesia）是 OpenAI 兼容的模型网关：
一个 `sk-` Key 加一个 Base URL 即可调用其账号可用的模型，按 Token 从预付余额扣费。
适配范围以官方 handbook（<https://howtofix.id/api/handbook/docs>，读取日期 2026-11-15）
为接口事实来源；该站是 `QuantumNous/new-api` 系发行版（`/api/status` 返回
`v1.0.0-rc.38`、`quota_per_unit = 500000`）。

## 接入参数

| 项目 | 值 |
| --- | --- |
| 服务商 ID | `builtin_howtofix`（迁移 049 写入，内置、默认启用） |
| kind | `howtofix` |
| Base URL | `https://api.howtofix.id/v1`（为空时使用该默认值，路径不再追加 `/v1`） |
| 认证 | `Authorization: Bearer sk-...`；官方 Anthropic 形状路由也接受 `x-api-key`，本适配器只用 Bearer |
| 协议 | 只发送 `POST /v1/chat/completions`；`GET /v1/models` 用于目录同步 |
| 余额 | `GET /v1/dashboard/billing/subscription` 与 `/v1/dashboard/billing/usage`（同一 Key） |

站点首页声明的 API host 是 `api.howtofix.id`；文档称 `howtofix.id/v1` 返回 404，
实测 apex 的 `/v1/*` 也会被该网关处理，但本适配器固定使用 `api.howtofix.id`，不依赖这处差异。

## 模型目录

目录按账号所属分组发放（`/api/pricing` 显示分组为 `cli`／`free`／`vip`／`default`，
例如 `hy4` 只在 `vip` 分组），因此**没有内置预设**：添加账号后网关会在后台自动同步一次；
也可在路由页选择该账号执行「同步官方模型」重新拉取。同步调用 `GET /v1/models`
并写入该账号的目录快照。

- 自动同步只在「该账号当前目录版本还没有任何模型」时发起，应用启动、账号增删改、
  服务商启停与手动同步成功后都会触发一次；同一运行内按「账号 + 凭据版本」去重，
  失败保留上次成功目录，不无限重试。
- 账号页的余额提示不再对 HowToFix 显示「余额尚未接入」，因为余额已接入（见下）。

- 上游若返回 `type`，只接受 `type=chat` 的条目；未返回 `type` 时全部收录。
- `context_window` 在 1–10,000,000 之间时作为上下文窗口保存；缺失或越界保持未知。
- `capabilities` 含 `vision` 时声明 `text` + `image` 输入，含 `tools` 时声明函数工具；
  没有 `capabilities` 字段时能力保持未知，不默认开启工具或图片。
- 目录上限 1000 条、正文上限 1 MiB；空目录、非 JSON 或非 200 一律按同步失败处理，
  保留上次成功目录，不静默降级为「无模型」。
- 公开价格页当前列出 12 个模型，其中只有 `deepseek-v4.1-flash`、`deepseek-v4.1-flash-1017`、
  `glm-5.3-flash` 标明 1M 上下文，`grok-4.6` 标明 500K，`glm-5.3-flash` 与 `grok-4.6`
  带 `Reasoning`／`Tools`／`Vision` 标记；其余模型的上下文与能力均未公布。
  这些数字只作说明，实际能力以账号同步结果为准。

## 协议与转换

- Chat Completions：原样转发客户端正文，响应（JSON／SSE）原样回传，Token 用量按上游
  `usage` 记账。
- Responses／Messages：网关先用既有 `agentprotocol` 转换成 Chat，再按调用方原协议编码回包。
  转换路径的 Token 用量保持未知（不推测数值）。
- handbook 的 overview 称 Anthropic `/v1/messages` 与 Gemini 路由是「原生直出」，
  但本适配器未经真实 Key 逐项验证，因此**不把转换说成原生**，也不发送 `x-api-key`。
- 不注入 `stream_options`、`reasoning_effort` 等上游未在本适配器验证的参数；
  不发布推理档位选择（公开价格页只有 `Reasoning` 标记，没有档位枚举）。

## 余额

HowToFix 有预付余额，账号卡与配额页按账号显示美元剩余额度：

- 剩余额度取 `GET /v1/dashboard/billing/subscription` 的 `hard_limit_usd`；该部署把
  `soft_limit_usd`／`system_hard_limit_usd` 返回成同一个值，因此不单独展示额度上限。
- 金额四舍五入到分只用于展示，不参与本应用记账；上游未返回时显示「—」，不当作零。
- 剩余额度为 0 或负数时标记为「不足」；查询失败显示固定错误码提示，不显示过期数值。
- `GET /v1/dashboard/billing/usage` 在本部署中始终返回 `total_usage: 0`，不反映真实支出，
  因此不读取、也不显示「已用」；本机 Token 用量仍按请求记账独立统计。

真实 Key 复核（2026-11-15，账号自带的测试 Key）：

- `GET /v1/models` 返回 12 个模型，每条字段为 `id`／`object`／`created`／`owned_by`／
  `supported_endpoint_types`（全部为 `["openai"]`），没有名称、上下文或能力字段，
  因此目录同步后能力保持未知；`hy4` 不在该账号分组内，说明目录确实按分组发放。
- `POST /v1/chat/completions`（`deepSeek-v4.1-flash-1017-sg`，`max_tokens: 1`）返回 200，
  `usage` 为 31 输入／1 输出 Token。
- 同一次请求后 `hard_limit_usd` 从 `50` 变为 `49.999998`，即余额按美元浮点返回、
  扣费粒度是上游整数额度单位（`quota_per_unit = 500000` 时约 $0.000002），
  所以界面按分展示只适合看余额量级，不适合核对单次请求支出。
- 未验证：Anthropic `/v1/messages` 与 Gemini 路由是否真为原生直出（本适配器不使用）。

## 边界与未接入

- 未接入：Embeddings、Rerank、Images、Audio、Videos、Moderations（网关没有对应协议位），
  以及 Files／Fine-tunes（上游返回 `501 api_not_implemented`）。
- 未接入：分组切换、每日签到、控制台会话接口（需要 apex 会话态，不是 `sk-` Key 的能力）。
- 限流：标准客户端 20 RPM、官方 CLI 120 RPM，超出返回 429 并带 `Retry-After`，
  沿用网关既有的可安全切换与冷却语义，不额外放大重试。
- 401／403／429 按既有错误语义回传；402（预付余额不足）原样回传客户端，
  不触发同池换号，避免用其他 Key 承担本次支出。
- 受管客户端（Claude Code、Codex、DSH、OpenCode 等）的模型目录来自「已配置的路由」，
  建好路由后自动出现，不需要为每个客户端单独接入。
- 真实 Key 的端到端调用由用户执行；仓库验证使用 loopback 假上游。
