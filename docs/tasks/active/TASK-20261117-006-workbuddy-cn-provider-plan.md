# TASK-20261117-006: WorkBuddy 国内版（copilot.tencent.com）接入规划

## Status

Planned

## Created

2026-11-17 15:45（Asia/Singapore）

## Request

用户问「目前只接入了国际版的吗？我现在想接入国内版本的，你看看该怎么写task」。

## Goal

形成 WorkBuddy **国内版**（腾讯 CodeBuddy 桌面版，`copilot.tencent.com`）作为 OmniProxy
原生内置订阅服务商的接入方案与任务拆分：明确它与已上线的国际版
（`builtin_workbuddy_ai`／`www.workbuddy.ai`）在协议、登录、目录、额度、协议路由上的差异，
给出首选实现方案、阶段划分、文件清单、验证与验收条件，以及待用户确认的决策点。

本记录只做规划，**不改任何代码**；实施另开任务记录。

## Background

### 参考实现（只读，社区插件 0.2.36）

包：`@magpie-community/opencode-workbuddy-auth`（npm，MIT）。本轮只读检查 npm tarball 的
`index.mjs`（5,003 行）与 `README.md`，**未执行其代码**；国际版接入时用的是同一份文件
（见 TASK-20261116-015）。插件一个包提供两个 provider：

| provider | 站点 | endpoint | 平台标识 | UA 中段（appName） |
| --- | --- | --- | --- | --- |
| `workbuddy`（国内版） | `copilot.tencent.com` | `https://copilot.tencent.com` | `workbuddy` | `WorkBuddy` |
| `workbuddy-ai`（国际版，已接入） | `www.workbuddy.ai` | `https://www.workbuddy.ai` | `workbuddy-ai` | `workbuddy-ai` |

`index.mjs` 依据：`SITES`（第 68–115 行）两站字段 `endpoint／platform／appName／messages／
noImages`；`sign()`（第 391–411 行）两站共用同一组客户端头，只有 `User-Agent` 的
appName 段与 `X-IDE-Version` 的来源不同；`cliAgent()`（第 418–420 行）。

### 国内版与国际版的差异（插件事实，**未经真实账号在 OmniProxy 验证**）

| 项 | 国际版（已接） | 国内版（本次） |
| --- | --- | --- |
| 浏览器登录 | `POST /v2/plugin/auth/state?platform=workbuddy-ai` | 同一接口，`platform=workbuddy`；`authUrl` 追加 `version=5.9.3` 与 `loginSessionId`（国内登录页在 `version < 5.8.0` 时隐藏企业微信入口，第 2103–2106 行） |
| 令牌与续期 | `/v2/plugin/auth/token`、`/refresh`、`/login/account` | 完全相同（同一 `platform` 参数只在 state 接口上） |
| 模型目录 | `GET /v3/config` | 同一路径；目录**含 Claude 模型**（`claude-opus-5.5`、`claude-sonnet-5.5`、`claude-opus-5`、`claude-opus-4.8-1m`，第 172–177 行实测） |
| 对话 | 只有 `POST /v2/chat/completions`（流式） | 同一 Chat 端点 **加** Anthropic Messages：`POST /v1/messages`（第 159–184 行）；Claude 模型在 Messages 上可用，chat 端点上「每个档位回答都一样」（第 163–166 行） |
| 推理档位 | 目录 `supportedEfforts`／只给 `effort` 的补全（TASK-20261117-004） | 同上；**Claude 模型额外追加 `max`**（第 4070–4078 行：Messages 上 `output_config.effort=max` 实测 200 且思考更多） |
| 每日签到 | 无 | 有：`GET /v2/billing/meter/checkin-activity-status` 与 `GET /v2/billing/meter/daily-checkin`（第 2619–2683 行），业务码 `1001` 已领／`1002` 无资格／`1003` 未开放，401 表示「签到只接受 WorkBuddy 自己的应用」 |
| 计费／Credits | `POST /billing/meter/get-user-resource-summary` | 同一路径（Credits 计点，不是人民币金额）；签到发放的也是 credits |
| 客户端指纹 | `WorkBuddy/5.9.3 workbuddy-ai/5.9.3 CLI/2.165.0` | `WorkBuddy/5.9.3 WorkBuddy/5.9.3 CLI/2.165.0` |
| 图片 | `glm-*` 实为纯文本（`noImages`） | 无 `noImages`：国内版 `glm-5.3-flash` 实测能读图（第 145–147 行） |
| 云托管企业 | 无 | 企业账号的 `domain` 可能是自家主机（`<ent>.sso.copilot.tencent.com`、`<ent>.copilot.qq.com`、`<ent>.sso.codebuddy.cn`），请求必须发到该主机（`endpointOf()`，第 230–255 行）；`tencent.sso.*` 两个 iOA 主机仍走 `copilot.tencent.com` |
| API Key | 未接 | 未接（国内版 Key 页 `www.workbuddy.cn/profile/keys`，仅记录，不纳入本次） |

