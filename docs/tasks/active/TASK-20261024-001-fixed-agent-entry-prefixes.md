# TASK-20261024-001: 固定 Agent 入口前缀（Codex 按 v1 方式接入）

## Status

In Progress（实现与本地验证完成，等待实机客户端验收）

## Created

2026-10-24 01:00（Asia/Singapore）

## Request

用户要求：把网关写给各 Agent 的 Base URL 从统一 `/v1` 改为「每个 Agent 一个固定前缀」，其中 **Codex 按旧版 v1 项目的方式处理**（`http://127.0.0.1:PORT/codex/v1`，保留官方 `auth.json` 与登录），其余客户端使用 `http://127.0.0.1:PORT/<客户端>/v1` 这类前缀地址。

## Goal

- 每个受管客户端有唯一、稳定的本地入口前缀，配置写出的地址与网关接受的路径来自同一张表。
- Codex 使用 `/codex/v1`，保留内置 `openai` 身份、官方 `auth.json` 与登录令牌，不再把接入 Key 放进路径。
- 其余客户端使用 `/<client>/v1`（Claude 系为根前缀 `…/claude`，Goose 为完整端点 `…/goose/v1/chat/completions`）。
- 协议处理、请求体上限、查询参数白名单、并发与统计口径保持不变。

## Background

- 现状：Codex 走 `/agent/codex/opk_<64hex>/v1`（ADR-005，为在保留订阅身份的前提下携带接入 Key）；其余客户端共用根地址 + `/v1`。
- 旧版 v1 项目（`E:\go\new\OmniProxy`）已采用前缀入口：`/codex/v1`、`/claude-desktop`、`/anthropic-router`、`/opencode-router`、`/pi-router`、`/gemini`。其 Codex 侧把占位 Key 写进 `auth.json` 的 `OPENAI_API_KEY`，网关侧**完全不校验**本地 Key。
- 本版不允许完全取消鉴权，因此 Codex 前缀采用「连接启用时按前缀放行」，其余前缀仍强制接入 Key。

## Scope

- 新增 `internal/core/agentendpoint`：客户端 → 前缀的唯一映射（含 `Clients(prefix)` 反查）。
- `internal/core/proxy`：路径折叠、前缀归属校验、`Options.PrefixAccess`、`client_mismatch` 错误码。
- `internal/application/agentconfig`：各 builder 改用固定前缀；Codex 配置不再写 `/agent/codex/<key>/v1`；`codex-models` 目录地址、Claude Desktop 就绪判定同步。
- `internal/transport/wails`：注入 `PrefixAccess`。
- 前端「手动接入」：按客户端生成前缀地址与配置模板。
- 文档：新增 ADR-021，修订 ADR-005 的传输细节，更新 `GATEWAY.md`／`LOCAL_AGENTS.md`。

## Out of Scope

- 不改协议实现、上游适配器、账号与路由、统计口径。
- 不为 Codex 启用 Responses WebSocket（`features.responses_websockets` 仍为 `false`）。
- 不移除 `/v1/...` 与 `/agent/codex/...` 旧路径（一个版本周期内保留兼容）。
- 不提交工作区中其他在办任务的未提交改动。

---

# Analysis

## Related Modules

- `internal/core/agentendpoint`（新增）
- `internal/core/proxy/agent_auth.go`、`handler.go`、`types.go`
- `internal/application/agentconfig/{patch,gateway_agents,opencode,qwen_code,harness,catalog_patch,desktop_ready,keys}.go`
- `internal/transport/wails/gateway.go`
- `frontend/src/features/gateway/gateway-client-configuration.ts`

## 关键事实（来自旧版 v1 与当前实现）

- 客户端自己追加协议路径：Codex `+ /responses`、Claude `+ /v1/messages`、ai-sdk 兼容客户端 `+ /chat/completions`、Goose **不追加**。因此前缀必须与后缀规则一起定义，不能统一拼 `/v1`。
- 旧路径必须归一化为协议规范路径后再进入既有校验，否则 `validModelQuery`（`beta=true`、`client_version`）、`requestBodyLimit`（16 MiB）、`ParseProtocolRequest`（协议与 `ValidClaudeModel`）会各按旧路径判断而静默失配。
- Codex 官方登录模式下 `Authorization` 携带 ChatGPT 令牌，无法用接入 Key 校验；因此该前缀需要一条独立的放行路径，且必须受 `agent_connections.enabled` 约束。

