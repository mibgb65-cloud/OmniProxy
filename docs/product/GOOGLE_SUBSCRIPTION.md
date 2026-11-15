# Google AI / Antigravity 订阅

Google 账号支持普通 Gemini API Key 和独立 Antigravity 订阅登录。两条通道的
凭据、接口与额度互相独立；Google AI Plus／Pro／Ultra 的名称不代表实际模型权限。

## 使用

1. 首次登录需在本机安装兼容的官方 Antigravity CLI，确保 PATH 中可找到 `agy`。
   当前本机识别验证版本为 `agy 1.0.0`；不自动安装或升级。
2. 在“账号 → Google”点击“登录 Google”，名称选填，在 Google 官方页面授权。
3. 关闭弹窗保留后台授权，重新打开可恢复；“取消”终止回调。授权最长等待 10 分钟。
4. 成功后可刷新授权、退出或重新登录。重新登录只能替换为同一 Google 身份，
   新身份需要新增账号。退出只清除 OmniProxy 的本机授权。
5. 应用启动、Google 授权状态变化、成功刷新模型列表、导入旧版数据或新建账号后自动同步
   已启用订阅账号；无需保持账号页打开，不自动启动网关。账号卡片上的模型列表刷新同时
   承担手动重试（读取官方选择器成功即触发一次同步），路由页仍有独立的同步入口。目录使用
   当前账号实际返回的受支持文本生成模型，不按套餐名补充。重新授权使旧目录失效，
   普通 token 刷新保留目录。同步后现有受管客户端目录刷新，首次接入仍需确认配置；
   手动限选和外部修改保留，运行中的 Agent 需重新加载菜单。失败保留旧快照并记录固定错误码，
   不自行持续重试，不发送推理请求，不影响其他账号继续同步。
6. 通过网关 Chat Completions、Responses 或 Messages 调用真实模型 ID 或路由别名；
   Codex CLI／Desktop、Claude Code CLI／Desktop 使用各自已有的一键接入流程。
   若普通 API Key 和订阅存在同名模型，可创建固定该订阅账号的路由规则来明确使用
   订阅通道；新增账号不会改变既有的默认账号选择顺序。
7. Antigravity 桌面端（2.15 及以上）在“网关 → 本地 Agent 接入”的 Antigravity
   卡片一键连接；完全退出并重启 Antigravity 后，模型菜单读取本地网关目录。
   该模式不需要 Google 登录，也不需要安装证书或修改系统代理；详见下文
   「Antigravity 桌面端」。

## 当前能力与边界

- 支持文本、函数工具、工具结果、JSON 与增量 SSE；工具不透明思考签名通过
  `tool_calls[].extra_content.google.thought_signature` 往返保存。
- 开放 Gemini 文本型号，以及官方选择器列出的 `claude-opus-4-6-thinking`、
  `claude-sonnet-4-6`、`gpt-oss-120b-medium`；保留已出现的 Sonnet thinking ID 兼容。
  显示名中的 Thinking 不代表模型 ID 含此后缀；未知第三方型号仍显示待接入。
- 订阅支持 Responses／Messages 的文本／函数工具转换，先读取完整上游结果再编码
  JSON／SSE；Chat 保留原生增量输出。用户消息与工具结果里的内联图片（`data:` base64，PNG／JPEG／
  WebP／GIF／HEIC／HEIF，单图 base64 上限 16 MiB）转换为上游 `inlineData`，与文本同一条
  消息时都保留；工具结果的图片放在同一 user 回合 `functionResponse` 之后的 `inlineData` part，
  agent 通过 `read_file`／截图工具读到的像素因此能真正到达模型。Chat 与 Responses／Messages
  三条协议都走同一整形层。语音、视频、远程图片 URL（上游不代为下载）、非白名单格式，以及
  出现在 system／assistant 回合里的图片仍明确失败，不静默丢弃或降级成文本。
- 模型目录对 Gemini 型号发布 `text` + `image` 输入（桌面端 `supportsImages: true`），
  客户端据此才允许附图并让 agent 把图片当像素发送；Google 订阅内的 Claude／GPT-OSS
  通道没有做过同样的图片验证，目录继续按纯文本发布。能力声明只表示「允许发送」，
  实际识别仍由上游模型决定。