### 仓库现状（只读核对，2026-11-17）

- 已接入的国际版：`internal/provider/workbuddyai`（kind `workbuddy_ai`、id
  `builtin_workbuddy_ai`、auth `workbuddy_login`、迁移 050／051），登录编排
  `internal/application/workbuddy`，网关转发 `internal/application/gateway/workbuddy.go`，
  Wails 接线 `internal/transport/wails/workbuddy_subscription.go`。
- 适配器把**国际版的站点常量写死在包级**：`client.go:34` 的 `signInStatePath`、
  `client.go:42` 的 `pluginPlatform = "workbuddy-ai"`、`client.go:45` 的 `bizUserAgent`，
  `workbuddyai.go:21-35` 的 `Kind／Name／DefaultBaseURL／AppVersion／CLIVersion`。
  国内版需要按「构建（build）」参数化，而不是复制一份适配器。
- **已具备的复用件**：Credits 三窗口解析（`quota.go`）、`/v3/config` 目录与档位
  （`discovery.go`，含 TASK-20261117-004 的补全规则）、gzip + 签名头 + 会话 id
  （`client.go`）、DPAPI 凭据与续期（`internal/application/account/workbuddy.go`）、
  订阅登录会话外壳（`internal/application/workbuddy`）、额度快照表 051。
- **按 kind 硬编码的位置**（新增第二个 kind 必须逐点处理，共约 10 处后端 + 8 处前端）：
  `provider/catalog.go`、`account/workbuddy.go`（`CheckWorkBuddyProvider`）、
  `routing/presets.go`、`routing/catalog.go`、`routing/model_entries.go`、
  `gateway/routing.go`、`gateway/agent.go`、`gateway/service.go`、
  `modelcatalog/{sync,workbuddy_auto}.go`、`transport/wails/{subscription,workbuddy_subscription,modelcatalog}.go`。
- **协议矩阵**：`routing/model_entries.go:60-168` 按 kind 决定 `Entry.Protocols`
  （`chat_completions／responses／messages`），网关据此把客户端协议转换成上游原生协议；
  `Metadata` 里没有「单模型上游协议」字段，只能用 `Protocols` 表达。
- 迁移最大编号 051，新迁移取 **052**。前端服务商能力判断是散落白名单
  （`subscription-login-config.ts`、`provider-labels.ts`、`provider-brand-icon.tsx`、
  `provider-accounts-view.tsx`、`add-account-dialog.tsx`、`quota-model.ts`、
  `use-gateway-preparation.ts`），无注册表。
- ADR-026 第 52 行写明「只接国际版」，国内版被列为「独立议题」——本次接入需要新增 ADR
  并在 ADR-026 里加指向。

## Scope（规划范围）

1. 新增内置服务商「WorkBuddy（国内版）」：kind、provider 行、迁移 052、认证方式复用
   `workbuddy_login`。
2. 适配器按 build 参数化（站点、UA、platform、Messages、签到、图片差异），国际版行为不变。
3. 目录同步：国内账号 `/v3/config`，Claude 模型追加 `max` 档位。
4. 协议路由：Claude 模型走 `/v1/messages` 原生；其它模型走 Chat；三种客户端协议经网关
   转成目标模型的原生协议。
5. Credits 三窗口复用；签到（手动）在 P2 评估。
6. 前端：服务商卡、登录弹窗、额度卡、目录与网关配置的 kind 支持；i18n。
7. 文档：`docs/product/WORKBUDDY.md`（国内版章节）、PRD、OVERVIEW、ADR-026 附录 + 新 ADR。
8. 验证：契约测试 + `task verify` + 用户授权的真实账号验证。

