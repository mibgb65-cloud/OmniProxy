# Product Requirements Document

## Overview

OmniProxy is a local-first desktop application that acts as an AI API gateway
and multi-account management tool. It runs on the user's machine, proxies AI
API requests through multiple provider accounts, and manages the local
configuration of AI coding clients.

The product exists because users of multiple AI providers and multiple
accounts need one local place to:

- unify access to several AI providers behind a single endpoint,
- distribute load across accounts of the same provider,
- survive quota exhaustion and account failures automatically,
- observe usage, quota and request history,
- configure local AI clients (Codex, Claude Code, Gemini CLI, …) safely.

## Goals

- Provide one local AI API gateway with model routing and account scheduling.
- Support multiple accounts per provider with automatic failover when a
  request fails or an account is rate-limited.
- Track quota, usage and request history per account and provider.
- Manage local client configurations (detect, read, back up, apply, restore,
  validate) without hand-editing config files.
- Run as a Windows desktop application with a system tray, single instance,
  autostart and automatic updates.
- Keep all data local: accounts, credentials, history and settings stay on the
  user's machine.

## Target Users

- Developers and power users with multiple AI provider subscriptions.
- Users of AI coding CLIs who need to switch providers/accounts frequently.
- Anyone who needs visibility into AI API usage and quota consumption.

## Core Use Cases

1. Point an AI client at the local OmniProxy gateway instead of a provider
   endpoint directly.
2. Add a provider account (API key or OAuth credential) once; OmniProxy
   schedules requests across eligible accounts.
3. Route a model name to the provider/account that should serve it.
4. Watch quota and usage; see which account served which request.
5. Detect and configure local AI clients (Codex, Claude Code, Gemini CLI,
   OpenCode) from the UI, with automatic backup and restore.
6. Control the gateway from the desktop UI or from external tools through the
   local Control API.

## Core Features

- 通用设置支持「启动软件时自动启动网关」，默认关闭，独立于 Windows 开机自启动；
  保存只影响下次软件启动，端口占用等启动失败不终止桌面应用，允许手动重试。
- v2 Beta 提供用户主动预览／确认的 v1 API Key 账号导入，保留旧文件及已有新版账号，明确订阅重新登录和不兼容范围；正式／Beta 检查支持持久选择和完整预发布排序。正式运行环境默认启动后及每六小时只检查版本，用户可关闭；侧栏升级标记打开版本浮窗，用户可下载／取消、校验并重启静默更新；自动下载默认关闭，显式开启后仅正式安装运行环境按已验证候选下载和校验，安装与重启仍需用户确认。详见 `docs/product/UPGRADE.md`、`UPDATE.md`。
- 侧栏底部“折叠”下方提供“设置与关于”统一入口，以带背景模糊的悬浮窗口展示左侧分类和右侧独立滚动内容；分类为通用设置、请求与日志、数据存储、检查更新、关于。通用设置紧凑展示启动／关闭行为与字号。检查更新独立为一级分类，按运行版本摘要、更新设置和离线当前版本日志分组，支持手动检查、可关闭的后台版本检查及默认关闭的自动下载开关；关于页展示官网、开源仓库、免责声明、使用协议、隐私政策五项入口，通过后端固定标识白名单在系统浏览器打开，等待时禁止重复操作，失败显示可重试的安全提示，不在应用内加载协议正文。官网三份独立静态协议已上线并核验标题与正文，不将 HTTP 200 首页回退当作成功。内容视口延伸至底部边框，底部留白随内容滚动，避免卡片提前截断。打开时保留当前工作页面，切换分类保留未保存输入，支持上下键选择分类、键盘关闭和焦点返回。旧 `/settings`、`/about` 链接继续打开对应分类。
- Multi-provider access via a unified adapter architecture (OpenAI,
  Anthropic, Google, DeepSeek, Zhipu AI, Moonshot AI/Kimi, MiniMax, Xiaomi MiMo,
  StepFun, Atria, River, Oxion AI, and OpenAI-compatible endpoints).
- Twelve named providers are enabled by default and can be switched off; users
  can add enabled-by-default custom OpenAI-compatible endpoints.
- 提供商卡片使用与账号一致的电源图标按钮控制启停：绿色表示启用，灰色表示停用，悬浮提示说明下一步操作。
- Provider cards show a curated model preview and an unauthenticated endpoint
  reachability/response-time check; actual model access depends on an account.
- 点击模型预览后在浮层中查看完整预设列表，卡片和同一行相邻卡片高度保持稳定；浮层支持键盘打开、Esc／外部点击关闭和窗口边缘避让。
- Multi-account management with OS-protected local credential storage; account
  metadata keeps only opaque references to secrets.
