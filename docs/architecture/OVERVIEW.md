# Architecture Overview

## OpenAI 账号轮换（2026-10-20）

迁移 035 在已有 Codex 设置表保存 queue／balanced，默认 queue 保留兼容。
`application/gateway` 在既有同池计划上做有界、并发安全的平滑加权首次选号，实际转发仍走原有状态／版本核验和不可重放边界；
额度权重只从 `application/subscription` 已验证的同版本、一分钟内完整观察读取，无新探测请求。
缺少可比快照时等权分配，固定账号与其他提供商不变；Core 无新增依赖或调度接口。见 ADR-018。

## Codex 自动激活的滑动目标观测（2026-10-19）

`quotaactivation` 将“未来日期”与“已开始”分开：以同身份的独立时间观测确认完整周期目标是否滑动，
在事务中保存原始观测与已确认周期，发送前仍复核原账号并按周期至多一次。迁移 034 只增加两个
非敏感观测字段；保留原登记与不确定结果保护，不改默认客户端登录／配置。观察未知与请求已完成
但窗口未确认都保持待复查；统一日志仅包含计数和固定错误码。详见 ADR-009、`product/SUBSCRIPTION.md`。

## 应用内更新与安装（2026-10-18）

`infrastructure/update` 增加平台资产精确配对、有界下载和受限重定向；`application/updater` 冻结候选、共享检查／下载／安装快照、持久缓存及安装前远端摘要复验。Wails 绑定只接收预期版本，正式运行环境可自动检查，下载默认由用户发起；用户可独立开启正式安装环境检查后的自动下载，安装仍必须手动确认。前端侧栏、浮窗和设置页共用 Query 本地快照，无正文 HTML 渲染。当前版本日志通过 Vite 静态收录 `docs/releases/v*.md` 并按 RuntimeInfo.version 匹配，不新增联网读取接口；开发服务仅额外开放该发行说明目录。

`infrastructure/appinstall` 在 main 初始化 profile 前分流受限辅助模式，按完整程序路径请求正常退出或超时终止；NSIS 沿用旧登记目录并保留数据。独立更新辅助进程等待提权安装结果，再以原用户令牌重新启动。Release 先 build／verify／package，再在 draft 上传完整资产并最终发布。无需新依赖或迁移，见 ADR-017、`product/UPDATE.md`、`development/RELEASE.md`。

## v1 导入与更新渠道（2026-10-18）

`infrastructure/legacy` 只读、有界解析固定旧文件并兼容 Windows v1 DPAPI entropy；`application/legacyimport` 预览脱敏结果、复核文件快照并逐账号调用既有创建事务。`account.ImportLegacyKey` 复用校验／密文保存／事务失败清理，确定性旧身份及目标 Key 检查提供幂等性。Wails 串行化导入操作，不自动扫描、不迁移订阅登录或改写客户端；无需数据库迁移。

`version.Compare` 保留预发布优先级；`infrastructure/update` 对 Beta 使用有界分页，稳定渠道使用正式端点，校验标签并构造可信发布链接。渠道配置复用 profile 原子文件和设置锁，未明确选择时跟随应用版本；前端仅按需联网并在渠道切换后清除旧结果。边界见 ADR-016、`product/UPGRADE.md`。

## 同通道账号故障切换（2026-10-18）

`application/routing.ResolvePlan` 复用既有 evaluate，并区分手工固定和兼容别名；`application/gateway` 在冻结服务商／认证／计费／地区及逐次版本检查后执行最多三次顺序尝试。明确拒绝在前一个响应关闭及凭据回调退出后才换账号，开始生成后不重放，固定 Codex 端点首事件明确额度拒绝的例外见 ADR-015。短期健康状态使用有界内存、失败代次及单恢复资格；单请求内存凭据摘要排除重复 Key，无后台探测或新增依赖。

迁移 033 保存开关与请求子尝试；`proxy.Trace` 延迟父账号绑定到最终消费／原上游 usage，`application/activity` 保留原 subject 保护、一次逻辑计数及累计，并提供可选详情／幂等恢复／级联清理。订阅读取在等待锁后复核配置版本。Wails 只提供设置／运行快照，UI 沿用现有查询和详情。范围及取舍见 ADR-015、`product/FAILOVER.md`。

## Kimi Code、Pi、Factory Droid 与 Goose（2026-10-18）

