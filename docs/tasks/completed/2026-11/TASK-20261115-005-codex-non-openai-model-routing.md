# TASK-20261115-005: Codex 入口把非 OpenAI 模型交回协议管线

## Status

Completed

## Created

2026-11-15 16:50（Asia/Singapore）

## Request

> codex客户端好像没法使用deepseek-v4.1-flash-1017，你看看是怎么回事。该怎么修复

用户截图：请求页 2026-11-15 16:28 的 `Codex Desktop` 记录，模型与账号都是 `—`／`未选择账号`，
状态 `失败`，其中两条显示「已切换 2 次」，错误「未报告」。用户当前在 Codex 桌面端选择
`DeepSeek-V4.1-Flash-1017`（River 通道）。

## Goal

Codex 客户端选择非 OpenAI 服务商的模型（River／DeepSeek 等）时，请求由网关的协议入口
处理（改写模型与档位、走适配器校验与转换），而不是被当成 OpenAI 上游的透明转发；
第三方凭据不得再被发往 `api.openai.com`。

## Background

排查证据（2026-11-15 16:36–16:46）：

- 客户端配置：`~/.codex/config.toml` 的 `openai_base_url = "http://127.0.0.1:3000/codex/v1"`、
  `model = "DeepSeek-V4.1-Flash-1017"`。`/codex/…` 之后全部进入透明转发
  （`internal/core/proxy/entry.go` 的 `isTransparentPath`，ADR-023）。
- 数据库 `request_attempts`：`9449e533…`、`4bdf1101…` 各 3 次尝试、`03fbe0e4…` 1 次尝试，
  服务商都是 `builtin_river`、上游模型都是 `deepseek-ai/DeepSeek-V4.1-Flash-1017`，
  状态码全部 401；路由与换号本身正常。
- 客户端日志：`sampling_error=unexpected status 401 Unauthorized: Unauthorized,
  url: http://127.0.0.1:3000/codex/v1/responses`。
- 同一时刻（16:36）DeepSeek Harness 用同一个 River 账号（`acct_225551…`）、同一个上游模型
  经 `/v1/chat/completions` 返回 200，证明 Key、额度与模型可用。
- `builtin_river` 的 `providers.base_url` 按设计为空串（适配器各自回落官方入口）；
  `attemptFromAPIKey`（`internal/application/gateway/passthrough_token.go`）在基址为空时
  一律回落到 `openai.DefaultBaseURL`，于是 River 的 Key 被发到 `https://api.openai.com/v1`
  → 必然 401，整池 7 个 River 账号被 401 冷却。
- 即使地址正确也不够：透明转发不改写正文。River 需要把公共 ID
  `DeepSeek-V4.1-Flash-1017` 换成官方 ID `deepseek-ai/DeepSeek-V4.1-Flash-1017`、把 Codex
  档位词汇还原成上游取值（`xhigh` → `max`）并做 River 的内容／工具校验，这些只存在于
  协议管线（`internal/application/gateway/river.go`、`reasoning.go`）。
- Codex 目录（`~/.codex/omniproxy-models.json`，71 个模型）发布了 River、DeepSeek、Google
  等全部 `responses` 能力模型，但 `/codex/v1` 实际只能服务 OpenAI 上游：目录与入口能力不一致。

## Scope

- `internal/core/proxy`：透明入口在读正文后询问宿主「本次能否按字节透明转发」；
  不能时把已读正文交回既有 `/v1/responses` 协议处理。仅 `/v1/responses` 与
  `/v1/responses/compact` 参与该判定，其它路径维持原透明语义。
- `internal/application/gateway`：实现该判定（路由目标是 `openai` 才透明转发）；
  `attemptFromAPIKey` 对非 OpenAI 的内置服务商拒绝回落 OpenAI 官方基址。
- 回归测试：非 OpenAI 模型经 Codex 入口改写模型与档位并在 River 基址完成请求；
  OpenAI 模型仍走原透明转发；非协议路径保持透明。
- 文档：`docs/product/GATEWAY.md`、`docs/product/RIVER.md`、`docs/plans/CURRENT.md`。

## Out of Scope

- 不改客户端配置、不改 Codex 模型目录的发布范围。
- 不处理 WebSocket 逐帧改写：握手没有正文，无法判定模型；开启 WebSocket 后非 OpenAI
  模型仍只能依靠客户端回退 HTTP（若仍选中则得到明确错误码，而不是 401）。
- 不新增数据库迁移，不改变 OpenAI 订阅／按量通道的透明转发语义。

## Existing Behavior

- `/codex/v1/…` 一律透明转发；按量账号基址为空时使用 `https://api.openai.com/v1`，
  River／DeepSeek／Kimi 等模型全部 401，并冷却整池账号。
- 协议入口 `/v1/responses` 才会改写模型／档位并使用对应适配器。

## Expected Behavior

- Codex 选择 River 模型时，请求在网关内改写为官方模型 ID、还原档位后发往 River，
  成功后按 Codex 的 SSE／JSON 形态返回。