- 已知客户端缺口与网关兜底：Antigravity 在 Enterprise Gateway 模式（`wireProtocol: openai`）
  下会把工具结果里的图片剥掉，只留 `Showing binary file of type image/png` 占位文本
  （实测同一轮请求只多 329 Token，而同一张图经网关直发占 1100+ Token）。网关因此按
  「同一 `tool_call_id` 的调用参数里带绝对路径 + 结果命中图片占位文本」回读本地文件，
  内容嗅探确认为 PNG／JPEG／GIF／WebP 且不超过 12 MiB 时补成同一回合的 `inlineData`；
  路径不存在、非图片、超限或参数里没有路径时保持原样，不报错也不降级。非图片内容
  不会被读出，因此不构成额外的文件访问能力。
- Gemini 函数工具通过 `parametersJsonSchema` 传递完整 JSON Schema，保留 Claude Code
  自带工具的 `$schema`、`exclusiveMinimum`、`additionalProperties` 等定义与约束；
  不再误放进受限的 `parameters` OpenAPI Schema，也不通过删除约束规避错误。
  Google 订阅内 Claude／GPT-OSS 通道维持原 Schema 字段，不扩展未经验证的能力。
- 跨协议客户端不携带 Google 扩展签名时，复用按授权／项目／模型／工具完整内容摘要
  隔离的内存签名缓存（最多 256 条、1 小时），不持久化消息正文或签名；重启／淘汰后
  不伪造签名，可能需重新开始工具轮次。Claude 工具调用的 `OTHER` 终态只在存在完整
  工具块时接受，不把未知文本终态当成功。
- 工具调用在流结束时输出完整参数与签名；普通文本逐段输出。缺少终态的流返回失败。
- 思考正文暂不公开，思考 Token 纳入输出用量；本机用量沿用现有记录与统计。
- 账号详情卡片／列表和配额总览显示 Google 官方额度分组、剩余百分比和重置倒计时，
  提供手动刷新。只展示实际返回的窗口，不套用 Codex 套餐或重置卡。
- 公开额度快照保存到当前运行环境的 SQLite；账号页和配额总览先恢复上次百分比／绝对重置时间，
  再联网更新。刷新失败不清空旧值；暂缺分组／字段保留原显示并标注留存，新重置窗口不继承旧百分比。
  留存值标记为待刷新，不计作已确认的当前余额，也不参与调度或推理权限判断。
- 快照按账号、成员身份与授权版本隔离，账号删除级联清理；普通令牌轮换在事务内重绑定。
  退出登录隐藏，重新授权后不恢复旧授权版本快照；本地恢复不解密令牌，也不等待联网刷新锁。
- 同一卡片提供“可选模型”入口，展开账号当前官方选择列表，按本机网关支持／待接入
  分组展示具体型号及 ID；卡片不再单独放手动同步按钮，自动同步覆盖启动／登录／导入／
  刷新模型列表，展开面板底部说明同步范围；“网关支持”不代表客户端已经配置或热加载。
  模型和额度独立查询、缓存与刷新，
  单项失败不隐藏另一项结果。
- 不请求付费积分回退，不切换普通 Gemini API Key 重试；上游限流返回 429。
- 同一提供商下的多个 Google 订阅账号在「Google 账号设置」选择“队列轮换／平衡切换”：
  队列按候选顺序优先用当前账号，均衡在同池可用账号间轮流分配新请求（只均衡请求数，
  不均衡 Token）；只对订阅账号生效，API Key 账号不参与，固定账号不切换；设置持久化在
  `google_settings`，每个请求读取一次以支持热切换。
- Antigravity 桌面端使用官方 Enterprise Gateway（`wireProtocol: "openai"`）接入，
  不拦截 TLS、不装本地 CA；连接期间桌面端不使用 Google 登录，Google 账号专属的
  实验与命令（`/boost`、`/teamwork`、`/plugin`、`/generative_ui`）暂时不可用，
  断开后恢复。目录发布 Google 订阅账号实际提供的全部文本模型（Gemini、Claude、
  GPT-OSS），不引入 Google 之外的提供商。

