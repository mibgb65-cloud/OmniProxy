# 本地 Agent 接入与模型同步

Google / Antigravity 订阅账号支持将实际发现的 Gemini、Claude Opus／Sonnet 4.6 及
GPT-OSS 文本型号同步到 Agent 目录，入口为账号卡片“同步到 AI Agent”或路由模型同步。
Responses／Messages 通过既有文本和工具转换接入，支持 JSON／有界 SSE；运行中的
客户端需重新加载模型菜单。普通 Gemini API Key 仍仅支持 Chat，不进入不兼容的
Agent 协议目录。详情见 [Google 订阅](GOOGLE_SUBSCRIPTION.md)。

## 使用

Kilo Code、Qwen Code、MiMo Code 各用一张 CLI／Desktop 卡片，提供本机引擎配置、模型
同步、独立 Key、备份恢复、检测和打开入口；MiMo 在同一卡片分别列出国内／国际桌面版。
Qwen 官方桌面应用使用普通 `qwen serve`
和用户设置；Kilo／MiMo 的桌面宿主可能覆盖引擎目录或内联设置，需以预览路径与实际
宿主为准，不能将全局配置成功当作所有桌面版本均已生效。见 [KILO_QWEN_MIMO_CODE.md](KILO_QWEN_MIMO_CODE.md)。

OpenCode CLI／Desktop 共用一张接入卡片；DeepSeek Harness 的 Desktop／Web 在同一卡片中
分别配置 profile。均支持本地 Chat 文本／函数工具目录、独立 Key、加密备份／恢复及目录
刷新，安装检测和固定官方入口共用原流程。详情见 [HARNESS_OPENCODE.md](HARNESS_OPENCODE.md)。

River API Key 可用于本地 Codex Responses 与 Claude Messages 文本／工具接入；Messages 转换需要缓冲，不是原生增量流。预设与发现模型均进入同一路由目录，限制见 [RIVER.md](RIVER.md)。

1. 在账号页登录 ChatGPT / Codex，或添加 DeepSeek／River／Atria／MiMo 普通 API Key。MiMo 可使用官方预设或在路由页同步账号模型，兼容边界见 [MIMO.md](MIMO.md)。
2. 在路由页选择账号，点击「同步官方模型」。目录按账号保存；失败保留原目录，更换凭据／工作区会使旧目录失效。改名、额度刷新不会清除目录。
3. 可以直接使用已同步模型，也可以新建精确路由并固定账号。
4. 网关页「客户端接入」全部使用统一的产品分组卡片：品牌与产品标题在上方，内部逐项展示 CLI／Desktop／Web 入口。每项单独显示对应软件的安装状态和实际配置的接入状态，点击进入配置页面，无需先启动网关。
5. 详情页宽屏采用左右两栏：左侧管理软件安装、配置路径、文件状态、接入 Key 复制与启停、原配置恢复；右侧默认展示实际已写入当前客户端的模型、默认模型标记、搜索与刷新。窄容器改为上下排列。点击「一键配置」后右侧切换为配置接入模型，自动加载预览并默认勾选全部适用模型；可以搜索名称／ID、逐项勾选、全选或清空，并从所选模型中指定默认模型。未调整时保持完整目录；全部取消会提示至少选择一个并禁用应用。确认数量与目录方式后应用，成功即返回已接入列表；取消不写入文件并返回原列表。重新启动对应客户端后，在客户端中选择模型；实际调用前需启动网关。

已接入列表从版本匹配的受管配置／本地目录读取，只返回公开 ID、名称和默认模型；未接管时显示空状态，外部修改或读取失败时展示错误，不把网关可用目录冒充为已接入列表。Codex 受管远程发现模式按当前远程目录展示，其他模式读取实际菜单；Key 停用时保留列表并显示停用提示。该列表不表示客户端已经重新加载或请求已验证成功。

已接入模型采用紧凑卡片，按容器宽度和字号自动分配每行数量，筛选后少量卡片保持原列宽。搜索旁的「仅看 [1m]」按钮按真实模型 ID 的精确后缀筛选，可与搜索叠加；再次点击恢复全部范围，并显示当前匹配数／总数。不修改接入模型、默认模型或配置文件。接入配置区不再展示网关未启动的通用提示，网关状态仍显示在公共控制区。

卡片网格按内容区实际宽度自适应，不再固定两列。最小卡宽为 22rem，窄容器按满宽收缩；
大屏和侧栏收起时自动增加列数，余宽分配给现有列，末行保持与上方列对齐，字号缩放时
同步调整排列。

Codex、OpenCode、Kilo、Qwen 的 CLI／Desktop 入口进入同一详情；MiMo CLI／国内 Desktop／国际 Desktop 也进入同一详情，共用各产品的接入状态。Claude 的 CLI／Desktop、Harness 的 Desktop／Web 分别进入其独立配置。安装状态始终按软件区分，不因共享 Key 就宣称对应软件已安装。

配置页面使用 `/gateway?tab=clients&client=客户端ID` 定位，支持 `codex`、`claude`、`claude-desktop`、`opencode`、`harness-web`、`harness-desktop`、`kilocode`、`qwen-code`、`mimo-code`。支持刷新与前进／后退；非法参数回到卡片目录。返回目录恢复原软件入口焦点（如从 Desktop 返回仍选中该 Desktop），选项卡往返保留正在进行的配置操作。点击入口只导航，不写入客户端配置或发起模型请求。

