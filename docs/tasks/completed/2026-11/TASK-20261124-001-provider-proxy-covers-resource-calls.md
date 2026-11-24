# TASK-20261124-001: 提供商代理覆盖额度、目录与授权续期等全部出站调用

## Status

Completed

## Created

2026-11-24 09:15

## Request

> 像如果配置了某个服务商的请求转发的代理，例如本地代理10808，我需要你对这个服务商的额度查询等等内容全部都要走代理，而不是只有请求走，比如说谷歌。

用户截图显示 Google 订阅账号卡片：额度数值仍然存在，但「可选模型」报「模型列表暂时不可读，请刷新重试」，
下方还有「操作未完成，请检查网络和登录组件后重试」。该账号的代理只在「使用代理的提供商」里按提供商
配置（未开启全局代理）。

## Goal

当某个提供商 `proxy_mode = proxy` 时，该提供商相关的**全部**出站调用都走同一个代理端点，至少包括：

- 账号额度／余额／积分查询（Google 订阅、Codex、WorkBuddy、MiniMax、Kimi、StepFun、HowToFix、DuMate）；
- 模型目录同步与可选模型读取（含 Google 订阅的 `fetchAvailableModels`、API Key 的 `/models`）；
- OAuth／令牌续期与身份、项目核对（Google 订阅 `oauth2.googleapis.com`、`token`、`userinfo`、
  `loadCodeAssist`；Codex `auth.openai.com`；WorkBuddy 刷新与轮询）；
- 登录流程中登录完成前的轮询与身份读取。

推理请求已经按提供商作用域分流，本任务不改变它的行为。

## Background

`internal/core/netproxy` 的判定顺序是「请求上下文里的提供商身份 → 目标主机名匹配 → 全局开关 → 直连」。
推理、透明转发和连通性探针会写入提供商作用域；额度、目录、登录与续期这些调用只包装了 Transport，
却没有把「这是哪个提供商」写进 `ctx`。于是：

- 全局代理关闭、仅对某提供商设置「走代理」时，这些调用落到主机名匹配；
- 主机名匹配只覆盖 `providers.base_url`（内置提供商取目录默认端点），
  Google 订阅实际使用的 `daily-cloudcode-pa.googleapis.com`、`cloudcode-pa.googleapis.com`、
  `oauth2.googleapis.com`、`www.googleapis.com`、`auth.openai.com`、`chatgpt.com` 都不在其中；
- 结果是额度/模型/续期悄悄直连：直连可达时能读到（截图中额度仍有数值），
  不可达时只报「模型列表暂时不可读」「操作未完成」，用户按提供商配置的代理形同虚设。

`docs/product/NETWORK_PROXY.md` §解析顺序第 5 条把「没有上下文的调用按目标主机名匹配」写成既有行为，
但这不足以表达「该提供商显式走代理」。显式 `proxy_mode = proxy` 不应被主机名回退绕过。

## Scope

- 在 `internal/application/account` 提供统一的「账号出站上下文」入口（把 `ProviderID` 写进 `ctx`）。
- 让所有提供商资源类服务在调用适配器前使用它：Google 订阅、Codex 订阅、模型目录、余额、
  MiniMax 额度、WorkBuddy、DuMate。
- 登录流程在提供商 ID 已知时同样写入作用域（Google 订阅 `Exchange`、WorkBuddy `BeginSignIn` /
  `WaitForToken` / `ReadIdentity`、DuMate 导入身份核对）。
- 网关透明转发的令牌续期（`attemptFromCodexAuth`）在选定目标后写入作用域。
- 回归测试：用假客户端断言 `netproxy.ScopeFrom(ctx)` 能读到账号所属提供商。
- 更新 `docs/product/NETWORK_PROXY.md` 与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不改变 `netproxy` 的判定顺序与主机名回退逻辑（继续作为兜底）。
- 不覆盖应用自身更新、Agent 安装下载（`docs/product/NETWORK_PROXY.md` 明确排除）。
- 不覆盖 Codex 官方组件（子进程）内部的网络调用：那部分不在本进程 Transport 内。
- 不新增代理出口分组、系统代理接管或代理软件管理。
- 不改动任何凭据存储格式与数据库结构。

