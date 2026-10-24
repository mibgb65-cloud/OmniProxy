# TASK-20261024-007: 修复 Codex WebSocket 首帧 401 重连，恢复官方 Ultra 档位

## Status

In Progress

## Created

2026-10-24 10:40（Asia/Singapore）

## Request

用户两点反馈（附 Codex 桌面端截图与「OpenAI 账号设置 › Codex WebSocket」截图）：

1. 会话中出现「正在重新连接 5/5」与
   `unexpected status 401 Unauthorized: unauthorized`。客户端 Codex 默认启用
   WebSocket，是否接受应由 OmniProxy 的「Codex WebSocket」开关决定（当时为开启）。
2. `GPT-6.1-Sol` 等 GPT 模型应有低／中／高／极高／Ultra 强度，Ultra 不能被去掉。

## Goal

- 固定前缀入口（`/codex/v1`，不写接入 Key、使用官方登录令牌）的 WebSocket 会话
  不再在首帧被网关判为 `unauthorized`，开关保持唯一决策点。
- Codex 目录重新发布官方已声明的 `ultra` 档位，并能在请求期原样还原。

## Background

### 现象与证据

- 本机 `~/.codex/config.toml`：`openai_base_url = "http://127.0.0.1:3000/codex/v1"`，
  配置中不写接入 Key（ADR-021 固定前缀），握手由 `PrefixAccess("/codex")` 放行。
- `request_records` 中 `client_id='codex'` 的记录：2026-10-23 有 26 条 `websocket`，
  2026-10-24 只有 `http`，没有任何 `websocket` 记录 —— 说明 WS 在
  `beginTrace` 之前就被拒绝，与「握手成功、首帧被判 unauthorized」一致。
- `internal/core/proxy/websocket.go` 每帧复核调用
  `s.opts.Authorize(ctx, key)`，与握手阶段的判定不一致：
  握手时官方登录令牌（超长 JWT）被跳过比较、由前缀放行；
  逐帧复核却把这个 JWT 直接交给接入 Key 比较（`Client` 只接受 68 字节 `opk_` 值），
  必然返回 `false` → `socketError(..., "unauthorized")` → 401 错误帧 → 客户端重连。
  同时缺 `handler.go` 里的 512 字节上限保护。
- 升级到固定前缀之前，Codex 使用 `/agent/codex/<opk_key>/v1/...`，逐帧复核命中真实接入 Key，
  因此历史上 WS 正常；前缀切换后即失效。

### Ultra 档位

- 本机 `account_models` 快照中 `gpt-6.1-sol` 的官方档位为
  `low / medium / high / xhigh / max / ultra`（`ultra` 描述为
  “Maximum reasoning with automatic task delegation”）。
- `internal/core/modelcatalog/codex_effort.go` 的客户端词汇表只有
  `minimal / low / medium / high / xhigh`，`ultra` 被当作未知值丢弃，
  且 `variantEffortOrder` 也不含 `ultra`，变体折叠与元数据校验同样不接受它。

## Scope

- `internal/core/proxy/handler.go`：把前缀授权方式传给 WebSocket 会话。
- `internal/core/proxy/websocket.go`：逐帧复核与握手判定一致（前缀授权 → `PrefixAccess`；
  接入 Key → `Authorize`，并保留 512 字节上限）。
- `internal/core/modelcatalog/codex_effort.go`：客户端词汇表补 `ultra`。
- `internal/core/modelcatalog/effort_variants.go`：档位顺序与 ` (ultra)` 变体后缀。
- 相关测试；`docs/product/LOCAL_AGENTS.md` 网关 WS 与档位说明。

## Out of Scope

- 不改变「Codex WebSocket」开关语义与默认值，不新增客户端配置项。
- 不把上游 `none`/`max` 之外的取值改写成 `ultra`：`max → xhigh` 映射保持不变，
  避免为第三方适配器凭空声明 Ultra。
- 不修改 Codex 身份、登录、会话或 `auth.json`。

---

# Implementation Plan

1. `handler.go`：记录本次请求是否由 `PrefixAccess` 放行，并随 WS 会话传入。
2. `websocket.go`：新增 `socketAccess` 与 `reauthorize`，逐帧按同一判定复核；
   前缀授权时复核前缀启用状态，接入 Key 时复核 Key，超长凭据不进入比较。
