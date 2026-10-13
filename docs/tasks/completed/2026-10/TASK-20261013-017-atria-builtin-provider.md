# TASK-20261013-017: Atria 内置提供商与原生三协议接入

## Status

Completed

## Created

2026-10-13 22:44（Asia/Singapore）

## Request

用户希望新增 Atria 内置提供商，要求先阅读 https://api.atria-asi.ai/docs，分析如何接入并给出规划。

后续明确授权：“我现在有一个正在进行的任务，在不影响的情况下，你继续接入吧”。

## Goal

按已完成的规划接入 Atria 内置提供商，完成最小充分验证，并保持另一项 GPT 订阅长上下文任务不受影响。

## Scope

- Atria 预设、升级迁移、原生 Chat Completions／Responses／Messages 适配器。
- API Key 账号、预设目录、默认／显式路由、网关统计与界面接入检查。
- 契约／集成／界面测试、完整质量门、隔离原生验收及文档交付。

## Out of Scope

- 不修改真实账号、密钥、数据库和客户端配置。
- 不停止既有开发应用，不切换原工作区分支，不覆盖 GPT 长上下文任务的修改。

# Progress

用户已授权继续实施，要求不影响进行中任务。已从 `a00ed05` 创建独立分支 `codex/atria-provider` 和工作区 `E:/go/new/wails3-v/OmniProxy-atria-provider`；全部实现、绑定生成、测试和构建在独立工作区进行。原工作区的 TASK-016 和代码保持不动；完成后保留独立提交，避免在其他任务工作区进行并发合并。

2026-10-14：适配器、目录和路由已完成，六组原生 HTTP／SSE 网关集成通过，并验证终态前到达流式首事件与官方 usage 累计。完整质量门及隔离原生窗口验收通过；测试进程已关闭。

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

- 新增第十家内置提供商 Atria 和追加迁移 019，保留旧账号、凭证引用及开关；018 留给并行任务。
- 原生三协议适配器校验协议、模型、密钥、输出上限与文本内容，禁止重定向，保留 SSE、工具正文、取消及 Retry-After。
- Atria API Key 账号自动派生官方预设，公开来源 preset 与 256000／文本元数据，不伪造发现时间或调用未知发现／余额 API。
- 默认映射、别名、账号资格、实际网关和客户端目录使用同一模型定义；复用原上游 usage 观察与统计。
- 提供商名称、路由保存、预设来源与接入检查完成中英文界面接入；品牌资源使用既有通用图标回退。
- 添加迁移／目录／路由／适配器／网关及前端行为验证，更新相关产品和架构文档，未增加依赖或修改 Core 的 GPT 长上下文实现。

## Files Changed

- `migrations/019_builtin_atria.sql`、`internal/provider/atria/`。
- `internal/application/{provider,modelcatalog,routing,gateway}/` 与相关测试、`main_test.go`。
- 提供商／路由／网关的前端行为、翻译字典及重新生成的目录类型绑定。
- PRD、OVERVIEW、TECH_STACK、GATEWAY、ROUTING、LOCAL_AGENTS、CURRENT 和本记录。

# Verification

- 定向 Go 测试：Atria 适配器、迁移、路由、目录和网关全部通过；包含六组原生非流式／SSE、工具历史、输出限制、图片／文件拒绝、认证、重定向、取消、usage 和凭证清理。
- `task verify`：PASS，格式、Go vet／staticcheck／ESLint、类型检查、Go 全套、前端 69 文件／369 测试、Windows 生产构建全部通过。
- 新工作区首次执行因缺少嵌入的 frontend/dist 停在 lint，先运行正常前端构建；后续修正新增提供商导致的旧九家数量预期、一个状态码常量及目录数量文案断言，最终完整质量门通过。
- 原生窗口验收：独立 test 配置、数据库和端口 45890／45891；添加占位 API Key 账号，验证第十家卡片、256000／预设来源、显式别名实际保存、Atria 接入检查与深浅主题。没有调用真实推理、没有写入用户客户端配置。
- 验收使用临时 Go overlay 仅指定 Wails 的独立 WebView 数据目录与本机 CDP 端口 45892；环境变量会被 SDK 重置，不能单靠变量启用。正式源码和正式构建不包含调试参数，验收进程已退出。
- 本地证据：独立工作区 `tmp/atria-qa/task-verify.log`、`native-verification.json` 和深浅截图；临时文件、占位凭证和验收二进制均不提交。
- `git diff --check`：PASS；文档与代码使用 UTF-8 无 BOM，未混合原工作区其他任务的改动。

## Remaining Risks

未提供有效 Atria Key，尚未验证真实上游和 Codex／Claude 客户端专有工具、续接参数及推理档位。上游余额、实时模型发现和 WebSocket 能力不宣称已支持。现有请求体容量保持不变，完整 256K 输入仍受本地字节上限约束。

# Documentation

已同步 PRD、架构概览、技术栈、网关、路由、本地 Agent 接入与当前看板；遵循既有 ADR，不新增架构决策。

# Deviations

迁移由暂拟 018 调整为 019，避免并行 GPT 任务争用版本。模型目录采用派生预设与可选来源字段，不依赖未确认发现接口。完整实现和提交保留在独立分支，未合入正在修改的原工作区。

# Git

`015cd6ab8baed81ff890809cbc23a5c78d4808f9 feat(provider): 接入 Atria 内置提供商与原生三协议`

分支：`codex/atria-provider`。原工作区正在执行 TASK-016，未切换其分支或合入本提交。归档和提交哈希记录由后续文档提交保存。

# Result

Atria 内置接入代码与本地验证完成。独立分支可供后续安全合入；另一任务的工作区、真实账号、开发应用和客户端配置保持独立。

## Integration

2026-10-14 用户明确要求同时合入 Atria 和 GPT 长上下文；已通过 TASK-20261014-001 在独立 main 工作区完成合并。合并提交 `7dce1b0`，合并后完整质量门与 369 项前端测试通过，保留本记录的真实 Key 验收限制。

# Completed

2026-10-14 00:16（Asia/Singapore）