- API Key 账号支持单个添加和批量添加。单个账号名称选填，留空时按 Key 的前最多 8 个 Unicode 字符命名；批量每行一个 Key、每次 1–100 个，只使用自动名称。
- Google 增加 Antigravity 独立订阅登录，提供浏览器授权、身份校验、刷新／退出、真实账号 Gemini／已适配 Claude／GPT-OSS 模型发现及文本／工具三协议 JSON／SSE；账号提供“同步到 AI Agent”，凭证走 DPAPI，不混用普通 API Key。账号详情／配额总览支持官方模型组额度、剩余百分比及重置时间；普通 API Key 项目资源内嵌对应账号卡片／列表。公开额度快照在 SQLite 按身份／授权版本保存，换页和重启先恢复再刷新，读取失败保留上次显示，不作实时调度依据；模型目录仍独立缓存。实际订阅权限及验证边界见 `docs/product/GOOGLE_SUBSCRIPTION.md`。
- OpenAI 支持通过本机官方 Codex 组件登录 ChatGPT 订阅账号，提供浏览器／设备码授权、多账号隔离、身份与套餐、刷新、退出和重新登录。系统凭证库与官方加密命名空间保存授权，已支持订阅 Responses 转发；详见 `docs/product/SUBSCRIPTION.md`、`docs/product/LOCAL_AGENTS.md`。
- OpenAI 账号设置提供应用级 Codex WebSocket 开关，默认开启、持久保存，控制当前网关的握手和连接内后续请求，不改写 Codex CLI／桌面端的提供商身份、认证或会话，HTTP/SSE 保持可用。
- OpenAI 账号设置在入口旁局部展开，自动激活 Codex 额度窗口默认关闭，切换开关直接持久保存并生效；对未开始的 5h／周窗口发送原账号最小消息并复查，持久去重防止重复消耗，已激活窗口不发送。
- 自动名称有意展示 Key 前缀，短于 8 个字符时使用实际长度；名称不写入日志。删除账号必须经过应用内、随主题切换的确认弹窗。
- 提供商账号详情支持卡片／列表切换并保存展示偏好；展示真实密钥的脱敏片段，用户可主动复制密钥。新的密钥读取接口只返回脱敏文本，复制在桌面端直接写入系统剪贴板。
- 账号支持编辑名称与替换 API Key；编辑弹窗不回填当前完整密钥，Key 留空时保留原凭证，名称清空继续自动命名。
- OpenAI 订阅支持 5h／一周任一窗口耗尽后自动跳过，使用同池其他启用账号；依据已校验额度快照和执行前明确额度拒绝，不跨服务商或计费通道，不重放已开始生成的请求，固定账号仍保留。
- OpenAI 订阅成功获取的额度和绝对重置时间按账号身份持久留存，换页／重启后先恢复，后续查询成功更新，失败或暂缺字段不清空旧显示；留存值标为待刷新，不作为调度或自动激活的新鲜依据。
- Account scheduling: eligibility, health, priority, load, quota, cooldown,
  concurrency and retry.
