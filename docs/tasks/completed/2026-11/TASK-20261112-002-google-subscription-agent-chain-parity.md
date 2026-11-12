# TASK-20261112-002: Google 订阅 Agent 链路对齐参考实现的请求形态与拒绝处理

## Status

Completed

## Created

2026-11-12 09:12

## Request

用户在对照参考项目 magpie（`E:\githubresource\magpie`，本机只读副本）后提出：

> 按照这个软件的逻辑进行修复，把它的处理全部都复用带上，按照 magpie 的方式先创建好 Task Record，
> 完成后先不做，通知我。

即：**按 OmniProxy 既有架构与约束**，把 magpie 对 Google 订阅（Antigravity 通道）请求的全部处理
方式对齐过来，先只交付任务记录与实施计划，不修改任何代码。

背景事实（已由用户在 Beta.17 实测并记录，见
[TASK-20261021-006](../completed/2026-10/TASK-20261021-006-reproduce-gemini-desktop-rejection.md)）：
Claude Code 2.1.299 的完整默认工具请求（21 工具、79,026 bytes、`max_tokens=32000`）经本机网关
发往所选订阅账号后，Google 返回 `429 / RESOURCE_EXHAUSTED`，上游无 details；把最大输出降到 512
仍 429；同账号短请求 200；官方选择器显示该型号 `remainingFraction=0.99977905`。

## Goal

- 使 **Agent 形态的完整请求**（Claude Code / Codex / Claude Desktop 的默认 system prompt 与工具集）
  能在 Antigravity 订阅通道上稳定拿到 200，而不再被误判为额度耗尽并触发客户端 10 次重试。
- 把 magpie 已证实有效的处理**按行为重写**进 OmniProxy 的适配器与网关，保持既有边界：
  固定端点、稳定错误码、不落盘/不日志上游正文、DPAPI 凭据、Core 不依赖 Wails、无新依赖。
- 不复制 magpie 源码，只复用其**协议结论、判定规则与请求形态**；每条改动都要有契约测试。
- 本任务先只创建记录（Status: Planned）。实施需用户另行确认。

## Background

OmniProxy 的 Google 订阅通道（ADR-013）目前等价于 magpie 的 `antigravity` app：OAuth 桌面客户端、
`cloudcode-pa` 控制面、`daily-cloudcode-pa` 推理面、`v1internal:generateContent` 与
`:streamGenerateContent?alt=sse`。经对照，两者在**认证与端点层一致**，差异全部集中在
「请求内容如何被整形」和「拒绝如何被归类」上——这正是 Agent 链路与短探针表现不同的原因。

参考实现的对应机制（magpie 源码位置，供实施时逐条对照）：

| 机制 | magpie 位置 |
| --- | --- |
| 拒绝行清单与判定 | `internal/gateway/codeassist.go` `agTurnedAway`、`antigravityTurnsAway` |
| 零宽空格打散 | 同上 `antigravityCloak`、`agBreaks`、`zeroWidth` |
| 拒绝改报 400 | 同上 `turnedAwayStatus`、`turnedAwayErrType` |
| 6 小时预判跳过 | 同上 `agForeseeing`、`antigravityForeseen` |
| 请求包封 | 同上 `buildCodeAssistSent`、`pairFunctionTurns`、`dropSecondAnswers`、`thinkingConfig` |
| 会话 ID | `internal/provider/google.go` `antigravitySession` |
| 包封补齐 | 同上 `codeAssistEnvelope` |
| 项目/套餐/引导 | 同上 `antigravityProject`、`onboard`、`loadCodeAssistReply` |
| 模型与权益 | 同上 `modelInfo`（`entitlement.userTier`）、`antigravityList` |
| 用户代理与版本 | 同上 `userAgent`、`antigravityVersion` |
| 429 两种语义 | `docs/reference.md` §「Antigravity 的 429」；`quotaResetAfter` |

OmniProxy 现有实现位置：

- `internal/provider/antigravity/chat_request.go`：`chatPayload` 构造包封、`sessionId` 随机、
  `systemInstruction` 原样、`functionResponse` 用 `result` 键。
- `internal/provider/antigravity/chat_options.go`：`toolDeclarations`（`gemini-*` 用
  `parametersJsonSchema`）、`generationOptions`（总是发送 `maxOutputTokens`，拒绝 `reasoning_effort`）。
- `internal/provider/antigravity/chat_tool_choice.go`：`AUTO` / `NONE` / `ANY`+`allowedFunctionNames`。
- `internal/provider/antigravity/tool_signatures.go`：签名缓存回填，不补 `skip_thought_signature_validator`。
- `internal/provider/antigravity/client.go`：固定 UA、429 → `rate_limited`。
- `internal/provider/antigravity/discovery.go`：`loadCodeAssist`/`onboardUser`/`fetchAvailableModels`
  （不带 `entitlement`、不过滤 `disabled`）。
