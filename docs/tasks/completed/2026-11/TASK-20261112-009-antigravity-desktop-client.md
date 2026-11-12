# TASK-20261112-009: Antigravity 桌面端接入本地网关

## Status

Completed

## Created

2026-11-12 19:20（Asia/Singapore）

## Request

用户要求给 Antigravity 桌面端做类似 ChatGPT 订阅的透传，使用多个 Google 订阅账号，
并且只适配 Google 账号已有的模型，不引入其他模型。

> 可以给 antigravity 的客户端做一个类似于 ChatGPT 的订阅的那种透传吗？
> 我现在有几个账号，想使用它，另外，这个 antigravity 就适配谷歌账号有的模型就可以了

补充确认：目标只是桌面端 Antigravity（不是 IDE，也不是 agy CLI）。

## Background

`E:\githubresource\magpie` 的 `internal/agent/antigravity.go` 记录了官方桌面端的
**Antigravity Enterprise Gateway**：

- 设置文件：2.30+ 为 `~/.gemini/config/config.json`，2.29 及以前为
  `~/.antigravity/settings.json`；两版各读自己的文件。
- 结构：`{"gateway":{"enabled":true,"url":…,"apiKey":…,"wireProtocol":"openai"},
  "models":{"defaultModelId":…,"listSource":…}}`。
- 模型目录：`GET <url>/v1/models?format=antigravity`，返回
  `{"list":[{modelId,displayName,maxTokens,maxOutputTokens,supportsImages,supportsThinking}]}`。
- 对话：`POST <url>/v1/chat/completions`，流式，`Authorization: Bearer <apiKey>`，
  `User-Agent: antigravity/hub/<version>`。
- 启用网关后不使用 Google 登录；Google 账号专属的实验与命令（`/boost`、`/teamwork`、
  `/plugin`、`/generative_ui`）在该模式下不可用。

因此不需要 Codex 式 Base URL 之外的协议转换，也不需要本地 CA / TLS 拦截。

## Goal

- 在网关新增 Antigravity 桌面端一键接入：写入/恢复两个设置文件，保留用户其他字段。
- 提供 `format=antigravity` 模型目录，发布 Google 订阅账号实际提供的全部文本模型
  （Gemini、Claude、GPT-OSS）。
- 复用现有 Google 订阅多账号路由、额度、故障切换与请求记录。
- 网关页提供 Antigravity 卡片：连接、断开、默认模型、重启提示与安全边界。

## Scope

- `internal/application/agentconfig`：新客户端 `antigravity`、双文件备份、配置生成。
- `internal/core/proxy`：`/v1/models?format=antigravity` 目录；桌面端 Chat 请求规整。
- `internal/core/modelcatalog`：Antigravity 目录 JSON 形状与 Google 订阅模型过滤。
- 前端网关页卡片、i18n、服务层客户端白名单。
- 文档：`GOOGLE_SUBSCRIPTION.md`、`CURRENT.md`、ADR（如边界变化）。

## Out of Scope

- Antigravity IDE（VS Code fork）与 agy CLI 的独立接入；二者不读同一版本文件时另行处理。
- 引入 Google 订阅之外的提供商；订阅内的 Claude／GPT-OSS 本任务一并发布。
- 图片、语音等多模态；订阅通道当前明确拒绝。
- 用 MITM / 本地 CA 拦截官方流量。

## Existing Behavior

- 13 个受管客户端通过 `agentendpoint` 固定前缀接入；`agentconfig.Clients()` 是白名单。
- Google 订阅账号已支持 Chat/Responses/Messages、额度与故障切换。
- `/v1/models` 只返回 OpenAI 列表；不支持 `format` 参数。
- 没有 Antigravity 客户端配置读写。

## Expected Behavior

- 网关页出现“Antigravity”卡片；检测到设置文件后可一键连接。
- 连接后 Antigravity 重启即读取本地网关模型与对话接口。
- 目录出现 Google 订阅账号实际提供的全部文本模型（Gemini、Claude、GPT-OSS）。
- 断开恢复原 `gateway` / `models` 块，不覆盖用户其他设置。

---

# Analysis

## Related Modules

- `internal/application/agentconfig`：`Clients`、`Paths`、`prepare`、`buildPlan`、
  `readClient`/`writeClient`、`ConfiguredModels`、`Restore`。
- `internal/core/proxy`：`handler.go`、`models.go`、`agent_auth.go`。
- `internal/core/modelcatalog`：目录形状与 Google 订阅模型过滤。
- `internal/application/gateway`：Google 订阅转发与请求规整。
- 前端 `features/gateway`、`services/agents.ts`、`i18n/locales/agents-*`。

## Key Decisions

- 网关根地址作为 `gateway.url`（`http://127.0.0.1:<port>`），不使用路径前缀；
  客户端归属由独立接入 Key 决定，避免依赖桌面端对 URL 路径的拼接行为。
- 发布 `providerKind=google` 的全部文本模型（Gemini、Claude、GPT-OSS）；目录仍由现有
  模型目录服务生成，不引入其他提供商。
- 桌面端请求里的 `reasoning_effort` 等 Google 订阅适配器不接受的字段，只在
  `client=antigravity` 时剥离；不改变其他客户端行为。
- 两个设置文件作为双文件 bundle 备份/恢复；连接时都写入，覆盖 2.29/2.30 两个版本。
- 删除 `models.list`（内联列表会遮住目录），断开时由备份恢复。

## Risks

- 桌面端版本升级可能改变设置文件位置或字段；`Inspect` 会报告配置变化，不静默覆盖。
- 桌面端启用的网关模式不提供 Google 登录态功能；需要在卡片上明确提示。
- Gemini／Claude／GPT-OSS 的上下文、输出、图片能力来自本地目录，未知字段不猜测。
- 桌面端真实请求形态未在本机抓包；先按 magpie 记录的 OpenAI Chat 形态实现，
  真机失败时按固定错误码补规整，不删除工具或截断内容。