`application/agentconfig` 增加四个精确客户端 ID，沿用 Chat 目录、独立接入 Key、模型选择持久化及加密备份；Kimi 根据当前／旧 Python 配置选择协议。Pi／Goose 使用固定关联路径的有界双文件快照，逐文件版本校验和失败回滚，不引入 SDK、运行时或数据库迁移。`infrastructure/agentinstall` 增加固定官方入口、CLI 版本与 Goose 桌面检测；前端复用产品分组及详情。实现边界见 `product/KIMI_PI_DROID_GOOSE.md`。

## Kilo／Qwen／MiMo Code 客户端（2026-10-17）

`application/agentconfig` 新增三个独立接入 ID，复用既有加密备份、版本校验、Key 鉴权
与目录刷新。Kilo／MiMo 复用 JSONC 提供商合并器；Qwen 使用用户设置的模型列表、专用
环境引用和 OpenAI Chat 认证选择，保留其他协议与 OAuth 文件。配置路径遵循各引擎
当前环境和文件优先级；桌面宿主的独立目录／内联覆盖不凭空推测。安装检测与打开使用
固定产品 ID，未经核实的安装包只提供官方入口。无新增依赖、迁移或跨层依赖。
MiMo 的国内／国际桌面目标各自检测和打开，使用已验证的 `Xiaomi MiMo`／`Xiaomi MiMo AI`
产品名与主程序名；兼容缺失 Publisher／InstallLocation 的 NSIS 项，保持两版状态独立。

Kilo 实际发送纯文本内容数组，`core/proxy` 在既有校验前将有界 `text` 部分合为字符串，
保留工具和扩展字段；图片／未知部分、超量或超长内容仍拒绝。字符串请求保持原字节。
详见 `product/KILO_QWEN_MIMO_CODE.md`。

## Harness／OpenCode 客户端（2026-10-17）

`application/agentconfig` 扩展 OpenCode 共用配置和 Harness 两个 profile，不修改核心代理
协议。OpenCode 使用 HuJSON 增量修改 JSONC；Harness 使用 YAML 节点保留设置，通过受控
文件快照将 profile 与单个凭据引用一起校验／备份／回滚，恢复不会覆盖共享凭据文件中的
其他项。沿用现有 DPAPI 备份表和精确鉴权白名单，无迁移；仅传递本地接入 Key。
安装器返回明确的可安装标记；未核实渠道的新目标只提供官方说明和固定本机打开入口，
不能由前端传递任意路径或命令。见 `product/HARNESS_OPENCODE.md`、ADR-005。

## Google 账号资源与 Agent 模型（2026-10-17）

API Key 项目资源内嵌对应账号卡片／列表，共享查询保持项目维度，配额总览仍按项目归组。
Antigravity 第三方文本型号只按官方选择器与精确适配白名单发布；订阅路由目录开放
Responses／Messages，普通 API Key 保持 Chat。网关复用 `core/agentprotocol` 的文本与
工具转换，Chat 保留增量 SSE，其他协议缓冲完整结果；256 条／1 小时的授权与内容摘要
隔离内存缓存恢复工具签名，不保存对话正文。账号“同步到 AI Agent”沿用 ModelSync
和受管菜单刷新，不新增接口、数据表或依赖。

## River 内置推理（2026-10-17）

迁移 029 追加第十一家内置服务商。`provider/river` 使用标准库原生 Chat／Responses HTTP／SSE，Messages 复用现有文本工具转换并缓冲完整结果。账号、DPAPI、目录凭据版本校验、路由和原上游 usage 沿用现有边界。模型预设与实时发现区分，仅开放已核对的三款文本模型；详情见 `docs/product/RIVER.md`。

迁移 030 将 River 旧优先级合并到逻辑型号；`provider/river` 集中维护公共 ID／短名称与上游 ID 的精确映射。路由先按逻辑型号选账号，再还原所选提供商的实际 ID；网关对默认映射和显式别名均改写正文。公共目录去重，兼容目录保留旧原始 ID 及 Claude Desktop 摘要；账号快照仍保存官方原始 ID。Claude 的 DeepSeek 普通／1M 双选与已实测订阅目标的独立规则分别维护。

## Google Antigravity 订阅（2026-10-16）

订阅额度通过独立 `GoogleSubscriptionQuota` → `application/googlesubscription.ReadQuota`
→ `provider/antigravity.ReadQuota` 读取 `retrieveUserQuotaSummary`，保留官方分组与窗口。
DTO 仅包含额度、标签与时间，比例／重置严格验证；读取前后核对授权版本，网络不占用
整个刷新互斥锁。前端与其他订阅共用实际 RPC 串行队列，详情／总览共享 60 秒缓存，
仪表盘默认不触发上游。没有新依赖、迁移、推理请求或额度历史持久化。

