# ADR 026: WorkBuddy AI 订阅通道与请求形态对齐

## Status

Accepted

## Date

2026-11-16

## Context

WorkBuddy AI 是腾讯 CodeBuddy 计划的国际版桌面 Agent（`www.workbuddy.ai`，bundle
`com.workbuddy.workbuddy-ai`）。它的模型不是通过公开 API Key 售卖，而是随桌面客户端登录的
订阅发放；协议只存在于客户端与 `https://www.workbuddy.ai` 之间（`/v2`、`/v3`、`/billing/meter`），
没有官方文档。

社区插件 `@magpie-community/opencode-workbuddy-auth`（MIT，npm 0.2.36）实现了该客户端协议：
浏览器登录轮询令牌、`/v2/plugin/auth/token/refresh` 续期、`/v3/config` 目录、
`/v2/chat/completions` 流式对话、`/billing/meter/get-user-resource-summary` 额度。
其请求带官方桌面客户端的 UA 与一整套指纹头（`User-Agent: WorkBuddy/5.9.3 …`、
`X-Product`、`X-IDE-*`、`X-Agent-*`、`X-Stainless-*`、gzip 请求体、会话 id 头），
UA 不匹配时上游返回 `10085`。

OmniProxy 已有三种订阅形态：Codex 的官方组件隔离登录（ADR-003／ADR-024）、
Google 的 OAuth + 网关续期（ADR-013）、DuMate 的登录态导入（Cookie）。WorkBuddy AI 与
它们都不同：令牌用浏览器轮询取得（非标准 OAuth2），所有权必须归网关，目录与额度都来自
同一份登录。

用户要求「完整订阅登录 + 只接国际版 + 首版含 Credits 三窗口」。

## Decision

- **原生 Go 适配器**，不引入 JS/Bun 插件运行时（沿用 DuMate 的结论）：
  `internal/provider/workbuddyai` 负责协议、`internal/application/workbuddy` 负责登录与续期，
  `internal/application/account/workbuddy.go` 负责受保护凭据。
- **独立认证类型** `workbuddy_login`，不复用 `subscription_login`（后者是 Codex 隔离登录的
  代号，见 ADR-024），避免 Codex 的额度、重置卡与清理逻辑把 WorkBuddy 账号当成自己的。
- **网关拥有令牌**：访问令牌、续期令牌与过期时间经 DPAPI 保存，SQLite 只存
  `credential_ref`、`auth_state`、账号标识与套餐；令牌只在后端回调内短暂解密，
  用完清零，不返回前端、不写日志。
- **续期语义**：请求前 5 分钟内到期才续期，单飞；续期成功在同一事务内重绑
  `account_models` 的授权版本，**普通续期不使目录失效**；重新登录（身份可能变化）不重绑旧目录。
- **请求形态向官方客户端对齐**：UA、指纹头、gzip、会话 id 按参考实现发送。该行为属于
  绕开上游客户端识别，与 ADR-013 的 system prompt 打散同级；风险写在产品文档，
  OmniProxy 不承诺账号不受限制。不提供「按原文发送」开关：UA 是硬要求，关闭只会让请求全部失败。
- **上游只接受流式**：所有请求以 `stream: true` 发出；Chat 客户端要流式时原样转发 SSE，
  要非流式或跨协议时由网关聚合成完整回包再按客户端协议编码。
- **额度只读展示，不参与调度**：Credits（计划内）、Extra credits、Total credits 三窗口
  来自官方 meter；`Total credits` 才代表账号是否可用。调度不得依据本机对上游额度的解读，
  避免解读错误影响可用性。免费模型每日 token 窗口（需按本机记账重算）不在首版。
- **只接国际版**：国内版（`copilot.tencent.com`）的 Claude 模型走 Anthropic Messages、
  另有签到语义，作为独立议题；桌面登录文件（5.6+ 加密）与 API Key 不做。

## Alternatives Considered

- **把 magpie 作为上游**（OmniProxy → magpie 网关 → 插件）：两个应用、两套路由与额度语义，
  自定义服务商当前也不可路由；只适合临时验证，不能作为产品通道。
- **嵌入社区插件**（Bun 或 Node sidecar）：引入第二个运行时与供应链面，且插件 API 是
  OpenCode 的，不是 OmniProxy 的；DuMate 已否决同类方案。
- **复用 `subscription_login`**：会让 Codex 的授权扫描、额度与清理逻辑误认账号。
- **复用 Codex 的续期回写**：`ReplaceCodexAuthTokensAt` 换引用时不重绑目录，
  会导致模型在下次续期后从路由里静默消失。
- **用桌面登录文件导入**：WorkBuddy 5.6+ 加密（`$wbEncrypted`），参考实现也读不了。

## Consequences

- 用户添加账号后即可在任一受管客户端使用 WorkBuddy AI 的模型；账号卡与配额页展示 Credits，
  账号卡保留刷新、重新登录与退出。
- 上游协议不是公开契约：端点、错误码与版本指纹随时可能变化，必须用固定错误码暴露失败，
  不静默降级；版本常量在发版后需要更新。
- 上游账号存在被识别与限制的风险，且未被验证；产品文档明确写出，不作为稳定性承诺。
- 免费 token 窗口与额度调度未接入：账号额度耗尽时仍可能被路由选中，直到上游拒绝；
  这是可接受的保守默认，后续单独立项。
- 真实账号验证会产生积分消耗，需用户授权后执行；在此之前所有行为均为契约测试与回环假上游。