---

# Implementation Plan

1. `agentendpoint` / `agentconfig.Clients()` / `Keys.Client` 白名单增加 `antigravity`。
2. `Paths` 增加两个设置文件；bundle 读写支持 antigravity。
3. `buildAntigravity` 写入 `gateway`、`models.listSource`、`models.defaultModelId`；
   `clientEntries` 返回 Google 订阅的全部文本模型。
4. 网关 `format=antigravity` 目录；`client=antigravity` 请求规整。
5. 前端网关页 Antigravity 卡片与 i18n。
6. 契约测试：配置生成/恢复、目录形状、请求规整、多账号路由。
7. `task verify`；更新文档与 CURRENT；提交。

---

# Progress

- 2026-11-12：核对 magpie 的 Antigravity 桌面端实现，确认 Enterprise Gateway 方案；
  建立任务记录，开始实现。
- 2026-11-12：完成后端配置读写、`format=antigravity` 目录、桌面端请求规整、前端卡片
  与契约测试；Go／前端质量门通过。真机请求验证待用户重启桌面端后确认。

---

# Implementation Result

- `internal/application/agentconfig`：
  - 新增客户端 `antigravity`，白名单、接入 Key 识别（`LIMIT 14`）、路径
    `~/.gemini/config/config.json` 与 `~/.antigravity/settings.json`。
  - 双文件 bundle 备份；`buildAntigravity` 写入
    `gateway: {enabled, url, apiKey, wireProtocol: "openai"}` 与
    `models: {defaultModelId, listSource}`，保留用户其他字段，删除会遮住远端目录的
    `models.list`。断开时按字段撤销并放回备份原值，保留桌面端在连接期间改写的
    `userSettings` 等设置；端口变更通过现有 `RebasePort` 同时改写两个文件。
  - `clientEntries` 发布 Google 订阅账号的全部文本模型（Gemini、Claude、GPT-OSS）；
    `ConfiguredModels` 报告写入的默认模型。
  - 设置文件不存在时返回 `client_not_found`，避免创建桌面端不会读取的文件。
- `internal/core/modelcatalog`：新增 `AntigravityJSON`，输出
  `{"list":[{modelId,displayName,maxTokens,maxOutputTokens,supportsImages,supportsThinking}]}`，
  保留 `providerKind=google` 的全部文本模型条目，不引入其他提供商。
- `internal/core/proxy`：
  - `/v1/models?format=antigravity` 返回桌面端目录。
  - `client=antigravity` 的 Chat 正文剥离 `reasoning_effort`／`service_tier`／
    `metadata`／`store`；桌面端 Chat 使用 16 MiB 请求与单条消息上限，其他客户端不变。
- 前端：网关页“客户端接入”新增 Antigravity 卡片，复用
  `AgentPreviewPlan`／`AgentApply`／`AgentRestore`，支持默认模型选择（Gemini／Claude／
  GPT-OSS）、
  断开恢复、重启与 Google 登录功能暂时关闭提示；i18n、服务白名单与错误码同步。
- 文档：`GOOGLE_SUBSCRIPTION.md`、PRD、CURRENT。

---

# Verification

- `gofmt -l main.go internal`：clean。
- `go vet ./internal/application/agentconfig ./internal/core/proxy ./internal/core/modelcatalog`：通过。
- `go test . ./internal/... -count=1`：全部通过（52 个包）；覆盖配置生命周期、
  桌面端改写 `userSettings` 后的字段级断开、端口改写、目录形状与请求规整。
- `pnpm typecheck`、`pnpm lint`（2 个既有 Fast Refresh warning）、`pnpm format:check`：通过。
- `pnpm vitest run src/features/gateway --maxWorkers=4`：30 文件／198 项通过。
- `pnpm test -- --maxWorkers=4`：147 文件／896 项通过。
- `pnpm build`：生产前端构建通过。
- 生产 Go 编译（`-tags production -trimpath -buildvcs=false -ldflags="-w -s -H windowsgui"`）：
  通过，临时二进制 23,712,768 bytes 已删除。
- 未执行：真实 Antigravity 桌面端请求。当前环境未安装桌面端；需要用户先启动一次
  客户端，再在网关页连接、完全退出并重启，确认模型菜单出现 Google 订阅账号实际
  提供的全部模型（Gemini、Claude、GPT-OSS）并完成一次文本／工具请求。

---

# Documentation

- [x] PRD
- [ ] Architecture（既有组件边界未变化）
- [x] ADR（无需新增；复用 Enterprise Gateway 官方能力）
- [x] `docs/product/GOOGLE_SUBSCRIPTION.md`
- [x] CURRENT

---

# Git

## Commit

`3225476f feat(agent): 接入 Antigravity 桌面端官方网关`

`9fc76a87 fix(proxy): 接受 Antigravity 目录的 Google API Key 头`

`8aee0628 fix(agent): Antigravity 断开按字段恢复并支持端口改写`

---

# Result

桌面端可以通过官方 Enterprise Gateway 使用本地网关的 Google 订阅账号池，不需要
TLS 拦截、本地 CA 或系统代理。配置写入两个版本文件，断开可恢复；目录发布
Google 订阅账号实际提供的全部文本模型（Gemini、Claude、GPT-OSS）。代码、契约测试
与前端质量门已通过；真实桌面端请求仍需用户按卡片提示
重启后确认。未处理组织级 `admin_settings.json` 或 `AGY_LLM_GATEWAY_*` 覆盖。

---

# Completed

2026-11-12 19:55（Asia/Singapore）
