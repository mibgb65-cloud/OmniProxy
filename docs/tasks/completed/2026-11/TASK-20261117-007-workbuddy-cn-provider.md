# TASK-20261117-007: 接入 WorkBuddy 国内版（copilot.tencent.com，P0+P1）

## Status

Completed

## Created

2026-11-17 15:45（Asia/Singapore）

## Request

用户确认按 [TASK-20261117-006](TASK-20261117-006-workbuddy-cn-provider-plan.md) 的规划实施：
方案 A（独立内置服务商）+ 首版 P0+P1（登录／目录／Chat + Claude 模型 Messages 原生路由）。

## Goal

新增内置订阅服务商「WorkBuddy（国内版）」：用户用浏览器登录国内版后，可在本机网关以
Chat Completions／Responses／Messages 调用其模型；目录来自账号自己的 `GET /v3/config`；
Claude 模型走上游原生 `POST /v1/messages`，其余模型走 `/v2/chat/completions`；Credits 三窗口
复用国际版口径。国际版行为不变。

## Background

差异与依据见 [TASK-20261117-006](TASK-20261117-006-workbuddy-cn-provider-plan.md)。要点：
站点 `https://copilot.tencent.com`、登录 `platform=workbuddy`、UA 中段 `WorkBuddy`、
`/v3/config` 目录含 Claude 模型、Claude 走 `/v1/messages`（chat 端点档位无效）、
Claude 档位追加 `max`、`glm-*` 在国内版能读图。

## Scope

- 适配器 `internal/provider/workbuddyai` 按 build 参数化（站点／platform／UA／Messages／图片差异），
  新增中国版 build 与 `/v1/messages` 流式转发；国际版常量与行为不变。
- 新增 provider：kind `workbuddy`、id `builtin_workbuddy`、迁移 `052`；认证方式复用 `workbuddy_login`。
- 账号层允许两个 kind；续期仍事务内重绑目录版本；Credits 三窗口复用。
- 目录同步：国内分支；Claude 模型档位追加 `max`。
- 路由与网关：国内分支的 `Entry.Protocols`（Claude → `messages` + `chat_completions`，其它 → chat）
  与协议分派转发。
- 前端：服务商卡、登录弹窗、额度卡、目录／网关配置、i18n。
- 文档：`WORKBUDDY.md`、PRD、OVERVIEW、ADR-026 附录 + 新 ADR。

## Out of Scope

- P2：手动签到、云托管企业域名（`<ent>.sso.copilot.tencent.com` 等）按域名切主机。
- API Key 认证（`ck_`／`pt_`）、桌面登录文件导入。
- 不改国际版 UA／端点／协议行为；不做 Chat→Messages 转换。
- 本轮不动用户数据、不安装、不打标签。

---

# Implementation Result

## Changes

- **适配器按构建参数化**（`internal/provider/workbuddyai`）：新增 `build.go`
  （`Build`＋`International`／`China`＋`BuildOf`、`UserAgent()`、`ShortUserAgent()`、
  `MessagesNative()`），`Client` 持有构建；`New` 仍是国际版，新增 `NewForBuild`、
  `NewClientAtFor`。登录 state 的 `platform`、授权页 `version`、UA、`X-IDE-Version`、
  计费/登录短 UA 全部改为读构建；错误前缀保持 `workbuddy_ai:`（前端按稳定码切词，与构建无关）。
- **图片声明按构建区分**：`normalizeModel` 增加构建参数——国际版 `glm-*` 仍按纯文本，
  国内版保留目录声明的图片输入。
- **原生 Messages**（`messages.go`）：`ForwardMessages` 把请求原样（不 gzip、不改写系统提示词、
  不改 `max_tokens`、不做 11128 重试）发往 `<base>/v1/messages`，头集与 Chat 相同，
  业务码 `10085/11128/11159/11158` 映射不变，未识别错误把正文交给网关。
- **新服务商**：kind `workbuddy`／id `builtin_workbuddy`／展示名 `WorkBuddy (China)`；
  迁移 `052_builtin_workbuddy.sql` 只追加 provider 行；`provider.Catalog` 增加条目。
- **账号层**：`CheckWorkBuddyProvider` 改为按 `BuildOf(kind)` 接受两个构建；新增
  `WorkBuddyBuild(ctx, providerID)` 供上层解析构建。
- **应用层**（`internal/application/workbuddy`）：`Service` 按构建缓存客户端
  （`clientFor`／`injected` 供测试注入），`buildOf(accountID)` 解析账号所属构建；
  登录会话记录所用客户端，续期、目录、额度都走对应构建。