## Out of Scope

- 不接国内版的 API Key 认证（`ck_`／`pt_`）与桌面登录文件导入（5.6+ 加密）。
- 不做自动签到（首版最多手动），不让额度参与调度。
- 不改国际版账号、路由与既有数据；国际版 UA／端点常量保持不变。
- 不做免费模型每日 token 窗口、不做国内版专属的图片描述兜底。
- 不在本记录里实现代码；实施按下面的阶段另开任务记录。

---

# Analysis

## 方案对比：服务商建模

| 方案 | 做法 | 优点 | 缺点 |
| --- | --- | --- | --- |
| **A（推荐）独立内置服务商** | 新增 `builtin_workbuddy`（kind `workbuddy`），与 `builtin_workbuddy_ai` 并列 | 账号、UA、站点、能力天然隔离；与国际版零耦合；路由/额度/目录都按 provider 走，用户可见范围清晰；与插件对两个 provider 的建模一致 | 需要把约 10 处 `kind == workbuddy_ai` 的硬编码扩成「两个 kind」；前端多一张卡 |
| B 账号级「地区」选项 | 沿用 `workbuddy_ai` provider，账号上加 `build`（国际／国内）选项，像 Kimi／MiniMax／StepFun 那样 | 前端只有一张卡；账号选项表可复用既有模式 | 登录前必须先选 build（`state?platform=` 不同），登录弹窗要加选择；UA／端点／Messages／签到都要按账号分叉；provider 展示名对国内账号是错的；国际版已发布，改动面反而更大、回归风险高 |
| C 只改 Base URL | 直接把 provider 地址换成 `copilot.tencent.com` | 改动最小 | 无法同时存在国际／国内账号；platform 与 UA 仍错，登录直接失败 |

**推荐 A**。理由：两个站点是两套独立账号体系与套餐（国际版 Free 100＋benefit 30 credits、
国内版另有签到），且客户端指纹、协议能力不同；A 让每个概念只有一个归属，避免 B 的
「一账号一 build」在网关、目录版本、额度快照三条链路上重复分叉。

## 关键设计点

1. **适配器参数化（不复制包）**
   - 引入 `Build` 配置：`Kind／Name（展示名）／BaseURL／Platform（state 的 platform 参数）／
     AppName（UA 中段）／AppVersion／CLIVersion／Messages（是否有 /v1/messages）／
     Checkin（是否有签到）／ImagesExcept（`glm-*` 是否实为纯文本）`。
   - `New(transport)` 保持国际版（既有调用点与测试不变）；新增 `NewChina(transport)` 或
     `NewForBuild(build, transport)`；`Client` 内部所有包级常量改成 build 字段。
   - 错误前缀仍用稳定码；`Error()` 文本里的 `workbuddy_ai:` 前缀需要按 build 输出
     （前端已能识别 `workbuddy_ai:`／`gateway:`／`subscription:`，国内版要对齐，否则会塌成
     「操作未完成」——这是 TASK-20261116-020 踩过的坑）。
2. **登录与账号层**：`CheckWorkBuddyProvider` 允许 `workbuddy_ai` 与 `workbuddy` 两个 kind；
   凭据结构不变（含 `domain`，等于 `copilot.tencent.com` 或云托管企业主机）；
   `RotateWorkBuddyCredentials`、`LogoutWorkBuddySubscription`、额度快照键都复用。
3. **目录与档位**：`workbuddyMetadata` 复用；国内版对 `claude-*` 追加 `max` 档位；
   目录版本仍由 `credential_ref` 派生（续期不丢目录）。
4. **协议路由（本次最需要设计的部分）**
   - `routing/model_entries.go` 新增国内分支：`claude-*` 声明 `messages`（原生）＋
     `chat_completions`（上游也能回，但档位无效）；其它模型 `chat_completions`。
   - 若客户端用 Responses 调 Claude 模型：网关把 Responses 转 Messages（沿用既有转换矩阵，
     实施前先核对 `agentprotocol` 是否已覆盖「Responses→Messages」）；
   - Chat 客户端调 Claude 模型：首版直连 chat（可用但档位不生效），在文档里写明；
     若要求档位生效，再做「Chat→Messages」转换后置任务。
   - `ForwardMessages`：正文按原样 gzip 转发，头用同一 `sign()` 集合（`Authorization`、
     `X-User-Id`、`X-Domain`、`X-Product`、`X-IDE-*`、`X-Agent-*`、SDK 指纹）；
     **不做**系统提示词改写、`max_tokens` 改写与 11128 重试（这些只针对 chat 端点，
     见插件第 4561–4569 行的实测）。
