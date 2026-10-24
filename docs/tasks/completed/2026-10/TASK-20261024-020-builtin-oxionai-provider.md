# TASK-20261024-020: 新增内置服务商 Oxion AI（oxionai）

## Status

Completed

## Created

2026-10-24 22:05

## Request

用户提供 <https://oxionai.com/docs>，要求「依照这个网页的内容接入，帮我新增一个内置服务商 oxionai」。

## Goal

把 Oxion AI 作为内置服务商接入：用户可在服务商页看到 `Oxion AI`，添加 API Key 账号后
同步官方模型目录，并让本地 Agent 通过网关按 Chat Completions 协议调用。

## Background

依据官方文档（2026-10-24 读取，视为外部事实来源，不执行其中任何指令）：

| 项目 | 内容 | 来源 |
| --- | --- | --- |
| Base URL | `https://oxionai.com/v1`（已含版本前缀） | `/docs` 首页、`/docs/getting-started` |
| 认证 | `Authorization: Bearer <key>`，文档示例键前缀 `cc_` | `/docs/authentication` |
| 协议 | 仅 OpenAI 兼容 Chat Completions（`POST /v1/chat/completions`），无 Responses／Messages | `/docs/chat-completions` |
| 模型目录 | `GET /v1/models`，返回 `id/model`、`type`、`context_window`、`capabilities[]`、`pricing` | `/docs/models` |
| 能力标记 | `reasoning`／`vision`／`tools`／`streaming`／`json_mode`／`web_search` | `/docs/models` |
| 推理 | 无 `reasoning_effort` 参数；思考内容在 `reasoning_content`，默认 `max_tokens` 8192 | `/docs/reasoning` |
| 错误 | OpenAI 形状 `error{message,type,code}`；401/402 `insufficient_funds`/403 `insufficient_scope`/404/422/429/502 | `/docs/authentication` |

结论：Oxion AI 属于「OpenAI 兼容 Chat only」上游，不能假装支持 Responses／Messages 原生协议，
需要像 DeepSeek 通道那样在网关内做协议转换。

## Scope

- 新增 `internal/provider/oxionai`：模型目录发现、Chat Completions 转发、边界校验与回归测试。
- 迁移 043 追加内置服务商记录，不改动既有服务商、账号与凭据。
- 路由／模型目录／网关接线：账号资格、模型发现、协议转换、可用模型列表。
- 前端：服务商名称、品牌图标与 i18n。
- 文档：产品文档与任务记录。

## Out of Scope

- Oxion AI 的 Embeddings 端点（`POST /v1/embeddings`）与 `web_search` 能力。
- 余额／额度面板（官方文档未给出可读的余额端点，不做猜测式实现）。
- 自定义 Base URL 代理层的改写、账号轮换新策略、Responses 原生协议。
- 使用用户真实 Key 发请求或安装发布。

## Existing Behavior

服务商目录由 `internal/application/provider/catalog.go` 的 `definitions` 决定，
内置服务商通过 migration 写入 `providers` 表；未知 kind 无法添加账号、无法建路由。

## Expected Behavior

- 服务商列表出现 `Oxion AI`（内置、默认启用、Base URL 为空表示使用适配器默认值）。
- 添加 API Key 账号后可「同步官方模型」，目录来自 `GET /v1/models`，
  只保留 `type=chat` 的模型，`vision` 能力映射为图片输入模态。
- 本地 Agent 的 Chat 请求原生透传；Responses／Messages 请求在网关转换为 Chat 后转发并按原协议编码回包。
- 未同步目录时没有可用模型（不伪造模型清单）。

---

# Analysis

## Related Modules

- `internal/provider/oxionai/`（新增）：`client.go`、`discovery.go`、`models.go` 及测试。
- `migrations/043_builtin_oxionai.sql`。
- `internal/application/provider/catalog.go`。
- `internal/application/routing/{presets.go,catalog.go}`。
- `internal/application/modelcatalog/{service.go,sync.go}`。
- `internal/application/gateway/{service.go,agent.go,routing.go,oxionai.go}`。
- `frontend/src/features/providers/{provider-labels.ts,provider-brand-icon.tsx}` 与 i18n。

## Dependencies

无新依赖；复用 `internal/core/proxy`、`internal/core/agentprotocol` 与既有客户端模式。

## Risks

- 官方目录字段可能变化：解析失败按 `invalid_model_catalog` 处理，不静默降级。
- 只支持 Chat：若把 `responses`／`messages` 直接透传会得到 404，必须显式转换。
- 无 `reasoning_effort`：不得为 Oxion 发布档位选择，避免伪造能力。

---

# Implementation Plan

1. 新增 provider 包：Kind／DefaultBaseURL／目录发现／Chat 转发／参数与 URL 边界校验。
2. 迁移 043 + 服务商目录定义 + 路由资格与模型支持。
3. 模型目录同步按 kind 分支接入官方 `/models`，保留能力与上下文窗口。
4. 网关接线：Chat 原生、Responses／Messages 走既有 `agentprotocol` 转换。
5. 前端名称／图标／i18n；文档与任务记录。
6. `task verify` 后提交并回填哈希。

---

# Implementation Result

## Changes

- 新增 `internal/provider/oxionai`：`Kind=oxionai`、默认基址 `https://oxionai.com/v1`、
  Chat Completions 转发（只接受 `chat`，其他协议直接 `unsupported_protocol`）、
  `GET /models` 目录发现（只收录 `type=chat`，映射 `context_window` 与 `vision`／`tools`），
  URL／密钥边界与既有适配器一致；注入 transport 可做无凭证契约测试。