`GoogleSubscriptionModels` 复用相同授权版本保护，独立读取官方模型选择器引用，
过滤内部条目并标记现有网关适配状态；前端模型缓存与额度缓存分离，详情／总览共用。
不改变路由既有 Gemini 模型及历史别名兼容行为。

`provider/antigravity` 使用标准库实现独立 Google OAuth、账号项目／模型查询与
Gemini Chat JSON／增量 SSE 转换；只接受固定官方 HTTPS 主机并保留工具签名。
`application/googlesubscription` 管理一次性回调、超时、取消及刷新互斥，
`application/account` 复用 DPAPI 保护 token bundle；模型目录的授权版本检查
拒绝旧登录响应，普通 token 轮换则在同一事务内保留原目录。Wails 既有订阅绑定
按 Google／Codex 分发，共享前端会话组件和变更事件，Core 不依赖两者。
无新依赖／迁移，不跨计费通道重试。见 ADR-013 与 `product/GOOGLE_SUBSCRIPTION.md`。

## Kimi Code 与默认使用顺序（2026-10-16）

迁移 028 追加账号计费字段及 Kimi 首选计费／地区表，旧账号按量身份和目录版本保持。`provider/kimi` 按账号固定解析普通／coding 两地区入口及原生模型映射，沿用保护凭据回调与原生三协议／SSE。路由共用计费筛选后应用地区顺序，目录使用实际账号能力，旧通道／Claude ID 仅兼容解析。Core 有界保留真实客户端 User-Agent，订阅上游原样使用，不依赖 Wails 或 React；订阅余额／发现不调用未核实接口。前端账号、设置、未知资源及受管目录复用既有组件。见 ADR-012、`product/KIMI.md`。

## StepFun 默认计费顺序（2026-10-16）

迁移 027 独立保存 StepFun 首选计费方式，初始 Step Plan 优先；与 MiMo 共用应用层合格候选筛选及前端设置保存组件，保留服务商优先级、账号稳定顺序与各自缓存／持久值。公开目录移除 StepFun 自动通道副本，旧通道及 Claude ID 沿用兼容目录解析；Router 能力、账号版本和原生协议边界不变。见 ADR-011、`product/STEPFUN.md`。

## MiMo 默认计费顺序（2026-10-16）

迁移 026 保存 MiMo 的应用级首选计费方式，初始 Token Plan 优先；路由快照读取持久设置，在每个服务商内过滤合格计费通道后复用核心账号选择，保留固定账号与服务商优先级。公共目录停止生成 MiMo 通道副本，兼容目录仅供旧 Claude 上下文／桌面编码 ID 解析。Wails 设置绑定复用受管目录刷新，前端沿用 TanStack Query 的读取、即时保存及互斥；无新的依赖、凭据处理或失败重试。见 ADR-010、`product/MIMO.md`。

## 小米 MiMo Token Plan 多集群（2026-10-15）

迁移 025 在账号元数据加入计费方式、集群及通道版本，并扩展统一目录版本视图。`provider/mimo` 在原生客户端允许 cn／sgp／ams 三个精确官方套餐主机，按通道校验 Key 和模型，复用原有协议与安全边界；普通按量入口保持兼容。应用层目录、路由、客户端菜单共用结构化别名与账号版本，转发在 `UseAPIKeyAt` 中执行，历史保存当次地区／计费方式。前端新增／批量／编辑及资源显示沿用既有模式，官方套餐额度保持未知、只打开固定平台页，无自动 Cookie 获取和新增依赖。见 `product/MIMO.md`。

## Kimi 普通 API 余额（2026-10-15）

`provider/kimi.ReadBalance` 复用固定地区及禁止重定向的原生 HTTP 客户端，严格校验成功状态和有界十进制响应。`application/balance` 在受保护账号版本回调内查询并复核地区／版本，以独立 Kimi DTO 返回可用、代金券及现金金额，保留负现金与精度；可用状态采用精确十进制比较。账号详情／配额和仪表盘沿用共享缓存与有界队列，无新增接口方法、迁移或依赖。见 `product/KIMI.md`、`product/BALANCE.md`。

## MiniMax 双平台／两类 Key（2026-10-15）