5. **额度与签到**：Credits 三窗口复用（同路径、同字段）；P2 手动签到：
   `checkin-activity-status` → `daily-checkin`，映射 `1001/1002/1003`／401 到稳定码，
   展示 credit／streak；不自动签到、不参与调度。
6. **风险边界**：国内版同样属于「请求形态对齐官方客户端」，写入 ADR-026 附录与新 ADR；
   真实账号验证需用户授权（消耗额度）。

---

# Implementation Plan

## 阶段划分

| 阶段 | 任务 | 交付物 |
| --- | --- | --- |
| P0 | 服务商接入 + 登录 + 目录 + Chat 转发 | 新 provider 行（迁移 052）、适配器 build 参数化、登录／续期／额度／目录接线、前端卡片与登录弹窗；国内外两版并行可用 |
| P1 | Claude 模型的 Messages 原生路由 | `/v1/messages` 转发、`routing` 协议声明与转换、目录档位追加 `max`、真实 Claude 模型验证 |
| P2 | 签到 + 云托管企业域名 | 手动签到接口与界面、`domain` 主机白名单（`<ent>.sso.copilot.tencent.com` 等）按企业域名切主机 |
| P3（可选） | Chat→Messages 转换 | 让 Chat 客户端调 Claude 模型时档位也生效 |

建议按 P0 → P1 → P2 拆成三个实施任务记录；每个任务各自 `task verify` 与提交。

## P0 文件清单（预估）

- 新增：`internal/provider/workbuddyai/build.go`（`Build` 定义与国际／国内两份配置）
- 修改：`internal/provider/workbuddyai/{workbuddyai,client,auth,discovery,quota,errors}.go`
  （常量迁移到 build，`NewChina`／`NewForBuild`；UA／platform／base 按 build 生成）
- 新增：`migrations/052_builtin_workbuddy.sql`（照 050 模式插入 provider 行）
- 修改：`internal/application/provider/catalog.go`（新增 builtin 条目）
- 修改：`internal/application/account/workbuddy.go`（`CheckWorkBuddyProvider` 允许两 kind）
- 修改：`internal/application/workbuddy/{service,login,operations,quota}.go`（按 provider kind 选 build）
- 修改：`internal/application/modelcatalog/{sync,workbuddy,workbuddy_auto}.go`（国内分支）
- 修改：`internal/application/routing/{presets,catalog,model_entries}.go`（kind 白名单 + 目录协议）
- 修改：`internal/application/gateway/{service,routing,workbuddy,agent,key_access}.go`（转发分支、kind 白名单）
- 修改：`internal/transport/wails/{subscription,workbuddy_subscription,modelcatalog,google_model_sync}.go`（登录分流、失败码、目录接线）
- 前端：`subscription-login-config.ts`、`provider-labels.ts`、`provider-brand-icon.tsx`（或复用图标）、
  `workbuddy-credits.tsx`、`add-account-dialog.tsx`、`provider-accounts-view.tsx`、
  `quota-model.ts`、`use-gateway-preparation.ts`、`i18n/locales/{zh,en}.ts`
- 测试：适配器 build 契约（UA／platform／端点／错误前缀）、账号保存与续期、目录同步与协议、
  routing 协议矩阵、gateway 转发、迁移 052、前端组件

## P1 文件清单（预估）

- 新增：`internal/provider/workbuddyai/messages.go`（`/v1/messages` 流式转发与错误码）
- 修改：`client.go`（新增协议常量与 `ForwardMessages`）、`routing/model_entries.go`（Claude→messages）、
  `gateway/workbuddy.go`（按 `request.Protocol` 分派）、`modelcatalog/workbuddy.go`（Claude 追加 `max`）
- 测试：Messages 正文原样透传（不改写 system）、SSE 转发、协议转换、档位 `max`

## P2 文件清单（预估）

