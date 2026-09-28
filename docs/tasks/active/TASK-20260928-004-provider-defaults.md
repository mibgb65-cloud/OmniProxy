# TASK-20260928-004: 内置提供商默认启用与自定义提供商

## Status

In Progress

## Created

2026-09-28 11:56

## Request

九家预设提供商不应要求用户手动添加，默认启用，并用滑块开关控制启停。用户可手动新增自定义提供商，自定义项默认启用，也使用滑块开关。用户确认自定义提供商本次按 OpenAI 兼容接口创建，填写名称和 Base URL。随后用户提供参考图，要求卡片加入 AI 图标、调整滑块位置、显示接入方式、移除“配置 Key”操作、加入测试连通与延迟的按钮，并显示可展开的模型列表。用户确认当前无账号密钥时按 API 地址的网络可达性及响应耗时测试，401/403 视为地址可达；模型列表先使用官方预设，不代表实时账号权限。

## Goal

安装或升级后，九家内置提供商自动存在且默认启用；启停状态持久化。内置项不提供新增、编辑或删除。自定义 OpenAI 兼容提供商可新增、编辑、启停和删除，新增后默认启用。升级时保留已有手动创建的配置及其账号关联。

## Existing Behavior

九家仅作为可选择的预设目录展示，用户仍需点击“添加配置”才能得到一条启用的数据库记录；自定义提供商尚无独立入口。

## Expected Behavior

预设卡片直接显示各自的开启状态和右上角滑块、AI 图标、接入方式、模型摘要及展开、连通/延迟测试；自定义区单独提供“新增自定义提供商”，要求有效 Base URL，创建后直接启用并可用滑块关闭。页面不出现“配置 Key”按钮。

---

# Analysis

## Options

1. 每次读取时虚拟补全内置项：无需迁移，但账号外键难以稳定引用内置提供商。
2. 新增 append-only SQL 迁移，将内置项以固定 ID 持久化，并用 `source` 标记内置/自定义：可保持现有 `accounts.provider_id` 外键和启停状态。

采用方案 2。现有手动创建的行升级后标记为 `custom` 并保持原 ID、字段与账号引用；迁移只新增九条内置行。

## Related Modules

- `migrations/`、`internal/application/provider`
- `internal/transport/wails/providers.go`
- `frontend/src/features/providers/`、提供商中英文文案与生成绑定

## Risks

- 迁移必须保留旧配置和账号；不能修改已应用的 `001_initial.sql`。
- 内置项不得经服务接口被删除或改写种类/地址；自定义项删除仍须保护账号凭证引用。
- 自定义 Base URL 属于外部输入，应继续强制 HTTPS 或本机 HTTP，并要求非空。
- 未录入账号凭证，连通测试不得宣称账号认证成功；请求不携带密钥、超时有界、不跟随重定向，并把 HTTP 401/403 视为已接通目标地址。
- 官方预设模型可能变动，界面应标注为预览而非实时账号可用列表。

---

# Implementation Plan

1. 新增迁移与内置/自定义服务模型，写升级、默认启用、切换状态和删除保护测试。
2. 将页面改为九家内置项的状态滑块，以及独立的自定义提供商列表、创建表单和状态滑块；按参考图调整卡片层级、图标、接入方式和可展开模型。
3. 使用官方 API 地址实现无凭证的网络连通/响应耗时测试，并明确结果含义、超时和 URL 边界。
4. 更新文案、架构与任务记录；运行 `task verify`，检查真实窗口与 Git 差异并提交。

---

# Progress

- 新增 append-only 迁移，九家内置提供商以固定 ID、启用状态写入数据库；旧手动配置通过 `source` 默认值保留为自定义。
- 服务层只允许内置项切换状态；自定义 OpenAI 兼容项创建时强制名称和 Base URL，默认启用，可编辑、启停及受账号关联保护的删除。
- 升级测试确认旧配置、旧账号引用未变化，九家默认启用；重复运行迁移不会重置用户关掉的开关。
- 前端改为内置卡片滑块与独立的自定义提供商区域；定向页面测试通过。
- 参考图布局已实现：通用 AI 图标、右上滑块、接入方式、可展开模型与底部状态/测试操作，无“配置 Key”按钮。
- 以官方文档核对九家 API 地址和代表性模型；连通测试使用无密钥 HEAD 请求，最多等待 6 秒，不跟随重定向，任何 HTTP 响应都归为地址可达并展示状态码。
- `task verify` 全部通过。实际 Wails 窗口视觉检查通过；OpenAI 地址测试返回 HTTP 404，界面正确显示“地址可达”和约 930 ms 响应耗时。