迁移 024 保存 MiniMax 账号地区、计费类型和版本；旧 URL 只能确定地区，计费类型待用户确认。`provider/minimax` 使用固定现行 `.cn`／`.io` 主机和受保护 Key 回调处理原生三协议与模型发现，不跨通道回退。目录版本、别名路由、在途快照和历史计费身份沿用 Step／Kimi 模式。`application/minimaxquota` 查询固定 `www` 官方订阅入口并在返回前复查账号，前端共享有界缓存；不把次数零值当满额，不将本机 Tokens 当官方剩余。普通余额与高级 Responses 工具保持未开放，详见 `product/MINIMAX.md`。

## 小米 MiMo 原生提供商（2026-10-14）

`provider/mimo` 使用普通按量 HTTPS 域名和 `api-key`，在已有 `UseAPIKey` 回调内转发 Chat、Responses 与独立 `/anthropic/v1/messages`，不引入 SDK。原生 SSE 与思考／工具上下文保持，复用 `proxy.ObserveResponse`。模型发现有界并筛选已核对对话模型；未同步账号提供标记为 preset 的目录，成功快照优先，凭据版本变化后失效。路由、客户端菜单和网关接入检查同步扩展，不新增数据库迁移。首版只公布文本能力与官方 1M 上下文；Token Plan 和多模态尚未接入，见 `product/MIMO.md`。

## Atria 原生提供商（2026-10-13）

迁移 019 追加默认启用的 Atria，018 预留给并行 GPT 长上下文任务。`provider/atria` 通过既有 `proxy.Upstream` 协议转发三种原生 HTTP／SSE，并在受保护 API Key 回调内管理资源和认证。路由与客户端目录共享精确模型 ID 和 256000／文本元数据；`application/modelcatalog` 派生账号预设，来源为 preset、不伪造发现时间。统计仍观察原上游 usage；未确认的发现／余额接口不调用。

## 代理统计（2026-10-09）

2026-10-17：`core/proxy.Trace` 使用单调时钟记录首次有效下游流式内容；HTTP 等完整 SSE 事件刷新，WebSocket 等消息写出，跳过元数据与空事件。迁移 032 保存可空明细及独立桶总和／成功样本，结束事务原子提交并保留幂等与溢出保护。`DashboardRead` 增加可选首字、全部未结束请求和累计提供商分布；网关与首页共用查询缓存／本地更新信号，界面不调用模型，无新增依赖。定义与取舍见 ADR-014。

迁移 009 增加独立 provider_id＋model 累计，与原用量差值事务同步。用量查询新增可选 providerModels 字段，按具体提供商配置展示模型表，保留原全局模型汇总。资源删除和请求淘汰不影响该累计；历史回填不能确定归属时保留历史名称分组。

`core/proxy.Trace` 统一 HTTP／WebSocket 请求 ID 与生命周期，原上游 usageReader 采集官方快照，避免跨协议默认零值污染统计。`application/activity` 实现 RequestObserver，迁移 008 事务更新请求明细及独立累计／每日汇总，记录身份快照，无资源删除外键。Wails 提供薄查询／保留设置绑定，用量与请求页通过 TanStack Query 读取；日志按 request_id 双向关联。见 ADR-006 与 `docs/product/ACTIVITY.md`。

## 本地 Agent 与模型目录（2026-10-09）

2026-10-12：`application/agentinstall` 管理只读检测缓存、单个后台安装任务与结果核对；`infrastructure/agentinstall` 只允许固定的 Windows WinGet 包，检测 CLI／注册表／当前用户 MSIX。Wails 提供快照、重新检测、安装和固定官方说明入口，前端仅在任务运行时轮询。安装不读取客户端认证或会话，不自动升级既有版本，不新增数据库迁移。Codex 安装成功后同步本机版本到配置目录适配逻辑。详见 `docs/product/LOCAL_AGENTS.md`。

2026-10-17：Claude Desktop 增加显式只读更新检查，应用层复核已安装版本并与安装互斥，基础设施只读取固定架构的官方 MSIX 发布重定向，不跟随下载；域名／路径／数字版本无法验证时返回未知，网络错误仅暴露稳定码。界面独立显示加载与结果，不自动检查或升级其他客户端。

发现更新后，同一按钮提供显式后台升级，通过固定 WinGet 商店包执行；复用单任务互斥、ConPTY 进度、轮询和退出取消。安装快照增加可选操作类型与目标版本，只有实际 MSIX 版本达到目标才完成，重新检测不会把旧安装误当升级成功；普通安装仍禁止升级。无需新依赖或迁移。

更新前复用五分钟内服务端验证的发布目标并重新核对本机状态，显式检查仍强制查新。ConPTY 保留脱敏退出码，将无适用更新识别为渠道不可用，提供固定官方下载页入口；前端错误白名单与组件回归共用实际解析，避免网络预检失败落入通用提示。