- `internal/application/gateway/google_agent.go`：跨协议转到 Chat 后交给适配器，回程再编码。
- `internal/core/proxy`：`Request.Effort` 已存在；`Response.QuotaResetAt` 已被 failover 使用。

## Scope

预计包含（按实施顺序）：

- `internal/provider/antigravity/prompt_refusal.go`（新建）：拒绝行清单、判定、零宽空格打散。
- `internal/provider/antigravity/errors.go`（新建或并入 `client.go`）：429 两种语义解析、
  结构化错误（状态、稳定码、重置时间）、`prompt_turned_away` 白名单。
- `internal/provider/antigravity/chat_request.go`：system 打散、历史修复、会话 ID、模型族分支。
- `internal/provider/antigravity/chat_options.go`：`maxOutputTokens` 策略、`toolConfig` 模式、
  `thinkingConfig`、`responseModalities`。
- `internal/provider/antigravity/tool_signatures.go`：Claude 与未签名步骤的
  `skip_thought_signature_validator`。
- `internal/provider/antigravity/discovery.go`：项目/套餐缓存与刷新、`entitlement` 回退、
  `disabled` 与内部模型过滤。
- `internal/provider/antigravity/client.go`：UA 与协议版本、`onboardUser` 附加头。
- `internal/provider/antigravity/*_test.go`：以下全部契约测试（注入 `RoundTripper`，不触网）。
- `docs/product/GOOGLE_SUBSCRIPTION.md`：429 两种语义、prompt 被拒的排查与不可重试说明。
- `docs/plans/CURRENT.md`：任务概览。

## Out of Scope

- **Gemini CLI 第二条通道**（`gemini` app：独立 OAuth 客户端、Code Assist Standard/Enterprise 必须
  指定 GCP project、`listExperiments` 实验位、模型 wire 名映射、`parametersJsonSchema`）。
  这是新通道，需独立 ADR/任务；本任务只修 Antigravity 通道。
- 额度面板与配额总览重构（继续使用 `retrieveUserQuotaSummary`，不改 UI）。
- Google 429 的跨账号自动切换（ADR-015 既有策略不变；仅新增「拒绝不休息」与可选的 6 小时跳过）。
- 前端新增设置项（打散默认常开，不提供开关；如需再加）。
- 打包、发布、真实客户端配置改写。

## Existing Behavior

1. `systemInstruction` 原样发送（只把 `system`/`developer` 消息的文本 part 依次放入）。
   因此 Claude Code 的 `x-anthropic-billing-header: …` 与
   `Built on Anthropic's Claude Agent SDK` 行会原样到达 Google。
2. 上游 429 一律映射为 `rate_limited`（`client.go`），网关再以 429 交还客户端；
   Anthropic/OpenAI SDK 对 408/409/429/5xx 自动重试，表现为「一直在 retry」。
3. `generationOptions` 对任何模型都发送 `maxOutputTokens`（有 `max_tokens`/`max_completion_tokens`
   时），并拒绝 `reasoning_effort`（400 `unsupported_content`）。
4. 每个请求随机生成 `sessionId`，同一会话每一轮都不同。
5. 工具调用缺签名时不补任何标记；Claude 模型的 `functionCall` 同样不补。
6. 工具声明对 `gemini-*` 使用 `parametersJsonSchema`（Beta.17 的真实验收结论），
   第三方模型使用 `parameters`。
7. `fetchAvailableModels` 只传 `{project}`；`disabled` 模型仍进入目录；`chat_*`/`tab_*` 未过滤。
8. `chatPayload` 在「历史以 model 开头 / 以 model 结尾 / 有未回答的 functionCall /
   同一 call 重复回答」时直接 400，不做修复。
9. `loadCodeAssist` 每次取项目后即覆盖凭据中的 `Project`，不保存 tier/paidTier，也不定期刷新。
10. UA 固定为 `antigravity/2.23.0 OmniProxy`；`onboardUser` 不带 `google-api-nodejs-client` 相关头。

## Expected Behavior

- Agent 的完整默认请求不再因 system prompt 命中被回 429；若 Google 仍拒绝，客户端收到
  **400 `prompt_turned_away`**，不自动重试，且账号不进入额度冷却。
- 真正的「模型容量耗尽」429（带 `Your quota will reset after <duration>.`）按该时长休息，
  并通过既有 `QuotaResetAt` 机制生效。