## Model Preview Sources

预设模型为截至 2026-09-28 的展示示例，不代表账号实时可用列表：

- [OpenAI 模型](https://developers.openai.com/api/docs/models/all)
- [Anthropic 模型](https://platform.claude.com/docs/en/models/overview)
- [Google Gemini 模型](https://ai.google.dev/gemini-api/docs/models)
- [DeepSeek 模型](https://api-docs.deepseek.com/quick_start/pricing)
- [智谱 / Z.ai 模型](https://zcode.z.ai/cn/docs/configuration)
- [Kimi API 模型](https://www.kimi.ai/help/kimi-api/api-troubleshooting)
- [MiniMax 模型](https://platform.minimax.io/subscribe/token-plan)
- [小米 MiMo 模型](https://mimo.mi.com/docs/zh-CN/quick-start/summary/first-api-call)
- [阶跃星辰模型](https://platform.stepfun.com/)

---

# Implementation Result

## Changes

- 新增 `002_builtin_providers.sql`：`providers.source` 区分内置/自定义，九家内置项以固定 ID 和默认启用状态写入；旧手动配置保留为自定义。
- 应用服务限制内置项只能切换启停；自定义 OpenAI 兼容项要求名称和 Base URL，新增时默认启用，支持编辑、开关和受账号关联保护的删除。旧版手动配置即使 Base URL 为空仍可编辑。
- 为九家内置项提供代码内默认 API 地址和代表性模型预览。模型列表可展开；自定义端点尚无模型预设。
- 新增不带密钥的 HEAD 连通测试：6 秒超时、不跟随重定向、读取 HTTP 响应头耗时，任何收到的 HTTP 状态都显示为地址可达；网络失败显示不可达。
- 参考图重排卡片：通用 AI 图标在左上、滑块在右上、接入方式标签和模型摘要居中，底部显示启停状态及测试入口；移除旧的“添加预设”流程和 Key 操作。

## Files Changed

- `migrations/002_builtin_providers.sql`
- `internal/application/provider/{catalog.go,service.go,mutations.go,probe.go,validation.go}` 及对应测试、`main_test.go`
- `internal/transport/wails/providers.go`、生成的 Wails 绑定
- `frontend/src/features/providers/{provider-card,provider-catalog,provider-list,provider-form,provider-switch,providers-view,use-providers}.*`
- `frontend/src/services/omniproxy.ts`、`frontend/src/i18n/locales/{zh,en}.ts`
- `docs/product/PRD.md`、`docs/architecture/{OVERVIEW,TECH_STACK}.md`、`docs/plans/CURRENT.md`、本任务记录

## Technical Notes

- 连通测试只反映目标地址的网络响应及耗时；即使 HTTP 401/403/404，也不代表账号认证或模型调用成功。
- 模型为可维护的代码内预览列表，实际可用范围取决于厂商和账号权限。自定义 OpenAI 兼容接口的模型将在账号接入后获取。
- 图标采用项目既有的 Lucide AI 图形与区分色，不声称是厂商官方商标。
- `providers-view.test.tsx` 略超 200 行，因为共用九家预设与多种页面流程的测试装配；拆分会重复装配，保留在同一文件便于审阅。

---

# Verification

## Tests

- [x] 旧数据升级、九家默认启用、开关持久化和重复迁移不会重置状态。
- [x] 内置项编辑/删除保护、自定义项必填地址和账号关联删除保护。
- [x] 无凭证 HEAD 请求、401 可达、不跟随重定向、网络失败。
- [x] 页面开关、自定义配置、模型展开及测试结果交互。
- [x] 实际 Wails 窗口视觉与连通测试。

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

`task verify` PASS：Go 测试通过；前端 14 个测试文件、59 个测试通过；Windows 生产构建成功。实际开发数据库 `builtin|9|9`，无自定义测试记录。Wails 窗口可见新卡片布局，OpenAI 无凭证地址测试返回 HTTP 404，界面显示地址可达、930 ms。`git diff --check` PASS。

---

# Documentation

PRD、架构概览、技术栈和 CURRENT 看板已同步默认提供商、静态模型预览和无凭证连通测试的边界；模型资料来源见本记录。

---

# Deviations

用户在开始后追加参考图及测试/模型需求；实现计划已同步扩展。技术方案仍使用既有 Wails、SQLite、React 与 Lucide 栈，没有引入新依赖。

---

# Git

## Commit

待完成。

---

# Result

九家内置提供商默认启用并可直接切换；自定义 OpenAI 兼容提供商可新增且默认启用。新卡片展示 AI 图标、接入方式、可展开模型与无凭证连通/延迟测试。
