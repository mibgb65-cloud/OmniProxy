# TASK-20261015-003: MiniMax 国内／海外按量与订阅接入规划

## Status

Completed

## Created

2026-10-15（Asia/Singapore）

## Request

用户希望接入 MiniMax，要求先制定计划，并询问是否有国内与海外两个平台；随后明确普通按量 API Key 和订阅 Key 都接。

## Goal

基于官方现行文档及已有 Kimi／Step 接入，明确双平台、两类 Key 的能力、资源边界、实施阶段和验收条件。

## Scope

本轮仅核对资料、代码和接入方案，形成可追溯规划与当前状态摘要。

## Out of Scope

适配器、数据库迁移、业务界面、真实 Key 读取、付费推理和客户端配置变更留待实施任务。

# Analysis

已阅读 PRD、架构、技术栈、编码、任务、Git 和测试规范；核对现有 MiniMax 预设及 Kimi／Step 的账号选项、目录版本、路由和网关实现。基线 `c4deaca`。

## 官方事实（2026-10-15 核对）

### 平台与端点

| 平台 | 控制台 | OpenAI 基础地址 | Messages 完整路径 |
| --- | --- | --- | --- |
| 国内 | `https://platform.minimax.cn` | `https://api.minimax.cn/v1` | `https://api.minimax.cn/anthropic/v1/messages` |
| 海外 | `https://platform.minimax.io` | `https://api.minimax.io/v1` | `https://api.minimax.io/anthropic/v1/messages` |

国内旧文档 `platform.minimaxi.com` 本次已重定向到 `platform.minimax.cn`，现行示例使用 `api.minimax.cn`。旧 `api.minimaxi.com` 出现在官方历史资料中；不能仅凭文档跳转认定旧 API 已停用或新旧请求完全等价。实施时识别旧配置，携 Key 的请求不跟随重定向，也不因失败自动切换主机。

