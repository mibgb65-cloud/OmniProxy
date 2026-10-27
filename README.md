# OmniProxy

**本地优先的 AI API 网关与多账号管理桌面应用。**

在一个地方管理 AI 提供商账号、模型路由、额度和请求记录，并将 Codex、Claude Code、
Gemini CLI、OpenCode 等客户端接入本机网关，减少反复修改配置和切换账号的成本。

[下载安装](https://github.com/mibgb65-cloud/OmniProxy/releases)
· [快速开始](#快速开始)
· [开发与构建](#开发与构建)
· [项目文档](docs/README.md)
· [社区致谢](#社区致谢)

## 功能概览

| 能力               | 说明                                                                                   |
| ------------------ | -------------------------------------------------------------------------------------- |
| 提供商与账号       | 内置多家提供商预设及自定义 OpenAI 兼容端点，统一管理 API Key 与已接入的订阅账号。      |
| 模型目录与路由     | 按账号同步实际模型目录，配置模型别名和固定账号，保留不同提供商的能力边界。             |
| 账号调度与故障切换 | 同通道内按规则选择账号，对明确拒绝进行有限换号、冷却和额度阻断；不静默跨提供商消费。   |
| 本地客户端接入     | 预览并确认客户端配置，管理接入模型和默认模型，提供受保护的配置备份与恢复。             |
| 请求与用量         | 查看请求结果、账号尝试、输入／输出／缓存 Token、耗时和结构化日志；未知用量不伪造成零。 |
| 额度与余额         | 按实际支持情况展示订阅额度窗口、官方余额或本机用量；本地估算不冒充官方账单。           |
| 提供商出站代理     | 支持 HTTP／SOCKS5、本机代理探测及按提供商分流，不修改系统代理或接管代理软件。          |
| 桌面体验与更新     | 系统托盘、主题与中英文界面、数据目录管理，以及稳定版／Beta 渠道的版本检查与确认更新。  |

内置提供商预设包括 OpenAI、Anthropic、Google、DeepSeek、智谱 AI、Moonshot AI／Kimi、
MiniMax、小米 MiMo、StepFun、Atria、River 和 Oxion AI；预设存在不等于所有推理路径均已接入。

网关提供 Chat Completions、Responses、Anthropic Messages 等接口；Codex 入口支持
HTTP／SSE 透明转发和 Responses WebSocket。**不同提供商、账号与模型支持的协议并不相同**，
具体范围见[网关说明](docs/product/GATEWAY.md)和[本地客户端接入](docs/product/LOCAL_AGENTS.md)。

## 下载安装

目前维护与发布的平台为 **Windows 10／11 x64**，需要 WebView2 Runtime。
Linux、macOS 及其他架构暂未作为已维护的发行目标。

1. 打开 [GitHub Releases](https://github.com/mibgb65-cloud/OmniProxy/releases)，选择需要的正式版或 Beta。
2. 下载对应的 Windows 安装包，核对该 Release 提供的 SHA256。
3. 运行安装向导，确认程序目录及数据存储位置，完成后启动 OmniProxy。

安装包使用**项目自签证书，而非公开受信任 CA 证书**，Windows 仍可能提示 SmartScreen
或未知发布者。请核对下载来源与校验值；安装器不会修改用户的证书信任库。
详情见[代码签名政策](CODE_SIGNING_POLICY.md)。

> 源码分支可能包含尚未发布的修复。已安装版本是否包含某项修改，以对应 Release 说明为准。

## 快速开始

1. **添加账号**：在提供商／账号页面添加自己的 API Key，或使用已支持的订阅授权入口。
2. **同步模型**：在路由页面按账号同步模型目录，按需配置模型别名或固定账号。
3. **启动网关**：在网关页面检查配置并启动本机服务。生产环境默认代理端口为 `3000`，以页面实际显示的地址为准。
4. **配置客户端**：进入「客户端接入」，选择已安装的客户端，预览并确认要写入的模型与配置。
5. **验证调用**：重新加载或重启对应客户端，发起请求，在请求页核对实际账号、结果与用量。

部分桌面客户端需要手动选择新建的配置条目；配置写入成功不等于客户端已经加载或上游请求成功。
具体步骤和版本限制见[本地客户端接入](docs/product/LOCAL_AGENTS.md)。

### 使用边界

- 需要使用者自行具备有效账号、API Key 或订阅权限；OmniProxy 不提供或出售上游额度。
- 显式固定的账号不会自动切换；故障切换不静默换模型，也不把订阅消费改为按量计费。
- 额度、余额和协议能力以所选账号及上游实际返回为准，不保证所有预设均支持全部功能。
- 账号配置、请求历史与本机统计在本地保存；实际推理请求仍会发送给所选上游服务商。

## 开发与构建

技术架构：**Go Core + Wails 3 桌面壳 + React 前端**。Core 不依赖 Wails 或 React；
桌面绑定与本地 HTTP Control API 通过各自的适配层接入业务能力。完整基线见[技术栈](docs/architecture/TECH_STACK.md)。

### 开发环境

- Go 1.26+，最低版本见 [`go.mod`](go.mod)。
- Node.js 22.x，见 [`.nvmrc`](.nvmrc) 和 [`frontend/package.json`](frontend/package.json)。
- pnpm，使用仓库提交的 `pnpm-lock.yaml`；不要混用其他包管理器生成锁文件。
- [Task](https://taskfile.dev) 与 Wails 3 CLI：

```bash
go install github.com/go-task/task/v3/cmd/task@latest
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.30
```

桌面开发同样需要 Windows 和 WebView2；生成 NSIS 安装包还需安装 NSIS。

### 初始化与启动

首次初始化会安装前端依赖、下载 Go 模块并安装静态检查工具：

```bash
task setup
task dev
```

`task dev` 提供 Go 重建与 Vite 热更新。开发环境默认代理端口为 `3001`，
与生产环境的数据目录和默认端口隔离。

如需覆盖运行 profile、日志级别或端口，可参考 [`.env.example`](.env.example) 创建本地 `.env`；
不要提交 `.env`、真实凭据或包含敏感内容的日志。

### 测试与打包

```bash
task test       # Go 测试 + Vitest 前端测试
task verify     # 格式检查 → lint → 类型检查 → 测试 → 构建
task build      # Windows production 程序 → bin/omniproxy.exe
task package    # 默认 Windows amd64 NSIS 安装包 → bin/OmniProxy-amd64-installer.exe
```

本地验证构建不等于公开 Release，也不会自动替换已安装的生产程序。
环境配置、运行 profile 和产物说明见[构建文档](docs/development/BUILD.md)，
正式发布流程见[发布指南](docs/development/RELEASE.md)。

## 项目文档

完整索引见 [`docs/README.md`](docs/README.md)。

| 文档                                      | 内容                         |
| ----------------------------------------- | ---------------------------- |
| [产品定义](docs/product/PRD.md)           | 项目目标、能力与非目标       |
| [架构概览](docs/architecture/OVERVIEW.md) | 组件与模块边界               |
| [账号故障切换](docs/product/FAILOVER.md)  | 换号范围、拒绝分类与消费边界 |
| [请求与用量](docs/product/ACTIVITY.md)    | 本机统计、留存与展示口径     |
| [网络代理](docs/product/NETWORK_PROXY.md) | 出站代理与提供商分流         |
| [安装与更新](docs/product/UPDATE.md)      | 检查、下载、校验与确认安装   |
| [开发规范](docs/development/)             | 编码、Git、测试、构建与发布  |
| [当前状态](docs/plans/CURRENT.md)         | 源码进展及待验收事项         |

AI 编程助手参与开发前，请先阅读 [`AGENTS.md`](AGENTS.md)；
每项开发工作须先创建 [Task Record](docs/tasks/README.md)，再实施并记录验证结果。

## 安全与隐私

- Windows 凭据通过 DPAPI 及现有系统凭据机制保护，数据库保存凭据引用，不保存明文 API Key。
- 日志经过统一脱敏；反馈问题时仍请自行检查并移除账号身份、接入 Key 和请求中的敏感内容。
- 使用过程中请遵守上游服务商的服务条款，不使用未经授权的账号或凭据。

详细政策见[隐私政策](PRIVACY.md)、[安全政策](SECURITY.md)与[代码签名政策](CODE_SIGNING_POLICY.md)。

## 社区致谢

特别感谢 **[LINUX DO 社区](https://linux.do/)**。

感谢社区在技术交流、知识分享与开源协作中营造的开放氛围，也感谢每一位愿意分享经验、
提出问题和参与讨论的社区成员。希望 OmniProxy 也能成为社区交流与实践中的一份小小回馈。

同时感谢原 OmniProxy 项目，以及 Go、Wails、React 和本项目所依赖的开源工具与库。
社区致谢不代表 LINUX DO 对本项目的官方背书。

## 许可证

本项目沿用原 OmniProxy 仓库的 **MIT License**，保留原有版权与许可声明，详见 [LICENSE](LICENSE)。
