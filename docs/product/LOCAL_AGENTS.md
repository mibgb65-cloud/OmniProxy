# 本地 Agent 接入与模型同步

Google / Antigravity 订阅账号支持将实际发现的 Gemini、Claude Opus／Sonnet 4.6 及
GPT-OSS 文本型号同步到 Agent 目录。启动应用、Google 授权状态变化或成功刷新账号
模型列表后会自动同步，无需逐账号点击按钮；账号卡片“重新同步到 AI Agent”及路由同步
仍可手动重试。后台串行处理，失败保留上次成功快照，不启动网关或发送推理请求。
只更新已经确认接入且未被外部改动的客户端；用户明确限定的模型范围不会自动扩张。
首次使用仍需在“网关 → 客户端接入”预览并确认一次配置。
Responses／Messages 通过既有文本和工具转换接入，支持 JSON／有界 SSE；运行中的
客户端需重新加载模型菜单。普通 Gemini API Key 仍仅支持 Chat，不进入不兼容的
Agent 协议目录。详情见 [Google 订阅](GOOGLE_SUBSCRIPTION.md)。

## 使用

Kimi Code、Pi、Factory Droid 和 Goose 已加入产品分组，复用独立 Key、模型选择、默认模型、加密备份与恢复；Goose 的 CLI／Desktop 共用配置、分别检测安装。Pi、Goose 的关联文件一起校验和备份，Kimi 区分当前版与旧 Python 版目录及协议。配置位置、官方依据与版本边界见 [Kimi／Pi／Droid／Goose](KIMI_PI_DROID_GOOSE.md)。

Kilo Code、Qwen Code、MiMo Code 各用一张 CLI／Desktop 卡片，提供本机引擎配置、模型
同步、独立 Key、备份恢复、检测和打开入口；MiMo 在同一卡片分别列出国内／国际桌面版。
Qwen 官方桌面应用使用普通 `qwen serve`
和用户设置；Kilo／MiMo 的桌面宿主可能覆盖引擎目录或内联设置，需以预览路径与实际
宿主为准，不能将全局配置成功当作所有桌面版本均已生效。见 [KILO_QWEN_MIMO_CODE.md](KILO_QWEN_MIMO_CODE.md)。

OpenCode CLI／Desktop 共用一张接入卡片；DeepSeek Harness 的 Desktop／Web 在同一卡片中
分别配置 profile。均支持本地 Chat 文本／函数工具目录、独立 Key、加密备份／恢复及目录
刷新，安装检测和固定官方入口共用原流程。详情见 [HARNESS_OPENCODE.md](HARNESS_OPENCODE.md)。

River API Key 可用于本地 Codex Responses 与 Claude Messages 文本／工具／图片接入；Messages 转换需要缓冲，不是原生增量流。图片（含截图工具结果）按官方 `image_url` 契约转发，目录标记 `text` + `image`。预设与发现模型均进入同一路由目录，限制见 [RIVER.md](RIVER.md)。

1. 在账号页登录 ChatGPT / Codex，或添加 DeepSeek／River／Atria／MiMo 普通 API Key。MiMo 可使用官方预设或在路由页同步账号模型，兼容边界见 [MIMO.md](MIMO.md)。
2. 在路由页选择账号，点击「同步官方模型」。目录按账号保存；失败保留原目录，更换凭据／工作区会使旧目录失效。改名、额度刷新不会清除目录。
3. 可以直接使用已同步模型，也可以新建精确路由并固定账号。
4. 网关页「客户端接入」全部使用统一的产品分组卡片：品牌与产品标题在上方，内部逐项展示 CLI／Desktop／Web 入口。每项单独显示对应软件的安装状态和实际配置的接入状态，点击进入配置页面，无需先启动网关。
5. 详情页宽屏采用左右两栏：左侧管理软件安装、配置路径、文件状态、接入 Key 复制与启停、原配置恢复；右侧默认展示实际已写入当前客户端的模型、默认模型标记、搜索与刷新。窄容器改为上下排列。点击「一键配置」后右侧切换为配置接入模型，自动加载预览并默认勾选全部适用模型；可以搜索名称／ID、逐项勾选、全选或清空，并从所选模型中指定默认模型。未调整时保持完整目录；全部取消会提示至少选择一个并禁用应用。确认数量与目录方式后应用，成功即返回已接入列表；取消不写入文件并返回原列表。重新启动对应客户端后，在客户端中选择模型；实际调用前需启动网关。

已接入列表从版本匹配的受管配置／本地目录读取，只返回公开 ID、名称和默认模型；未接管时显示空状态，外部修改或读取失败时展示错误，不把网关可用目录冒充为已接入列表。Codex 受管远程发现模式按当前远程目录展示，其他模式读取实际菜单；Key 停用时保留列表并显示停用提示。该列表不表示客户端已经重新加载或请求已验证成功。

已接入模型采用紧凑卡片，按容器宽度和字号自动分配每行数量，筛选后少量卡片保持原列宽。搜索旁的「仅看 [1m]」按钮按真实模型 ID 的精确后缀筛选，可与搜索叠加；再次点击恢复全部范围，并显示当前匹配数／总数。不修改接入模型、默认模型或配置文件。接入配置区不再展示网关未启动的通用提示，网关状态仍显示在公共控制区。