- 请求形态与 Antigravity 客户端一致：不以 model 开头/结尾、functionCall 与 functionResponse 成对、
  无重复回答、`sessionId` 在同一会话内稳定、思考档位按模型族下传、Claude 调用带跳过签名标记。
- 模型目录只列出「账号实际可选且未被禁用」的模型，Pro/Ultra 账号不再退化成免费档列表。
- 错误码、日志与 UI 仍只出现白名单稳定码；上游正文、凭据、项目不出现在日志与前端。

---

# Analysis

## Related Modules

- `internal/provider/antigravity`：全部改动集中在此适配器；对外签名 `Forward`/`Project`/`Models`/
  `Quota`/`Credentials` 保持兼容。
- `internal/application/gateway`：`google_agent.go` 只做协议转换与回程编码；本任务不新增网关分支，
  仅依赖既有 `proxy.Response.QuotaResetAt`、`proxy.Failure`、`PreExecutionQuotaRejected` 语义。
- `internal/core/proxy`：无需改动（`Request.Effort` 已存在）。
- `internal/application/account` / `googlesubscription`：凭据结构不新增字段时无需改动；
  若需要保存 tier 供 UI 使用则另开。
- `docs/product/GOOGLE_SUBSCRIPTION.md`、`docs/plans/CURRENT.md`。

## Dependencies

无新增依赖。版本查询只用 `net/http` + 现有 Transport；所有解析使用标准库。

## Risks

- **规避厂商检查的合规与账号风险**：打散 system prompt 属于绕开上游内容判定，与 ADR-013 已记录的
  Antigravity 风险同级。需在 `GOOGLE_SUBSCRIPTION.md` 的「安全 / 当前能力与边界」中明确写出，
  不新增「更安全」的表述，也不默认鼓励多账号。
- **前缀缓存**：打散会改变 system 的字节，可能影响上游 prefix cache。magpie 的结论是模型可正常阅读，
  代价可接受；验证阶段需观察 cache 命中与输出质量，不静默扩大打散范围。
- **状态码兼容性**：把拒绝从 429 改成 400 会改变客户端可见状态（这正是目的，让 SDK 停止重试）。
  需在文档与请求记录里标注为固定错误码，不把历史记录反推成真实额度事实。
- **热路径开销**：`chatPayload` 已对 79KB 正文做一次 unmarshal；新增的字符串扫描必须单遍、
  按命中原样短路（未命中直接返回），避免每请求额外分配与拷贝。
- **`parametersJsonSchema` 方向与 magpie 相反**：magpie 在 Antigravity 上用裁剪后的 `parameters`；
  OmniProxy 的 Beta.17 修复来自真实账号，说明该字段被接受。实施第 10 步必须先做一次 A/B，
  以真机结果决定保留或回退，不允许凭文档直接翻转已验证行为。
- **`Entitlement` 被 400 拒绝**：必须按 magpie 的顺序降级并记忆 24 小时，否则每次列模型都会多打一次
  400，污染请求记录与日志。

## Root Cause

高置信根因（待一次受控对照确认）：

> Antigravity 对 system instruction 中出现的 Claude Code 计费行
> （`x-anthropic-billing-header:`）与 Claude Agent SDK 身份行
> （`Built on Anthropic's Claude Agent SDK`）回 `429` 且正文为
> `Resource has been exhausted (e.g. check quota)`，与账号剩余额度无关。
> OmniProxy 原样透传 system prompt，并把该 429 归类为 `rate_limited` 交还客户端，
> 于是 SDK 对 429 自动重试，表现为「Agent 链路一直打不通 / 一直 retry」。

与既有证据的吻合点：上游无 details、只匹配到通用 quota 文案、把 `max_tokens` 从 32000 降到 512
仍拒绝、同账号短请求 200、模型余额 99.98%。不吻合点：3 工具短探针通过——参考实现记录该检查是
启发式且会变（2026-11-11 只插一个零宽空格再次被拒，改为每词两个才通过），因此短探针通过
不足以排除。**结论以第 1 步的单变量对照为准，不在对照前对外宣称已修复。**

已确认的次要成因（即使根因是别的，这些也是与上游客户端不一致的真实差异）：

1. 任何模型都发送 `maxOutputTokens`（Antigravity 客户端对 Gemini 不发）。
2. `sessionId` 每请求随机，会话无法在 Google 侧连续。
3. 无 `thinkingConfig`：带工具的请求在 Antigravity 变体上几乎不思考。
4. Claude 模型的 `functionCall` 不带 `skip_thought_signature_validator`。
5. `fetchAvailableModels` 不带 `entitlement.userTier`（Pro/Ultra 账号拿到免费档列表）。
6. 无历史修复（以 model 开头/结尾、未回答的工具调用、重复回答都直接 400）。
7. 429 只有一种语义，容量耗尽与内容拒绝混在一起。