## Existing Behavior

- 提供商 `proxy_mode = proxy` 只对网关推理、透明转发、连通性探针和已显式写入作用域的模型窗口查询生效。
- 额度、模型目录、登录与续期依赖主机名匹配；Google 订阅与 Codex 订阅使用的官方主机不在匹配表内，
  因此按提供商配置的代理对这些调用无效。

## Expected Behavior

- 账号所属提供商一旦设置为「走代理」，该账号的额度、余额、积分、目录、登录轮询、身份核对与令牌续期
  全部经同一代理端点发出；代理不可用时返回 `proxy_unavailable` 等既有代理错误码，而不是悄悄直连。
- 提供商设置为「直连」时，上述调用同样直连。
- 全局代理开关与 `no_proxy` 语义保持不变。
- 推理请求行为不变。

---

# Analysis

## Related Modules

- `internal/application/account`：账号元数据入口，新增出站上下文帮助函数。
- `internal/application/googlesubscription`：Google 订阅额度／目录／续期／登录。
- `internal/application/subscription`：Codex 额度 HTTP 读取与令牌准备。
- `internal/application/gateway`：透明转发的 Codex 令牌续期。
- `internal/application/modelcatalog`：按提供商同步模型目录。
- `internal/application/balance`、`minimaxquota`、`workbuddy`、`dumate`：余额／额度／积分查询。
- `internal/core/netproxy`：只消费已有的 `WithProvider`，不改判定逻辑。

## Dependencies

- 复用 `netproxy.WithProvider` 与 `proxy_mode` 快照，不新增依赖。

## Risks

- 作用域写早于账号复核：若账号在调用期间被重新登录，作用域仍指向同一 `ProviderID`，
  代理策略不会因此改变（`ProviderID` 不随重新登录变化），风险低。
- 登录完成前的调用没有账号记录：直接使用输入里的 `ProviderID`，与保存后的账号一致。
- 测试注入的假 Transport 不会被 `netproxy.Wrap` 包装，因此回归测试断言 `ctx` 中的作用域，
  而不是真实拨号路径。

---

# Implementation Plan

1. `account.Item` 增加 `OutboundContext(ctx)`，返回带 `ProviderID` 的上下文。
2. `googlesubscription`：`credentials` 返回账号项并在回调内使用作用域；额度／模型读取、
   `Models`、登录 `Exchange` 使用作用域。
3. `subscription`：额度 HTTP 读取与令牌准备使用作用域。
4. `gateway`：透明转发的令牌续期在选定目标后使用作用域。
5. `modelcatalog.Sync`：解析出账号与提供商后一次性写入作用域。
6. `balance`、`minimaxquota`、`workbuddy`、`dumate`：在校验账号后写入作用域；
   WorkBuddy 登录阶段使用输入里的提供商 ID。
7. 各包补充作用域回归测试。
8. 运行 `task verify`，更新文档与 `CURRENT.md`。

---

# Progress

- 确认根因：主机名回退只认识 `providers.base_url`，Google 订阅与 Codex 订阅的官方主机不在其中；
  显式「走代理」因此被绕过。
- 新增 `account.Item.OutboundContext`，并接入 Google 订阅、Codex 订阅、模型目录、余额、MiniMax、
  WorkBuddy、DuMate、激活请求与透明转发续期。
- 补齐 8 组作用域回归断言，全部 Go 测试通过。
- 质量门按 `task verify` 的五个阶段逐项执行（`task verify` 的 POSIX `format:check` 配方在当前
  Windows `cmd` 下无法解析），全部通过。

---

# Implementation Result

## Changes

实际完成：

- `account.Item.OutboundContext(ctx)`：把账号所属 `ProviderID` 写入 `netproxy` 作用域，
  空提供商不伪造作用域。
- Google 订阅：令牌续期、身份／项目核对、额度摘要、可选模型、账号目录与登录授权码交换全部带作用域；
  `credentials` 同时返回账号项，供读取方复用同一身份。