安装进度通过后台 ConPTY 的 WinGet 输出读取，转换成有界的阶段／百分比／字节数，原文不进入 UI 或日志。安装快照带起止时间，支持无数值时的动态进度反馈。配置重装恢复通过显式重新读取、文件版本校验和迁移 013 的旧备份引用归档完成；新恢复基准与应用版本同事务提交，失败回滚当前文件。

2026-10-10：`core/modelcatalog` 增加中立目录、边界校验与 Codex 格式适配。迁移 011 保存官方元数据和受管目录备份引用；路由按实际选中账号关联目标能力。配置服务自动生成完整目录、保留可用原默认，预览修订绑定目录方式／默认／端口；本地目录与配置双文件版本提交失败时回滚。Codex 内置身份采用本地兼容目录，适用自定义 Provider 可用远程目录；Claude Desktop 使用完整显式列表。见 ADR 008。

`application/agentconfig` 管理固定客户端配置位置、字段合并、版本冲突、DPAPI 备份及独立接入 Key；`application/modelcatalog` 将账号官方目录存入迁移 007 的 `account_models`，路由按凭据／工作区和启用状态校验。Wails 只提供薄绑定。

2026-10-10：`application/routing` 从目录派生默认同名映射，迁移 012 独立保存模型服务商偏好，读取默认列表不创建手工规则。精确规则保持强制目标语义；未命中时由独立核心账号选择函数将优先服务商排在原创建时间／ID 顺序之前。无可用优先候选时回落到其他同名模型候选，网关、预览和客户端目录共享此决策，不引入请求失败重试。

`core/agentprotocol` 负责文本和工具的协议转换；`core/proxy` 负责鉴权、HTTP/SSE 与 WebSocket 生命周期，不读取凭据。`application/gateway` 协调 DeepSeek API Key 和隔离 ChatGPT 订阅回调。`infrastructure/codex` 在内存兼容官方 keyring／age 授权；`provider/chatgpt` 只转发到固定官方端点。原始 Responses 保留流式能力，跨协议有界收集完整结果后输出目标格式。 Codex Compaction V2 通过普通 Responses 保留经校验的功能标记和回合元数据；legacy 独立 Compaction 使用受信请求标志，经固定原生订阅端点返回 JSON。两者均不进入跨协议转换、不修改加密上下文。受管原生 Codex 目录显式声明 freeform 补丁能力，文件变更事件由客户端核心产生。

2026-10-13：迁移 015 在既有模型偏好记录增加有界服务商顺序，将旧首选迁移为单项列表，保留旧字段和首选接口。核心账号选择先按服务商顺序，再按创建时间／ID；不可用的服务商跳过，未排序候选沿用自动选择。Wails 增加顺序保存接口，默认映射返回可用性与完整顺序，预览返回实际优先级；网关与客户端目录继续复用同一解析，无请求失败重试。

客户端会话身份保护及官方加密存储兼容见 ADR 005，第一版能力边界见 `docs/product/LOCAL_AGENTS.md`。

## 阶跃星辰账号级通道（2026-10-14）

迁移 021 保存 Step 专用区域／计费方式和配置版本，`account_catalog_versions` 将凭据与配置共同绑定模型快照。`provider/stepfun` 只解析四类固定官方 HTTPS 入口，复用受保护 Key 回调与原生三协议 HTTP／SSE。路由从账号选项生成通道别名，混合池的裸模型拒绝歧义，已解析请求和取 Key 时均检查账号快照；不跨平台或计费模式重试。迁移 022 保存当次地区／计费方式，账号用量身份按通道分离。

普通余额采用独立 DTO 保留 prepaid／postpaid、十进制金额及累计分项，不伪造币种或订阅 Credit。Step Plan 目录为官方预设，剩余资源未知；详情和总览共用缓存及固定官方入口。见 `docs/product/STEPFUN.md`。

## Kimi 普通开放平台（2026-10-14）

迁移 023 保存账号级国内／海外选项，并扩展统一目录版本视图；旧空地址保留原海外语义，未知自定义地址不猜平台。`provider/kimi` 使用固定官方主机及原生协议，Messages 为 `/anthropic/v1/messages`；K3 Responses 无状态并保留命名空间／自定义工具及推理历史，K2 Responses 不声明支持。地区别名与裸名歧义共享真实路由决策，在途请求／目录同步校验版本；复用 DPAPI、SSE、缓存 Token 口径及当次地区记录，未新增 SDK。见 `product/KIMI.md`。