依据：[国内 OpenAI SDK](https://platform.minimax.cn/docs/api-reference/text-openai-api)、[国内 Anthropic SDK](https://platform.minimax.cn/docs/api-reference/text-anthropic-api)、[海外 OpenAI SDK](https://platform.minimax.io/docs/api-reference/text-openai-api)、[海外 Anthropic SDK](https://platform.minimax.io/docs/api-reference/text-anthropic-api)、[官方历史 MCP 平台配置](https://github.com/MiniMax-AI/MiniMax-MCP)。

Key 必须匹配签发地区；官方 CLI 的 401 排查也要求 region 与 Key 来源一致。本项目应显式保存地区，不通过试发 Key 探测两个平台。

### 普通与订阅 Key

- 普通 Key 消耗按量账户余额；订阅 Key 使用订阅额度／积分包。两类 Key 独立，即使请求使用相同地区的 API 入口，也不能交换计费身份。
- 官方现行订阅名为 M Plan；旧 Token Plan 用户可保留原权益。旧 Coding Plan 名称仅作用户识别兼容，不能据名称硬编码成相同套餐。
- M Plan 公开 5h／周窗口与 `https://www.minimax.cn/v1/token_plan/remains`、`https://www.minimax.io/v1/token_plan/remains` 查询入口。路径虽然仍含 token_plan，也不能据此把旧套餐数据结构和权益当成 M Plan。
- 订阅额度用完后，上游可能继续扣已购买积分包；不会自动扣普通账户余额。OmniProxy 不自动换普通 Key，也不能声称能阻止上游按订阅规则扣积分。

依据：[国内 M Plan FAQ](https://platform.minimax.cn/docs/m-plan/faq)、[海外 M Plan FAQ](https://platform.minimax.io/docs/m-plan/faq)。查询响应字段、旧套餐返回值和权限仍需在实施中验证；公开 FAQ 只提供调用入口，不足以确定 DTO。

### 模型与协议

- 官方提供原生 Chat Completions、Messages、Responses 的 HTTP／SSE 文档。Responses 使用 `/v1/responses`，不是只有 Chat 可用；首版按地区、Key 类型和具体模型建立能力矩阵。
- 对话候选采用真实 ID：`MiniMax-M3`、`MiniMax-M2.7`、`MiniMax-M2.7-highspeed`、`MiniMax-M2.5`、`MiniMax-M2.5-highspeed`；旧 M2／M2.1 是否纳入首批待实施时复核，不把它们直接称为下线模型。
- `MiniMax-M3.1-Flash-Preview` 当前仅经 M Plan／MiniMax Code 提供，不放进普通按量账号的可调用预设；旧 Token Plan 也不能自动视为具有该权限。M3 系列标称 1,000,000，M2.x 为 204,800；标称窗口与实际验证分开记录。
- 保留完整思考和工具历史，包括 `reasoning_content`、`<think>`、thinking 块／signature、工具 ID。M3 与 M3.1 的思考开关／档位有区别，Responses 与 Chat 的默认也不同，不套用一个全局 effort 表。
- 原生 Responses 参考包含完整历史输入，store 固定 false；首版按完整历史处理，服务端 ID 续接、WebSocket、后台任务和 Codex custom／namespace 工具能力须单独确认，不直接复制 Kimi 的支持结论。
- 国内／海外均公开 `GET /v1/models`。官方定义是平台支持模型列表，成功返回不等于该 Key 拥有全部订阅权限，也不提供完整上下文能力；目录和能力元数据分别建模。
- 首版接文本／思考／函数工具。语音模型从对话目录排除；图片／视频输入、生成、文件及 MCP 独立安排。

依据：[国内 Responses](https://platform.minimax.cn/docs/api-reference/responses-create)、[海外 Responses](https://platform.minimax.io/docs/api-reference/responses-create)、[Messages 认证与请求](https://platform.minimax.cn/docs/api-reference/text-chat-anthropic)、[国内模型列表](https://platform.minimax.cn/docs/api-reference/models/openai/list-models)、[海外模型列表](https://platform.minimax.io/docs/api-reference/models/openai/list-models)。具体模型窗口及思考参数同时参考上方 SDK 文档。

普通余额：本次两站公开索引未找到可确认的受支持余额查询接口，标记 TBD；不能从 Step 复制 `/v1/accounts`，也不抓控制台 Cookie 或私有 RPC。[国内索引](https://platform.minimax.cn/docs/llms.txt)、[海外索引](https://platform.minimax.io/docs/llms.txt)

## 现有实现与方案比较

`application/provider/catalog.go` 已有 `kind=minimax`、`builtin_minimax`、图标和海外默认地址，但模型为展示名且混入 Speech；网关没有 MiniMax 转发分支。账号／目录／路由可参考已完成的 Kimi 与 Step，实现时不重写这些模块。

1. 只改提供商 Base URL：无法让同一品牌下的国内／海外账号同时使用，计费身份也不清晰。
2. 拆四张内置卡片：重复品牌并影响现有按 kind 展示与固定提供商身份。
3. **推荐一个品牌入口＋账号级地区／Key 类型**：保留现有 ID、启停和图标；新增／批量／编辑支持 `region=cn/global`、`billing_mode=payg/subscription`。订阅仍走 `auth_method=api_key` 和 DPAPI，不新增 OAuth。

两种计费类型可以共用协议路径，资源隔离必须依赖账号配置和实际凭证，不能靠 URL 推断。界面明确提示用户提供对应类型的 Key；未确认 Key 类型不标为已经验证，不能仅按前缀保证真实计费来源。

## 数据、兼容性与影响范围

- 新增 MiniMax 账号选项表及单调配置版本，沿用事务、凭据清理和并发更新模式。修改地区／Key 类型会失效模型和资源快照；名称修改不改变身份，Key 轮换沿用已有凭据版本。
- 旧空地址保留原 `.io` 海外语义；已识别 `.cn`／旧 `.com` 官方配置可确定国内地区，未知地址保留数据并要求明确选择。**旧地址无法区分按量与订阅，计费类型必须确认后才成为路由候选**，不能像 Step 那样按路径迁移为普通按量。
- 扩展 `account_catalog_versions`、模型目录、路由候选、网关取 Key 前复查；同步中途切平台或类型时禁止写回旧结果。当前最新迁移是 023，实施时重新检查并追加编号，不改既有迁移。
- 按通道生成别名，例如 `minimax-cn-api/MiniMax-M3`、`minimax-global-plan/MiniMax-M3`；匹配后仅发送真实模型 ID。同名裸模型跨通道有歧义时要求显式路由或固定账号，不按创建时间隐式选计费方式。既有精确规则优先。
- 请求固定所选地区／Key 类型，401、429、额度不足或网络错误不跨通道重试；本轮不扩建全局调度器。历史记录当次地区／计费方式，不随账号编辑被改写。
- 涉及 `internal/provider/minimax`、`application/{account,provider,modelcatalog,routing,gateway,activity,quota}`、数据库迁移、Wails 绑定、账号／资源前端和客户端目录过滤；复用标准库、Redactor、DPAPI 与现有 HTTP／SSE，不新增 SDK 或大型依赖。

# Implementation Plan

## P1：账号与通道隔离

保存双平台／两类 Key 及配置版本，完善单个／批量／编辑和通道标识；迁移保留旧数据，未确认类型明确提示。统一固定官方主机解析、模型／资源缓存身份与路由别名。默认新账号国内／按量，但必须在表单中明确展示该选择。

## P2：对话、目录与客户端

新增 MiniMax 原生三协议适配器，按已确认能力开放模型；保持增量 SSE、取消、思考和两轮函数工具关联。认证头由受保护回调构造，客户端头不透传，禁止重定向。Responses 不支持的状态／工具类型明确报错，不静默丢字段或伪造支持。

模型发现严格限制响应大小、条目数、ID 和重复项，只纳入已支持对话模型；未发现时使用带 preset 来源的官方候选，成功快照也不冒充订阅授权验证。按能力生成 Codex／Claude 目录及接入检查，保留现有客户端认证身份。

## P3：用量与订阅资源

复用 usage 观察器，核对各协议输入、缓存和推理 Token 的包含关系，累计快照按差值记录；缺失／截断不伪造零值。本机用量与官方剩余分别显示。

订阅仅向已选地区的固定 `www` 主机查询官方 remains 入口；不能从推理 Base URL 拼出管理域名。先核对响应契约与新旧套餐，再实现有界解析、5h／周窗口、积分包和共享缓存；未知字段保持未知，失败保留旧值并标过期，配置变更使快照失效。普通余额在没有受支持接口前显示未接入及官方入口。多个 Key 没有稳定资源池 ID 时不把额度相加。

## P4：真实验收与交付

四类通道分别验收短提示、JSON／SSE、两轮工具、思考和用量；Codex Responses 与 Claude Messages 分别确认工具兼容，避免仅靠单轮聊天宣称 Agent 可用。覆盖 M Plan 与旧套餐差异，必要时标注未验证通道；长窗口需单独控制成本验证。实施完成后运行 `task verify`、更新产品／架构文档并提交记录。

## 最小充分验证

1. 迁移保留原 ID、启停、凭据引用；旧海外默认、旧国内地址、未知地址及未知计费类型分别覆盖。
2. 四组合单个／批量／编辑、非法枚举／Key 格式边界、失败回滚与凭据清理、并发版本冲突。
3. 主机／路径／认证、通道别名、裸名歧义、固定账号、禁止跨区／跨计费回落及凭据重定向。
4. 切地区／计费类型／Key 时目录与资源缓存失效；同步期间配置变化不写旧数据，已解析请求不能跟随新配置。
5. 各模型三协议能力、思考／工具历史、真正增量 SSE、取消、截断、usage 差值与错误分类；未知工具不冒充支持。
6. remains 新旧套餐字段、窗口／积分与 Token 单位、空／超界／错误数据、共享额度不重复加总、刷新有界及脱敏。
7. 原生深浅主题检查账号和资源 UI、现有供应商回归；实施后 `task verify`。真实 Key 和长窗口另列实际验证结果。

## 待确认项

旧 API 域名的直接请求行为；两类 Key 的可验证身份信号；普通／M Plan／旧 Token Plan 的实际模型权限；remains 响应 schema、单位和稳定资源池身份；普通余额受支持接口；Responses 高级工具与实际客户端能力。以上不以推测填入已实现契约。

# Progress

- 用户已明确国内／海外、按量／订阅都纳入方案；本轮仍仅规划。
- 官方旧国内文档跳转至 `.cn`，现行资料新增 M Plan／M3.1；已补充旧域名和旧订阅兼容，避免沿用过时假设。
- 已确认原生 Responses、模型发现与订阅 remains 入口；缺少完整契约的资源字段保留 TBD。

# Implementation Result

已形成官方事实、方案比较、账号／目录／路由／资源边界、四阶段实施与验收清单。仅修改任务记录和状态看板，业务代码和运行时数据未修改。

# Verification

- 官方网页／索引人工核对平台、路径、Key 类型、模型与原生协议、订阅查询入口；未读取真实 Key 或发送推理。
- PASS：两份文档均为 UTF-8 无 BOM，CURRENT 相对任务链接存在，`git diff --check` 通过；已人工复查已确认事实与待验证边界。
- 本轮只有说明文档，无可执行配置或命令变更，不运行 `task verify`；实施任务必须运行质量门。

# Documentation

仅同步任务记录和 CURRENT；PRD、架构、技术栈继续反映已经实现的能力。

# Deviations

用户补充两类 Key 都接，方案扩展为四种组合。BrowserOS 技能已读取但本轮无对应 MCP 工具，公开资料使用网页检索工具；Markdown 页面抓取受限时读取官方 HTML，未使用第三方结论替代官方事实。

# Git

待文档审查后提交；既有未跟踪任务记录不属于本任务。

# Result

接入规划完成；建议沿用一张 MiniMax 卡片，按账号选择平台与 Key 类型，分 P1–P4 实施并验收。接入业务代码尚未实施，真实平台调用与资源响应契约仍待验证。

# Completed

2026-10-15（Asia/Singapore）；完成状态仅指本轮接入规划。