- **目录同步**：`modelcatalog` 的 WorkBuddy 分支覆盖两个 kind；新增
  `StartKindsAutoSync` 让一次后台扫描同时处理两版；`workbuddyMetadata(model, messagesNative)`
  对国内版 `claude-*` 追加 `max` 档位。
- **路由与网关**：`routing` 的白名单、`supportedProvider`、`hasPresetAccount` 与目录账号过滤
  都覆盖两个 kind；网关按 kind 选择客户端，`request.Protocol == messages && 构建为
  MessagesNative(model)` 时走原生 Messages，否则仍走 Chat（含 Responses／Messages→Chat 聚合）。
- **Wails 接线**：`SubscriptionBegin`／`SubscriptionCurrent` 用 `workbuddyProvider()` 同时接受
  两个内置 provider；额度、刷新、退出、目录同步本来就按账号解析，无需分叉。
- **前端**：`subscription-login-config`（新增 `workbuddy` 品牌与文案）、`provider-labels`、
  `provider-brand-icon`（复用同一品牌图）、账号卡登录判定、网关准备列表、配额归类、
  Credits 卡片标题按构建区分，中英文案补齐；新增两条前端测试。

## Files Changed

- 适配器：`internal/provider/workbuddyai/{build.go,messages.go,build_test.go,messages_test.go,workbuddyai.go,client.go,auth.go,discovery.go,auth_test.go}`
- 应用层：`internal/application/{provider/catalog.go,account/workbuddy.go,workbuddy/*,modelcatalog/*,routing/{presets,catalog}.go,gateway/{service,routing,agent,workbuddy}.go}`
- 迁移：`migrations/052_builtin_workbuddy.sql`
- Wails：`internal/transport/wails/{subscription.go,workbuddy_subscription.go}`
- 测试：`internal/application/{provider/service_test.go,provider/workbuddy_migration_test.go,account/workbuddy_test.go,modelcatalog/workbuddy_test.go,gateway/workbuddy_messages_test.go}`、`main_test.go`
- 前端：`features/accounts/*`、`features/providers/*`、`features/gateway/use-gateway-preparation.ts`、
  `features/quota/quota-model.ts`、`i18n/locales/workbuddy-{zh,en}.ts`
- 文档：`docs/product/WORKBUDDY.md`、`docs/product/PRD.md`、`docs/architecture/OVERVIEW.md`、
  `docs/decisions/ADR-026-workbuddy-ai-subscription.md`、`docs/decisions/ADR-027-workbuddy-china-build.md`

## Technical Notes

- 国内版与国际版共用同一套插件 API 路径（登录、续期、`/v3/config`、`/v2/chat/completions`、
  `/billing/meter`），差异只有站点、`platform`、UA 与 Messages 能力，因此用 `Build` 参数化
  一份实现，而不是复制适配器包。
- Messages 请求刻意不压缩：社区插件只对 chat completion 做 gzip。
- 国内版 `claude-*` 追加 `max` 的依据是社区插件在真实账号上的实测（Messages 上
  `output_config.effort=max` 返回 200 且思考更多）；目录本身只到 `xhigh`。
- 后台自动同步用 `StartKindsAutoSync` 一次覆盖两个 kind，避免两套扫描各自刷新客户端目录。

---

# Verification

## 质量门（`task verify`，2026-11-17 16:04–16:27 Asia/Singapore）

```text
task verify → exit 0
```

| 阶段 | 结果 |
| --- | --- |
| format | gofmt 无输出；prettier 通过（首次失败于 5 个前端文件格式，已 `prettier --write`） |
| lint | `go vet`／`staticcheck` 无新增告警；`eslint` 0 error |
| typecheck | `tsc --noEmit` exit 0 |
| test | Go 全包通过；前端 158 files / 967 tests |
| build | 绑定 + vite + Windows production 构建通过 |

首次 `task verify` 还暴露 `main_test.go` 把内置服务商数量写死为 15（应为 16），已同步修正。

## 契约测试覆盖

| 行为 | 测试 |
| --- | --- |
| 两个构建的站点、UA、`platform`、Messages/图片差异 | `workbuddyai/build_test.go` |
| 国内版登录 `platform=workbuddy` 与授权页 `version`/`loginSessionId` | `TestChinaBuildUsesItsOwnPlatformAndVersion` |
| 目录图片：国内版保留 `glm-*`、国际版仍按文本 | `TestChinaBuildKeepsGLMImages` |
| Messages 原样转发、不 gzip、头集、会话 ID | `TestForwardMessagesRelaysBodyAndHeaders` |
| Messages 协议/构建校验与业务码映射 | `TestForwardMessagesRejectsWrongBuildAndProtocol`／`MapsBusinessCodes` |
| 网关 Chat 与 Messages 的分派 | `gateway/workbuddy_messages_test.go` |
| 目录元数据 Claude `max` | `modelcatalog/workbuddy_test.go` |
| 账号挂载与构建解析 | `account/workbuddy_test.go` |
| 两版 provider 预设与迁移 052 幂等 | `provider/{service_test,workbuddy_migration_test}.go`、`main_test.go` |
| 前端品牌/文案/构建隔离 | `provider-labels.test.ts`、`subscription-login-config.test.ts` |