- OpenAI 模型继续透明转发：路径、头部、正文原样。
- 非协议路径（模型目录、分析事件等）继续透明转发，行为不变。
- 任何情况下第三方凭据都不会被发往 `api.openai.com`。

---

# Analysis

## Related Modules

- `internal/core/proxy/handler.go`、`passthrough.go`、`passthrough_types.go`
- `internal/application/gateway/passthrough.go`、`passthrough_token.go`
- 既有协议入口实现：`handler.go` 后半段、`routing.go`、`river.go`

## Risks

- 透明转发是 Codex 的唯一入口，任何误判都会影响已经工作的 OpenAI 调用；因此判定只在
  模型明确路由到非 `openai` 服务商时返回「走协议入口」，其余（无模型、路由失败、账号
  不可用）保持原行为。
- 重复授权：透明分支与协议分支各调用一次 `authorizeEntry`；它只比较接入 Key，没有副作用。
- 正文已读：协议分支通过 `io.NopCloser(bytes.NewReader(body))` 复用同一份正文，仍受
  `MaxBytesReader` 容量约束；只在透明分支清零正文。

---

# Implementation Plan

1. 创建本记录。
2. 新增 `proxy.TransparentPolicy` 可选接口；`servePassthrough` 读正文后询问宿主，
   返回「不能透明」时把正文写回 `r.Body` 并让 `handler.go` 继续走协议流程。
3. `gateway.Transparent`：解析正文模型 → `resolvePreview` → `selectRoutedTarget`，
   目标 kind 不是 `openai` 时返回 false；无法判定时返回 true（保持原行为）。
4. `attemptFromAPIKey`：内置服务商基址为空且 kind 非 `openai` 时返回
   `unsupported_provider`，不再回落 OpenAI 官方地址。
5. 测试：`core/proxy` 三个用例（改走协议、保持透明、非协议路径不参与判定）；
   `gateway` 一个 River 端到端用例（模型与档位改写、命中 River 基址）。
6. `go test ./internal/core/proxy/ ./internal/application/gateway/`，再跑 `task verify`。
7. 更新文档与看板，提交并回填本记录。

---

# Progress

- 2026-11-15 16:46：完成证据链定位（`request_attempts` 的 7 次 401 全部是 River 目标，
  `attemptFromAPIKey` 回落 OpenAI 基址），确认还需要模型／档位改写。
- 2026-11-15 16:50：创建记录，开始实现。
- 2026-11-15 16:52：Core 侧加入 `TransparentPolicy` 判定与正文回交；Gateway 侧实现判定并
  给按量账号加兜底拒绝；三个新单测通过。
- 2026-11-15 16:56：真实链路验证（只读生产库、真实 River Key）：`/codex/v1/responses`
  + `DeepSeek-V4.1-Flash-1017` 非流式 1907 ms、流式 833 ms，均 200。
- 2026-11-15 17:02：第一次 `task verify` 在 `internal/transport/wails` 的
  `TestSetProxyPortStillAvoidsForeignOccupiedPort` 报 `port_unavailable`；该用例单独重复 5 次
  全通过，属并行端口竞争的偶发失败，未改该包任何代码。
- 2026-11-15 17:18：`task verify` 重跑 EXIT=0。

---

# Implementation Result

## Changes

- `internal/core/proxy`：`PassthroughBinding` 之外新增可选接口 `TransparentPolicy`。
  `servePassthrough` 读正文后先问宿主「能否按字节透明转发」，且只有
  `/v1/responses`、`/v1/responses/compact` 参与判定；宿主返回 false 时把正文写回
  `r.Body` 并返回 false，`ServeHTTP` 继续走同一函数的协议入口（解析、路由、适配器转换）。
  其余路径、WebSocket 握手（无正文）与所有无法判定模型的请求保持原透明语义。
- `internal/application/gateway`：
  - 新增 `Service.Transparent`：按正文解析模型 → `resolvePreview` → 服务商 kind，
    只有 `openai` 才透明转发；无模型、未命中路由、路由或提供商读取失败都返回 true，
    不改变既有失败语义。
  - `attemptFromAPIKey`：内置第三方服务商（River／DeepSeek／Kimi 等）基址为空时返回
    `unsupported_provider`（400），不再把它们的 Key 发到 `api.openai.com`。
- 测试：
  - `internal/core/proxy/passthrough_policy_test.go`：判定为 false 时改走协议入口且透明会话
    不取号；判定为 true 时正文与路径原样送出；非协议路径不参与判定。
  - `internal/application/gateway/passthrough_provider_test.go`：Codex 入口 + River 公共模型
    的端到端用例（非流式／流式），断言上游路径 `/v1/responses`、River 基址与 Key、
    官方模型 ID、档位 `xhigh` → `max`、请求记录模型；另有按量账号默认基址的兜底拒绝用例。
  - `internal/application/gateway/live_codex_test.go`：`OMNIPROXY_LIVE_RIVER=1` 才运行的真实
    链路验证，复用抽出的 `liveProfile`（只读生产库）。

## Files Changed

