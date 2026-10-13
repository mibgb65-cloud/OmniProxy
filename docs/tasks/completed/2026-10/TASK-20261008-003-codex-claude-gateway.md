# TASK-20261008-003: Codex 与 Claude 网关接入

## Status

Completed

## Created

2026-10-08 10:32（Asia/Singapore）

## Request

网关页面先提供给 Codex CLI／Desktop 和 Claude CLI／Desktop 使用的接口。去掉「运行本地 DeepSeek 代理，检查接入并验证实际请求。」这句说明。

用户补充：本次先完成页面、地址和配置说明，后端协议随后实现；Claude Desktop 指 Claude Code 桌面功能。

## Goal

网关页面清晰呈现 Codex、Claude 两类接入方式，提供经过验证的协议地址与配置，并准确说明客户端支持范围。

## Analysis

- 现有网关仅支持 DeepSeek Chat Completions。Codex 的 Responses 与 Claude 的 Messages 均显示为协议待实现，与实际网关运行状态分开。
- Codex CLI 与 Desktop 通过 `config.toml` 配置带 `/v1` 的 Base URL；Claude Code CLI 的 `ANTHROPIC_BASE_URL` 使用网关根地址，客户端追加 `/v1/messages`。
- Claude Code Desktop 按官方 Third-Party Inference 配置入口说明，不声称 CLI 的环境变量或 `settings.json` 会自动作用于桌面端。
- 保留现有网关状态、地址复制、检查和 DeepSeek 测试能力；仅新增客户端接入预览及配置说明，不修改任何客户端本机配置。
- 相关网关文件为之前任务已有的未提交文件。保存修改前副本，本次保留工作区结果，避免提交整份文件时混入既有任务实现。

## Implementation Plan

1. 新增 Codex 与 Claude 两个接入卡片：Base URL、请求路径、协议待实现状态和复制功能。
2. 提供 CLI／Desktop 配置切换，使用模型与网关鉴权占位符；配置为预览，不写入用户客户端文件。
3. 移除原页面副标题及无用字典项，补齐中英文文案。
4. 验证地址格式／端口边界、配置格式、复制反馈、客户端模式、加载和错误状态及双语布局，运行 `task verify`。
5. 更新网关能力说明与任务记录，保留其他任务既有修改。

## Implementation Result

- 网关页新增 Codex 与 Claude 两个接入卡片：分别生成带 `/v1` 的 Codex Base URL 和不带 `/v1` 的 Claude Base URL，显示计划的 Responses／Messages 请求地址。
- 提供地址与配置复制、独立的 CLI／Desktop 模式、可见的协议待实现状态；即使现有网关运行中，也不将新协议显示为可用。
- Codex 提供 TOML 配置预览，Claude CLI 提供 JSON 预览，Claude Code Desktop 提供 Third-Party Inference 操作步骤及官方文档链接。模型与网关凭证均为占位值。
- 删除旧副标题及中英字典项。新增客户端组件、配置生成器及两个测试文件，补齐双语文案；现有网关启停、检查和 DeepSeek 测试仍保留。
- 仅本机规范地址可生成模板；复制失败使用安全文案、支持重试，加载时禁用复制，复制中阻止重复提交。
- 更新 `docs/product/GATEWAY.md` 的接入预览和能力边界。后端协议与客户端配置文件读写均不在本次范围内。

## Verification

- 网关专项初次 6 个文件、56 项测试通过；随后增加复制中禁止重复操作的回归测试，并纳入完整门禁验证。
- 完整前端测试 27 个文件、168 项通过；覆盖地址与端口边界、异常 URL、JSON 模板、TOML 字段顺序、地址／配置复制、重试、加载、实际运行端口、双语切换及桌面配置差异。
- 浏览器检查通过：960、1280、1920、2560px 窗口，中文亮色／英文暗色共八组；两个卡片在 1280px 起并排，窄窗口堆叠，没有卡片内部或文档横向溢出。
- 使用 `tomllib` 与 JSON 解析器核对实际页面展示的配置；检查 Claude 桌面模式的三步说明，并确认不会展示 CLI 配置文件作为桌面配置。
- `task verify BIN_DIR=C:/Users/mimanchi/AppData/Local/Temp/omniproxy-client-gateway-qa-bed838b3/quality-bin` PASS：同一质量门禁的格式、lint、类型检查、全部 Go／前端测试、绑定生成和 Windows 生产构建通过。只将构建输出目录改到临时路径，保留正在运行的开发程序。
- 开发进程 PID 13324 与后台任务 PID 28384 持续运行；Vite 首页 HTTP 200，Control API 健康状态为 `ok`／`development`，桌面窗口可见。
- 既有网关目录与修改前副本逐文件核对，仅 `gateway-view.tsx` 被修改；其他变更集中于本任务新增文件、网关字典和相关文档。Git diff 核对及 `git diff --check` 通过。
- 临时配置验证脚本、布局结果和截图位于 `C:/Users/mimanchi/AppData/Local/Temp/omniproxy-client-gateway-qa-bed838b3/`。

## Documentation

已同步网关能力说明与 `docs/plans/CURRENT.md`。产品核心、架构、技术栈和后端接口没有变化。

## Sources

- [Codex CLI／Desktop 网关配置](https://learn.chatgpt.com/docs/enterprise/connect-to-a-gateway)
- [Codex 配置参考](https://learn.chatgpt.com/docs/config-file/config-reference)
- [Claude Code CLI 与 Desktop 接入](https://code.claude.com/docs/en/llm-gateway-connect#desktop-app)

## Deviations

按用户确认，本次只完成页面、地址和配置说明，协议实现留待后续。质量门禁改用临时产物目录，保持开发窗口运行。

## Remaining Risks

Responses、Messages、客户端模型目录及本地鉴权尚待实现；本次模板只能作为接入预览，不代表请求可用。实际 CLI／Desktop 调用验证随协议实现开展。

## Git

Not committed by Agent. 被修改的网关页面和字典原本属于其他任务的未提交文件，直接暂存整份文件会包含此前实现，而单独提交本次新增组件会缺少既有未提交依赖。本次完整结果保留工作区，未改变 HEAD 或暂存区。

## Result

两个客户端接入页面、地址、配置预览与说明完成，副标题已移除，实际布局与完整质量门禁通过；后端协议按用户要求暂未实现。

## Completed

2026-10-08 10:59（Asia/Singapore）