## System Overview

2026-10-13：迁移 017 和 `application/quotaactivation` 管理默认关闭的自动额度激活授权、周期登记和单后台检测循环。复用订阅身份校验／隔离令牌和固定 ChatGPT Responses 适配器，绕过模型路由固定原账号。发送前事务登记，未知结果只复查，独立额度信号更新前端共享缓存。生命周期由 Wails 启动／取消／有界关闭；决策见 ADR-009。

2026-10-13：迁移 016 保存应用级 Codex WebSocket 开关，默认开启。`application/gateway` 设置服务通过 Core 的只读策略回调，在鉴权后控制新握手及连接内后续创建响应；读取失败拒绝访问并返回稳定错误码。Wails 提供薄读写接口，OpenAI 账号设置共用 Query 缓存，不改写客户端身份／认证或配置。

仪表盘统计由 `application/activity` 的迁移 010 时间桶承担。开始、用量差值、结果及恢复同事务更新，读取根据客户端本地日历的 UTC 边界聚合。桌面适配层合并只含变更信号的 `activity:updated` 事件；首页资源只订阅现有缓存，主动操作才请求官方接口。决策见 ADR-007，页面边界见 `docs/product/DASHBOARD.md`。

OmniProxy is a local desktop application built as three layers:

```text
┌────────────────────────────────────────────┐
│ Desktop Shell (Wails 3 + React frontend)   │  internal/transport/wails + frontend/
├────────────────────────────────────────────┤
│ Application Services (use cases)           │  internal/application/<feature>
├────────────────────────────────────────────┤
│ Core (gateway, routing, scheduler, …)      │  internal/core/…
├────────────────────────────────────────────┤
│ Infrastructure (db, credentials, logger)   │  internal/infrastructure/…
└────────────────────────────────────────────┘
```

The Core must stay independent of Wails and React: if both were deleted, the
Core would still compile and run behind the local HTTP Control API.

## Major Components

| Component            | Location                              | Responsibility                                            |
| -------------------- | ------------------------------------- | --------------------------------------------------------- |
| Desktop shell        | `internal/transport/wails`            | Wails app, window, typed bindings, lifecycle              |
| Control API          | `internal/transport/httpapi`          | Local loopback HTTP API (chi) for CLI/agents/MCP          |
| Application services | `internal/application/<feature>`      | Use-case orchestration shared by both transports          |
| Core                 | `internal/core/…`                     | Gateway, router, scheduler, quota, usage, model registry  |
| Provider adapters    | `internal/provider/<name>`            | Per-provider capability behind one adapter interface      |
| Client integrations  | `internal/integration/<client>`       | Detect/read/backup/apply/restore/validate client configs  |
| Infrastructure       | `internal/infrastructure/…`           | SQLite database + migrations, credential store, logger    |
| Runtime profile      | `internal/platform/profile`           | Production/development/test isolation (ports, data dirs)  |
| Frontend             | `frontend/src/…`                      | React SPA: pages, features, components, hooks, services; all user-facing copy goes through `frontend/src/i18n` (zh/en) |

## Component Responsibilities

DeepSeek 余额查询通过 Wails `AccountBalance` → `internal/application/balance` → `internal/provider/deepseek` 实现。应用层核对账号归属，在 account.UseAPIKey 回调内请求固定官方地址；返回金额文本与币种，不暴露凭据。无需启动网关，无余额历史落库，前端按账号版本单独缓存。见 `docs/product/BALANCE.md`。

订阅额度查询通过 Wails `SubscriptionQuota` → `internal/application/subscription` → `internal/infrastructure/codex` 的官方 stdio 协议读取，仅返回窗口剩余百分比与重置时间。账号元数据列表不触发网络查询；前端单独按需缓存额度，既有操作互斥限制子进程数量，失败不改变授权。具体边界见 `docs/product/SUBSCRIPTION.md`。

手动重置卡使用通过同一官方组件完成，账号应用服务维护迁移 006 的操作记录。调用前持久化幂等键和身份归属，兑换结果不确定时保留 pending，核对复用同一键；完成回执和刷新失败独立处理，不由前端推算额度恢复。见 ADR-004。

### Desktop shell (`internal/transport/wails`)

Creates the Wails application and the main window, registers bound services
(the `AppService`), and manages startup/shutdown. It contains no business
logic. It is the only package allowed to import the Wails runtime.