## 真实账号验证（2026-11-17 16:39–16:52，用户账号，国内版）

| 项 | 结果 |
| --- | --- |
| 浏览器登录 | 已走通：DB 里 `builtin_workbuddy` 账号（kind `workbuddy`、`workbuddy_login`、`authenticated`），令牌域名 `www.workbuddy.cn` |
| 目录自动同步 | 已走通：登录完成后后台自动同步 60 个模型（`glm-5.3`、`glm-5.3-flash`、`deepseek-v4.1-flash`、`kimi-k3-1`、`minimax-m3.1-flash`、`hy4std`、`wb-lite`、`codewise-*` 等），`account_catalog_versions` 已绑定凭据版本 |
| 真实对话（Chat） | 已走通（用户授权）：`glm-5.3-flash`、单轮短问题，经 `NewForBuild(China)` 出站 `200` + SSE，回答 `OK`，`usage` 26/3/29；说明国内版 UA／`platform` 未被 `10085` 拒绝 |
| 账号目录里的 Claude | 该账号 `/v3/config` 60 个模型里**没有 `claude-*` ID**；显示名带 Claude 的是 `default-1.1`（Claude-3.7-Sonnet）与 `default-1.2`（Claude-4.0-Sonnet）两个别名，按 ID 规则不会走 Messages（与社区插件 `nativeOf` 口径一致） |
| 路由 | `model_routes` 仍为 0，需用户在路由页建规则后网关才暴露模型 |

## 未验证

- 国内版 Messages 原生路由需要账号目录里真有 `claude-*` 模型；本账号没有，因此真实链路的
  Messages 转发只由契约测试覆盖。
- 国内版客户端指纹、`platform` 与 Messages 端点来自社区插件 0.2.36，非官方契约。
- 签到与云托管企业域名不在本期（P2）。

---

# Documentation

- [x] `docs/product/WORKBUDDY.md`（两版对照、协议边界、目录档位、未接入）
- [x] `docs/product/PRD.md`、`docs/architecture/OVERVIEW.md`
- [x] `docs/decisions/ADR-027-workbuddy-china-build.md`（新）+ ADR-026 指向
- [x] `docs/plans/CURRENT.md`、`docs/releases/v2.3.1-beta.21.md`（随构建任务）

---

# Deviations

- 规划建议 P0／P1 拆成两个实施任务；实际合并为一个任务记录：Messages 原生路由与 P0 的
  协议分派共用同一处 `forwardWorkBuddy` 与同一套测试，拆开会产生两次全量验证与半成品状态。
  文档仍保留 P0／P1 的阶段说明。
- 规划中的「Chat→Messages 转换（P3）」未做：Chat 客户端调 Claude 模型仍走 Chat，档位不生效，
  已在文档与发行说明写明。
- 规划里写了「前端约 8 处白名单」，实际还需要改 Credits 卡标题与新增两条前端测试，
  比预估多两处。

---

# Git

## Commit

eb7e82f8 feat(provider): 接入 WorkBuddy 国内版（copilot.tencent.com）

---

# Result

WorkBuddy 国内版已作为独立内置服务商接入：`builtin_workbuddy`（kind `workbuddy`、迁移 052）
与 `builtin_workbuddy_ai` 并列，账号/令牌/目录/额度按构建隔离。适配器只有一份实现，按
`Build` 区分站点（`copilot.tencent.com`）、登录平台（`platform=workbuddy`）、
UA（`WorkBuddy/5.9.3 WorkBuddy/5.9.3 CLI/2.165.0`）与能力；国内版 `glm-*` 保留图片输入。
国内版 Claude 模型在客户端用 Messages 协议时原样转发到原生 `POST /v1/messages`（不 gzip、
不改写系统提示词、不做 11128 重试），目录为 `claude-*` 追加 `max` 档位；Chat 与 Responses
客户端仍走既有 Chat 聚合/转换链路。Credits 三窗口与后台/手动目录同步覆盖两个 kind，
国际版的 UA、端点与行为未变。

`task verify` exit 0（Go 全包、前端 158 文件／967 项测试、Windows production 构建）。
真实账号验证待用户执行：先登录＋目录，再授权一次对话。签到与云托管企业域名属 P2，未接入。

---

# Completed

2026-11-17 16:32（Asia/Singapore）