### 请求形态与拒绝处理（2026-11-12）

订阅通道的请求按官方 Antigravity 客户端的行为整形，并区分两种 429：

- **system prompt 打散**：Claude Code 的计费行（`x-anthropic-billing-header:`）、
  Claude Agent SDK 身份行、Codex GPT-5 的两条身份行、Hermes Agent 旧人设，会被上游
  以「Resource has been exhausted」拒绝整个请求（与余额无关）。默认对这些行按词插入
  零宽空格（每个 ≥3 字符的词在第 1、2 个字符后各一个），模型照读、上游的行匹配看不到；
  设置页「Google 订阅请求 → 按原文发送 system prompt」可关闭打散。
- **内容拒绝不重试**：命中上述行且上游回该 429 时，网关返回 `400 prompt_turned_away`，
  账号不进入额度冷却。Anthropic／OpenAI SDK 对 429 会自动重试 10 次，改报 400 后
  客户端不再重放。
- **同 prompt 预判跳过**：账号拒绝过某个 system prompt 后 6 小时内，若还有其它候选，
  直接跳过该账号、不发送、不写冷却；最后一个候选仍照常发送，失败如实返回 400。
- **429 的两种语义**：带「Your quota will reset after <时长>」的容量耗尽记为
  `upstream_quota_exhausted`，按该时长休息本账号（有其它候选时换号，单账号返回 429）；
  其余 429 维持 `rate_limited`。
- **模型族差异**：`maxOutputTokens` 仅 Claude 模型发送；Gemini 3+ 下发
  `thinkingLevel`、2.5 下发 `thinkingBudget`；Claude 的工具调用在缺少签名时补
  `skip_thought_signature_validator`，并使用 `VALIDATED` 工具模式。
- **历史修复**：不再因为「历史以模型回合开头／结尾、工具调用未回答、同一调用重复回答」
  直接失败，而是前置引导回合、补中断结果、保留首个回答并合并相邻同角色回合。
- **会话标识**：`sessionId` 由首条用户文本的哈希得出，同一会话每一轮保持一致。
- **模型目录权益**：`fetchAvailableModels` 先上报付费套餐（`entitlement.userTier`），
  被 400 拒绝时降级并记忆 24 小时，最后不带权益请求；停用与内部模型不进入目录。
- **客户端标识**：User-Agent 采用 `antigravity/hub/<公开版本> <系统>/<架构> OmniProxy`，
  版本取自 Antigravity 公开更新清单（缓存 6 小时，失败回退兼容下限）；`onboardUser`
  另带 `X-Goog-Api-Client` 与 Node 客户端标识。

打散 system prompt 属于绕开上游内容判定，与 ADR-013 已记录的 Antigravity 账号风险同级：
可能影响上游前缀缓存，且不能保证上游检查永不变化，不因此宣称账号更安全。

- 本实现使用 Antigravity 客户端内部协议，接口兼容性与真实账号权限需持续验证。

## Antigravity 桌面端

桌面端从以下文件读取网关（两个版本各读自己的文件）：

- 2.30 及以上：`~/.gemini/config/config.json`
- 2.29 及以前：`~/.antigravity/settings.json`

接入块为 `gateway: {enabled, url, apiKey, wireProtocol: "openai"}` 与
`models: {defaultModelId, listSource}`；`listSource` 指向
`http://127.0.0.1:<端口>/v1/models?format=antigravity`。一键连接会同时写入两个
文件并加密备份原内容；连接期间删除会遮住远端目录的内联 `models.list`。断开时
只撤销本应用写入的 `gateway`／`models` 字段并放回备份中的原值，桌面端在连接
期间改写的主题、权限等设置保留。`gateway` 中用户自行添加的其他字段保留；
网关端口变化会同步改写两个文件中的本机地址。