3. `codex_effort.go`／`effort_variants.go`：词汇表与档位顺序加入 `ultra`。
4. 回归测试：无接入 Key 的前缀 WS 会话可用、撤销前缀授权后下一帧 401、
   超长令牌不进入比较、`ultra` 发布与还原。
5. `go test` 定向包 + `task verify`。

---

# Implementation Result

## WebSocket 逐帧复核

- `internal/core/proxy/handler.go`：新增 `byPrefix`，记录本次握手是否由 `PrefixAccess`
  放行；WS 分支改为传入 `socketAccess{key, prefix, client, byPrefix}`。
- `internal/core/proxy/websocket.go`：新增 `socketAccess` 与 `(*Server).reauthorize`：
  - `byPrefix` 为真 → 复核 `PrefixAccess(prefix)`，停用即 `401 unauthorized`；
  - 否则 → 保留 512 字节上限后复核 `Authorize(key)`；
  - 复核结果同步回 `access.client`（客户端归属），不再使用独立 `client` 变量。
- 逐帧复核不再把官方登录令牌送进接入 Key 比较，与握手判定一致。

## Ultra 档位

- `codex_effort.go`：客户端词汇表补 `ultra`（`minimal/low/medium/high/xhigh/ultra`）。
- `effort_variants.go`：`variantEffortOrder` 补 `ultra`，`variantSuffixes` 补 ` (ultra)`，
  变体折叠与 `ValidateMetadata` 都接受该档位。
- `max → xhigh`、`none → low` 映射保持不变，不为第三方适配器凭空声明 Ultra。

## 测试

- 新增 `internal/core/proxy/websocket_prefix_auth_test.go`：
  - `TestCodexWebSocketSurvivesPrefixedAuthorization`：1400 字节官方登录令牌 + `…/codex/v1`
    的会话首帧完成；断言超长令牌 0 次进入接入 Key 比较；停用前缀后下一帧 `401 unauthorized`。
  - `TestCodexWebSocketRevokesDisabledAccessKey`：带接入 Key 的会话在 Key 停用后下一帧 `401`。
- `codex_effort_test.go`：新增 official-gpt 用例（`low/medium/high/xhigh/max/ultra`
  → 发布 `low/medium/high/xhigh/ultra`），并断言 `xhigh`／`ultra` 各自原样还原。
- `effort_variants_test.go`：未知档位改用 `turbo`，新增 `ultra` 变体合法的断言。

# Verification

```text
gofmt -l internal cmd main.go                    空
go vet . ./internal/...                          无输出
staticcheck ./internal/core/proxy ./internal/core/modelcatalog   无输出
go test ./internal/core/proxy ./internal/core/modelcatalog        PASS
go test ./internal/application/{gateway,modelcatalog,agentconfig,routing} \
        ./internal/infrastructure/codex                          PASS
go test . ./internal/...                        （见下）
```

缺陷复现：临时恢复旧的逐帧复核（直接 `Authorize(key)`）后，
`TestCodexWebSocketSurvivesPrefixedAuthorization` 立即失败
（`prefixed socket rejected on first frame: "error"`），确认用例覆盖真实根因。

本机数据佐证：`request_records` 中 `client_id='codex'` 在 2026-10-23 有 26 条
`websocket`，2026-10-24 为 0 条（同时段 `http` 正常），与「WS 在记账前被拒」一致；
`codex_settings.websocket_enabled=1` 说明用户截图中的开关确为开启。

未执行：实机 Codex Desktop 重连与 Ultra 菜单验收（需用户本机客户端）；
未执行前端质量门（本次无前端改动）。

# Remaining Risks

- 客户端重连次数与提示文案由客户端决定；本修复只保证网关不再错误拒绝合法会话。
- `ultra` 能否在客户端菜单实际渲染取决于客户端版本是否认识该值；目录已按
  official catalog 声明原样发布，网关请求期按同一值还原。
- 历史 `websocket` 失败记录里没有 `unauthorized`（它在记账前被拒），因此这次修复
  不能回补旧记录，只对新会话生效。

# Git

## Commit

```text
94985cb feat(gateway): Codex 透明转发、网关自有登录与 WebSocket 握手预算
e8f0756 feat(frontend): 激活失败原因、Codex 运行时标签与请求页字段
90681e0 docs: 同步 Codex 透明转发决策、产品文档与 2026-10 任务记录
```

代码随 94985cb 落库；记录随 90681e0 归档。

---
