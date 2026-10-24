# TASK-20261024-008: Codex 入口改为 v1 形态的透明转发流程

## Status

In Progress

## Created

2026-10-24 10:33

## Request

用户要求把 v2 的 Codex 处理流程换成 v1（旧项目 `E:\go\new\OmniProxy`）的处理流程：

- 保留 `openai_base_url = "http://127.0.0.1:PORT/codex/v1"`；
- 其余（转发路径、请求头、请求体、WebSocket、令牌刷新）按 v1 的做法改。

## Goal

Codex 客户端打到 `/codex/v1/...` 的请求，按 v1 的语义转发到官方订阅端点：

1. 前缀之后的任意子路径原样透传（不再有端点白名单），只做 v1 的路径归一
   （去掉 `/v1` 等版本段后拼接上游基址）；
2. 客户端请求头整体透传（含 `originator`、`session_id`、`conversation_id`、
   `OpenAI-Beta`、`User-Agent`），只替换鉴权头；
3. 请求体不再改写（不再强制 `store=false` / `stream=true`、不再删采样参数）；
4. WebSocket 升级真正拨到上游并双向中继；
5. 访问令牌由网关自己续期（v1 的 `refresh_token` grant），失败按 v1 语义
   换账号重试。

## Background

- v2 现状：固定端点 + 固定请求头 + 协议复刻（见
  `internal/provider/chatgpt/client.go`、`internal/core/proxy/handler.go`）。
- v1 行为：透明反向代理 + 自身 OAuth 刷新 + WS 桥（见
  `E:\go\new\OmniProxy\OmniProxyBackend\internal\proxy`）。
- 相关决策：ADR-003（官方登录）、ADR-005（会话标识）、ADR-021（固定前缀，
  其中「不引入通用 URL 转发」一条被本任务取代）。

## Scope

- `internal/core/proxy`：新增透明转发与 WS 中继，入口解析按 v1 语义放宽。
- `internal/application/gateway`：新增 v1 形态的转发会话（候选账号、冷却、
  换号重试），上游基址可配置。
- `internal/infrastructure/codex`：新增 v1 形态的 `refresh_token` 自行续期。
- `migrations/`：`codex_settings` 增加上游基址列。
- 文档：新增 ADR，更新 GATEWAY 产品文档与 CURRENT 看板。

## Out of Scope

- 不覆盖用户 `~/.codex/auth.json`（v1 的「把池账号写进本地登录」不再需要：
  透传时身份仍由网关替换）。
- 不改 Anthropic / Google / DeepSeek 等其它入口的协议流程。
- 不改前端页面与设置文案。

## Existing Behavior

`/codex/v1/...` 只接受 `POST /v1/{responses,responses/compact,chat/completions,messages}`
与 `GET /v1/{models,codex-models}`；其余 404。请求头被丢弃后重建，请求体被改写，
WS 在本地终结后转成 HTTP 调用。

## Expected Behavior

`/codex/v1/...` 前缀之后的内容按 v1 语义处理：路径透传、请求头透传、请求体
原样、WS 真正桥接；`chat/completions` 与 `messages` 仍按协议转换（v1 对
`chat/completions` 也做转换），`/v1/codex-models` 保留本地目录。

---

# Analysis

## Related Modules

- `internal/core/proxy`（入口解析、转发、WS）
- `internal/application/gateway`（账号租约、失败切换、设置）
- `internal/infrastructure/codex`（凭据读取、OAuth 续期）
- `internal/application/agentconfig`（客户端配置写入，保持 `/codex/v1`）

## Dependencies

- 已有依赖 `github.com/coder/websocket`（WS 中继），不新增第三方包。

## Risks

- 透传后请求头数量与体量放宽，需要保留 loopback / Host / 前缀归属校验。
- 令牌进入 `http.Header` 参与网络请求（v1 语义），必须在使用后清零，
  且不得写入日志。
- 上游基址可配置意味着可以指向任意地址；默认仍为官方域名。

---

# Implementation Plan

1. 新增 `core/proxy` 透明转发：路径归一、头部复制/替换、响应原样回传、
   WS 中继；`Options` 增加转发绑定。
2. `handler.go` 改为先解析入口：`/codex/...` 走转发，`chat/completions`
   与 `messages` 继续走协议转换，`/v1/codex-models` 保留本地。
