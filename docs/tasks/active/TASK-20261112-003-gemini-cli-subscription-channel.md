# TASK-20261112-003: 新增 Gemini CLI 订阅通道（Code Assist Standard / Enterprise）

## Status

Planned

## Created

2026-11-12 09:15

## Request

用户确认：Gemini CLI 的第二条订阅通道「可以另外建立任务」。

背景：`TASK-20261112-002` 只修 Antigravity 通道（`antigravity` app）。OmniProxy 目前没有
Gemini CLI 自己的登录（`gemini` app），因此「Gemini 订阅」在本产品中等价于 Antigravity 订阅；
使用 Gemini Code Assist Standard / Enterprise 的用户无法把该订阅接入网关。

## Goal

- 新增独立 `gemini` 订阅 provider：OAuth 用 Gemini CLI 的 installed-app 客户端，
  推理走 `cloudcode-pa.googleapis.com` 的 Code Assist `v1internal`，与 Antigravity 通道互不影响。
- 账户必须指定 Google Cloud project（Code Assist Standard/Enterprise 才发放会话；个人账号不服务）。
- 模型列表按 Gemini CLI 的 `listExperiments` 实验位与 `retrieveUserQuota` 桶决定，不按套餐名推断。
- 复用 `TASK-20261112-002` 的传输层与错误语义（429 两义、`prompt_turned_away`、结果码白名单），
  但**不打散** system prompt（magpie 明确：Gemini CLI 的登录按原文发送）。

## Background

参考实现（magpie `internal/provider/google.go`）把 Google 的两个 app 分成
`geminiApp` 与 `antigravityApp`，差异固定为：

| | gemini | antigravity |
| --- | --- | --- |
| OAuth client | Gemini CLI 的 installed-app client | Antigravity 桌面 client（`agy` 内读取） |
| 推理端点 | `cloudcode-pa.googleapis.com` | `daily-cloudcode-pa.googleapis.com` |
| 控制端点 | 同上 | `cloudcode-pa.googleapis.com` |
| `loadCodeAssist` metadata | `{ideType: IDE_UNSPECIFIED, platform, pluginType: GEMINI}` | `{ideType: ANTIGRAVITY}` |
| 未开通时 | `onboardUser` + `tierId` + `cloudaicompanionProject` | `onboardUser` + `tier_id` + `ide_*` |
| 实验位 | `listExperiments`（模型可用性） | `fetchAvailableModels`（picker + 权益） |
| 模型 wire 名 | `gemini-3.5-flash` → `gemini-3.8-flash` 等映射 | 直接用变体 ID |
| 工具 schema | `parametersJsonSchema`（完整 JSON Schema） | `parameters`（受限子集，见 002 结论项） |
| UA | `GeminiCLI/<ver>/<model> (<os>; <arch>; terminal)` | `antigravity/hub/<ver> <os>/<arch>` |
| 包封附加 | `user_prompt_id` | `userAgent`/`requestType`/`requestId`/`sessionId` |

magpie 还要求：Gemini CLI 的登录只对 Standard/Enterprise 生效，未指定 project 时给出
「name it with …」的可操作错误；个人 Google 账号应改用 Antigravity。

## Scope

- `internal/provider/gemini`（新包）或复用 `antigravity` 的传输层抽出公共部分
  （实施时决定，优先「一个 app 参数化」而不是复制代码）。
- OAuth：独立的 client id/secret 约束、PKCE、loopback 回调、身份校验。
- 项目选择：`GOOGLE_CLOUD_PROJECT` 语义（本产品为账号级字段与设置入口）。
- 模型：`listExperiments` 实验位 + `retrieveUserQuota` 桶 + 模型 wire 映射。
- provider 注册、账号类型（`AuthSubscription`）、额度读取复用现有结构。
- 文档：`docs/product/GOOGLE_SUBSCRIPTION.md` 增加第二条通道与差异；新增 ADR 记录
  「同一 Google 订阅拆两条通道」的边界。

## Out of Scope

- 导入 Gemini CLI 本机已有登录（`~/.gemini/oauth_creds.json`）；
  参考实现支持，本任务先只做 magpie 自己发起的登录，导入另立任务。
- 把两条通道合并成一个 provider、或自动在两者之间切换。
- Antigravity 通道的请求形态修复（属于 TASK-20261112-002）。

## Existing Behavior

`internal/provider/antigravity` 只实现 Antigravity；`googlesubscription` 服务的 `CheckGoogleProvider`
把 Google 订阅限定为单一 provider 类型。没有 `gemini` 的 OAuth 客户端常量与端点。

## Expected Behavior

- 账号页可新增「Gemini CLI」订阅账号，指定 project 后可用。
- 网关以 `gemini/<model>` 形式暴露其模型，Chat/Responses/Messages 三种协议可用。
- 未指定 project 或账号不被服务时，返回可操作的固定错误码，不猜测、不降级到 Antigravity。

---

# Analysis

## Related Modules

- `internal/provider/antigravity`（可抽取的传输、错误、签名、流式解码）
- `internal/application/googlesubscription`、`internal/application/account`（provider 白名单）
- `internal/transport/wails/google_subscription.go`（登录绑定与额度）
- `internal/application/modelcatalog`、`internal/application/routing`
- 前端账号新增流程（provider 选择、project 输入）

## Dependencies

无新增依赖；复用 `net/http`。

## Risks

- Code Assist 个人账号已停止发放会话，失败信息必须明确指向 Standard/Enterprise 与 project。
- 两条通道共享 Google OAuth 端点但 client 不同，token 不可互换；凭据必须标记 channel。
- 若从 `antigravity` 抽公共层，需先有 002 的测试基线，避免把 Antigravity 行为带进 Gemini 通道
  （尤其：system prompt 打散只能用于 Antigravity）。

---

# Implementation Plan

1. 抽出两条通道共用的传输与错误分类（`internal/provider/googlecodeassist` 或参数化 `antigravity`），
   保持 `antigravity` 现有对外行为不变。
2. 实现 `gemini` app 的 OAuth（独立 client、PKCE、loopback、身份校验）与项目字段。
3. 实现 `loadCodeAssist`（GEMINI metadata）/`onboardUser`（`tierId` + project）与
   「需要 project」的可操作错误。
4. 实现 `listExperiments` 实验位、`retrieveUserQuota` 桶、模型 wire 名映射与目录输出。
5. 注册 provider、账号类型、额度读取与前端新增流程。
6. 契约测试：OAuth、项目缺失、实验位组合、wire 映射、429 两义（复用 002 的语义）。
7. `task verify`；文档与 ADR；提交。

---

# Progress

- 2026-11-12：按用户确认建立记录；等待 TASK-20261112-002 完成后开始（复用其传输层与测试基线）。
- 2026-11-12：[TASK-20261112-002](../completed/2026-11/TASK-20261112-002-google-subscription-agent-chain-parity.md)
  已完成（`task verify` PASS），其 `prompt_refusal` / `upstream_error` / `refusal_marks` /
  `chat_*` 可直接复用；本任务开始前需先确认「一个 app 参数化」还是「独立包」的抽取方式。

---

# Implementation Result

（未开始）

---

# Verification

（未开始）

---

# Documentation

- [ ] PRD
- [ ] Architecture
- [ ] ADR（新增：Google 订阅两条通道）
- [ ] CURRENT

---

# Git

## Commit

（未开始）

---

# Result

（未开始）

---

# Completed

（未完成）