- 新增：`internal/provider/workbuddyai/checkin.go`（状态查询 + 领取 + 错误码映射）
- 修改：`internal/application/workbuddy/quota.go`（签到编排与稳定码）、
  `internal/transport/wails/workbuddy_subscription.go`（绑定）、
  前端额度卡／账号卡「签到」按钮与结果文案、`workbuddy-{zh,en}.ts`
- 云托管企业：`build.go`／`endpoint` 解析加域名白名单（仅 `*.sso.copilot.tencent.com`、
  `*.copilot.qq.com`、`*.sso.codebuddy.cn`，其余域名拒绝并提示），补安全测试

## 待用户确认

1. **服务商建模**：采用推荐方案 A（独立内置服务商「WorkBuddy（国内版）」）？还是 B（账号级地区选项）？
2. **首版范围**：只要 P0（登录 + 目录 + Chat），还是 P0+P1（含 Claude 模型的 Messages 原生路由）？
   签到（P2）是否要进首版？
3. **真实账号验证**：是否有可用的国内版账号（个人／企业）授权登录并发一次对话（消耗额度）？
   若是企业账号，是否属于云托管域名（`<ent>.sso.copilot.tencent.com` 等），决定 P2 优先级。

## 验收标准

- [ ] 账号页出现「WorkBuddy（国内版）」卡片，浏览器登录走 `platform=workbuddy`，
      出站 UA 为 `WorkBuddy/<app> WorkBuddy/<app> CLI/<cli>`；登录、续期、退出、重新登录可用
- [ ] 国内账号目录同步成功，Claude 模型与其它模型的能力／上下文／档位与 `/v3/config` 一致
      （Claude 追加 `max`）；续期不让目录失效
- [ ] Chat 协议经网关可用；Claude 模型走 `/v1/messages` 原生且档位生效（若纳入 P1）
- [ ] Credits 三窗口展示正确；签到（若纳入）可手动领取并给出 credit／streak／失败码
- [ ] 国际版账号、路由与既有目录不回归；`task verify` 通过
- [ ] 真实账号登录 + 一次真实对话（用户授权）；未验证项写入文档
- [ ] 文档与 ADR 更新：`WORKBUDDY.md`、PRD、OVERVIEW、ADR-026 附录 + 新 ADR

## 风险与缓解

| 风险 | 缓解 |
| --- | --- |
| 国内版客户端指纹／头部与插件记录不一致（`10085`、`11128`） | 首版只按插件实测形态发送；真实账号先跑最小 chat 验证形态；失败按稳定码暴露，不静默降级 |
| Claude 模型在 chat 端点档位无效，用户以为档位坏了 | 目录与文档写明「Claude 走 Messages 才有时档位」；P1 落地后按 Messages 路由 |
| 云托管企业账号请求发错主机（`endpointOf` 语义） | 首版仅支持 `domain=copilot.tencent.com` 的个人／标准企业账号，云托管域名给明确提示；P2 再按白名单切主机 |
| 新增 kind 漏改白名单（历史上 `routing-editor` 漏过 HowToFix） | 实施时用「kind 是否可路由／可登录／可同步目录」的表格逐项对照，并补测试兜底 |
| 签到接口未在 OmniProxy 实测 | 按插件错误码实现并做契约测试；真实账号验证后补文档；失败不影响对话与额度展示 |
| Credits 语义与币种误解 | 国内版同样是 credits 计点（不是人民币）；沿用国际版三窗口语义，不换算金额 |

---

# Implementation Result

待实施（本记录只做规划）。

---

# Verification

- 本轮为规划：只读核对参考实现 `index.mjs`（SITES／sign／cliAgent／nativeOf／effortsOf／
  checkinOf）与仓库现状（适配器常量、kind 硬编码位置、`routing/model_entries.go` 协议矩阵、
  迁移编号、前端白名单）。未运行真实账号、未发送上游请求、未改代码。

---

# Documentation

- 本记录（规划）
- 实施时需更新：`docs/product/WORKBUDDY.md`、`docs/product/PRD.md`、`docs/architecture/OVERVIEW.md`、
  `docs/decisions/ADR-026-workbuddy-ai-subscription.md`（附录）＋新增 ADR（国内版接入边界）

---

# Git

## Commit

4bcbde3f docs(tasks): 新增 WorkBuddy 国内版接入规划（规划提交；实施提交待后续任务记录）

---

# Result

TBD

---

# Completed

TBD