配置时复用同款卡片与网格，每张卡片增加复选框、选中样式和上下文信息，点击整张卡片或聚焦复选框后按空格均可切换。配置搜索也支持「仅看 [1m]」；筛选不改变隐藏模型的勾选和默认模型，全选／清空仍作用于完整目录，实际应用仍以全部已选模型为准。

卡片网格按内容区实际宽度自适应，不再固定两列。最小卡宽为 22rem，窄容器按满宽收缩；
大屏和侧栏收起时自动增加列数，余宽分配给现有列，末行保持与上方列对齐，字号缩放时
同步调整排列。

Codex、OpenCode、Kilo、Qwen 的 CLI／Desktop 入口进入同一详情；MiMo CLI／国内 Desktop／国际 Desktop 也进入同一详情，共用各产品的接入状态。Claude 的 CLI／Desktop、Harness 的 Desktop／Web 分别进入其独立配置。安装状态始终按软件区分，不因共享 Key 就宣称对应软件已安装。

配置页面使用 `/gateway?tab=clients&client=客户端ID` 定位，支持 `codex`、`claude`、`claude-desktop`、`opencode`、`harness-web`、`harness-desktop`、`kilocode`、`qwen-code`、`mimo-code`、`kimi-code`、`pi`、`factory-droid`、`goose`。支持刷新与前进／后退；非法参数回到卡片目录。返回目录恢复原软件入口焦点（如从 Desktop 返回仍选中该 Desktop），选项卡往返保留正在进行的配置操作。点击入口只导航，不写入客户端配置或发起模型请求。

配置预览、保存与网关运行状态独立。没有适用的模型／路由时，应先同步账号模型或配置可用路由。目录在预览后变化时，需要重新预览；首次接入使用稳定默认，仍可用的原模型优先保留。

公共启停按钮与侧栏电源按钮共享操作互斥和反馈。快速成功时启停反馈至少保留约 650ms，慢操作不额外叠加完整等待；真实请求和结果缓存立即执行，失败立即提示。固定尺寸图标采用柔和淡入淡出和较慢旋转，支持减少动态效果。后台状态同步不会带动「刷新接入检查」按钮变化，只有手动刷新才显示其加载反馈，首次读取期间保留禁用。

选择部分模型后，目录刷新只更新所选且仍可用的模型，不会自动加入取消的模型或新增模型；所有所选模型失效时保留原文件并报告无可用模型。选择保存在受管备份记录（迁移 031），重启应用后仍生效，恢复原配置时一起清除。全选应用恢复动态完整目录。Codex 部分模型使用本地目录并关闭远程发现，避免远程完整目录覆盖选择；不改变网关路由或接入 Key 的模型调用权限。

Harness 保持显示建议模型，不提供默认模型写入；保存后仍需在 Harness 中选择模型。

右侧配置面板正文和模型目录统一使用细圆角滚动条，保留固定轨道占位，默认隐藏，悬停轨道或拖动时淡入、移开后淡出；滚轮和键盘滚动可用，减少动态效果模式下取消过渡。模型目录高度按视口限制，标题和操作按钮位于正文滚动层之外。保存／取消后恢复配置按钮焦点；编辑时左侧配置操作暂时禁用，避免并发覆盖。

应用成功后，右侧返回已接入模型列表，顶部悬浮显示“配置已保存，原配置已加密备份。”，无需滚动到内容底部；约 5 秒后自动收起，可手动关闭，鼠标悬停或键盘聚焦时暂停计时。提示适配深浅主题与减少动态效果，通过温和的状态播报告知结果；收起通知不会改变已保存状态。

## Codex Fast 服务档位

OpenAI 订阅模型同步保留官方 `model/list` 公布的 `serviceTiers`，按账号写入现有
模型快照，再导出为 Codex 目录的 `service_tiers`。官方确认支持 Fast 的 GPT 模型
及其路由别名保留相应入口；档位名称与额度说明沿用官方信息，不根据模型名称推断。
固定账号别名只继承该账号的能力，不借用其他账号的 Fast 声明。

旧模型快照缺少此字段时仍可使用，但不会凭空声明 Fast。升级后在路由页对 OpenAI
账号执行「同步官方模型」，成功会刷新已接入且未被外部修改的受管客户端目录；
运行中的 Codex 需要重新加载模型菜单，必要时重新打开客户端。
同步失败保留上次成功快照，不读取或覆盖用户默认官方缓存。

此修复只恢复能力声明，不自动启用 Fast，不修改现有 `service_tier` 选择，也不为
第三方兼容模型声明 OpenAI 服务档位。实际可用性、速度和额度消耗由官方账号及上游决定；
原生 Responses 转发保留用户选择的服务档位。

## Codex WebSocket 接入开关

账号 → OpenAI →「OpenAI 账号设置」提供应用级「Codex WebSocket」开关，默认开启。只控制当前 OmniProxy 网关是否允许 Responses WebSocket，所有 OpenAI 账号共用；设置保存在本机 SQLite（迁移 016），无需启动或重启网关即可保存。关闭后新握手返回 `403 websocket_disabled`；已连接会话在下一次创建响应时被拒绝，进行中的请求可以正常结束。普通 HTTP/SSE 保持可用，客户端是否自动回退取决于自身版本。