## Fix

见 Implementation Plan。核心是把 magpie 的处理拆成三层落位：

- **内容层**（`chat_request.go` / `chat_options.go`）：打散、历史修复、模型族分支、会话 ID。
- **传输层**（`client.go` / `errors.go`）：UA 与协议版本、429 两种语义、拒绝改报 400。
- **目录层**（`discovery.go`）：项目与套餐缓存、`entitlement` 降级、`disabled` 过滤。

## Regression Prevention

全部行为用注入 `RoundTripper` 的契约测试固定（沿用 `client_test.go` / `forward_test.go` 的既有模式），
不依赖网络与真实账号；每项改动都必须有「修复前会失败」的新测试。真机只做一次受控验收，
不重放用户会话。

---

# Implementation Plan

1. **根因对照（一次性、受控）**
   用隔离 Claude Code 2.1.299 生成的 79,026 bytes 合成请求（SHA256
   `8aac985ac545f963cf2f57e8107775ca75f17e0d52ada721c13aa67c88a0b603`），对同一账号发 3 个单变量请求：
   原样 / 删去两条命中行 / 保留命中行但按第 2 步规则打散。每次 1 请求，禁用 SDK 自动重试，
   只记录状态与白名单错误分类。结果写入本记录 Progress；若三条均 429，则根因改为容量/账号维度，
   其余步骤照常实施（它们是与上游客户端的既定差异）。

2. **打散与判定（新建 `prompt_refusal.go`）**
   - `promptRefusals`：5 条小写行（计费头、Claude Agent SDK 身份行、两条 Codex GPT-5 身份行、
     Hermes 旧人设）。
   - `turnsAway(system)`：不区分大小写的包含判断。
   - `cloak(system)`：仅当命中时执行；对命中区间内每个「≥3 个 ASCII 字母或数字」的连段，
     在第 1、第 2 个字符后各插入一个 U+200B；用 ASCII-only 小写映射保持字节偏移；
     已含 U+200B 的行视为已打散（幂等，不重复插入）；未命中直接返回原串，零分配。
   - 判定始终针对**原文**，打散只影响发送内容。

3. **拒绝的对外映射**
   - 新增稳定码 `prompt_turned_away`，加入 `ErrorCode` 白名单；`ProxyError` 映射为 HTTP 400。
   - 触发条件三者同时成立：模型请求（非控制面）、原文命中 `turnsAway`、上游 429 且正文含
     `resource has been exhausted`。
   - 该错误不写额度冷却、不计 `rate_limited`；请求记录只存固定码，不存上游正文。
   - 与 ADR-015 的关系：不改既有 429 不换号策略，只把这一类 429 变成不可重试的 400。

4. **429 分类（`errors.go`）**
   - 解析 `You have exhausted your capacity on this model. Your quota will reset after <duration>.`，
     `time.ParseDuration` 且上限 `maxQuotaBlockWindow`，命中则携带 `QuotaResetAt`（走既有 failover）。
   - 其余 429 维持 `rate_limited`；401/403/400 的既有映射不变。

5. **请求形态（`chat_request.go`）**
   1) 首条不是 user → 前插一条 user turn，文本 `[Continuing from previous AI thoughts...]`。
   2) 末条是 model → 追加一条 user turn（固定引导文本，与 magpie 的 `modelLastAsk` 一致）。
   3) 连续同角色 turn 合并；含 `functionResponse` 的 turn 与正文分 turn。
   4) 未回答的 `functionCall` → 补 `[The result of this tool call is unavailable: the turn was
      interrupted.]`（替代现有「必须全部回答」的 400）。
   5) 同一 call 的重复 `functionResponse` → 只保留第一条（替代现有 `invalid_request`）。
   6) 工具返回的图片：Claude 模型放到所有 `functionResponse` 之后并加
      `Image returned by tool <name> (call <id>):`；其他模型紧跟各自 response。