- Model routing rules.
- 模型路由首版支持精确别名映射、具体提供商配置与可选固定账号；规则持久化并动态生效，提供不读取密钥的匹配预览，模型发现接口同步可解析别名。
- 路由页依据已有模型目录自动展示同名初步映射，无需逐条新增规则；同名模型可保存多个服务商的优先级顺序。精确规则优先，默认映射按顺序选择可用服务商，跳过不可用配置，其余候选沿用自动选择；预览／网关／客户端目录保持一致。本次覆盖现有 DeepSeek 和 ChatGPT/Codex 适配器。
- 网关支持 DeepSeek 与 ChatGPT / Codex 的文本／工具调用，可通过 Chat Completions、Responses、Messages 和 Codex WebSocket 接入；跨协议先收完整结果再输出 SSE。已支持默认开启、最多三次的同服务商／认证／计费／地区账号切换，保留固定账号、200 后不重放及一次请求统计；OpenAI 订阅另支持默认队列／平衡轮换，平衡只重排同池请求，不增加生成后重试。具体拒绝类型和 Google 配额限制见 `docs/product/FAILOVER.md`。更广泛的健康／负载调度仍待实现。
- River 支持 API Key、四款官方预设与账号模型同步，原生 Chat／Responses JSON 和 SSE、思考与函数工具调用，Messages 通过既有文本／工具转换接入。NVIDIA／GLM 使用短名称，原版 Flash 与直连 DeepSeek 共用服务商优先级，1017 版本独立显示及路由；DeepSeek Flash／Pro 保留普通和 1M 客户端选项。账号页提供持久化的队列／均衡使用设置，同池持续失败 Key 暂时冷却并供后续请求切换，不重放结果不确定的当前请求。目录、路由、客户端配置与用量统计共用；已按官方契约接入文本与图片输入（含 Claude 工具结果中的截图），不承诺音频、文件、余额或 1M 真实长输入验收。详见 `docs/product/RIVER.md`。
- Atria 支持 API Key 账号及 `Atria-Dawn-Preview` 官方预设，256000 上下文、仅文本；Chat Completions／Responses／Messages 使用原生 HTTP 和 SSE，复用路由、客户端目录与用量统计。预设不代表账号认证或额度已经验证，余额及实时模型发现尚未接入。
- Oxion AI 支持 API Key 账号（`cc_` 前缀）与官方 `GET /v1/models` 目录同步：只收录官方 `type=chat` 的模型，上下文窗口与 `vision`／`tools` 能力按目录声明，未同步前没有可用模型。上游只有 OpenAI 兼容 Chat Completions，Chat 请求原生透传，Responses／Messages 由网关转换后转发并按原协议回包；官方未声明 `reasoning_effort`，因此不发布档位选择。Embeddings、`web_search` 与余额未接入。详见 `docs/product/OXIONAI.md`。
- 小米 MiMo 支持普通按量及 Token Plan 个人／团队 Key、套餐中国／新加坡／欧洲集群，按账号选择计费通道和区域。账号设置可保存 Token Plan／API Key 默认顺序，同名模型统一发布；无合格首选账号时选下一通道，固定账号／旧显式通道继续优先，请求失败不跨通道重试。普通三个 V2.6 对话模型、套餐 Pro／Flash，官方模型同步、路由和客户端目录共用；原生三协议保留文本、思考、函数调用及 JSON／SSE，并记录当次区域／计费归属。预设与账号发现区分，套餐官方剩余未知并提供平台入口；余额、多模态及语音接口未接入，真实 Key／长窗口待验证。详见 `docs/product/MIMO.md`。
- 阶跃星辰支持账号级国内／海外和按量 API／Step Plan 四组合，账号设置可保存 Step Plan／普通 API Key 默认顺序，同名模型统一发布，固定账号和旧显式通道继续优先；原生文本协议按模型开放，请求失败不跨通道重试。支持普通模型同步、官方普通余额和本机用量，Step Plan 官方剩余保持未知。详见 `docs/product/STEPFUN.md`；真实 Key 与长窗口验收待完成。
- Kimi 支持普通 API／Kimi Code 订阅 Key 与国内／海外账号、默认计费及地区顺序，同款模型统一展示，固定账号／显式通道及旧 ID 保持兼容。订阅原生三协议、型号映射与真实客户端身份保留；普通 K3 三协议、K2.7 Code／K2.6 Chat／Messages、高速版 Chat 能力保持。普通 API 官方可用／代金券／现金余额按地区查询，国内 CNY、海外 USD；订阅剩余额度未知并提供控制台入口，不推断套餐或 1M 权限，不跨通道失败重试。真实 Key／长窗口仍待验证。详见 `docs/product/KIMI.md`。
- MiniMax 支持账号级国内／海外 × 按量／订阅 Key，五款普通模型与订阅同步发现的 M3.1 预览、原生文本三协议／思考／函数工具／SSE、目录及通道路由。提供官方订阅窗口查询与本机用量；旧账号需确认计费类型，普通余额及高级 Agent 工具未接入，真实 Key／长窗口待验证。详见 `docs/product/MINIMAX.md`。
- 网关页支持 Codex CLI／Desktop、Claude Code CLI／Desktop、OpenCode CLI／Desktop、DeepSeek Harness Web／Desktop，以及 Kilo／Qwen／MiMo Code 本机引擎配置和 CLI／Desktop 入口；支持独立接入 Key、预览、加密备份和恢复。保留原有设置与官方会话，桌面宿主覆盖范围见产品说明。路由页可同步 ChatGPT／DeepSeek 官方模型目录并按账号选择；详见 `docs/product/LOCAL_AGENTS.md`。
- 一键接入默认配置全部适用模型／路由，也可搜索、逐项选择并指定所选范围内的默认模型，禁止空选择；优先保留仍可用的原默认。目录记录官方已确认元数据并保留未知能力，Codex 按身份／核心版本适配本地或远程目录，部分选择使用本地目录；Claude Desktop 提供所选显式列表。受管目录刷新保留选择，支持安全更新和恢复。
- Agent 详情页采用安装／接入管理与模型两栏，右侧展示实际受管模型目录并支持搜索／刷新；配置按钮将右侧切换为内联模型设置，保存或取消后返回列表。窄容器上下排列，不以可配置模型替代已接入状态。
- Quota monitoring and usage statistics.
- DeepSeek、阶跃星辰与 Kimi 普通 API Key 账号支持官方余额查询，按各自协议展示币种、金额及分项，提供手动刷新；详见 `docs/product/BALANCE.md`。
- Codex 订阅账号详情可查询真实额度窗口、剩余百分比及重置时间；Plus／Team 普通套餐展示 5h 与周额度，其他套餐仅周额度。窗口下以当前窗口保留的本机 Token 成本／已用比例粗估美元总额与剩余，至少已用 5%，按实际模型基础参考价及缓存计费；周估算覆盖完整一周窗口，不受 5h 重置或历史周基线裁剪影响。美元估算以紧凑表格只显示总额／剩余单值，无额外提示；缓存未知时不计折扣，未知价格不猜，零星缺用量按已确认样本估算、超过两成缺用量才显示占位。下方三列展示输入／输出／缓存 Token，移除“本机留存”及缓存／完整性说明，保留未知值和部分缓存下界，缓存不重复计入总量。估算不是官方美元余额，Token 只覆盖本机留存、不代表其他客户端消费，详见 `docs/product/SUBSCRIPTION.md`。配额页集中展示跨账号订阅额度及 DeepSeek／阶跃星辰／Kimi 余额、异常概况、搜索／筛选和受控刷新，与账号详情共享缓存；额度历史和推理权限验证仍待后续实现。详见 `docs/product/QUOTA.md`。
- Request history with success/failure classification.
- 网关运行概览展示真实请求指标、成功／失败趋势、可筛选最近记录、累计上游分布和接入信息。新增首字耗时按下游有效流式内容写出计时，区分旧数据／非流式未知与真实 0，均值显示成功流式样本数；共用本地记账与独立聚合，不调用模型。详见 `docs/product/GATEWAY.md`。
- 仪表盘提供本地今日／7 天／30 天运行总览、请求／Token／成功率／完整耗时、结果趋势、待处理问题、最近请求及资源／客户端配置概况；资源默认读缓存，官方查询由用户主动触发。独立时间桶保留明细清理后的聚合，历史缺口明确标示。详见 `docs/product/DASHBOARD.md`。
- 已实现持久请求明细、输入／输出 Token 累计及每日／成员／模型汇总；删除账号与明细淘汰不减少累计。请求默认保存 10000、设置可调整，日志缓存最多 5000，request_id 双向关联。Team 同时按邮箱和 account_id 识别。详见 `docs/product/ACTIVITY.md`。
- Provider health detection.
- Client integration management for external AI tools.
- Kimi Code、Pi coding agent、Factory Droid CLI 与 Goose CLI／Desktop 支持本地网关模型接入、默认模型、独立 Key、备份恢复和官方软件入口；Pi／Goose 的关联文件共同校验。详见 `docs/product/KIMI_PI_DROID_GOOSE.md`。
- Local HTTP Control API for CLI/agent/script/MCP access.
- Windows desktop UI, system tray, single instance, autostart, auto-update.