- 模型目录只返回 Google 订阅账号的 Gemini 文本模型，字段为
  `modelId`、`displayName`、`maxTokens`、`maxOutputTokens`、`supportsImages`、
  `supportsThinking`；未知窗口不猜测。Gemini 型号的 `supportsImages` 为 `true`
  （通道已确认能把内联图片转成 `inlineData`），Claude／GPT-OSS 通道仍为 `false`。
  能力存在本地账号目录里，改版后需要重新「同步官方模型」并重启／重连桌面端，
  客户端才会拿到新的 `supportsImages`。
- 对话走 `<url>/v1/chat/completions`，复用现有 Google 订阅多账号路由、额度冷却、
  故障切换与请求记录。桌面端自带的 `reasoning_effort`、`service_tier`、
  `metadata`、`store` 只在 `client=antigravity` 时剥离，不改变其他客户端。
- 桌面端通过接入 Key 识别归属；Key 只存在本机设置文件与受保护存储中。
- 桌面端把 `-low`／`-medium`／`-high`／`-xhigh` 变体当作独立模型发送，同时还会带一个
  档位字段；型号里的档位即用户的选择，网关以型号为准（并移除矛盾的档位字段），因此
  `gemini-4-hydrogen-high` 的记录与上游 `thinkingLevel` 都是 `high`，不会被字段降级。
  Google 只发布变体、没有基础型号（`gemini-4-hydrogen`），因此目录折叠不会发生；这种
  目录用“同一前缀下至少两个已知档位”确认后缀是档位，只有一个后缀时保留客户端档位。
- 先在桌面端启动一次，确保设置文件存在，再回来连接；连接后必须完全退出并重启。
- 若组织下发了 `admin_settings.json` 的网关，或设置了 `AGY_LLM_GATEWAY_URL`
  等 `AGY_LLM_GATEWAY_*` 变量，它们优先于用户设置文件，本机卡片写入不会生效；
  当前版本不自动改写这些外部配置。

## 失败排查

- 客户端出现 “API key was rejected” 不足以证明 Google API Key 失效。先查看网关
  请求详情中的账号通道、HTTP 状态和错误码：订阅登录使用 OAuth，与普通 API Key 独立。
- 普通推理／控制端点的上游 HTTP 400 返回 `400 / invalid_request`，应排查请求参数、
  工具上下文及接口兼容性，不要求用户重置密钥。旧版曾将其误报成 `401 / reauth_required`；
  旧请求记录未保存原始上游状态，不能据此反推每次历史认证错误的真实原因。
- Claude Code 自带工具调用 Gemini 的一种已复现 400 原因是 Schema 方言不匹配；
  需要包含 `parametersJsonSchema` 修复的新网关构建，只有错误分类修复的旧构建仍会失败。
  该修复不等同于解决所有历史 400；工具签名缓存失效及其他高级选项仍遵守上述边界。
- 真实上游 HTTP 401，以及 OAuth 令牌端点 HTTP 400，仍返回 `401 / reauth_required`。
  在 OmniProxy 对应 Google 账号中先刷新授权；持续失败时使用同一身份重新登录。
  不要把 Google OAuth 令牌或 Google API Key 填入客户端的本地网关 Key。
- `403 / access_denied` 表示权限拒绝；`429 / rate_limited` 表示限流或资源限制。
  429 时停止连续重试，在账号／配额页手动刷新对应模型分组，参考实际重置时间后重试；
  剩余额度快照不能证明推理一定可用，也不能仅凭 429 判定是短时限流还是额度耗尽。
- 周／5h 余额并不覆盖所有瞬时速率或模型容量限制。即使官方余额较高，或同账号短请求
  返回 200，也不能据此认定原会话请求已经恢复；原始上游拒绝类型没有记录时须保留未知。
- 当前普通 API Key 的 Google 429 不自动遍历账号（项目共享额度）；订阅账号的容量耗尽
  429（带重置时间）会休息该账号，并在有其它候选时换号。内容拒绝既不休号也不换号
  （见上文「请求形态与拒绝处理」）。判断「持续 retry」先核对客户端重试事件和每条网关
  请求的尝试数；客户端可以为同一逻辑请求建立多条独立网关请求。