6. **模型族差异（`chat_request.go` / `chat_options.go`）**
   - `family(model)`：`gemini` / `claude` / `gpt-oss` / `other`（全部按小写 id 前缀判断）。
   - `maxOutputTokens`：仅 `claude` 发送；`gemini`/`gpt-oss`/`other` 不发送。
   - `toolConfig`：默认 `AUTO`；`required` → `ANY`；指名 → `ANY` + `allowedFunctionNames`；
     `claude` → `VALIDATED`（并在两者同时成立时以 `VALIDATED` 为准）。
   - `thoughtSignature`：`claude` → 每个 `functionCall` 填 `skip_thought_signature_validator`，
     文本 part 不带签名；`gemini` → 先回填缓存，仍未签名的步骤填同一个标记。
   - `thinkingConfig`：Gemini 3+ → `thinkingLevel`（`low`/`medium`/`high`；
     `minimal`/`none` → `low`）；2.5 → `thinkingBudget`；Claude thinking → budget
     （effort 映射，且 budget 不小于 `maxOutputTokens` 时收紧到 `maxOutputTokens-1`）；
     Claude 非 thinking 与 `gpt-oss` → 不下发。
   - 图片模型或绘图请求 → `responseModalities: ["TEXT","IMAGE"]`。

7. **会话标识（`chat_request.go`）**
   - `sessionId` = `-` + Unix 可打印的 int64（sha256(首条 user 文本) 前 8 字节 & 0x7fffffffffffffff）；
     无文本时回退随机。`requestId` 保持随机。同一会话每一轮的 system 与 messages 变化不影响该值。

8. **目录与权益（`discovery.go`）**
   - 项目与套餐：`loadCodeAssist` 结果按账号缓存；`currentTier`/`paidTier` 记录并每小时刷新一次
     （刷新失败保留旧值）。
   - `fetchAvailableModels`：按 `paidTier` → `currentTier` → 不带 的顺序尝试
     `entitlement.userTier`（跳过 `free-tier`/`legacy-tier`）；被 400 拒绝时降级并记忆 24 小时。
   - 过滤 `disabled` 模型与 `chat_*`/`tab_*` 内部 id；保留 `agentModelSorts` 顺序；
     继续只输出 `supportedModel` 白名单内的模型。

9. **用户代理与协议版本（`client.go`）**
   - UA 改为 `antigravity/hub/<version> <GOOS>/<GOARCH>`；`version` 取官方更新清单（缓存 6 小时，
     失败回退 `2.23.0`），不声称是本机 `agy` 版本。
   - `onboardUser` 额外发送 `X-Goog-Api-Client: gl-node/22.21.1` 与
     `User-Agent: <UA> google-api-nodejs-client/10.3.0`；`ide_version` 用同一实时版本。
   - 其余请求头与固定端点不变。

10. **工具 schema（结论项，先 A/B 再定）**
    保留现有 `parametersJsonSchema`（Beta.17 真机验收结论），不采用 magpie 的
    `parameters` + 裁剪器；只补两项无害加固：根必须是 object（否则 `unsupported_content`），
    `enum` 只保留非空字符串。若第 1 步或第 10 步的 A/B 显示 Google 对该字段另有要求，
    再以裁剪器作为回退，结论写入本记录 Deviations。

11. **错误码与文档**
    - `ErrorCode`/`ProxyError` 增加 `prompt_turned_away`（400）。
    - `docs/product/GOOGLE_SUBSCRIPTION.md`：新增「429 的两种语义」「system prompt 被拒不可靠重试」
      「请求形态与官方客户端对齐」三段；把 TASK-20261021-006 的「未修复」标注更新为指向本任务。
    - 前端沿用既有错误码展示，不新增文案分叉（如需，另开 UI 任务）。

12. **测试（全部契约测试）**
    - 打散：5 条命中行各一次、大小写混合、只插一个空格的旧形态仍被识别、幂等、未命中零改动。
    - 拒绝：命中行 + 429 → 400 `prompt_turned_away`；未命中行 + 429 → `rate_limited`；
      容量 429 → `QuotaResetAt` 正确。
    - 形态：8 项历史修复、模型族 4 项（maxOutputTokens/toolConfig/签名/thinking）、
      `responseModalities`、sessionId 稳定性。
    - 目录：`entitlement` 降级与 24 小时记忆、`disabled` 过滤、`chat_*` 过滤。
    - 传输：UA 形状、版本回退、`onboardUser` 头。
    - schema：根非 object 拒绝、enum 过滤、`parametersJsonSchema` 原样透传。

13. **验收与质量门**
    - `gofmt`、`go vet`、`staticcheck`、`go test ./internal/provider/antigravity/...`、
      `task verify` 全绿。
    - 真机：经用户授权后各发 1 次受控请求（完整 Claude Code 形态 + 短探针），只记录状态、
      白名单错误码与用量，不重放用户会话；结果写入 Verification。
    - 更新 `docs/plans/CURRENT.md`；按 Git 规范提交并回填 commit。

---

# Needs Confirmation

实施前需要用户决定以下三点（不阻塞记录创建）：