On Windows, the shell also owns the system tray and the window-close hook.
Closing the main window exits by default; a persisted setting can hide it to
the tray instead. The tray menu restores the window or quits through the normal
shutdown path. Login autostart uses Wails' per-user Windows registration and
is exposed only in the production profile.

### Control API (`internal/transport/httpapi`)

A chi router bound to `127.0.0.1:<control port>`. External clients use it to
query health and (later) drive the gateway. The desktop UI does **not** call
it; the UI uses Wails typed bindings.

### Application services (`internal/application/<feature>`)

One package per feature area (account, gateway, provider, quota, routing,
settings). Both transports call into these services; neither transport
duplicates business logic.

> Current state: `internal/application/provider` manages eleven default-enabled
> built-in providers and custom OpenAI-compatible endpoints in SQLite and is
> exposed through Wails bindings. It also provides a bounded, unauthenticated
> endpoint reachability check. `internal/application/account` manages API Key
> account metadata and protected credentials through Wails bindings. The Control
> API still exposes only `/healthz`. The gateway forwards DeepSeek,
> ChatGPT / Codex subscription, River, Atria, MiMo, StepFun, Kimi, MiniMax, and Google Chat requests; other provider adapters are pending.

账号单个创建接受空名称并生成 Key 前最多 8 个 Unicode 字符的名称；批量创建只接受提供商 ID 和 1–100 个 Keys，先完整校验，再通过 SQLite 事务统一保存元数据。任一写入失败时回滚本批元数据并清理已写入的凭证；清理失败会向调用方返回错误。自动名称包含用户明确要求的 Key 前缀，不得将名称写入日志。

前端新增账号弹窗提供单个／批量方式，批量每行一个 Key、不提供名称输入；总览和详情共享 Radix AlertDialog 删除确认，只有确认后才调用删除服务。

账号详情的卡片与列表复用时间显示、启停、删除和密钥预览组件，展示偏好保存在 Zustand。查询缓存仅存新的密钥接口返回的脱敏字符串。复制由 Wails 适配层在受控凭证回调内直接写入系统剪贴板，复制方法不返回密钥内容；解密字节在回调结束后清理。较短密钥的预览全部遮掩，凭证读取失败时提供重试，余额未接入时继续显示待接入。

账号编辑复用统一弹窗，只提交名称和可选的新 Key。替换时先写新凭证，再通过事务与原引用／更新时间条件切换数据库引用；提交失败清理新凭证，原凭证不受影响。提交成功后清理旧凭证，清理失败由 UpdateResult 的警告标志报告，避免将已更新误报为未保存。前端提交结束清理 mutation 输入状态，并刷新元数据和脱敏预览。

`internal/application/subscription` 管理 OpenAI 的 ChatGPT 订阅授权生命周期；`internal/infrastructure/codex` 通过本机官方 app-server 的 stdio 协议实现浏览器／设备码登录、取消、刷新和退出。每个账号使用固定缓存根目录下的随机 CODEX_HOME，强制 keyring，实际令牌由官方组件管理；DPAPI 引用只保存隔离配置定位。迁移 004 保留 API Key 默认值并增加安全身份元数据。重新授权先暂存新配置，成功切换后清理旧配置，失败清理保留引用／标记以便重试。Wails 事件只携带变更信号，前端订阅后补读会话状态；不会读取用户默认 Codex 登录。详见 `docs/product/SUBSCRIPTION.md` 和 ADR-003。

### Core (`internal/core/…`)

Pure Go business logic: the proxy server, request parsing, model routing,
account scheduling, quota tracking and usage collection. No HTTP framework
types, no Wails types, no database types in its public API.

当前 `internal/core/proxy` 使用标准库 `net/http` 实现 loopback 监听、启停串行化、请求校验、资源边界和普通／SSE 响应。`Upstream` 只使用 context、字节和 io.Reader 等 Go 类型，不携带凭证。`internal/application/gateway` 每次重新检查提供商与账号启用状态，在已有 UseAPIKey 回调内调用 `internal/provider/deepseek`。精确路由、同池故障切换与 OpenAI 订阅队列／平衡轮换已接入；更广泛的负载调度尚未实现。

`internal/core/routing` 提供模型名校验、精确规则匹配、顶层 model 改写和纯元数据账号选择；`internal/application/routing` 通过迁移 003 的 model_routes 表保存规则，校验具体提供商与固定账号归属，并生成同源的列表诊断和只读匹配预览。网关使用可选 RequestRouter／ModelCatalog 接口解析后检查实际模型，并在 /v1/models 动态公布可解析别名。规则保存后即时生效，异常规则不静默回退。目标标识有意保留而不设置删除外键，资源删除后显示异常，保持既有删除 API 行为。详见 `docs/product/ROUTING.md`。

