# OmniProxy

<div align="center">

**本地优先的 AI API 网关、账号调度器与额度观测控制台**

把 Codex、Claude Code、Claude Desktop、OpenCode、Pi Coding Agent、DeepSeek-TUI、Gemini CLI 以及 OpenAI / Anthropic 兼容客户端统一接入本机代理，由 OmniProxy 在本地完成账号选择、鉴权注入、失败重试、额度刷新、用量统计和客户端配置写入。

[English](README_EN.md) · [发布记录](docs/releases) · [Releases](https://github.com/mibgb65-cloud/OmniProxy/releases)

![Release](https://img.shields.io/github/v/release/mibgb65-cloud/OmniProxy?include_prereleases&label=release)
![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS-0078D4)
![Go](https://img.shields.io/badge/Go-1.26-00ADD8)
![Vue](https://img.shields.io/badge/Vue-3-42B883)
![Wails](https://img.shields.io/badge/Wails-v2-DF0000)

</div>

## 为什么需要 OmniProxy

本地 AI 开发工具越来越多，账号、Base URL、模型和额度状态却散落在不同配置文件里。OmniProxy 把这些分散状态收拢到一个本地桌面控制台中：

- 多个账号不再手动切换，由调度器按状态、选择范围和并发占用自动挑选。
- 客户端只连接 `127.0.0.1`，真实上游 Token 留在本机，由代理按厂商注入。
- Codex、Claude Code、Claude Desktop、OpenCode、Pi Coding Agent、DeepSeek-TUI 等工具可以一键写入稳定网关入口；后端服务商、凭据类型和默认模型在「网关路由」页面切换。
- 请求历史、模型 Token、失败原因、额度重置时间、API Key 余额和本地账单统计统一可见。

OmniProxy 不是云端中转服务。它面向个人本地开发场景，默认只监听 loopback 地址，凭据保存在本机数据目录中。

## 核心能力

| 能力 | 说明 |
| --- | --- |
| 本地透明代理 | 暴露 OpenAI、Anthropic、Codex、Pi、TokenRouter、AnyRouter、Zo Computer、Prem 等本地入口，自动注入上游鉴权。 |
| 网关路由 | 为 Codex、Claude、OpenAI 兼容和 Gemini 客户端提供稳定入口，后端厂商选择集中在网关层完成，切换厂商不再重写客户端配置。 |
| 多账号调度 | 支持队列模式、优先平衡使用、账号选择范围、低额度跳过和并发占用避让。 |
| 失败自动切换 | 上游返回 `429`、`502`、`503`、`504` 等可重试错误时，自动换账号重试。 |
| 额度观测 | 展示 API 余额、订阅额度、重置时间、Codex Free 周额度、Coding Plan 用量、OpenRouter 余额和按币种汇总的 API Key 余额。 |
| 用量统计 | 记录请求历史、客户端来源、模型、输入 / 输出 / 总 Token、失败原因、每日账单快照和账单明细洞察。 |
| 客户端配置 | 一键配置 Codex、Claude Code、Claude Desktop、Gemini CLI、OpenCode、Pi Coding Agent、DeepSeek-TUI，并支持恢复原配置。 |
| 现代桌面控制台 | Gemini 风格浅色 / 深色主题，统一卡片、弹窗、下拉框、滚动和消息提示，适合长时间观察本地代理状态。 |
| Claude 模型槽位 | 可从 DeepSeek、MiMo、Kimi、GLM、Zo Computer 模型中选择最多 4 个模型写入 Claude Code / Claude Desktop。 |
| Zo Computer 网关 | 通过本地 `/zo` 和 `/zo/v1` 入口适配 OpenAI Chat Completions、OpenAI Responses、Anthropic Messages 和模型列表。 |
| 本地安全存储 | Windows 使用当前用户 DPAPI；macOS 使用 Keychain 保存主密钥并加密账号凭据，导出备份时保持显式可控。 |

## 架构

```mermaid
flowchart LR
  subgraph Clients["本地客户端"]
    Codex["Codex"]
    Claude["Claude Code"]
    ClaudeDesktop["Claude Desktop"]
    OpenCode["OpenCode"]
    Pi["Pi Coding Agent"]
    DeepSeekTUI["DeepSeek-TUI"]
    API["OpenAI / Anthropic Client"]
  end

  subgraph OmniProxy["OmniProxy Desktop"]
    Console["Vue 控制台"]
    Proxy["Local Gateway :3000"]
    Scheduler["Token Pool + Scheduler"]
    Logs["History / Billing / Quota"]
  end

  subgraph Providers["上游服务"]
    OpenAI["OpenAI / Codex"]
    Anthropic["Anthropic"]
    DeepSeek["DeepSeek"]
    Kimi["Kimi"]
    Mimo["Xiaomi MiMo"]
    More["Zhipu / MiniMax / Gemini / OpenRouter / TokenRouter / sub2api / new-api / AnyRouter / Zo / Prem / Custom"]
  end

  Clients --> Proxy
  Console --> Scheduler
  Proxy --> Scheduler
  Scheduler --> Logs
  Scheduler --> OpenAI
  Scheduler --> Anthropic
  Scheduler --> DeepSeek
  Scheduler --> Kimi
  Scheduler --> Mimo
  Scheduler --> More
```

## 最新变化

- **Codex 浏览器登录**：账号管理页新增「登录 Codex」，通过 PKCE 和本机回调完成浏览器授权，并自动新增或更新 OpenAI `auth.json` 账号。
- **Claude 浏览器登录**：Anthropic 账号页新增「登录 Claude」，通过 PKCE 和动态本机回调完成授权，自动新增或更新 Claude OAuth 账号并支持令牌续期。
- **Codex 额度刷新卡**：额度页可查看刷新卡数量、发放/到期/使用记录，并可在二次确认后消耗刷新卡重置 5 小时额度。
- **Codex 额度窗口自动激活**：可在「全局设置」中显式开启；检测到 5 小时或周额度尚未开始计时时，会发送一次最小 Codex 请求并复查，已激活账号不会重复发送。
- **Codex 额度激活可观测性**：开启自动激活后，保存设置会立即扫描 Codex 账号；额度页 OpenAI 分组提供统一的「检测并激活」入口，逐个检查当前账号，激活开始、成功、跳过和失败会写入请求历史。
- **Gemini 风格界面重构**：桌面控制台统一为现代极简样式，覆盖仪表盘、额度、账号管理、请求历史、实时日志、用量趋势、费用账单、一键配置、全局设置和 OpenRouter 对话等页面。
- **桌面交互优化**：统一下拉框、弹窗、全局消息提示、滚动条、按钮和卡片样式；各子页面独立记录滚动位置，实时日志改为仅展示最近 5 分钟并在页面内部滚动。
- **网关路由配置**：Codex、Claude Code、Claude Desktop、OpenCode、Pi、DeepSeek-TUI 和 Gemini CLI 默认写入稳定本地入口；上游服务商、凭据类型和默认模型改为在「网关路由」页面选择。
- **Zo Computer 网关**：新增 Go 原生 Zo Computer 适配，支持 `/zo/v1/chat/completions`、`/zo/v1/responses`、`/zo/v1/messages` 和模型列表兼容接口。
- **AnyRouter / Prem 接入**：AnyRouter、Prem 等第三方上游继续支持直连路径，也可以作为网关路由的后端厂商，由 OmniProxy 负责多账号调度和鉴权注入。
- **Claude Desktop 与 DeepSeek-TUI**：新增 Claude Desktop 3P Gateway Profile 和 DeepSeek-TUI 本地配置写入 / 恢复。
- **API Key 余额汇总**：厂商额度页和账号管理页支持按币种汇总 API Key 余额，GLM 等资源包明细会保留展示。
- **账单明细增强**：费用账单右侧明细区新增费用洞察、模型占比和未纳入模型摘要，并优化暗色模式海报预览。
- **Codex Chat Completions 兼容入口**：新增 `/codex/v1/chat/completions`，可用 OpenAI `auth.json` 账号接入 OpenAI Chat Completions 客户端，内部自动转换到 Codex Responses 后端。
- **Codex 流式响应转换**：Codex Responses 的 SSE 事件会转换为 `chat.completion.chunk`，非流式请求会汇总为 `chat.completion` 响应。
- **Codex 模型与参数适配**：支持 GPT-5.6 Sol / Terra / Luna，以及 `gpt-5.6-sol-high` 等 Codex CLI 模型别名，并保留 `max_completion_tokens`、`reasoning_effort`、tools / function calling 等常用参数。
- **Codex 请求体兼容**：支持解码 Codex 发往本地 Responses 入口的 zstd / gzip 压缩请求体。

## 快速开始

### 下载使用

1. 从 [GitHub Releases](https://github.com/mibgb65-cloud/OmniProxy/releases) 下载安装包。Beta 版本可能提供未签名 macOS DMG 供测试，正式日常使用仍优先选择 Windows 安装包或已签名版本。
2. 启动 OmniProxy，在「账号管理」添加至少一个上游账号。
3. 在「网关路由」确认各客户端后端厂商和默认模型，在「全局设置」确认代理端口和厂商 Base URL。
4. 启动本地代理。
5. 将客户端 Base URL 指向 `http://127.0.0.1:3000`，或使用「一键配置」写入本地客户端配置。

### 从源码运行

依赖：

- Go
- Node.js
- Wails v2 CLI

```powershell
cd .\OmniProxyBackend
C:\Users\mimanchi\go\bin\wails.exe dev
```

或使用仓库脚本：

```powershell
.\scripts\dev.ps1
```

## 本地入口

| 协议 / 客户端 | 正式版地址 | Dev 版地址 |
| --- | --- | --- |
| OpenAI compatible | `http://127.0.0.1:3000` | `http://127.0.0.1:3001` |
| Codex backend | `http://127.0.0.1:3000/backend-api/codex` | `http://127.0.0.1:3001/backend-api/codex` |
| Codex Chat Completions | `http://127.0.0.1:3000/codex/v1` | `http://127.0.0.1:3001/codex/v1` |
| Claude router | `http://127.0.0.1:3000/anthropic-router` | `http://127.0.0.1:3001/anthropic-router` |
| Gemini router | `http://127.0.0.1:3000/gemini` | `http://127.0.0.1:3001/gemini` |
| Pi router | `http://127.0.0.1:3000/pi-router/v1` | `http://127.0.0.1:3001/pi-router/v1` |
| TokenRouter | `http://127.0.0.1:3000/tokenrouter/v1` | `http://127.0.0.1:3001/tokenrouter/v1` |
| AnyRouter Codex / OpenAI | `http://127.0.0.1:3000/anyrouter/v1` | `http://127.0.0.1:3001/anyrouter/v1` |
| AnyRouter Claude Code | `http://127.0.0.1:3000/anyrouter/anthropic` | `http://127.0.0.1:3001/anyrouter/anthropic` |
| Zo Computer | `http://127.0.0.1:3000/zo/v1` | `http://127.0.0.1:3001/zo/v1` |
| Prem | `http://127.0.0.1:3000/prem/v1` | `http://127.0.0.1:3001/prem/v1` |
| Control API | `http://127.0.0.1:3890/api` | `http://127.0.0.1:3891/api` |

Prem 需要先运行官方 `pcci-proxy`，OmniProxy 默认把 Prem 上游指向 `http://127.0.0.1:3100/v1`，可在「全局设置」中修改。Prem 账号只需要保存 API Key；多 Key 由 OmniProxy 选择可用账号并注入到转发请求中。

Codex、Claude、OpenAI compatible、Pi router 和 Gemini router 是面向客户端的稳定入口。它们接到请求后会按「网关路由」配置选择 OpenAI、Anthropic、Gemini、DeepSeek、MiMo、sub2api、new-api、AnyRouter、Zo、Prem 或自定义网关等后端；表中的 TokenRouter、AnyRouter、Zo、Prem 等路径仍保留为高级用户的固定后端入口。

默认数据目录：

| 版本 | 数据目录 | 指针文件 |
| --- | --- | --- |
| 正式版 | `~/.omniproxy` | Windows: `%APPDATA%\OmniProxy\bootstrap.json`；macOS: `~/Library/Application Support/OmniProxy/bootstrap.json` |
| Dev 版 | `~/.omniproxy-dev` | Windows: `%APPDATA%\OmniProxyDev\bootstrap.json`；macOS: `~/Library/Application Support/OmniProxyDev/bootstrap.json` |

## 支持矩阵

| 厂商 | 凭据类型 | 主要能力 |
| --- | --- | --- |
| OpenAI | API Key | OpenAI 兼容请求、rate-limit header 余量记录。 |
| OpenAI / Codex | `auth.json` | 自动解析邮箱、access token、account id，刷新 Codex 订阅额度，支持 Codex Responses 与 Chat Completions 兼容转换。 |
| Anthropic | API Key | Anthropic 原生请求和 Claude Code 路由。 |
| Anthropic / Claude | OAuth JSON | 支持浏览器 OAuth 登录、自动导入与刷新，也可手动粘贴包含 `access_token` / `refresh_token` 的 Claude OAuth JSON。 |
| DeepSeek | API Key | OpenAI 兼容入口和 Anthropic router。 |
| Kimi | API Key | Kimi Code 相关路由和订阅用量刷新。 |
| Xiaomi MiMo | API Key | 按量 API Key，通常以 `sk-` 开头。 |
| Xiaomi MiMo | Token Plan | Token Plan Key，通常以 `tp-` 开头，支持订阅额度展示。 |
| Zhipu GLM | API Key / Coding Plan | OpenAI 兼容、Anthropic router、Coding Plan 用量刷新。 |
| MiniMax | API Key | OpenAI 兼容入口和 Anthropic router。 |
| Gemini | API Key | Gemini API 路由和 Gemini CLI 一键配置。 |
| OpenRouter | API Key | 模型列表、余额查询、桌面端对话。 |
| TokenRouter | API Key | OpenAI 兼容路由，API Key 通常以 `tr_` 开头。 |
| sub2api | API Key | OpenAI / Anthropic / Gemini 兼容网关，可作为网关路由后端或固定后端入口。 |
| new-api | API Key | OpenAI / Anthropic / Gemini 兼容网关，默认 `http://127.0.0.1:3000`，通过 `/api/usage/token/` 刷新 Key 额度。 |
| AnyRouter | API Key | Codex/OpenAI 与 Claude Code/Anthropic 兼容网关，默认 `https://anyrouter.top`。 |
| Zo Computer | Access Token | OpenAI Chat Completions、OpenAI Responses、Anthropic Messages、模型列表和客户端模型预设。 |
| Prem | API Key | 通过 Prem 官方 `pcci-proxy` 本机 OpenAI 兼容服务转发，支持多 Key 调度。 |
| 自定义网关 | API Key | OpenAI / Anthropic 兼容网关。 |

## 客户端一键配置

| 客户端 | 支持内容 |
| --- | --- |
| Codex | 优先从 auth 池选择一个 OpenAI `auth.json` 账号作为 ChatGPT 登录身份；没有可用账号时自动切换为本地 API 登录。两种模式都写入稳定 `/codex/v1` 网关入口，实际请求仍由后端路由和凭据池独立调度。 |
| Claude Code | 写入稳定 Anthropic router，并可选择最多 4 个模型槽位；后端厂商在网关路由中选择。 |
| Claude Desktop | 写入 3P Gateway Profile，可复用 Claude 模型选择结果；后端厂商在网关路由中选择，配置后需要重启 Claude Desktop。 |
| Gemini CLI | 写入稳定 Gemini router，后端厂商和默认模型在网关路由中选择。 |
| OpenCode | 只写入 `omniproxy` provider 和 `/opencode-router/v1`，后端厂商在网关路由中选择。 |
| Pi Coding Agent | 只写入 `omniproxy` provider 和 `/pi-router/v1`，后端厂商在网关路由中选择。 |
| DeepSeek-TUI | 写入 `omniproxy` provider 和 `/opencode-router/v1`，不再绑定 DeepSeek 内置 provider。 |

## 控制 API

桌面前端优先通过 Wails 绑定调用后端。HTTP 控制 API 保留给本地脚本和调试工具使用。`GET /api/control-token` 仅允许桌面端可信来源读取；其它接口需要携带当前运行期 `X-OmniProxy-Control-Token`，也支持 `Authorization: Bearer <token>`。

常用端点：

| 类型 | 端点 |
| --- | --- |
| 账号 | `GET /api/tokens`、`POST /api/tokens`、`POST /api/tokens/import-api-keys`、`PUT /api/tokens/{id}`、`DELETE /api/tokens/{id}` |
| 调度 | `PUT /api/tokens/{id}/selected`、`PUT /api/tokens/{id}/exclusive`、`DELETE /api/tokens/{id}/exclusive` |
| 验证 | `POST /api/tokens/{id}/validate` |
| 代理 | `GET /api/proxy/status`、`POST /api/proxy/start`、`POST /api/proxy/stop`、`GET /api/proxy/active-requests` |
| 配置 | `GET /api/config`、`PUT /api/config`、`GET /api/data-directory` |
| 历史 | `GET /api/logs`、`GET /api/history`、`DELETE /api/history/clear` |
| 账单 | `GET /api/billing/usage`、`GET /api/billing/dates`、`DELETE /api/billing/clear` |
| 客户端配置 | `POST /api/codex/configure`、`POST /api/codex/restore`、`POST /api/claude/models/configure`、`POST /api/claude/restore`、`POST /api/claude/desktop/models/configure`、`POST /api/claude/desktop/restore`、`POST /api/deepseek-tui/configure`、`POST /api/deepseek-tui/restore`、`POST /api/gemini/configure`、`POST /api/gemini/restore`、`POST /api/opencode/configure`、`POST /api/opencode/restore`、`POST /api/pi/configure`、`POST /api/pi/restore` |
| 更新 | `GET /api/update/check`、`POST /api/update/download`、`GET /api/update/download/status`、`GET /api/update/diagnostics`、`POST /api/update/install` |

`/selected` 用于把账号加入或移出所属厂商的调度选择集合。没有已选账号时，调度器默认轮换该厂商全部可用账号；存在已选账号时，只在已选账号内轮换。

## 开发与验证

```powershell
cd .\OmniProxyBackend
go test ./...
```

```powershell
cd .\frontend
npm test
npm run build
```

正式构建：

```powershell
cd .\OmniProxyBackend
C:\Users\mimanchi\go\bin\wails.exe build
```

macOS universal 构建必须在 macOS runner 或 Mac 机器上执行，Windows 不能交叉编译 Wails 的 Darwin 包：

```bash
cd OmniProxyBackend
wails build -clean -platform darwin/universal -ldflags "-X main.appVersion=v1.1.9"
```

当前 beta 发布流程会生成 ad-hoc 签名的 `OmniProxy-<tag>-darwin-universal-unsigned.dmg` 测试附件。没有 Apple Developer ID 签名和公证前，该 DMG 仅用于测试，macOS 可能显示 Gatekeeper 警告。

可与正式版共存的 Dev 构建：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build-dev.ps1 -Version dev -OutputName OmniProxy-dev.exe
```

Dev 版使用 `omniproxy_dev` build tag，应用标题、单实例 ID、数据目录和默认端口都与正式版隔离，适合在已安装正式版的机器上并行验证。

## 项目结构

```text
.
├── OmniProxyBackend/              # Wails 桌面主工程与 Go 后端
│   ├── internal/config/           # 本地配置、数据目录、默认值
│   ├── internal/clientconfig/     # 本地客户端配置文件写入工具
│   ├── internal/logs/             # 请求和诊断日志
│   ├── internal/proxy/            # 代理、路由、鉴权、用量解析、WebSocket
│   ├── internal/storage/          # JSON / SQLite 本地持久化
│   ├── internal/token/            # 账号模型、Token 池、调度、额度状态
│   └── frontend-dist/             # 前端构建产物嵌入目录
├── frontend/                      # Vue 3 + Vite + Element Plus 前端
│   ├── src/features/              # 页面级功能模块
│   ├── src/domain/                # 前端共享领域规则
│   └── src/components/            # 复用 UI 组件
├── docs/releases/                 # 人工整理的发布说明
├── scripts/dev.ps1                # 桌面开发启动脚本
├── scripts/build-dev.ps1          # 可共存 Dev exe 构建脚本
├── README.md                      # 中文文档
└── README_EN.md                   # English README
```

## 发布通道

| 通道 | Tag 示例 | GitHub Release 行为 |
| --- | --- | --- |
| Stable | `v1.2.0` | 正式 Release，适合日常使用；当前公开附件仍以 Windows 安装包为主。 |
| Beta | `v1.2.1-beta.1` | Pre-release，适合验证新功能和回归修复；可附加未签名 macOS DMG 供测试。 |
| Dev | `dev-*` | 本地构建版本，不作为公开 Release。 |

发布说明位于 `docs/releases/`。正式版发布前按 [发布检查清单](docs/release-checklist.md) 复核安装、更新、卸载和资产校验。Beta 版本会标记为 GitHub Pre-release，正式版保留给稳定发布。
发布正式版后，工作流会自动删除同版本线的 Beta GitHub Release 及附件，但保留 Beta Git 标签和 `docs/releases/` 中的发布说明。

## 安全模型

- 默认只绑定 `127.0.0.1`，不面向公网或局域网暴露。
- 控制 API 使用本地控制令牌保护，桌面端自动获取和携带。
- Windows 上账号凭据使用当前用户 DPAPI 加密写入本地数据目录；macOS 上使用 Keychain 保存主密钥，再用本地加密信封写入数据目录。
- 导出的账号池备份、Codex `auth.json` 和客户端配置备份可能包含真实凭据，请只保存到可信目录。
- 分享日志、截图或 Issue 前，请检查账号名、路径、请求 ID、Base URL 和 provider metadata。

## 路线图

- 更细粒度的额度趋势图和跨厂商对比视图。
- 更完整的 SSE、WebSocket、并发调度和异常恢复测试。
- 更多厂商、更多客户端工具和更多协议适配。
- 更严格的控制 API 本地访问边界。
- 更清晰的前端组件边界和可维护的设计系统。

## 贡献

欢迎提交 Issue 和 Pull Request。一个高质量问题报告通常包含：

- 操作系统、OmniProxy 版本和运行方式。
- 使用的客户端工具，例如 Codex、Claude Code、OpenCode、Pi Coding Agent 或自定义 API 客户端。
- 相关 provider、路由路径、模型名和错误日志。
- 预期行为、实际行为以及最小复现步骤。

提交 PR 前建议至少运行：

```powershell
cd .\OmniProxyBackend
go test ./...
```

```powershell
cd .\frontend
npm test
npm run build
```

## Star

如果 OmniProxy 改善了你的本地 AI 开发流程，欢迎点一个 Star。真实使用场景里的问题反馈、配置样例和回归用例，比泛泛的路线图更有价值。