配置预览、保存与网关运行状态独立。没有适用的模型／路由时，应先同步账号模型或配置可用路由。目录在预览后变化时，需要重新预览；首次接入使用稳定默认，仍可用的原模型优先保留。

选择部分模型后，目录刷新只更新所选且仍可用的模型，不会自动加入取消的模型或新增模型；所有所选模型失效时保留原文件并报告无可用模型。选择保存在受管备份记录（迁移 031），重启应用后仍生效，恢复原配置时一起清除。全选应用恢复动态完整目录。Codex 部分模型使用本地目录并关闭远程发现，避免远程完整目录覆盖选择；不改变网关路由或接入 Key 的模型调用权限。

Harness 保持显示建议模型，不提供默认模型写入；保存后仍需在 Harness 中选择模型。

右侧配置面板正文和模型目录统一使用细圆角滚动条，保留固定轨道占位，默认隐藏，悬停轨道或拖动时淡入、移开后淡出；滚轮和键盘滚动可用，减少动态效果模式下取消过渡。模型目录高度按视口限制，标题和操作按钮位于正文滚动层之外。保存／取消后恢复配置按钮焦点；编辑时左侧配置操作暂时禁用，避免并发覆盖。

应用成功后，右侧返回已接入模型列表，顶部悬浮显示“配置已保存，原配置已加密备份。”，无需滚动到内容底部；约 5 秒后自动收起，可手动关闭，鼠标悬停或键盘聚焦时暂停计时。提示适配深浅主题与减少动态效果，通过温和的状态播报告知结果；收起通知不会改变已保存状态。

## Codex WebSocket 接入开关

账号 → OpenAI →「OpenAI 账号设置」提供应用级「Codex WebSocket」开关，默认开启。只控制当前 OmniProxy 网关是否允许 Responses WebSocket，所有 OpenAI 账号共用；设置保存在本机 SQLite（迁移 016），无需启动或重启网关即可保存。关闭后新握手返回 `403 websocket_disabled`；已连接会话在下一次创建响应时被拒绝，进行中的请求可以正常结束。普通 HTTP/SSE 保持可用，客户端是否自动回退取决于自身版本。

该开关不修改 Codex CLI／Desktop 的配置、提供商身份、登录、会话或环境变量。Codex 0.155.1 已将旧 responses_websockets／v2 标记为 removed，内置 openai 禁止覆盖；不能将旧标志的 false 当作真实关闭客户端传输。用户选择控制网关以保持 ADR-005 的身份保护。参考：[官方配置参考](https://learn.chatgpt.com/docs/config-file/config-reference)、[开发者设置](https://learn.chatgpt.com/docs/developer-settings)。

只影响当前运行的新版网关；旧应用进程和其他端口不联动。客户端若仍指向旧版 `/codex/v1` 地址，应使用新版网关页的一键接入，不能宣称只打开本开关已修复其他进程的 403。

## 客户端安装

卡片显示真实的软件安装状态，与网关接入 Key 状态分别展示。详情页的「软件安装」支持只读检测、重新检测、安装和官方说明。Codex CLI 与桌面应用分别检测；Claude Code CLI 和 Claude Desktop 也各自检测。

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

- Codex 保留内置 `openai` 身份时使用应用管理的 `omniproxy-models.json`。支持远程目录的自定义提供商使用独立的 `/v1/codex-models` 地址和模型发现开关；旧核心使用本地兼容目录，不自动升级。
- Claude Desktop 写入完整的 `inferenceModels`，支持从其模型选择器切换网关模型和别名，不依赖只识别 Claude ID 的自动发现。
- Claude Code CLI 自动配置默认模型及完整 `/model` 菜单；也可通过完整模型 ID 切换，旧版客户端需支持 `modelPicker` 才会显示列表。

模型同步和路由／资源变更会更新未被外部修改的受管目录，客户端重启后生效；不会结束正在进行的会话。本地目录使用加密备份和文件版本保护，与原客户端配置一同恢复。目录位置已有未受管文件时拒绝接管。目录刷新失败记录安全错误，原有配置的外部修改不会被覆盖。

每个客户端独立接入 Key，可以停用、重新启用。后台使用本应用管理的上游账号凭据；客户端不会取得这些凭据。

## 会话保留

Codex 不改 model_provider，官方登录和会话根目录不变。内置 openai 通过专用本地接入路径；自定义提供商保留表名。认证助手／活跃配置 profile 冲突时拒绝自动修改。

Claude CLI 合并接入 env、完整模型菜单及已确认的压缩设置，不退出原登录。`modelPicker.options` 使用预览中的模型 ID 和名称，`replaceBuiltInOptions=true` 替换重复档位和不可用的内置模型，保留 Default 行；不伪装模型身份，不覆盖已有 `availableModels` 限制。受管目录刷新同时更新 CLI 菜单，仍要求文件未被外部修改；原菜单随加密备份恢复。本机 Claude Code 2.1.292 已确认支持，旧版未显示时可更新或输入完整模型 ID。[官方模型菜单说明](https://code.claude.com/docs/en/model-config#add-a-custom-model-option)。

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