- `internal/core/proxy/handler.go`
- `internal/core/proxy/passthrough.go`
- `internal/core/proxy/passthrough_types.go`
- `internal/core/proxy/passthrough_policy_test.go`（新增）
- `internal/application/gateway/passthrough.go`
- `internal/application/gateway/passthrough_token.go`
- `internal/application/gateway/passthrough_provider_test.go`（新增）
- `internal/application/gateway/live_test.go`（抽出 `liveProfile`）
- `internal/application/gateway/live_codex_test.go`（新增）
- `docs/product/GATEWAY.md`、`docs/product/RIVER.md`、`docs/product/LOCAL_AGENTS.md`、`docs/plans/CURRENT.md`

## Technical Notes

- 判定只看服务商 kind，不预先解析账号是否可用：账号不可用仍由后续路径报出原有
  `route_*` 错误码，避免在判定阶段改写失败原因。
- 走协议入口时会再授权一次（`authorizeEntry`）。真实 Codex 客户端发送的是官方登录令牌
  （>512 字节，跳过接入 Key 比较）或命中前缀放行，两次都是单条 SQLite 查询，开销可忽略；
  该重复只发生在非 OpenAI 模型的请求上。
- 正文在改走协议入口时不清零（协议层还要解析它），与协议入口既有处理一致；透明分支仍清零。
- 残留限制：透明 WebSocket 握手没有正文，无法判定模型；开启「Codex WebSocket」后
  非 OpenAI 模型只能依赖客户端回退 HTTP，若仍强行走 WS，会因按量账号兜底拒绝拿到
  `unsupported_provider` 而不是 401。

---

# Verification

## Tests

- [x] Unit Tests：`go test ./internal/core/proxy/`、`go test ./internal/application/gateway/` 通过（含新增 3 个 Core 用例、2 个 Gateway 用例）。
- [x] Go 全量：`go test . ./internal/...` 54 个包全部通过。
- [x] Integration Tests：`TestCodexEntryRoutesNonOpenAIModelThroughProtocolPath`（非流式／流式）用假上游验证模型与档位改写、River 基址与凭据、记账模型。
- [x] 真实链路：`OMNIPROXY_LIVE_RIVER=1 OMNIPROXY_PROFILE=production` 运行
      `TestLiveCodexRiverModel`（只读生产库、不写请求记录、真实 River Key）——
      `entry=/codex/v1/responses model=DeepSeek-V4.1-Flash-1017 stream=false status=200 latency_ms=1907`、
      `stream=true status=200 latency_ms=833`，回复包含完整 `response.completed`。
      同一请求在修复前是 3 次 401 后 502。
- [x] E2E Tests：本机运行的是已安装的 `v2.3.1-beta.3`（不含本修复），真机需新构建后复测。

## Quality Checks

- [x] Format
- [x] Lint（0 errors；`reset-credit-list.tsx` 2 条既有 react-refresh 警告）
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task verify  →  EXIT=0
format check / lint（0 errors）/ typecheck 通过
go test . ./internal/...    54 个包全部通过
frontend                     154 files / 935 tests passed
windows production build     bin/omniproxy.exe（24,197,120 bytes）
```

第一次 `task verify` 在 `internal/transport/wails` 的
`TestSetProxyPortStillAvoidsForeignOccupiedPort` 失败于 `gateway: port_unavailable`；
该用例单独 `-count=5` 全部通过，本次未修改该包，判定为全量并行时的临时端口竞争，
重跑通过。

---

# Documentation

- [x] `docs/product/GATEWAY.md`：Codex 透明转发只服务 OpenAI 通道，非 OpenAI 模型改走协议
      入口；协议上限只在判定为协议处理后生效。
- [x] `docs/product/RIVER.md`：Codex 入口的 River 模型同样还原官方 ID 与档位。
- [x] `docs/product/LOCAL_AGENTS.md`：Codex 入口按正文模型判定。
- [x] `docs/plans/CURRENT.md`
- [ ] PRD／Architecture／Tech Stack／Coding Rules／Testing／Build／ADR：不需要（沿用 ADR-023
      透明转发决策，只收窄其适用范围）。

---

# Deviations

原计划在判定阶段调用 `selectRoutedTarget` 复核账号；实际改为直接读提供商 kind。
原因：判定只需要服务商类型，账号可用性由后续路径报出原有错误码更准确，也少两次数据库查询。

---

# Git

## Commit

```text
6ae6dcae fix(gateway): Codex 入口把非 OpenAI 模型交回协议管线
```

---

# Result

Codex 客户端在 `/codex/v1` 入口选择 River／DeepSeek 等非 OpenAI 模型时不再被当成 OpenAI
上游透明转发：网关按正文模型判定后回到协议入口，把公共 ID 换成官方 ID、把 Codex 档位词汇
还原成上游取值、走对应适配器校验与转换，并用该服务商自己的基址与凭据发送；第三方凭据
不会再被发往 `api.openai.com`。OpenAI 模型、模型目录、分析事件等路径的透明语义不变。
真实 River Key 的只读验证与非流式／流式端到端用例均已通过；本机已安装版本不含该修复，
需重新构建安装包后才能实机复测。

---

# Completed

2026-11-15 17:20（Asia/Singapore）
