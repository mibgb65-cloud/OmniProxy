# TASK-20261112-005: Google 订阅覆盖 DSH 等非 Claude/Codex 客户端

## Status

Planned

## Created

2026-11-12 11:55（Asia/Singapore）

## Request

用户指出：`TASK-20261112-002` 只按 Claude Code／Codex 的请求形态做了对齐与真机对照，
「dsh 等其它客户端的接入」还没有处理，要求先明确该怎么处理。

> 你还没有处理 dsh 等其他的客户端的接入，看看该怎么处理呢

## Goal

让 DSH（DeepSeek Harness Web／Desktop）以及其它受管客户端在 Google 订阅（Antigravity）通道上
既能选到模型、也能真实跑通，并把「哪些客户端已确认可用」写成可核对的结论，而不是只覆盖
Claude Code／Codex 两种请求形态。

## Background

`TASK-20261112-002` 的请求整形与真机对照都基于抓取到的 Claude Code Agent 请求
（Messages 协议，经 `agentprotocol.ToChat` 转成 Chat 后发给 Antigravity）。其余客户端
从未在该通道上验证，也没有针对它们的错误提示与档位支持。

当前代码事实（2026-11-12 核对）：

- 受管客户端共 13 个，入口前缀见 `internal/core/agentendpoint`（`bindings`）：
  `codex`、`claude`、`claude-desktop`、`opencode`、`kilocode`、`mimo-code`、`qwen-code`、
  `harness-web`、`harness-desktop`、`kimi-code`、`pi`、`factory-droid`、`goose`。
- 协议分布（`internal/application/agentconfig/clients.go` 的 `isChatClient`）：
  - Chat Completions：OpenCode、Kilo Code、MiMo Code、Qwen Code、Harness Web／Desktop、
    Kimi Code、Pi、Factory Droid、Goose（10 个）；
  - Messages：Claude Code、Claude Desktop；
  - Responses：Codex。
- Google 订阅只提供 Chat 适配（`internal/application/gateway/google_agent.go`）：
  Chat 直接透传，Responses／Messages 先转 Chat、拿到完整结果后再编码回原协议。
- 模型目录对客户端按协议过滤（`internal/core/modelcatalog/compatibility.go` 的
  `ForProtocol`）；`internal/application/routing/model_entries.go` 只给订阅账号的条目
  追加 `responses`／`messages`，Chat 客户端本来就包含订阅模型。
- 因此 DSH 在**协议与目录层面已经能连到 Google 订阅**，缺的不是新的适配器。

## Scope

- 只针对 Google 订阅（`internal/provider/antigravity`）通道的客户端可用性。
- 验证 DSH（Harness Web 为主，Desktop 同配置）在真实订阅账号上的纯文本与工具调用请求。
- 按验证结果决定是否补齐下列缺口，并只在有实测依据时改动。
- 文档：`docs/product/GOOGLE_SUBSCRIPTION.md` 增加「客户端与协议」矩阵与 DSH 的使用步骤。

## Out of Scope

- Gemini CLI 的第二条订阅通道（`TASK-20261112-003`）。
- 图片／语音等多模态能力（当前两通道都明确拒绝）。
- 其它 provider（OpenAI、River、Kimi 等）的客户端矩阵。
- 新增客户端或改客户端安装／检测逻辑（13 个客户端都已存在）。

## Existing Behavior

DSH 的受管配置写入 `$DSH_HOME/profiles/{web,desktop}/cordis.patch.yml`，提供商声明为
`api: openai-completions`、`baseURL: http://127.0.0.1:<port>/harness-web/v1`（或
`/harness-desktop/v1`）、`maxTokensField: max_tokens`、`supportsDeveloperRole: false`。
模型菜单来自本机目录，Google 订阅模型在里面；真机是否 200、以及失败时的错误码没有记录。

## Expected Behavior

- DSH 的模型选择器出现 Google 订阅模型，选中后请求返回 200（含工具调用续接）。
- 失败时返回可操作的固定错误码，并在 `GOOGLE_SUBSCRIPTION.md` 的排查表里能查到对应处理。
- 每个受管客户端在矩阵里都有「已确认／未验证」的明确状态，不用推断代替结论。

---

# Analysis

## Related Modules

- `internal/provider/antigravity/chat_request.go`（字段白名单、system 打散、`reasoning_effort` 拒绝）
- `internal/provider/antigravity/chat_options.go`（`generationOptions` 允许字段）
- `internal/provider/antigravity/prompt_refusal.go`（5 条实测拒绝行）
- `internal/application/gateway/google_agent.go`、`google_rejection.go`
- `internal/application/agentconfig/{harness.go,harness_effort.go,client_entries.go}`
- `internal/application/routing/model_entries.go`、`internal/core/modelcatalog`
- `docs/product/GOOGLE_SUBSCRIPTION.md`