该开关不修改 Codex CLI／Desktop 的配置、提供商身份、登录、会话或环境变量。Codex 0.155.1 已将旧 responses_websockets／v2 标记为 removed，内置 openai 禁止覆盖；不能将旧标志的 false 当作真实关闭客户端传输。用户选择控制网关以保持 ADR-005 的身份保护。参考：[官方配置参考](https://learn.chatgpt.com/docs/config-file/config-reference)、[开发者设置](https://learn.chatgpt.com/docs/developer-settings)。

只影响当前运行的新版网关；旧应用进程和其他端口不联动。客户端若仍指向旧版 `/codex/v1` 地址，应使用新版网关页的一键接入，不能宣称只打开本开关已修复其他进程的 403。

## 客户端安装

卡片显示真实的软件安装状态，与网关接入 Key 状态分别展示。详情页的「软件安装」支持只读检测、重新检测、安装和官方说明。Codex CLI 与桌面应用分别检测；Claude Code CLI 和 Claude Desktop 也各自检测。

已安装的 Claude Code Desktop 提供「检查更新」，点击后重新核对本机版本，并读取[官方 Windows 发布入口](https://support.claude.com/en/articles/12622703-deploy-claude-desktop-for-windows)的最新 MSIX 版本。结果区分发现新版本、暂无新版本、无法确认与失败；检查不下载安装包，不改写配置，打开客户端仍可使用。该结果针对官方公开发布版本，商店或组织分发可能有不同的发布时间；本次不扩展其他客户端的更新渠道。版本格式或发布地址变化时不判断最新，可查看官方说明。

发现新版后，同一按钮切换为「更新」；再次点击才通过固定 Microsoft Store 包启动 WinGet 后台升级。五分钟内复用服务端已验证的发布结果，但始终复核本机安装；过期、版本变化或显式检查时重新查询，减少重复联网失败。界面显示「更新中…」和进度，防止重复提交，并与安装任务互斥；完成后核对真实版本达到官方检查目标，再显示更新成功并恢复检查按钮。更新失败或被中断时保留具体原因供重试／重新检测，不把仍安装着的旧版本当作成功。

WinGet 返回无适用更新／升级版本不更新时，显示「商店更新暂不可用」，同一按钮改为「官方下载」，仅打开固定 `https://claude.com/download`；用户完成官方安装后重新检测，打开页面本身不算更新成功。未找到 WinGet 时普通更新按钮禁用，仍可查看官方说明。既有安装操作继续使用 `--no-upgrade`；更新不接受任意 URL／包名，不强制关闭客户端、卸载旧版或绕过完整性校验。

Windows 使用固定 WinGet 包与安装源：

| 软件 | 软件包 ID | 安装源 |
| --- | --- | --- |
| Codex CLI | `OpenAI.Codex` | `winget` |
| Codex Desktop | `9PLM9XGG6VKS` | `msstore` |
| Claude Code CLI | `Anthropic.ClaudeCode` | `winget` |
| Claude Code Desktop（Claude 应用的 Code 入口） | `9P6K58THS811` | `msstore` |

只有点击「安装」才启动任务；点击安装表示接受软件及安装源协议，必要的系统授权由 Windows 处理。安装一次只执行一个任务，保留既有版本、不自动升级，使用 WinGet 默认完整性校验。界面显示准备／下载／安装／核对阶段、进度条及已用时间。WinGet 报告数值时显示阶段百分比和当前下载量；未报告时显示动态进度，不推算全流程百分比。下载 100% 不等于安装完成，最终结果仍以实际检测为准。失败后可以重新检测或重试；检测异常不会被当作未安装。没有 WinGet 时提供官方说明入口。

CLI 检测 PATH 和固定用户安装目录；桌面应用检测卸载注册表和当前用户 MSIX 包。Claude Desktop 兼容商店新版 `AnthropicPBC.Claude` 和旧版 `Claude` 的精确包名，新旧包共存时优先识别健康版本。只检测安装元数据，不读取登录、凭据和会话。安装器输出不进入 UI 或日志。退出应用中断任务后不会宣称安装已回滚，下一次启动重新检测结果。Windows 原生安装与 WSL 内的软件分别管理，本版不安装 WSL 客户端。

进度通过后台 Windows ConPTY 读取，兼容 WinGet 的终端进度指令和有界文本帧；只向界面传递阶段、百分比和字节数。无法建立终端且尚未启动安装进程时才降级为普通输出读取，避免重复安装。未提供进度的安装器仍显示阶段和时间。

安装和接入配置相互独立，仍允许先准备配置。安装完成后自动刷新状态；Codex 核心版本同步到模型目录兼容逻辑。

Claude Code Desktop 是官方 Claude 桌面应用中的 Code 功能，与 Chat 共用安装包。该卡片的「打开 Claude Code」通过固定的 `claude://code/new` 进入编程页面，不附带任务、项目路径或凭据；官方说明链接指向 Code 桌面文档。一键配置会启用第三方网关模式并选中「OmniProxy · 本地网关」，完全退出并重启 Claude 后进入 Code。原有官方设置、配置条目和会话文件保留，恢复配置可还原原模式与选择。尚未激活本地网关时，打开按钮提示先应用一键配置，避免进入官方登录页。[官方桌面说明](https://code.claude.com/docs/en/desktop)、[第三方配置说明](https://claude.com/docs/third-party/claude-desktop/in-app-configuration)、[Code 入口链接](https://support.claude.com/en/articles/14729294-open-claude-desktop-with-a-link)。

Windows 的第三方配置入口为 `%LOCALAPPDATA%\Claude-3p\configLibrary`，启动模式写入共享的 `Claude` 与 `Claude-3p` 目录。Claude 商店安装包明确将 `Claude-3p` 排除出文件系统虚拟化，因此 `Packages\...\LocalCache\Local\Claude-3p`（可能映射到 `D:\WpSystem`）中的残留配置不能作为路径选择依据。开发版也使用同一客户端入口，无需先安装 OmniProxy；默认网关端口为 3001。若旧版本曾写入错误位置，重启修复后的 OmniProxy，使用「重新读取配置」重新预览并应用，原加密备份按现有归档流程保留。

若配置提示成功但 Claude 重开后仍使用旧地址，检查 OmniProxy 是否从工具的虚拟文件环境启动。需从正常桌面终端启动开发版后重新读取和应用；相同路径字符串并不保证两进程访问同一实体。重新读取建立新主配置备份时，启动模式和选中配置也同步采用当前恢复基准。详见 `docs/development/BUILD.md` 的原生验收步骤。

配置错误分别提示读取、写入、文件占用与备份失败，不将文件写入问题误报为网关异常。Windows 加密配置目录拒绝同目录重命名时，使用独占文件句柄再次校验版本并写入，保留文件加密与 ACL；失败恢复原字节，恢复也失败时明确报错。此兼容分支提供进程内失败回退，无法保证断电或强制结束进程时的原子性，应用前仍保存加密备份。

安装渠道依据：[Codex CLI 官方说明](https://learn.chatgpt.com/docs/codex/cli)、[Codex Windows 官方说明](https://learn.chatgpt.com/docs/windows/windows-app)、[Claude Code 官方说明](https://code.claude.com/docs/en/setup)、[Claude Desktop 官方下载](https://claude.com/download)、[WinGet 安装命令](https://learn.microsoft.com/en-us/windows/package-manager/winget/install)。

网关允许 CLI 和 Electron 不携带 Origin 的原生请求，包括 `Sec-Fetch-Site: none`；仍校验 loopback Host、端口及独立接入 Key，拒绝浏览器跨来源、跨站和同站请求。安装客户端并不放宽接入认证。

## 模型目录

Claude Code Desktop 的配置使用稳定兼容路由名，标签显示实际模型／用户路由名称。Messages 请求先用当前可用目录解析兼容名，再进入现有精确路由和账号选择；目录或路由移除后，旧配置不能继续调用。兼容名不写入用户路由表，其他客户端目录继续使用原模型 ID。

Claude CLI / Desktop 为 DeepSeek Flash／Pro 同时保留普通和 `[1m]` 两种选项，例如 `deepseek-flash`／`deepseek-flash[1m]`、`deepseek-v4-pro`／`deepseek-v4-pro[1m]`；已有普通默认保持普通，已有 1M 默认保持 1M。River Flash 归入同一逻辑模型并共用服务商优先级；NVIDIA、GLM 使用短名称，旧完整 ID／Desktop 摘要和上下文选项保持兼容。未知上下文不生成变体，已实测 GPT 订阅的仅 1M 规则保持独立。这里的 `[1m]` 是客户端选项，不作为上游模型 ID 发送。

Desktop 使用独立的长上下文基础兼容 ID，以及原生 `supports1m=true` / `prefer1m=true`，避免配置归一化合并选项或选中标签后仍按 200K 调用。客户端去掉后缀后网关仍可识别；上一版普通和共享基础 ID 的请求保留解析兼容，不改上游真实模型。应用与目录刷新复用同一格式。实际桌面菜单和上下文显示必须验收，写入条目数量或 CLI 参数验证不能替代原生结果。

Windows 商店版已有 `LocalCache/Local/Claude-3p/configLibrary` 时优先配置该真实库，避免 MSIX 虚拟副本遮蔽共享目录。系统把包数据重定向到其他盘 `WpSystem` 时，通过规范路径定位同一配置文件，继续保留版本、加密与链接保护；只有 Roaming 官方设置或空包缓存时仍使用原共享入口。两种商店包家族及未打包版本分别验证。

现有原生客户端可能仍缓存旧配置。本次文件更新和完全重启后仍出现旧 8 项，最终经原生编辑器更新模型字段、保存重启才确认 9 项生效。自动刷新这份原生缓存尚未验证，不能仅凭一键配置文件写入成功宣称菜单已更新。

GPT 订阅的长窗口必须经当前账号的真实请求确认，不能从官方 API 的总窗口或其他账号缓存推断。确认只绑定当前授权版本，普通目录同步保留该能力；更换凭据／工作区后失效。已确认模型在 Claude 中仅提供 `[1m]`，Desktop 一键配置使用同一原生字段，应用和刷新保留仍可用的原默认模型。本机旧 Codex 核心可能尚未列出 GPT-6.1 Sol；该模型完成真实验证后可补入当前账号目录，后续同步保留，其余消失模型仍正常移除。

本次六款 GPT（6.1 Sol、6 Astra / Sol / Luna、5.6 Sol / Luna）均实际完成 900019 个输入 token；百万输入返回长度错误，不能把 1M 显示解释为允许整整百万输入。Claude CLI 一键配置为已确认 GPT 的 `modelSettings.<模型ID>.autoCompactWindow` 设置 850000，为输出和工具文本留余量，保留已有更小阈值及其他模型设置，不覆盖用户全局变量，也不限制 DeepSeek。读取同一用户设置的本地 Desktop 会话使用该阈值；客户端可能把 850K 压缩边界显示在用量环中，模型本身仍为 1M 选项。Terra 本次返回上游过载／429，尚未确认。真实验证会消耗订阅额度，不能把 HTTP 200 但流内失败当作通过。[Claude 自动压缩说明](https://code.claude.com/docs/en/model-config)。

Messages 正文上限为 16 MiB，以容纳长文本与 JSON 编码；继续在读取前鉴权并占用有界并发名额，其余协议仍使用 2 MiB 上限。

模型目录按账号和授权版本隔离，保存官方已确认的描述、推理档位和可选能力。路由别名继承实际目标账号的元数据；未知上下文／工具能力保持待确认，不借用其他模型的数值。客户端目录排除已确认不支持所需协议、文本或工具的模型。

### 推理强度（Claude）

Claude Code CLI／Desktop 的模型菜单每个模型只占一行（普通 + 可选 `[1m]`），推理强度由客户端原生 effort 控件选择（`/effort`、`/model` 的强度滑杆），不再按档位派生出「`模型 · 高强度推理`」这类伪模型条目。

- `modelPicker.options` 对已确认档位枚举且公布默认档位的 River 模型附加 `behavesAs`，让客户端按已知模型解析上下文窗口、身份与默认档位；菜单标签和请求中的模型 ID 都不变，仍显示真实模型名，不改变计费归属。
- 同时写入 `modelSettings.<模型ID>.effortLevel` 与 `maxEffortLevel`。**两个键只接受 `low`／`medium`／`high`／`xhigh`，`max` 不是合法取值**：上游默认档位为 `max` 时写 `xhigh`，模型只支持 `max` 时不写上限。写成 `max` 会被客户端忽略，并把实际发送值压回 `high`（本机 Claude Code 2.1.292 实测：`low`／`medium`／`xhigh` 原样送出，`max` 退化为 `high`）。
- 网关按模型真实枚举再次校验 effort：非法档位返回 `unsupported_reasoning_effort` 400，不静默丢弃、不降级、不猜测；客户端未显式选择时使用上游公布的默认档位。
- 旧版本写入的 `omniproxy-effort-*` 选择仍可解码回「基础模型 + 档位」，已保存的旧会话不会变成 `unsupported_model`；这些 ID 不再出现在菜单中。
- 强度控件是否出现由客户端按模型识别结果决定；`behavesAs` 需要 Claude Code 2.1.257 或更高版本。未识别为已知模型的网关 ID 会收到 `unrecognized_model` 提示，并按 200K 上下文处理。[官方模型菜单与能力字段](https://code.claude.com/docs/en/settings-reference#fields-for-modelpicker)、[模型配置与 effort](https://code.claude.com/docs/en/model-config#adjust-effort-level)。

### 推理强度（DeepSeek Harness／Pi）

Harness 只在模型**声明了 reasoning levels** 时，才在模型选择器里显示 **Effort** 菜单；未声明时菜单中没有该项，由端点自身默认值决定是否思考。

- 目录按上游已确认枚举输出 `reasoningEfforts` 映射，键是菜单档位，值是线路上发送的 `reasoning_effort` 拼写；未确认档位的模型继续写 `reasoningEfforts: false`。
- `off` 显式映射为 `none` 而不是留空：留空等于不发参数，而 River 的 DeepSeek／GLM 在不带参数时默认仍在思考，留空会让「关闭」无效。
- 可关闭思考的 River 模型同时写模型级 `compat.thinkingFormat: deepseek`，使档位切换同步发送 `thinking: {type: enabled/disabled}`。GLM 没有关闭档位，因此不写该开关，也不伪造 `off`。
- 网关仍按各模型真实枚举复核；声明只表示「允许客户端选择」，实际生效由上游决定。依据 [Harness 模型配置](https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/user/guide/providers.md#reasoning-effort)。

### 图片输入（本地 Agent）

客户端目录按上游已确认能力声明图片模态，未确认的模型保持纯文本：

| 客户端 | 字段 |
| --- | --- |
| Harness Web／Desktop、Pi | 模型级 `input: [text, image]`（pi-ai） |
| OpenCode、Kilo、Qwen、MiMo Code | `models.<id>.modalities.input` |
| Goose | `models[].supports_vision` |

pi-ai 的 `input` 一旦显式非空就优先于安装目录，因此必须如实输出；省略会回落到路由 `defaultInput`（默认 `[text]`），同样看不到图片。声明只表示「允许客户端发送」，模型是否真的能识别由上游决定，`system`／`assistant` 消息中的图片仍由上游按官方限制拒绝。依据 [Harness 模型配置](https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/user/guide/providers.md#image-input) 与 [OpenCode 自定义提供商](https://github.com/anomalyco/opencode/issues/8875)。

- Codex 保留内置 `openai` 身份时使用应用管理的 `omniproxy-models.json`。支持远程目录的自定义提供商使用独立的 `/v1/codex-models` 地址和模型发现开关；旧核心使用本地兼容目录，不自动升级。
- Claude Desktop 写入完整的 `inferenceModels`，支持从其模型选择器切换网关模型和别名，不依赖只识别 Claude ID 的自动发现。
- Claude Code CLI 自动配置默认模型及完整 `/model` 菜单；也可通过完整模型 ID 切换，旧版客户端需支持 `modelPicker` 才会显示列表。

模型同步和路由／资源变更会更新未被外部修改的受管目录，客户端重启后生效；不会结束正在进行的会话。本地目录使用加密备份和文件版本保护，与原客户端配置一同恢复。目录位置已有未受管文件时拒绝接管。目录刷新失败记录安全错误，原有配置的外部修改不会被覆盖。

每个客户端独立接入 Key，可以停用、重新启用。后台使用本应用管理的上游账号凭据；客户端不会取得这些凭据。

## 会话保留

Codex 不改 model_provider，官方登录和会话根目录不变。内置 openai 通过专用本地接入路径；自定义提供商保留表名。认证助手／活跃配置 profile 冲突时拒绝自动修改。

Claude CLI 合并接入 env、完整模型菜单及已确认的压缩设置，不退出原登录。`modelPicker.options` 使用预览中的模型 ID 和名称，`replaceBuiltInOptions=true` 替换重复档位和不可用的内置模型，保留 Default 行；不覆盖已有 `availableModels` 限制。受管目录刷新同时更新 CLI 菜单，仍要求文件未被外部修改；原菜单随加密备份恢复。本机 Claude Code 2.1.292 已确认支持，旧版未显示时可更新或输入完整模型 ID。[官方模型菜单说明](https://code.claude.com/docs/en/model-config#add-a-custom-model-option)。

Desktop 应用时添加并选中「OmniProxy · 本地网关」第三方推理配置条目，合并两份启动配置的 `deploymentMode=3p`。原模式和选择保存为可恢复的非敏感状态；官方登录和原会话文件不改写。组织托管策略仍优先。[官方 Desktop 接入说明](https://code.claude.com/docs/en/llm-gateway-connect#desktop-app)。

「恢复原配置」按加密备份恢复原文件，原本没有文件时删除本次创建的文件。外部修改会阻止覆盖；Desktop 同时恢复原启动模式和配置选择，保留接入后新增的非受管设置。主配置版本和激活状态同事务提交，多文件写入或提交失败会回退；外部改变受管模式或当前配置时拒绝恢复。恢复同时停用 Key，重新应用可启用。旧版没有模式快照的配置仍保留先切回原配置再移除的保护。

## 重新读取配置

详情页「接入配置」标题右侧和一键配置弹窗底部均提供「重新读取配置」。重装或外部编辑后，重新读取当前版本、清除旧预览，然后再次预览／应用。只有明确重新读取后才允许接入已变化的文件；应用仍核对读取的文件版本，后续外部改动会再次被阻止。

重新接入会保留当前文件中的非网关设置，为当前内容创建新的加密恢复基准。迁移 013 的 `agent_backup_history` 保留旧备份引用；切换恢复基准与应用版本同事务提交，失败回滚当前文件并保留旧备份。该入口只读取配置元数据，完整内容和凭据不返回前端。

## 第一版能力

Gemini 当前只提供 Chat API，虽可在路由页同步官方模型和限制，但不参与 Codex／Claude 的一键配置模型列表。Messages／Responses 转换与完整工具签名桥接尚未实施，不能以输入上限直接宣称原生客户端已启用 1M。

- Windows 本地客户端配置；ChatGPT / Codex 官方 model/list 和 DeepSeek 官方 GET /models。
- Atria 账号自动提供 `Atria-Dawn-Preview` 官方预设和 256000／仅文本元数据，目录注明预设来源，不查询尚未确认的 `/v1/models`。Codex 使用本地 Responses，Claude CLI／Desktop 使用本地 Messages；Atria 原生 SSE 保留增量与工具关联。真实 Key 和客户端专有扩展仍需实机验证，不为 Atria 生成 1M 变体。
- GET /v1/models 动态合并可用原始模型与路由；所有请求校验本地接入 Key。
- GET /v1/codex-models 提供 Codex 专用元数据；只接受有界 `client_version` 参数。普通模型列表支持有界 `limit`、`after_id`、`before_id`；Messages 支持 Claude Code 实际发送的单个 `beta=true` 查询参数，重复、异常或未知参数仍拒绝，其他推理协议不接受查询参数。
- POST /v1/chat/completions、POST /v1/responses、POST /v1/messages。
- Codex Responses WebSocket，连接内上下文续接、会话和请求并发上限、空闲超时、重置 Key 状态检查。
- ChatGPT Responses 保留原生文本、工具及流式事件；跨协议支持文本和工具调用，包括 Codex namespace／custom 工具及 Claude tool_use／tool_result。Claude Code 在 messages 中附加的 system 文本提醒保留角色、位置和正文，系统消息中的工具及其他不支持内容仍拒绝。
- 跨协议返回完成后的 JSON 或 SSE；当前不会逐 token 翻译。DeepSeek 跨协议思考关闭，多模态及不支持的托管工具明确返回 unsupported_content。
- HTTP Responses 为无持久化请求，调用方需提供完整上下文；没有调度、静默重试或故障切换。

其他提供商模型目录、跨平台配置及高级协议能力留到后续任务。


## Codex WebSocket 预热与错误状态

已通过本机 Codex CLI 0.163.0 的隔离 loopback 契约测试验证：客户端先发送
`response.create` / `generate=false` 预热帧，再通过 `previous_response_id` 创建正式响应。
网关在本地回复预热事件，只保存当前连接上下文，不发起模型调用、不产生模型请求或用量记录；
正式请求合并上下文后移除 `generate` 等传输字段再转发。预热仍受鉴权、协议开关、请求大小及路由校验约束。

HTTP 与 WebSocket 共用上游失败分类：400 请求拒绝、401／403 认证失败、402 余额不足、
429 限流；5xx 统一为安全的 502 上游不可用。WebSocket 错误帧保留真实分类后的状态、稳定错误码及
合法的 `retry-after`，不再把全部上游错误或路由错误改写为通用流失败。收到 `response.completed`
即完成请求，不等待上游连接关闭。上游原始错误正文、凭据与提示词不会透传或记录。

本地可选实机客户端契约测试：设置 `OMNIPROXY_CODEX_TEST_EXECUTABLE` 为已安装 Codex 的绝对路径，
执行 `go test ./internal/core/proxy -run TestCodexCLIIsolatedWebSocket -count=1`。
测试使用临时 `CODEX_HOME`、假密钥、独立工作目录和随机 loopback 端口，不修改用户真实配置或登录，
只验证本机客户端与模拟上游，不代表真实账号权限或额度已通过验收。


## Codex CLI／Desktop 的模型设置层次

Codex CLI 与 Desktop 使用同一用户配置体系（实际根目录受 `CODEX_HOME` 控制），
OmniProxy 继续共用 `codex` 接入入口，不另改 Desktop 的登录、会话或首选项存储。
两者可能使用不同的内核版本，不能仅用 PATH 中 CLI 的结果声称 Desktop 界面已经生效。

需要区分三个状态：

1. 模型目录：`model_catalog_json`／远程发现决定可选模型；Desktop 后端通过 `model/list` 获取。
2. 用户默认：配置中的 `model` 决定未显式指定模型的新会话默认值。
3. 会话覆盖：CLI 的 `-m` 和 Desktop 后端新会话的显式 `model` 可以覆盖用户默认；已有会话保留原模型，不因默认文件改变而被强制切换。

`model/list.isDefault` 是模型推荐标记，不是当前会话或用户配置中的已选模型。
不要仅依据菜单推荐项判断一键配置失败，也不要将列表加载成功当作真实转发已经启用。
模型目录与上游入口分别配置；列表存在并不代表内置 OpenAI 入口已指向本机网关。

本机隔离验证覆盖 CLI 0.163.0 与 Desktop 0.163.0-alpha.10.2：使用真正的
`Service.Apply` 生成临时配置，内置 OpenAI／自定义提供商均验证了目录、默认模型、新会话、
显式覆盖及实际 loopback 请求中的模型；文件修改后新会话读取新默认，已有会话仍使用旧模型。
这不是 Desktop 外壳界面的逐项验收，也不代表真实订阅权限或模型额度已验证。

可选原生核心契约测试：同时设置 `OMNIPROXY_CODEX_CATALOG_QA=1` 与
`OMNIPROXY_CODEX_TEST_EXECUTABLE`（目标 CLI 或 Desktop 内核的绝对路径），运行
`go test ./internal/infrastructure/codex -run TestInstalledCodexLocalModelCatalog -count=1`。
配置、认证与会话都在临时目录，不会切换正在运行的用户客户端。

官方依据：[配置优先级](https://learn.chatgpt.com/docs/config-file/config-basic)、
[App Server 模型目录与会话模型](https://learn.chatgpt.com/docs/app-server)。


## 订阅 Responses 缺失媒体类型的兼容边界

真实官方订阅 Responses 端点可能返回 HTTP 200 和合法 SSE，却不带 `Content-Type`。
只在固定订阅适配器内兼容：状态必须为 200、类型头缺失、首个完整事件必须是可识别的
Responses 生命周期事件或错误事件。最多检查 2 MiB，不等待整流结束，所有预读字节原样回放。
显式 JSON／HTML／其他类型头不被覆盖；未知事件、无效 JSON、空正文和不完整首事件不会被认可。
Responses HTTP 与 WebSocket 的单事件容量统一为 2 MiB，避免工具定义回显形成的大首事件被旧的 512 KiB 上限误拒绝；其他协议的单行容量不变。
通用媒体类型校验、后续事件校验及流完整性检查保持；开始生成后不跨账号重放。固定订阅端点首事件明确额度拒绝的识别与切换边界见 `FAILOVER.md`。

网关日志仅记录 `upstream_media_type` 的固定分类与 `upstream_content_type_inferred` 布尔值，
不记录原始响应头、正文、提示词或凭据。补头不会绕过用量观察和首字计时。

2026-10-18 的修复构建已在独立临时数据库、随机 loopback 端口和只读隔离账号凭据下完成
真实上游验收：gpt-5.6-luna 与 gpt-6.1-sol 的 HTTP／WebSocket，另有实际安装的
Codex CLI 0.163.0 与 Desktop 内核 0.163.0-alpha.10.2 均完成请求；不是 Desktop GUI 外壳验收。
测试没有替换生产软件、刷新账号登录或改写用户 Codex 配置；修复不会自动进入正在运行的旧进程。


## 原生 OpenAI Responses 图片输入

Codex 目录只在目标是原生 OpenAI 订阅 Responses、且上游元数据明确包含 `image` 时声明
`input_modalities=["text","image"]`，别名沿用实际目标能力；未知／仅文本目标保持仅文本。
其他适配器的上游模型即便支持图片，也不代表网关的跨协议转换已支持图片，不能一律宣传多模态。

Responses HTTP／WebSocket 的 JSON 总量上限为 16 MiB，包括 Base64 图片、文本、工具上下文和
合并后的历史，不是单张图片的原始文件上限；较大图片的 Base64 约增加三分之一体积。
HTTP 超限返回 413；WebSocket 过大帧按 1009 关闭，历史合并／路由改写后的超限返回
`413 / request_too_large`，不会调用上游。普通 Chat Completions 继续使用 2 MiB 上限，
JSON 嵌套／重复字段、鉴权、并发、读取超时和只监听 loopback 等约束不变。

升级后需要让 Codex 读取新的受管模型目录；若配置或目录存在外部改动，应在接入页预览并
重新应用，不能绕过哈希保护直接覆盖共享配置。随后重新加载客户端或新建会话，以免继续使用
旧的仅文本能力快照。无需切换 `model_provider` 或重新登录。

2026-10-19 已用无隐私自建 PNG 验证 gpt-6.1-sol：约 3.16 MB 的图片 JSON 分别通过真实
HTTP／WebSocket，并由实际 CLI 0.163.0／Desktop 内核 0.163.0-alpha.10.2 附件请求识别
正确颜色与形状，共四次完整图片请求，用量及首字均记录。不是 Desktop GUI 外壳逐项验收；
测试使用独立数据库／端口／客户端目录，只读使用应用隔离凭据，没有改动用户真实配置或图片。

## Codex Desktop 文件变更与上下文压缩

Codex 的对话变更摘要依赖客户端核心生成的 `fileChange`／`turn/diff/updated`，不是网关从回答文本或 Git 日志推算。受管 Codex 模型目录为原生 OpenAI Responses 条目声明 `apply_patch_tool_type=freeform` 并提示优先用原生补丁工具进行手工编辑；其他提供商不推断该能力。仅通过脚本、格式化器或外部程序写文件，不保证形成相同的原生补丁事件。

目录修复需要刷新受管模型目录并重启／重新加载 Codex 后才能进入新工具列表；旧会话已发生的终端写入不会因此补生成历史变更摘要。新核心的回环契约覆盖新增／修改／删除三种文件操作，不能将此等同于所有 Desktop 外壳版本的视觉验收。

当前 Codex 0.163 核心的独立压缩采用 Compaction V2，经普通 Responses 请求携带 `remote_compaction_v2` 与回合压缩元数据，返回 SSE 中的加密 compaction 项。网关通过长度、JSON／重复字段和功能白名单校验保留这两项协议头，仅发送给固定 ChatGPT 端点且不记入日志；WebSocket 使用当前帧元数据，避免沿用旧压缩回合标记。真实本机 CLI／Desktop 核心的离线契约覆盖完整压缩生命周期；不将此当作真实大上下文或外壳 UI 已验收。

网关支持固定的 `POST /v1/responses/compact` 和带独立接入 Key 的 Codex 等价路径。压缩复用账号路由、授权、既有网关请求期限、并发／请求大小限制和用量记账；它不是普通的流式 Responses 推理。仅原生 ChatGPT 订阅适配器访问固定官方 `responses/compact` 端点，完整保留不透明压缩项和历史输入，不修改成 `stream=true`。其他适配器返回 `compaction_not_supported`，不通过 Chat／Messages 转换伪造摘要。

自动压缩触发阈值由 Codex 的 `model_auto_compact_token_limit` 与模型实际能力决定，独立于请求传输压缩 `enable_request_compression`。用户原有全局上下文窗口／压缩阈值不会被本轮修复改写；配置较大窗口不等于官方已确认该模型可承载同样容量。接口修复不会自动修复既有失败会话或证明真实大上下文压缩已成功。

协议对照：[官方 App Server 文档](https://learn.chatgpt.com/docs/app-server)、[Codex 配置参考](https://learn.chatgpt.com/docs/config-file/config-reference)、[Responses 压缩说明](https://developers.openai.com/api/docs/guides/compaction)。实际桌面内核的 V2 路径以本机离线契约确认，不根据公共 API 的 legacy 示例推断。