## Dependencies

无新增第三方依赖。

## Risks

- `/codex/v1` 在连接启用时不再要求接入 Key：本机其他进程可复用该入口（仍受 loopback Host 校验与协议/大小限制）。这是对齐 v1 的明确取舍，记录在 ADR-021。
- 已有用户配置中的旧地址不会自动迁移，需要重新「预览 → 应用」；旧地址继续可用，不会立即失效。
- `/claude` 由 Claude CLI 与 Claude Desktop 共用，二者只能靠接入 Key 区分归属。

---

# Implementation Plan

1. `agentendpoint` 包：`Prefix`／`Clients`／`BaseURL` + 单元测试。
2. `proxy`：`resolveAgentPath` 折叠前缀（精确分段匹配），`client_mismatch` 校验，`PrefixAccess` 兜底放行。
3. `agentconfig`：各 builder 以 `BaseURL(client, port)` 为根；Codex 写 `/codex/v1`；`codexCatalogURL`。
4. `wails`：注入 `PrefixAccess`。
5. 前端：连接模板与前缀地址。
6. 测试：前缀折叠、上限保持、跨客户端 Key 拒绝、Codex 匿名放行、`beta=true` 仍放行。
7. 文档与 ADR。

---

# Implementation Result

已完成。

- 新增 `internal/core/agentendpoint`：13 个客户端 → 固定前缀的唯一映射表；`/claude` 同时允许 `claude` 与 `claude-desktop`。
- `internal/core/proxy/agent_auth.go`：新增 `resolveAgentPath`（返回规范路径、前缀、允许客户端）与 `splitAgentPrefix`（完整路径段匹配，不接受 `/codexanything`）；旧 `/agent/codex/<key>/v1` 继续解析且限定 codex。
- `internal/core/proxy/handler.go`：使用折叠后的路径；已鉴权客户端的 Key 与入口不符时返回 `403 client_mismatch`；无 Key 时由 `Options.PrefixAccess` 判定前缀放行；超过 512 字节的凭据不进入 Key 比较（Codex 官方登录令牌是超长 JWT），按无 Key 处理。
- `internal/core/proxy/types.go`：新增 `Options.PrefixAccess`。
- `internal/application/agentconfig`：`patch.go`（Codex `openai_base_url` 直接写 `…/codex/v1`，Claude 系写 `…/claude`）、`gateway_agents.go`／`opencode.go`／`qwen_code.go`／`harness.go`（`…/<client>/v1`）、`catalog_patch.go`（`codexCatalogURL`）、`desktop_ready.go`（按前缀判定就绪）、`keys.go`（`PrefixAccess` 仅在 codex 连接启用时返回 codex）。
- `internal/transport/wails/gateway.go`：注入 `PrefixAccess`。
- 前端 `gateway-client-configuration.ts`：Codex `…/codex/v1`、Claude `…/claude`；Codex 模板不再出现 `<gateway-api-key>` 路径占位。
- 新增测试：`agentendpoint` 映射、`proxy` 前缀折叠/上限/跨客户端拒绝/Codex 匿名放行/`beta=true`、`agentconfig` 前缀放行边界；更新受影响断言。

# Verification

```text
go build ./internal/... .            PASS
go test ./internal/...               PASS
go test ./internal/core/proxy/       PASS
go test ./internal/application/agentconfig/  PASS
pnpm vitest run src/features/gateway PASS（29 files / 193 tests）
gofmt -l internal cmd main.go        空
```

未执行：`task verify` 全量（含 staticcheck 与打包）。工作区存在其他在办任务的未提交改动，全量门禁结果不宜归因于本任务。

实机验收待做：Codex CLI／桌面在 `/codex/v1` 下的登录身份、长会话（16 MiB 上限）、模型目录；Claude CLI／Desktop 在 `/claude` 下的实际请求。

# Git

## Commit

```text
94985cb feat(gateway): Codex 透明转发、网关自有登录与 WebSocket 握手预算
e8f0756 feat(frontend): 激活失败原因、Codex 运行时标签与请求页字段
90681e0 docs: 同步 Codex 透明转发决策、产品文档与 2026-10 任务记录
```

代码随 94985cb 落库；记录随 90681e0 归档。

---