## Gaps

| 编号 | 缺口 | 依据 | 影响客户端 |
| --- | --- | --- | --- |
| G1 | 没有 DSH 等客户端的真机结论，只有 Claude Code／Codex 形态 | `GOOGLE_SUBSCRIPTION.md` 验证记录 | 全部非 Claude/Codex 客户端 |
| G2 | Chat 客户端无法表达推理档位：正文带 `reasoning_effort` 直接 `400 unsupported_content` | `chat_request.go` 的字段拒绝列表、`chat_options.go` 的 `allowed`（两处都不含 `reasoning_effort`）；`proxy.ReasoningEffort` 虽然会读该字段，但请求在更早一步已失败 | DSH、OpenCode、Kilo、Qwen、MiMo、Kimi、Pi、Factory Droid、Goose |
| G3 | 目录不发布推理档位：`harnessReasoningEfforts` 只对 River 模型输出，Google 模型一律 `false` | `internal/application/agentconfig/harness_effort.go` | DSH：Effort 菜单对 Google 模型不出现 |
| G4 | 拒绝行只有 5 条（Claude Code 计费行、Agent SDK 身份行、Codex ×2、Hermes 旧人设） | `prompt_refusal.go` 的 `refusals` | DSH 等自有 system prompt 若命中，只会表现为 `429 rate_limited`（不匹配已知行就不会改报 400） |
| G5 | 文档只写了 Codex／Claude Code 的一键接入 | `GOOGLE_SUBSCRIPTION.md`「使用」第 6 条 | 用户不知道 DSH 能不能用、怎么配 |

## Dependencies

无新增依赖。G2 的档位映射复用现有 `thinkingOptions` / `thinkingBudgets` / `levelOf`。

## Risks

- G2 改动位于 Chat 入口的字段白名单，必须只对 Google 订阅生效；其它 provider 的
  `reasoning_effort` 语义不同（如 River 已有 Messages／Chat 链路），不能顺带放开。
- G3 一旦为 Google 模型声明档位，DSH 会显示 Effort 菜单；若上游对某档位不接受，
  用户会看到一个能点但会失败的菜单。需要只声明实测或契约测试覆盖的档位。
- G4 只允许补「实测命中」的行；猜测会把正常 prompt 也打散，并掩盖真实拒绝。
- 真机验证需要用户本机 DSH 实例与真实订阅账号，不能由契约测试替代。

---

# Implementation Plan

1. **真机验证（阶段 1，先做）**：在用户本机 DSH（Harness Web）选择 Google 订阅模型，
   发一条纯文本与一条带工具的请求；记录模型菜单内容、HTTP 状态、错误码与网关请求详情。
   若用户不方便操作，则由用户在网关里给出 `/harness-web` 的接入 Key，由本机代跑同等请求。
2. **文档矩阵（阶段 2）**：无论验证结果如何，在 `GOOGLE_SUBSCRIPTION.md` 增加
   「客户端与协议」矩阵（13 个客户端 × Chat/Messages/Responses × 已确认/未验证），
   并写清 DSH 的配置入口与选模型步骤。
3. **按错误码定点修复（阶段 3，只在验证失败时）**：
   - `unsupported_content`（`reasoning_effort` / `thinking`）→ G2：Chat 正文接受
     `reasoning_effort`，按模型族映射到 `thinkingLevel`／`thinkingBudget`，并从白名单拒绝列表移除；
     同时评估是否给目录补 G3 的档位声明。
   - `400 prompt_turned_away` 或 `429 rate_limited` 且可复现 → G4：采集 DSH system prompt 中
     实际命中的行，仅补实测行，并保留 6 小时预判跳过行为。
   - 其它错误 → 记录到本记录，必要时另立任务，不在本任务里顺手改传输层。
4. `task verify`；更新文档与 `docs/releases/` 下一条发行说明；提交。

---

# Progress

- 2026-11-12：按用户要求建立记录，只做代码事实核对（协议分布、字段白名单、拒绝行、目录档位），
  未修改任何代码。
- 待用户确认：DSH 里的具体现象（菜单没有模型／请求报错／还没配置），以及是否允许代跑验证。

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
- [ ] ADR（无需新增，除非改变通道边界）
- [ ] `docs/product/GOOGLE_SUBSCRIPTION.md`（客户端与协议矩阵、DSH 步骤）
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
