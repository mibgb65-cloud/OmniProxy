# ADR 021: 每个本地 Agent 使用固定入口前缀

Status: Accepted

Date: 2026-10-24

Supersedes: ADR-005 的「内置 openai 指向 `/agent/codex/<本地接入Key>/v1`」一条传输细节

## 背景

ADR-005 为了让 Codex 在不重定义 `[model_providers.openai]`、不动官方登录的前提下携带本地接入 Key，把 Key 嵌进了路径：`http://127.0.0.1:PORT/agent/codex/opk_<64hex>/v1`。其余客户端共用根地址 + `/v1`。

旧版 v1 项目一直使用「每个客户端一个固定前缀」的入口（`/codex/v1`、`/claude-desktop`、`/anthropic-router`、`/opencode-router`、`/pi-router`、`/gemini`），用户要求 Codex 恢复该形态，其余客户端同样按前缀区分。

约束：新版 Codex 禁止覆盖内置 `openai` 表，因此拿不到 `experimental_bearer_token`／`env_key`；官方登录模式下它发送的是 ChatGPT 令牌，不是本应用的接入 Key。

## 决策

- 每个受管客户端有唯一固定前缀，由 `internal/core/agentendpoint` 单一维护；客户端配置写出的 Base URL 与网关接受的路径来自同一张表。
- Codex 使用 `http://127.0.0.1:PORT/codex/v1`（客户端追加 `/responses`），配置不再包含接入 Key；官方 `auth.json`、登录方式、`model_provider` 与会话目录保持原样。
- 其余客户端使用 `/<client>/v1`；Claude CLI 与 Desktop 共用 `…/claude`（客户端追加 `/v1/messages`）；Goose 使用完整端点 `…/goose/v1/chat/completions`。
- 网关先把前缀折叠成协议规范路径 `/v1/...`，再进入既有的协议解析、查询参数白名单、请求体上限、并发与统计。前缀不改变任何协议语义。
- 带接入 Key 的请求必须与入口前缀的受管客户端一致，否则返回 `403 client_mismatch`。
- 无法携带接入 Key 的客户端（当前只有 Codex）由 `Options.PrefixAccess` 判定：仅当该受管连接处于启用状态时按前缀归属放行。超过 512 字节的凭据一律不进入接入 Key 比较，按「无 Key」处理——Codex 官方登录令牌本身就是超长 JWT。
- 旧 `/v1/...` 与 `/agent/codex/<Key>/v1` 路径保留兼容，归属继续由接入 Key 决定。前缀按完整路径段匹配，`/codexanything` 之类近似路径不被接受，不引入通用 URL 转发。

## 后果

- 已应用的客户端配置需要重新「预览 → 应用」才能拿到新地址；旧地址在保留期内仍可用。
- `/codex/v1` 在连接启用时不再要求接入 Key：本机其他进程理论上可复用该入口。这是对齐旧版 v1 的明确取舍，缓解手段仍是 loopback 监听、Host 校验、协议与大小限制、无上游凭据透传。
- `/claude` 由两个 Claude 客户端共用，二者只能靠接入 Key 区分归属与统计。
- Codex 的 Responses WebSocket 继续关闭（`features.responses_websockets=false`），本决策不改变该点。
- 前缀是路由分组而非配额边界：并发槽与请求期限仍是全局的。