3. `gateway` 实现转发会话：模型路由候选 → 账号租约 → 借出令牌 → 冷却与换号。
4. `codex_settings` 增加 `upstream_base_url`（迁移 042）+ 服务读写。
5. `infrastructure/codex` 增加 v1 形态的 `refresh_token` 续期与内存缓存，
   接到 `subscription` 的令牌借出路径。
6. 补测试、跑 `go build ./...` 与定向 `go test`，更新文档。

---

# Progress

- 已还原 v1 与 v2 两侧实现，确认差异点（路径/头部/正文/WS/续期）。
- 核心透传与 WS 中继已落地，入口解析与转发会话已接入。
- 令牌自行续期与可配置上游基址已接入。

---

# Implementation Result

## Changes

- `internal/core/proxy`：新增
  - `passthrough.go`：透明转发的读取、尝试循环与失败回传；
  - `passthrough_send.go`：构造上游请求、原样回写状态／头部／流；
  - `passthrough_path.go`：v1 的路径归一（剥离 `/backend-api/codex`、
    `/codex`、`/v1` 后拼接基址）与 WS 地址换算；
  - `passthrough_headers.go`：头部透传、鉴权头替换、令牌清零；
  - `passthrough_socket.go`：上游 WebSocket 拨号与双向中继；
  - `entry.go`：入口解析与准入（`/codex/...`、`/agent/codex/<Key>/...`）。
  - `handler.go` 先按入口分流；`types.go` 增加 `Options.Passthrough`；
    `server.go` 增加不跟随重定向的转发客户端。
- `internal/application/gateway`：新增 `passthrough.go`（上游基址、候选取号、
  借出令牌）与 `passthrough_session.go`（冷却、换号、拒绝分类）；上游基址来自
  `codex_settings.upstream_base_url`。
- `internal/infrastructure/codex`：新增 `oauth_refresh.go`（v1 形态的
  `refresh_token` 续期与过期判断）、`auth_json.go`（auth.json 解析与过期判断）、
  `token_refresh.go`（官方登录路径的内存续期缓存）、`auth_file.go`
  （池账号 auth.json 的解析与客户端可加载形态）；`subscription` 的令牌借出改走
  `codex.UseFreshToken`。
- **登录所有权（v1 语义，ADR-024）**：
  - `account` 新增 `codex_auth_json` 凭据类型（`codex_file.go`）：auth.json 存进
    受保护凭据存储，SQLite 只留可展示身份；同一官方账号重复导入只更新凭据；
    `ReplaceCodexAuthTokens` 续期回写不改账号版本与身份。
  - `gateway` 新增 `passthrough_token.go`：池账号分支自己续期并回写，
    与官方隔离登录（`subscription_login`）并存于同一套路由／冷却／换号逻辑。
  - `agentconfig` 新增 `codex_login.go`：一键配置把池账号写成客户端
    `~/.codex/auth.json`（先留 `.omniproxy.bak`），恢复配置时还原并删除备份；
    没有池账号时保留客户端原登录。
  - `transport/wails` 新增 `codex_login.go`：`ImportCodexAuthFromClient` 把本机
    `~/.codex/auth.json` 收进池；启动时把池账号查询装进一键配置。
  - 路由与目录把 `codex_auth_json` 与 `subscription_login` 视为同类 Codex 登录
    凭据（`routing.codexLogin`、`gateway.codexLoginMethod`）。
- `migrations/042_codex_upstream_base_url.sql`：`codex_settings` 增加上游基址列。
- 文档：ADR-023、ADR-024、`docs/product/GATEWAY.md`、`docs/plans/CURRENT.md`。

## Files Changed

- `internal/core/proxy/handler.go`、`types.go`、`server.go`、`agent_auth.go`
- `internal/core/proxy/passthrough.go`、`passthrough_send.go`、
  `passthrough_path.go`、`passthrough_headers.go`、`passthrough_socket.go`、
  `entry.go`（含各自测试）
- `internal/application/gateway/passthrough.go`、`passthrough_session.go`、
  `passthrough_token.go`、`passthrough_flow_test.go`、
  `passthrough_codex_auth_test.go`、`service.go`、`codex_settings.go`、`routing.go`、
  `account_rotation.go`
- `internal/application/account/codex_file.go`、`codex_file_test.go`
- `internal/application/routing/catalog.go`、`plan.go`
- `internal/application/agentconfig/codex_login.go`、`codex_login_test.go`、
  `apply.go`、`restore.go`、`types.go`