## Non-Goals

Explicitly out of scope for the current stage:

- Cloud sync, multi-user accounts, or any server-side component.
- Mobile applications (iOS/Android build assets exist in the Wails template but
  are not product targets).
- A billing or payment system.
- Prompt management, conversation storage, or any LLM content itself.
  OmniProxy moves requests; it does not store their content beyond metadata.
- Support for provider-specific features beyond the unified adapter interface.
- macOS/Linux desktop support (planned after the Windows product is complete;
  the architecture keeps this possible).
- Replacing or wrapping the provider SDKs beyond request/response translation.

## Constraints

- Local-first: the gateway binds to loopback; no data leaves the machine except
  to the AI providers the user configured.
- 密钥字段保存在受保护凭证存储中，数据库使用不透明引用；用户要求的自动账号名称允许保存前最多 8 个字符。日志不得包含密钥或这些名称。
- Development and production environments must not share data directories.
- The Go core must remain independent of Wails and React so it stays testable
  and reusable (e.g. behind a future CLI).

## Success Criteria

- A configured client can complete an AI API request through the local gateway.
- When one account fails or is exhausted, requests are retried on another
  eligible account without user intervention.
- The user can see per-account quota, usage and recent request outcomes.
- Client configurations can be applied and restored from the UI without manual
  file editing.
- The app starts on boot, runs in the tray, and updates itself.

TBD: concrete performance targets, exact quota-refresh semantics per provider,
and the first released version number.