- Codex 订阅：`chatgpt.com` 额度与重置卡、`auth.openai.com` 令牌续期带作用域。
- 网关透明转发：借令牌（可能触发 OAuth 续期）前写入已选定目标提供商的策略。
- 模型目录：解析出账号与提供商后一次性写入作用域，覆盖 OpenAI／Google／DeepSeek／River／
  Oxion／HowToFix／MiniMax／Kimi／StepFun／MiMo／WorkBuddy 的目录同步。
- 余额与额度：DeepSeek／Kimi／StepFun／HowToFix 余额、MiniMax 订阅额度、WorkBuddy Credits。
- DuMate：积分、签到与导入身份核对。
- 激活流程发送的最小请求。
- 回归测试：账号作用域、Google 订阅额度／可选模型／续期、Codex 额度、模型目录、余额、
  MiniMax、DuMate、激活请求。

## Files Changed

主要修改文件：

- `internal/application/account/provider_scope.go`
- `internal/application/googlesubscription/{operations,quota,login}.go`
- `internal/application/subscription/{quota,token}.go`
- `internal/application/gateway/passthrough_session.go`
- `internal/application/modelcatalog/sync.go`
- `internal/application/{balance,minimaxquota,workbuddy,dumate,quotaactivation}` 对应服务
- `docs/product/NETWORK_PROXY.md`、`docs/architecture/OVERVIEW.md`、`docs/plans/CURRENT.md`

## Technical Notes

- 作用域仍由 `netproxy.Decide` 消费，判定顺序与主机名回退未改；主机名匹配保留给既无账号也无
  提供商身份的调用（应用更新、Antigravity 版本清单）。
- 登录完成前没有账号记录，直接使用输入里的 `ProviderID`；该值与保存后的账号一致。
- 测试注入的 `RoundTripper` 不会被 `netproxy.Wrap` 包装，因此回归断言检查 `ctx` 中的作用域，
  而不是真实拨号。

---

# Verification

## Tests

- [x] Unit Tests
- [x] Integration Tests
- [ ] E2E Tests

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify（按 `task verify` 的五个阶段逐项执行；标准任务配方中的 POSIX `format:check` 在当前
  Windows `cmd.exe` 下无法解析，改用等价的 `gofmt -l` + `pnpm format:check`）

## Verification Result

```text
gofmt -l main.go internal                  → 空
pnpm format:check                          → All matched files use Prettier code style!
go vet . ./internal/...                    → 通过（无输出）
staticcheck . ./internal/...               → 通过（无输出）
pnpm lint                                  → 0 errors, 2 个既有 react-refresh 警告
pnpm typecheck                             → 通过
go test . ./internal/...                   → 全部 PASS
pnpm test                                  → 164 files / 1006 tests passed
task build                                 → EXIT=0（Windows production，bin/omniproxy.exe）
```

日志：`.gotmp/verify-20261124-001-{vet,staticcheck,fe-format,fe-lint,fe-typecheck,fe-test,build}.log`。

说明：Vitest 在 `workspace-write` 沙箱下因 Vite 子进程管道被拒绝而 `spawn EPERM`，
本次在完整权限模式下运行并通过；该限制与本次改动无关。

---

# Documentation

- [x] PRD（无需改动）
- [x] Architecture：`docs/architecture/OVERVIEW.md` 出站代理段落补充作用域来源
- [ ] Tech Stack
- [ ] Coding Rules
- [ ] Testing
- [ ] Build
- [ ] ADR（沿用 ADR-025，无新决策）
- [x] CURRENT：新增本任务进展

No new ADR required: 判定顺序与三态路由未变，只是补齐作用域写入。

---

# Deviations

None.

---

# Git

## Commit

```text
71fc1664 fix(proxy): 额度、目录与续期等资源调用按提供商走代理
```

---

# Result

按提供商配置的代理现在覆盖该提供商的额度、余额、积分、模型目录、登录轮询、身份核对与令牌续期，
不再依赖 `providers.base_url` 主机名回退；显式「走代理」不会被静默绕过。

---

# Completed

2026-11-24 09:49