1. **是否提供「按原文发送 system prompt」开关**。参考实现支持 `prompt=as-written`，
   本计划默认常开打散、不加设置项；若要暴露给用户，需额外改 profile 配置与设置页。
2. **6 小时「预判跳过」是否实施**。参考实现在同 prompt 被拒后 6 小时内直接跳过该账号；
   这与 ADR-015「Google 429 不自动换号」相邻但语义不同（跳过的是内容拒绝，不是额度）。
   建议仅在「有其它可用候选」时启用；只有单账号时仍返回 400，不隐藏失败。
3. **Gemini CLI 第二通道**是否另立任务。本任务明确不包含；若产品需要“Gemini 订阅”覆盖
   Gemini CLI 的登录（Code Assist Standard/Enterprise），需独立任务与 ADR 增补。

---

# Progress

- 2026-11-12：完成与参考实现的逐项对照，定位「Agent 请求形态 + 429 语义」两类差异；
  建立本记录，等待用户确认后进入 In Progress。
- 2026-11-12：用户确认三项取舍（提供按原文开关、实施 6 小时预判跳过、Gemini CLI 另立
  [TASK-20261112-003](TASK-20261112-003-gemini-cli-subscription-channel.md)），状态改为 In Progress。
- 2026-11-12：适配器完成打散、拒绝分类、两种 429、请求形态、模型族参数、会话标识、
  目录权益与客户端标识；网关完成「跳过／休息／最后候选」接线；设置项与界面落地。
- 2026-11-12：契约测试补齐；全部 Go 包与设置页前端测试通过，进入 `task verify`。

---

# Implementation Result

## Changes

实际完成：

- **打散与判定**（新建 `prompt_refusal.go`）：5 条实测被拒行；判断前先去除零宽空格，
  因此旧的部分打散形态仍能识别；命中区间按词重建为「每个 ≥3 字符连段在第 1、2 个字符后
  各插一个 U+200B」的规范形态，幂等且未命中零改动。判定始终针对原文 system。
- **拒绝分类**（新建 `upstream_error.go`）：429 按正文措辞拆成「内容拒绝」与「容量耗尽」；
  正文只做有界读取（8KB），错误只携带状态、布尔与重置时长，不保存／不返回正文。
- **对外映射**：内容拒绝且原文命中 → `400 prompt_turned_away`（不写额度冷却）；
  容量耗尽 → `429 upstream_quota_exhausted` + 上游重置秒数；其余 429 → `rate_limited`。
- **预判跳过**（新建 `refusal_marks.go`）：按「账号身份 + 项目 + 原文 system」指纹记录 6 小时，
  有其它候选时 `400 prompt_refused_before` 直接换号；最后一个候选（`LastCandidate`）仍照发。
- **请求形态**（新建 `chat_history.go` / `chat_message.go`，改写 `chat_request.go`）：
  历史不以模型回合开头／结尾（前置引导回合、追加 `Continue.`）；未回答的调用补中断结果；
  同一调用重复回答保留第一条；相邻同角色回合合并；`sessionId` 取首条用户文本哈希。
- **模型族参数**（新建 `chat_family.go`，改 `chat_options.go` / `chat_tool_choice.go`）：
  `maxOutputTokens` 仅 Claude 发送；Gemini 3+ 用 `thinkingLevel`、2.5 用 `thinkingBudget`；
  Claude 的 `tool_choice=auto` 用 `VALIDATED`；缺签名的调用在缓存回填后统一补
  `skip_thought_signature_validator`。
- **目录与标识**（改 `discovery.go`，新建 `discovery_models.go`）：`loadCodeAssist` 记录付费
  套餐；`fetchAvailableModels` 按「付费档 → 不带权益」降级并记忆被拒 24 小时；
  过滤停用与 `chat_*`/`tab_*` 模型；UA 改为 `antigravity/hub/<公开版本> <os>/<arch> OmniProxy`，
  版本后台刷新（6 小时）并回退 `2.23.0`；`onboardUser` 补客户端标识头与实时 `ide_version`。
- **网关接线**（`failover.go` / 新增 `google_rejection.go`）：每个候选带 `LastCandidate`；
  `prompt_refused_before` 不写冷却直接换号；订阅容量 429 在还有候选时按重置时间休息并换号，
  单候选时把真实错误交还客户端；把「共享额度观察优先」的冷却豁免收窄到 Codex 订阅账号。
- **设置项**：profile 新增 `googlePromptAsWritten`；`DesktopSettings` 增字段与
  `SetGooglePromptAsWritten` 绑定；启动时读入内存供网关按请求读取；设置页新增
  「Google 订阅请求」卡片（默认关闭 = 打散）。