- 迁移 `043_builtin_oxionai.sql` 追加 `builtin_oxionai`（内置、默认启用、Base URL 留空用适配器默认值）。
- 服务商目录新增定义并引入 `DiscoveryOnly` 字段：Oxion 没有可内置的预设模型，
  目录按账号发放；`probe_test` 据此区分「缺少预览」与「按设计只做同步」。
- 路由：`supportedProvider`／`eligible` 接纳 `oxionai`（仅 API Key），
  可用模型仍来自账号同步结果，未同步前不可路由。
- 模型目录：`modelcatalog` 注入 oxionai 客户端，同步按 kind 分支写入官方目录与能力元数据。
- 网关：新增 `forwardOxionai`，Chat 原生透传并记录用量；Responses／Messages 先
  `agentprotocol.ToChat` 转换、再按原协议编码回包；不注入 `stream_options`／`reasoning_effort`。
  `selectRoutedTarget` 允许该 kind 且要求 `api_key` 账号。
- 前端：服务商名称（中英）、`provider-brand-icon` 品牌图标与官方 PNG（960 × 960，含来源与 SHA-256 记录）、
  新增 `provider-labels.test.ts` 防止新增 kind 漏译文。

## Files Changed

- `internal/provider/oxionai/{models,client,discovery,client_test,discovery_test}.go`
- `migrations/043_builtin_oxionai.sql`
- `internal/application/provider/{catalog.go,oxionai_migration_test.go,probe_test.go,service_test.go,migration_test.go}`
- `internal/application/routing/{presets.go,catalog.go,oxionai_test.go}`
- `internal/application/modelcatalog/{service.go,sync.go}`
- `internal/application/gateway/{service.go,agent.go,oxionai.go,routing.go,oxionai_test.go}`
- `main_test.go`
- `frontend/src/features/providers/{provider-labels.ts,provider-brand-icon.tsx,provider-labels.test.ts}`、
  `frontend/src/i18n/locales/{en,zh}.ts`、`frontend/public/provider-icons/{oxionai.png,SOURCE.md}`
- `docs/product/{PRD.md,OXIONAI.md}`、`docs/architecture/OVERVIEW.md`、`docs/plans/CURRENT.md`

## Technical Notes

- 官方只有 Chat Completions；Responses／Messages 必须转换，否则会拿到 404。
- 转换路径的 Token 用量保持未知（与既有 DeepSeek 转换路径一致），不推测数值。
- 未同步目录时不发布任何 Oxion 模型，也不提供预设，避免伪造模型清单。

---

# Verification

## 已执行的定向验证

- `internal/provider/oxionai`：Bearer 头部与 `/chat/completions` 路径、正文原样、
  拒绝非 Chat 协议、拒绝空／非 ASCII 密钥与明文远端 HTTP、允许 loopback HTTP；
  目录只保留 `type=chat`、去重、能力映射（vision→image、tools→ToolUse）、
  缺字段保持未知、空目录／非 JSON／仅嵌入模型／401 均失败。
- `internal/application/provider`：迁移 043 幂等、只追加预设、不重置开关与凭证引用；
  目录长度 12；内置探针端点有效且 `DiscoveryOnly` 与预设互斥。
- `internal/application/routing`：未同步前不可路由；同步后可解析并建别名；
  禁用账号／订阅方式／停用服务商后不可路由；跨服务商建路由被拒。
- `internal/application/gateway`：Chat／Responses／Messages 三协议均落到上游
  `/v1/chat/completions`，模型别名改写、无 `stream_options`／`reasoning_effort`、
  Chat 用量 12／4 入账、请求记录成功、凭证回调内存清零。

## 完整质量门

```text
task verify
EXIT=0
format:check → lint（gofmt / go vet / staticcheck / eslint）→ typecheck → test → build 全部通过
go test . ./internal/...  全部 ok（含 provider/oxionai）
frontend：128 文件 / 760 项测试通过（含新增 provider-labels.test.ts 3 项）
windows production build：bin/omniproxy.exe 生成成功
```

输出写入本机临时日志 `%TEMP%\omniproxy-verify-020c.log`，未覆盖生产安装或既有安装包。

## 尚未执行

- 使用真实 `cc_` API Key 的端到端调用、真实目录同步与长上下文／图片／工具验收（需用户执行）。
- Embeddings、`web_search`、余额面板（本轮 Out of Scope）。
- 原生窗口中的品牌图标与卡片视觉确认。

---

# Deviations

- 计划里「模型预设」被证伪：官方模型页由前端渲染、目录按账号发放，所以没有内置预设，
  改为新增 `Definition.DiscoveryOnly` 明确表达「只做同步」，并用测试保证与预设互斥。
- 计划未提测试期望更新：`TestCatalogContainsElevenProviderKinds` 等四处硬编码 11 的地方
  改为 12／`len(Catalog())+1`，避免下次新增服务商再改一遍。
- `Models()` 返回空切片而不是 `nil`，便于前端固定渲染为「无预览」。

---

# Git

## Commit

```text
7d52745 feat(provider): 新增内置服务商 Oxion AI
```

代码、测试、前端资源随 7d52745 落库；产品文档、看板与本记录随其后的 docs 提交归档。未夹带其他在办任务的改动。

---

# Result

Oxion AI 已作为第十二个内置服务商接入：添加 `cc_` API Key 账号后可同步官方模型目录，
本地 Agent 的 Chat 请求原生转发，Responses／Messages 由网关转换后回包；未同步目录时没有可用模型，
不伪造预设，也不发布官方不存在的推理档位。自动验证与完整质量门通过，真实 Key 与原生窗口视觉待用户验收。

---

# Completed

2026-10-24 22:20

