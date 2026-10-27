# TASK-20261013-017: Atria 内置提供商接入规划

## Status

Completed

## Created

2026-10-13 22:44（Asia/Singapore）

## Request

用户希望新增 Atria 内置提供商，要求先阅读 https://api.atria-asi.ai/docs，分析如何接入并给出规划。

## Goal

形成与现有实现一致、可执行和可验证的接入计划。当前请求范围为调研和规划。

## Scope

- 核实官方接口、认证方式和模型能力。
- 检查提供商预设、账号、模型目录、路由、网关和客户端配置的现有扩展点。
- 记录推荐方案、实现步骤、验收条件与待确认信息。

## Out of Scope

- 本轮不实现提供商、迁移或网关代码。
- 不修改真实账号、密钥、数据库和客户端配置。

# Analysis

## 官方信息

核实日期：2026-10-13。来源：[Atria API 官方文档](https://api.atria-asi.ai/docs)。

| 项目 | 文档确认内容 |
| --- | --- |
| 提供商名称 | Atria |
| 推荐内部标识 | `kind = atria`、`id = builtin_atria`（本项目设计） |
| 默认 Base URL | `https://api.atria-asi.ai/v1` |
| 当前模型 ID | `Atria-Dawn-Preview`，大小写必须保留 |
| 上下文 | 256K；官方 Codex 示例使用 `256000` |
| 输入模态 | 仅文本；图片、PDF 等二进制附件不受支持 |
| 输出限制 | 整数 1–65536；与上下文总容量分别约束 |
| Chat Completions | `/v1/chat/completions`，Bearer Key |
| Responses | `/v1/responses`，Bearer Key |
| Messages | `/v1/messages`，`x-api-key` 和 `anthropic-version: 2023-06-01` |
| 流式响应 | 三种协议均提供 SSE；需要保留各协议的结束事件和 usage |
| 限流 | 同一官方账号的所有 Keys、三种 API 共享 RPM，429 可表示限流或额度耗尽 |

文档给出客户端接入示例，但完整工具类型、工具结果关联、推理参数、会话续接及实际 SSE 事件兼容性仍需契约测试和真实账号验证。

## 当前实现与影响范围

| 模块 | 当前实现与计划影响 |
| --- | --- |
| `internal/application/provider/catalog.go` | 九家预设的名称、默认 URL、展示模型和认证方式；新增 Atria 定义后，现有内置 ID 校验和无凭证探测可复用 |
| `migrations/` | 002 只在首次迁移写入九家预设；必须追加迁移才能覆盖已有安装，禁止修改 002。当前最高版本 017，暂拟 `018_builtin_atria.sql`，实施前重新核对编号 |
| `internal/application/account/` | 已有 API Key 单个／批量添加、编辑和受保护凭证回调，可复用；控制台 Google 登录不等于本应用需要新增 OAuth |
| `internal/provider/` | 实际适配器只有 DeepSeek 和 ChatGPT；拟新增独立 `atria` 包，复用已有代理接口和标准库 HTTP |
| `internal/application/gateway/` | `Service`、`forwardAgent`、`selectRoutedTarget` 当前限定 DeepSeek／订阅 OpenAI；必须增加明确的 Atria 分支，不能让未知种类落入 DeepSeek 转发 |
| `internal/application/routing/` | `validation.go`、`preview.go`、`catalog.go`、`default_routes.go`、`model_entries.go` 均有提供商／模型限制；新增 Atria 后需保持解析、预览、默认映射和实际网关一致 |
| `internal/application/modelcatalog/` | 目前账号同步只支持 DeepSeek 官方发现和 ChatGPT 订阅；Atria 首版采用官方预设，不依赖尚未确认的模型发现接口 |
| `internal/core/modelcatalog/`、`internal/application/agentconfig/` | 已有中立元数据、Codex 目录和 Claude Desktop 稳定别名；复用现有格式适配与配置备份，核对 256000 与文本模态传递 |
| `frontend/src/features/providers/`、`frontend/src/i18n/locales/` | 增加名称、品牌资源映射及中英文文案；保持既有卡片与账号流程 |
| `frontend/src/features/routing/` | `routing-editor.tsx` 提交校验仅允许 DeepSeek，字段选项与 `routing-view.tsx` 模型集合也有限制；需要一起扩展。`model-catalog.tsx` 同步账号筛选仅允许 DeepSeek／订阅 OpenAI |
| `internal/core/proxy/`、`internal/application/activity/` | 已有三协议响应复制、usage 观察、请求生命周期、Retry-After 和提供商统计；优先复用，仅在契约暴露实际问题时修改 |

代码和测试优先于状态文档：CURRENT.md 部分 Known Issues 仍写“订阅推理待接入”，但当前代码已经有 ChatGPT Responses 转发，不按旧描述规划。

## 方案取舍

1. 只加提供商预设：改动最少，可以添加账号，但不能完成实际请求，不作为可用接入的交付标准。
2. 只接 Chat Completions，再转换其他协议：可快速复用现有转换，但跨协议当前先收完整结果再生成 SSE，额外参数和工具语义也可能损失。
3. **推荐：独立 Atria 适配器，三种原生协议直接转发。** 上游本身支持三个端点，现有 Core 已支持三个下游协议，可以保留流式输出并减少转换。

不为单个提供商建立插件框架或重写现有适配器。遵循 ADR-005 的客户端身份保护和 ADR-008 的中立目录，不需要改变技术栈或新增大型依赖。

## 首版能力边界与待核实信息

- 首版包括默认启用的内置卡片、API Key 账号、官方预设模型与元数据、默认／显式路由、三协议文本与经验证的工具调用、SSE、用量统计和既有客户端接入。
- 本地公布的模型表示配置可路由，不表示 Key 已通过认证或官方账号具有调用额度。
- 未在当前文档确认 `/v1/models`。首版直接使用官方预设；模型目录区分“官方预设”和“账号查询结果”，不得把预设当成已通过账号查询，不伪造查询时间。若后续确认发现接口，再加有界解析、凭据版本校验和失败保留旧目录。
- 未在当前文档确认余额／配额查询 API，首版维持未接入状态，不展示推算余额或假零值。
- 官方 Codex 示例列出 low／medium／high 推理档位，但请求参数语义与实际响应需验证，不能据此承诺完整推理兼容。
- Atria 只提供三个 HTTP API 的文档；不宣称上游支持 WebSocket。现有本地 Codex WebSocket 如继续桥接 HTTP Responses，需单独验证续接与终态语义。
- 现有 Chat／Responses 请求体上限为 2 MiB，Messages 为 16 MiB；上下文元数据不等于所有请求都能容纳完整 256K。首版保留现有边界，验收大文本；如出现确实无法支持的代表性输入，再调整有界容量。
- RPM 按官方账号共享，增加同账号 Key 不能提升上限；不硬编码文档中的 RPM 示例值，不新增调度或自动重试。

## Risks

- URL 重复拼接 `/v1`、模型大小写改变或 Messages 认证头错误，会导致所有请求失败。
- 客户端专有工具类型、图片块、Responses 续接参数或 SSE 事件差异，可能使“兼容 API”的简单文本示例通过而实际 Agent 失败。
- 仅修改后端提供商白名单，可能出现默认映射可用但前端无法保存显式路由；必须覆盖整个链路。
- 使用跨协议转换而未说明缓冲行为，可能让用户误以为模型没有流式响应；推荐原生转发避免这个问题。
- 与另一项活动任务的 Codex 长上下文目录工作可能涉及相同文件；实施时重新检查工作树，保留其他任务修改。

# Implementation Plan

1. **预设与升级迁移**
   - 在提供商定义新增 Atria、官方 Base URL、精确模型 ID 和 `api_key`。
   - 追加新迁移，只插入 Atria；不重建九家预设，不覆盖已有启停状态、账号、密钥引用或自定义配置。
   - 补充全新数据库、从 017 升级及重复启动的验证；调整现有写死九家／十条的预期。
2. **原生适配器**
   - 新增 `internal/provider/atria/`：集中声明模型 ID、元数据、允许的协议端点和认证头。
   - Base URL 包含 `/v1`，只拼接 `/chat/completions`、`/responses`、`/messages`；使用固定官方默认地址，不接受正文指定上游地址。
   - 在 `UseAPIKey` 回调内发请求；禁止重定向和客户端认证头透传，保留超时、取消、响应关闭和受限 Retry-After。
   - 按协议验证文本输入块、工具字段和输出限制；拒绝已知不支持的图片／音频／文件输入。校验整数参数 `max_tokens`、`max_completion_tokens`、`max_output_tokens` 的适用范围，边界为 1–65536。
   - 原生 JSON／SSE 返回给已有 Core，并按实际协议调用 `proxy.ObserveResponse`；Chat 流式 usage 的请求开关以契约验证为准。
3. **目录、路由和网关**
   - 在 Atria 模型元数据记录 `ContextWindow = 256000`、`InputModalities = ["text"]`；未知能力保留空值。
   - 从官方预设派生目录，支持启用的 Atria API Key 账号；模型目录 UI 明确预设来源，必要时增加可选来源字段并重新生成 Wails 绑定。
   - 扩展目标验证、账号资格、默认同名映射、显式别名、模型发现与目录元数据继承；各模块调用同一 Atria 模型定义。
   - 添加网关 Atria 分支，重新核对账号归属和启用状态。保持现有 DeepSeek／ChatGPT 分支行为与无隐式重试语义。
4. **界面与客户端接入**
   - 增加提供商中英文名称；优先使用官方品牌资源，未找到时沿用现有通用图标回退，不伪造品牌图案。
   - 修正路由页提供商选项、模型集合及提交校验中的限制，保持后端为最终校验方；覆盖默认映射和手工别名保存。
   - Codex 继续指向本地网关 Responses，Claude Code／Desktop Code 继续指向本地 Messages；上游 Key 只由 OmniProxy 保管，客户端使用既有独立接入 Key。
   - 检查生成目录、256K 与文本模态、Claude Desktop 稳定别名和既有备份／恢复；不整体替换用户原模型目录或改写身份与会话。
5. **验证与文档交付**
   - 先完成下面的最小充分验证，再运行完整 `task verify`。
   - 有本机安全保存的有效 Atria Key 后进行真实客户端验收；不在聊天、测试夹具或日志中保存真实 Key。
   - 实现完成时更新 PRD、架构／技术栈的提供商数量、GATEWAY、ROUTING、LOCAL_AGENTS 中受影响的能力描述及 CURRENT；记录实际测试结果、风险和实现提交。

## 验收条件

- 新安装和旧安装升级后均出现 Atria 内置卡片，默认启用，重新启动不重置用户停用选择；原有提供商与账号数据不变。
- 添加 Atria Key 后，原始模型可被路由，`/v1/models` 和客户端目录包含 `Atria-Dawn-Preview`；没有可用账号时不把它公布为可调用模型。
- 默认映射和显式别名均选择正确提供商／账号，停用或替换账号后即时生效，不能路由到 DeepSeek 或订阅 OpenAI。
- 三协议分别覆盖非流式与 SSE、文本、工具调用及工具结果关联；流式首个有效事件在完整响应结束前到达。
- 请求详情和用量页将官方输入／输出 usage 计入 Atria；缺少 usage 时保持未知，不记为假零或重复累计。
- 契约测试覆盖 URL／认证、未知协议、输出参数类型与边界、非文本输入、401／403／429／5xx、Retry-After、重定向、超时、取消和截断 SSE。
- Codex CLI／Desktop、Claude CLI／Desktop Code 通过现有本地配置流程完成文本及工具任务；Key 不进入客户端目录，原配置可恢复。
- `task verify` 通过；原有 DeepSeek 和 ChatGPT 关键链路的回归测试通过。

# Implementation Result

## Changes

- 已阅读官方文档并检查现有提供商、迁移、账号、模型目录、路由、网关和客户端格式适配。
- 已完成方案比较、分步实施计划、验收条件和风险记录。
- 接入代码尚未实施；当前状态保持 Planned。

## Files Changed

- `docs/tasks/active/TASK-20261013-017-atria-builtin-provider.md`

# Verification

- 官方文档：已人工核对协议路径、认证、精确模型 ID、上下文／输出限制、文本输入及限流说明。
- 本地实现：已人工核对模块扩展点、迁移机制、前后端提供商限制、目录来源及现有代理容量。
- 计划中的 `task verify` 已与 Taskfile 实际定义核对。
- 本轮仅规划，未发送推理请求，未进行真实账号验收；单测、集成测试、构建和 `task verify` 不适用。
- 本任务记录以 UTF-8 无 BOM 保存；交付前检查 Git 工作树和文档差异。

# Documentation

仅新增本任务规划记录。产品能力尚未改变，其余当前能力文档在实施阶段同步。

# Git

Not committed by Agent.

# Result

规划完成。推荐按“预设与迁移 → 原生三协议适配器 → 目录／路由／网关 → 界面／客户端 → 验证”完成可用闭环，首版不依赖尚未确认的上游模型发现或余额查询接口。

---

# Archival (v2.0.0)

- 原状态：Planned
- 2026-10-27：v2.0.0 正式版已公开（标签源码 `c9c48b4a31988a6cc3fa2f4b1d423e61bd10f71b`），
  本文件是 TASK-20261013-017 的**规划草稿**，其实现已由同名 Completed 记录
  `completed/2026-10/TASK-20261013-017-atria-builtin-provider.md`（提交 `015cd6a`）取代。
- 归档动作：文件名追加 `-planned-draft` 后归档，不覆盖既有实现记录，也不删除内容
  （保留当初的调研与规划依据）。状态先按「重复草稿」记 Cancelled，后按用户
  2026-10-27「v2.0.0 之前的任务全部标记为已完成」的要求改为 Completed。

## Completion Note

同一 Task ID 的规划草稿与实现记录重复；实现与验收以同名 Completed 记录为准。按用户 2026-10-27 的要求，v2.0.0 之前的 Task Record 一律标记为 Completed，本草稿据此改为 Completed 归档；它不承载独立交付。