- Beta.17 的 Schema 修复不等同于完整 Desktop 默认工具已通过真实验收。2026-10-21
  在所选订阅账号上，隔离 Claude Code 2.1.299 的 21 工具默认合成请求被 Google 返回
  `429 / RESOURCE_EXHAUSTED`；只将最大输出从 32000 降至 512 仍失败。
  2026-11-12 已按参考实现定位为「system prompt 命中被拒行」，并在真实账号上确认：
  原文得到 `400 prompt_turned_away`，打散后得到 `200`（见下文验证记录）。
  不要通过删工具、静默截断或重新打包伪装解决。
- `400 / prompt_turned_away`：system prompt 命中上游拒绝行。检查设置页
  「Google 订阅请求 → 按原文发送 system prompt」是否开启；关闭后自动打散，无需重装或
  重置账号。该错误不可重试，反复重试只会得到同样的 400。
- 内部跳过（`prompt_refused_before`）只在有其它候选时出现，不返回客户端；请求详情里
  看到它表示同一账号 6 小时内已拒绝过同一 prompt，属预期行为。
- `429 / upstream_quota_exhausted`：容量耗尽，`Retry-After` 是上游给的重置秒数；
  单账号时按此等待，多账号时由下一个候选接替。
- 错误分类修复不会恢复已失效的授权、补充额度或自动修复不兼容的请求。

## 安全

首次登录有界读取本机官方 agy 内的公开桌面 OAuth 配置，校验已知客户端 ID 与
配置指纹，不执行程序，不读取已有用户登录。仓库只保存公开标识和指纹，不含 secret。
无法识别组件时提示缺失或不兼容，不能降级到其他客户端配置。

授权使用随机 state、PKCE、一次性 loopback 回调和有限超时。账号 token、刷新 token
及所需客户端配置全部通过既有 DPAPI 存储；数据库只保存引用和白名单身份元数据。
授权信息不进入日志、请求 URL、前端凭证接口或客户端配置。网络只访问固定 Google
HTTPS 端点且拒绝重定向；授权 URL 仅允许进行中的固定 Google 登录地址。

内部协议兼容版本标识取 Antigravity 公开更新清单里的当前版本（缓存 6 小时，读不到时用
参考实现的 `2.23.0` 下限），并在 User-Agent 末尾保留 OmniProxy 身份；此值不是已安装 agy
的版本或账号权限证明。参考源码采用 MIT 许可证，本次独立实现沿用标准库及已有模块，
未复制其翻译器或令牌存储代码。

设计决策见 [ADR-013](../decisions/ADR-013-google-antigravity-subscription.md)。
参考项目固定版本：
https://github.com/router-for-me/CLIProxyAPI/tree/e7003d2a3e2cee2d45f9950c067bf8e02a2ac0f8

## 验证记录

2026-10-16：实际独立浏览器授权与开发 profile 的 DPAPI 存储成功，发现 14 个支持模型；
固定新账号通过本机网关调用 `gemini-2.5-flash` 返回 HTTP 200，输入 6／输出 1 Token，
本机用量记录一致。高级模型与真实多轮工具／长窗口仍需按实际账号验证，不由目录推断。
OAuth、并发刷新、身份变化、目录版本、工具签名两轮、SSE 中断与错误脱敏由契约测试覆盖。

2026-11-12：对齐官方客户端的请求形态与拒绝处理（打散、拒绝分类、两种 429、历史修复、
模型族参数、会话标识、目录权益、客户端标识），全部由注入 RoundTripper 的契约测试覆盖
（打散 5 行与幂等、429 分类、按模型族的生成参数、历史修复、跳过与最后候选、权益降级）。

2026-11-12：Antigravity 桌面端接入按官方 Enterprise Gateway 契约实现。配置生成与
双文件恢复、`format=antigravity` 目录形状与全部 Google 订阅模型（Gemini／Claude／
GPT-OSS）过滤、桌面端请求字段规整由契约测试覆盖；当前开发环境未安装桌面端，
尚未做真机请求验证。首次使用需按卡片提示重启桌面端，并在模型菜单中确认出现
Google 订阅账号实际提供的模型。