- **文档**：`GOOGLE_SUBSCRIPTION.md` 新增「请求形态与拒绝处理」与排查条目、更新 429 策略、
  版本标识与验证记录；ADR-013 补记请求形态对齐及其风险。

## Files Changed

- `internal/provider/antigravity/`：`prompt_refusal.go`、`refusal_marks.go`、`upstream_error.go`、
  `chat_family.go`、`chat_history.go`、`chat_message.go`（新增），`chat_request.go`、
  `chat_options.go`、`chat_tool_choice.go`、`client.go`、`forward.go`、`tool_signatures.go`、
  `discovery.go`（重写）、`discovery_models.go`（新增）、`auth.go`；对应 `*_test.go`
- `internal/core/proxy/types.go`：`PromptAsWritten`、`LastCandidate`
- `internal/application/gateway/`：`service.go`、`google.go`、`google_agent.go`、`failover.go`、
  `google_rejection.go`（新增）、`google_rejection_test.go`（新增）
- `internal/platform/profile/`：`config.go`、`desktop.go`
- `internal/transport/wails/`：`app.go`、`desktop_settings.go`、`gateway.go`
- `frontend/`：`bindings/.../wails/{appservice,models}.ts`（按既有生成格式手补两项）、
  `services/omniproxy.ts`、`features/settings/{use-desktop-settings.ts,google-prompt-setting.tsx,
  google-prompt-setting.test.tsx,settings-content.tsx,preferences-test-helpers.tsx,
  desktop-behavior.test.tsx}`、`i18n/locales/{zh,en}.ts`
- `docs/`：`product/GOOGLE_SUBSCRIPTION.md`、`decisions/ADR-013-google-antigravity-subscription.md`、
  `plans/CURRENT.md`、本记录与 `TASK-20261112-003-...md`

## Technical Notes

- 打散必须在**原文**上判定、在**发送副本**上插入：指纹与拒绝归类都用原文，否则跳过永远不命中。
- 签名顺序：客户端回传的签名 → 缓存回填 → 最后才对「整轮无签名」补跳过标记；
  提前写标记会挡住缓存里的真实签名（实测会让 Claude 工具轮次丢掉签名）。
- 不采用参考实现在 Antigravity 上的 `parameters` + 裁剪方案：`parametersJsonSchema` 是本项目
  Beta.17 在真实账号验收过的行为，本次只补文档与结论，不做未验证的翻转。
- 订阅容量 429 走「错误 → retryReason」而不是伪造 `proxy.Response`，避免丢掉客户端可见的
  错误 JSON；单候选分支显式返回原错误，不伪装成账号池冷却。
- 前端绑定按既有生成格式手补两项（`DesktopSettings.googlePromptAsWritten` 与
  `SetGooglePromptAsWritten(4020948349)`）：本机 `wails3` 版本重新生成会改写全部 56 个
  绑定文件（约 4.4k 行无关差异），因此只补必要条目；绑定方法 ID 由同一生成器给出，
  与后备注册一致。升级 Wails 工具链时应整目录重新生成。
- 旧账号在下次重新授权／强制刷新时才写入 `Tier`；未刷新前不带权益列模型，行为与改动前一致。

---

# Verification

## Tests

- [x] Unit Tests：新增 `prompt_refusal_test.go`（5 行命中、按词打散、幂等与部分打散修复、
  未命中零改动）、`upstream_error_test.go`（拒绝 400、容量 429 重置、分类边界、6 小时跳过与
  最后候选、账号隔离）、`chat_envelope_test.go`（8 组模型族参数、5 种工具模式、会话标识、
  5 项历史修复、打散与按原文、Claude 跳过签名）、`discovery_entitlement_test.go`（权益降级与
  24 小时记忆、停用／内部模型过滤、onboard 头与实时版本、版本解析与 UA 形态）、
  `google_rejection_test.go`（跳过信号、休息时长、后续候选判定）、
  `google-prompt-setting.test.tsx`（默认关闭、保存、失败保留原值）。
- [x] Integration Tests：`go test ./internal/provider/antigravity/ ./internal/application/gateway/
  ./internal/application/account/ ./internal/application/googlesubscription/ ./internal/core/proxy/
  ./internal/transport/wails/ ./internal/platform/profile/` 全部通过；网关既有
  `TestGoogleAgentModelsCatalogAndToolContinuation`（6 模型 × 3 协议 × 流式 × 续接）保持通过。