DeepSeek 适配器在回调结束前关闭响应，禁止上游重定向，仅接受 HTTPS 或明确配置的本机 HTTP 端点。客户端 Header 不透传；上游错误只向客户端返回稳定代码。代理日志只记录模型、流式标记、分类和耗时，不保存提示词、回复或密钥。

桌面绑定提供 GatewayInfo／GatewayStart／GatewayStop／GatewayTest。状态变化通过类型化 `gateway:status` 事件通知前端重新读取快照，避免事件乱序覆盖真实状态；无轮询。关闭网关和退出应用都会取消在途请求。HTTP Control API 仍只提供健康检查，与代理端口、桌面绑定分离。

### Infrastructure

- `infrastructure/database`: opens SQLite via `database/sql` (pure-Go
  `modernc.org/sqlite` driver) and applies embedded SQL migrations from
  `migrations/`. The schema only changes through migrations; `schema_version`
  records what has been applied.
  启动编排在写入前调用只读历史布局识别：已知不兼容统计库隔离到原目录的 `v2` 子目录，持久目录配置不触发搬迁；已有有效 v2 库沿用，混有账号／后期迁移或未知布局时明确失败。Windows 在桌面框架初始化前也能显示失败提示，并先关闭日志文件确保诊断落盘；见 `product/UPGRADE.md`。
- `infrastructure/logger`: structured `log/slog` setup with a central
  Redactor; sinks are console, rotating file and an in-memory UI buffer.
  The UI buffer exposes bounded structured snapshots with sequence cursors.
  The Wails adapter coalesces `logs:updated` notifications; the logs page reads
  deltas through bindings and handles pause, filtering and cache eviction.
  See `docs/product/LOGS.md` for export and redaction boundaries.
- Credential storage: per-user DPAPI-protected files under each Windows
  profile's data directory (ADR-002). The database stores only a
  `credential_ref`, never a plaintext secret.

### Runtime profile (`internal/platform/profile`)

Production, development and test profiles resolve their own app ID, data
directory, proxy port, control port and log directory. Dev and production
never share a data directory. The active profile comes from the build tag
(`production` for `wails3 build`) and can be overridden with
`OMNIPROXY_PROFILE`.

## Data Flow

Request path (target architecture):

```text
AI client
  → HTTP Gateway (core/proxy)
  → Request parser
  → Model router (core/routing)
  → Account scheduler (core/scheduler)
  → Provider adapter (provider/<name>)
  → HTTP transport
  → AI provider
```

Response path:

```text
AI provider
  → Provider adapter
  → Response transformer
  → Usage collector (core/usage)
  → client
```

UI path:

```text
React
  → Wails typed binding (frontend/bindings, generated)
  → Application service
  → Core / Infrastructure
```

External client path:

```text
CLI / agent / MCP
  → HTTP Control API (loopback)
  → Application service (same services as the bindings)
  → Core / Infrastructure
```

Real-time updates reach the UI through Wails events (e.g. `proxy:started`,
`request:completed`, `quota:updated`), not through polling.

## External Dependencies

- AI providers: OpenAI, Anthropic, Google, DeepSeek, Zhipu AI, Moonshot AI/Kimi,
  MiniMax, Xiaomi MiMo, StepFun, Atria, River, and OpenAI-compatible endpoints. Eleven named
  provider kinds are persisted and enabled by default, with user-controlled
  switches. Custom OpenAI-compatible endpoints can be added separately;
  request adapters other than DeepSeek, ChatGPT / Codex, River, Atria, MiMo, StepFun, Kimi, MiniMax, and Google Chat are pending.
- Local AI clients whose configs are managed: Codex, Claude Code, Gemini CLI,
  OpenCode.
- GitHub Releases + Wails updater for application updates.
- Windows DPAPI for local credential protection; Keychain / Secret Service
  adapters may be added when those platforms are supported.

## Architecture Boundaries

1. Core does not depend on Wails.
2. Core does not depend on React.
3. The UI never touches SQLite directly.
4. The UI never touches the credential store.
5. Providers never render UI.
6. The HTTP API and the Wails bindings share the same application services.
7. Business logic never lives inside React components.
8. Credentials never reach logs (central Redactor).
9. Real-time state uses events, not polling.
10. New providers are added as adapters, not by modifying the gateway core.

These boundaries are enforced by review; several are additionally enforced by
package layout (`internal/…` import rules).