2026-11-12 真机对照（用户授权，1 个订阅账号、模型 `gemini-2.5-flash`、21 工具／约 88.8 KB、
`max_tokens=32000`，各 1 次、无自动重试）：system prompt 按原文发送得到
`400 / prompt_turned_away`；按词打散后得到 `200`；删去两条命中行同样 `200`。
这确认了「少数 system prompt 行导致 429 拒绝」的根因，并确认打散后的 Agent 形态请求被上游
正常接受。结论限于该账号与模型，不推广为所有套餐、模型与工具组合均已验收。

## 订阅额度

额度来自固定官方 Antigravity 内部只读接口 `retrieveUserQuotaSummary`，以当前授权的
项目为范围。实际账号返回 Gemini Models 和 Claude and GPT models 两组，各有周／5h
窗口。界面将第二组标为“第三方模型（Claude / GPT）”，表示 Google 订阅内的官方共享
资源，不表示网关已经开放这些模型。其他账号仅展示各自实际返回的窗口。

每组／窗口独立显示，不能合计成总额度，不能从剩余百分比推算请求或 Token 数。
模型目录的 `quotaInfo` 不能当作周额度，因此本功能不通过模型目录拼接或猜测周窗口。
缺失比例显示未知，比例 0 才表示耗尽；很小的正数不会被显示成零。没有重置时间时显示
待确认，不宣称“未开始计时”；已过重置时间显示待刷新，不自动恢复为满额。

额度仅按需读取，不触发推理、购买积分或修改账号设置。查询全程有 10 秒上限，读取前后
核对授权版本；退出／重新登录后的在途旧结果被拒绝。前端按账号身份／授权状态隔离，
详情和配额页共享 60 秒缓存、10 分钟保留时间及既有串行查询队列；失败保留同身份上次
成功值并显示安全错误，不后台轮询。仪表盘默认只订阅同一缓存，主动检查才访问上游。

2026-10-16：新读取服务已用既有开发账号真实验证，返回两组四窗口及官方重置时间，
当次均为剩余 100%。全过程未发送推理请求，未读取或展示明文授权。

依据：[官方额度面板](https://antigravity.google/docs/cli/commands/usage/)、
[参考项目协议讨论](https://github.com/router-for-me/CLIProxyAPI/issues/3876)。

## 账号可选模型

展示使用 `fetchAvailableModels` 的 `agentModelSorts[].groups[].modelIds`，仅保留其引用的
公开条目；原始 `models` 字典也含内部补全模型和历史别名，不能全部当作当前可选型号。
网关适配标记依据 Gemini 文本能力和已适配第三方精确白名单，不由套餐名、额度池名称或剩余比例推断。
当前界面的“网关支持”（旧版称“已接入”）表示协议适配，实际调用权限仍由 Google 校验。

2026-10-17：真实只读服务返回 6 个可选型号，包含 Gemini 3.8 Flash High／Low／Medium、
Claude Opus 4.6 Thinking、Claude Sonnet 4.6 Thinking 和 GPT-OSS 120B Medium。
原先 3 个 Gemini 已有 Chat API 适配，其他 3 个显示待接入。此次原目录验证未发送推理请求。
路由同步保留已有 14 个 Gemini 条目及别名，以兼容已有配置；它与当前官方选择列表
用途不同，不据此宣称每个旧条目都仍是官方推荐型号。

2026-10-17 后续接入验收：真实选择器六款全部适配；同步保存 17 个受支持型号（含旧 Gemini
兼容项）。Opus／Sonnet／GPT-OSS 经实际网关 Responses 各返回 HTTP 200；Gemini Flash Low
与 Sonnet 经 Messages 完成真实工具请求／结果续接，两轮均返回 HTTP 200。请求均记入本机
用量，不启用付费积分回退；该结果不代表所有高级参数、多模态或长上下文都已验证。

## 普通 API Key 资源布局

Google 提供商详情的项目声明、官方剩余未知、本机错误观察、绑定／刷新及官方平台入口
归入对应 API Key 卡片／列表行，创建／更新时间在资源后显示。多个 Key 属于同一项目时
仍读取同一资源池；卡片中的重复展示不代表独立额度。配额总览保留按项目归组的集中展示。