- `internal/infrastructure/codex/oauth_refresh.go`、`auth_json.go`、`auth_file.go`、
  `token_refresh.go`、`oauth_refresh_test.go`
- `internal/application/subscription/service.go`、`internal/transport/wails/gateway.go`、
  `internal/transport/wails/codex_login.go`
- `migrations/042_codex_upstream_base_url.sql`
- docs

## Technical Notes

- `chat/completions` 与 `messages` 仍走协议转换：订阅端点只接受 Responses，
  v1 对 `chat/completions` 也做了同样转换。
- 客户端配置继续写 `openai_base_url = http://127.0.0.1:PORT/codex/v1`，
  并保持 `features.responses_websockets=false`：WS 通道已可用，但默认不让
  客户端启用，避免未验证的协议版本被强制打开。
- 令牌为参与网络请求而进入 `http.Header`（v1 语义）；Core 在每次尝试结束后
  清零该头部，且不写日志、不落库。
- 续期回写只替换 `credential_ref`，不动 `updated_at`：否则进行中的请求会被
  `route_configuration_changed` 打断。

---

# Verification

## Tests

- [x] Unit Tests（`internal/core/proxy` 透传路径/头部/响应）
- [ ] Integration Tests
- [ ] E2E Tests

## Quality Checks

- [x] Format
- [x] Build
- [ ] Lint
- [x] Test（Go 全包）
- [ ] Verify

## Verification Result

```text
gofmt -l internal                PASS（无输出）
go vet . ./internal/...          PASS
go test . ./internal/...         PASS
task verify                      未运行（需要前端依赖与打包链，另行执行）
```

新增用例：

- `internal/core/proxy`：路径归一（含基址已带 `/v1` 的情形）、头部透传与鉴权替换、
  429 换号后成功、400 原样回传不换号、WebSocket 双向中继。
- `internal/application/gateway`：`…/codex/v1/responses` 用设置里的基址转发、
  `/v1/responses` 归一为 `/responses`、其他前缀仍按白名单 404；
  池账号（`codex_auth_json`）直接提供上游身份，客户端自带登录被替换。
- `internal/application/account`：导入只把 auth.json 放进受保护存储、同一官方账号
  重复导入不新增账号、续期回写不改版本与身份、不可用载荷一律拒绝。
- `internal/application/agentconfig`：一键配置把池账号写成客户端登录并留备份、
  恢复后还原用户原登录、无池账号时不接管登录。
- `internal/infrastructure/codex`：临近过期判定、`refresh_token` 续期请求字段、
  缺少账号标识时拒绝、池账号 auth.json 的邮箱／套餐解析。

---

# Documentation

- [ ] PRD
- [ ] Architecture
- [x] ADR（ADR-023 透明转发、ADR-024 登录所有权）
- [x] CURRENT
- [x] 产品文档 `docs/product/GATEWAY.md`

---

# Deviations

- 计划：客户端配置打开 `responses_websockets`。
  实际：保持关闭，网关侧 WS 中继可用。
  原因：默认强制开启未验证的 WS 协议版本会让客户端在握手失败时整体不可用。
- 计划：完全照搬 v1 的"导入即覆盖"（无条件用池账号覆盖客户端登录）。
  实际：只在池里存在 `codex_auth_json` 账号时覆盖，并保留 `.omniproxy.bak`
  供恢复；官方隔离登录（`subscription_login`）账号仍按 ADR-003 使用。
  原因：两种凭据来源要能共存，且覆盖用户登录必须是可逆操作。

---

# Git

## Commit

```text
94985cb feat(gateway): Codex 透明转发、网关自有登录与 WebSocket 握手预算
e8f0756 feat(frontend): 激活失败原因、Codex 运行时标签与请求页字段
90681e0 docs: 同步 Codex 透明转发决策、产品文档与 2026-10 任务记录
```

代码随 94985cb 落库；记录随 90681e0 归档；未夹带其他在办任务的改动。

---

# Result

Codex 入口回到 v1 形态：路径、头部、正文、WS、令牌续期与登录所有权都按 v1
语义处理（网关自己保管 auth.json、自己续期并回写、并把它下发给客户端），
客户端 Base URL 仍是 `http://127.0.0.1:PORT/codex/v1`。