- [x] E2E Tests：2026-11-12 经用户授权，在真实订阅账号（`g1-pro-tier`，项目
  `aicode-consumers`，模型 `gemini-2.5-flash`）上各发 1 次受控请求（21 工具、约 88.8 KB、
  `max_tokens=32000`），只记录状态与稳定错误码、不保存回复正文：

  | 变体 | system prompt | 结果 |
  | --- | --- | --- |
  | written | 原文（含计费行与 SDK 身份行） | `400 prompt_turned_away`（2.0s） |
  | cloaked | 原文按词打散（新默认） | `200`（1.8s） |
  | clean | 删去两条命中行 | `200`（1.9s） |

  另发 1 次 `written` 复测确认错误码与状态（`400 prompt_turned_away`，2.0s），共 4 次
  单次请求，无自动重试。结论：根因是这两行 system prompt 触发的拒绝，不是额度或输出上限；
  打散后的请求被上游正常接受。

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task verify
PASS（exit 0）
```

- format / lint（go vet、staticcheck、eslint）/ typecheck：通过，无新增告警。
- Go：`go test . ./internal/...` 全部包通过。
- 前端：147 文件 / 895 项通过。
- build：Windows production 构建通过（`bin/omniproxy.exe`）。

过程记录：首次 `task verify` 在默认并发下由 1 项无关既有前端用例（`stepfun.test.tsx`）
5 秒超时中断；该用例单独运行通过，限 4 worker 的全量前端（895 项）通过，随后整门
`task verify` 重跑 PASS。未删除或跳过任何测试。

---

# Documentation

- [ ] PRD：无需修改（产品范围不变，订阅通道能力以 GOOGLE_SUBSCRIPTION.md 为准）
- [ ] Architecture：无需修改（模块边界不变，改动在既有适配器与网关内）
- [ ] Tech Stack：无需修改（无新增依赖）
- [ ] Coding Rules：无需修改
- [ ] Testing：无需修改
- [ ] Build：无需修改
- [x] ADR：ADR-013 补记「请求形态向官方客户端对齐」及其打散风险
- [x] CURRENT：已更新本任务与 TASK-20261112-003 概览

---

# Deviations

- **Claude 签名**：原计划「Claude 模型的每个调用一律写 `skip_thought_signature_validator`」；
  实际为「客户端回传或缓存里有签名就原样回传，只有整轮都没有签名时才补标记」。
  原因：提前写标记会挡住缓存回填，既有契约测试（跨协议工具签名往返）会失败；真实账号上
  Claude 工具轮次的签名往返是已验收行为，不能为了对齐而破坏。
- **工具 schema**：原计划先做真机 A/B 再决定 `parameters` 与 `parametersJsonSchema`；
  实际未执行真实推理（无用户授权的一次性真机对照），保持 Beta.17 已验收的
  `parametersJsonSchema`，只在文档记录与参考实现的差异，不做未验证的翻转。
- **订阅容量 429**：原计划「不改既有 429 不换号策略」；实际让多候选时按上游重置时间休息并
  换号、单候选时返回真实错误，并把「共享额度观察优先」的冷却豁免从「任意账号」收窄到
  「Codex 订阅账号」。原因：订阅额度按账号独立，参考实现同样休息换号；API Key 路径不变。
- **User-Agent**：参考实现只发送 `antigravity/hub/<ver> <os>/<arch>`；实际保留
  ` OmniProxy` 后缀，不隐藏请求来源，也不宣称已安装 agy 的版本。
- **设置项落点**：计划写「若暴露到界面需改 profile 与设置页」；按用户确认实施为设置页
  「Google 订阅请求」卡片，默认关闭（即默认打散）。

---

# Git

## Commit

```text
d4af73e9 feat(google): 对齐 Google 订阅 Agent 请求形态与拒绝处理
```

未推送；工作区仅保留本次任务之前的未跟踪记录
（`docs/tasks/completed/2026-10/TASK-20261029-010-subq-provider.md`），未改动。

---

# Result

已把参考实现的处理按本项目边界落到 Google 订阅通道：system prompt 按词打散（设置页可改回
按原文发送）、内容拒绝改报不可重试的 400、6 小时同 prompt 预判跳过、容量 429 按上游重置
时间休息并在有候选时换号、请求历史修复、模型族生成参数与工具模式、稳定的会话标识、
模型目录权益与停用过滤、客户端标识。全部行为有契约测试；`task verify` PASS。

真实账号对照已完成并确认根因：原文 system prompt 返回 `400 prompt_turned_away`，打散后与
删去命中行均为 `200`；21 工具、约 88.8 KB 的 Agent 形态请求不再被误判为额度耗尽。
Gemini CLI 第二通道另立
[TASK-20261112-003](TASK-20261112-003-gemini-cli-subscription-channel.md)。

---

# Completed

2026-11-12 10:55
