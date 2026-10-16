# Kimi 普通 API 与 Kimi Code 订阅 Key

沿用 `builtin_kimi` 内置提供商与图标，账号可选择普通按量 API／Kimi Code 订阅及国内／海外平台；新账号默认国内普通 API，单个／批量／编辑均支持。Key 保存在受保护凭证存储中，不进入数据库、日志或客户端目录。订阅接入使用用户在官方控制台创建的专用 Key，不包含 OAuth 登录。

| 平台 | OpenAI 风格基础地址 | Messages 完整路径 |
| --- | --- | --- |
| 国内 | `https://api.moonshot.cn/v1` | `https://api.moonshot.cn/anthropic/v1/messages` |
| 海外 | `https://api.moonshot.ai/v1` | `https://api.moonshot.ai/anthropic/v1/messages` |

两平台的 Key 不通用。提供商卡片的默认地址只是国内预设；实际转发与发现使用所选账号的平台。遇到 401、限流、网络错误或额度不足均不会尝试另一个平台。

## Kimi Code 订阅

订阅 Chat／Responses 基础地址为国内 `https://api.kimi.com/coding/v1`、海外 `https://api.kimi.ai/coding/v1`；Messages 使用同一地址后的 `/messages`，与普通 API 的 `/anthropic/v1/messages` 不同。三种协议保持原生文本、思考／工具历史及 JSON／SSE；普通 API 的既有协议限制保持。

| 统一目录 ID | 订阅上游 ID | 说明 |
| --- | --- | --- |
| `kimi-k3` | `k3` | 与普通 API K3 共用逻辑名称；未核实会员档位时目录保守公布 256K |
| `k3-256k` | `k3-256k` | 订阅专用 256K 版本 |
| `kimi-for-coding` | `kimi-for-coding` | K2.8 Preview，独立保留，不替换普通 K2.7 Code |
| `kimi-k2.7-code-highspeed` | `kimi-for-coding-highspeed` | 同款 K2.7 Code 高速版，共用逻辑名称 |

Kimi Code 官方的 `k3`、`kimi-for-coding-highspeed` 也接受直接请求，只解析到订阅账号，不因按量优先误用普通 Key；公开目录不重复列出原生别名。`kimi-for-coding` 标称 1M、`k3-256k`／高速版标称 256K；K3 1M 权益取决于会员档位，预设不当作账号授权验证。订阅 K3／K3-256k 默认 high，K2.8 默认 max，公布 low／high／max；高速版不声明多档强度。首版仅文本和函数工具，不宣称图片／视频支持；拒绝关闭思考导致订阅上游静默换模型、持久响应引用、后台或未接入的服务端状态能力。

真实客户端 User-Agent 经过单值、长度和 ASCII 边界校验后原样转发到订阅服务，不透传客户端认证头，也不伪装 Codex／Claude。没有客户端标识时诚实标识为 OmniProxy；官方是否接受该工具及 Key 权限以上游结果为准。

订阅目录使用明确标记的预设，未确认 Key 级模型发现接口时禁止发起 `/coding/v1/models`；订阅余额同样不调用普通 `/users/me/balance`。账号卡片／列表及配额页显示订阅额度未知，并可打开固定的官方控制台。会员共享额度、加油包和真实模型权限不从 Key 数量或本机用量推断，不修改官方加油包设置。

## 当前模型和协议

| 模型 | 标称上下文 | Chat | Messages | Responses |
| --- | --- | --- | --- | --- |
| `kimi-k3` | 1048576 tokens | 原生 | 原生 | 原生 |
| `kimi-k2.7-code` | 262144 tokens | 原生 | 原生 | 未确认 |
| `kimi-k2.7-code-highspeed` | 262144 tokens | 原生 | 未确认 | 未确认 |
| `kimi-k2.6` | 262144 tokens | 原生 | 原生 | 未确认 |

Responses API 当前明确仅支持 K3。Messages 参考主要描述 K3，官方 Claude Code 集成另明确 K2.7 Code／K2.6；高速版尚未明确出现在 Messages 集成中，首版只开放其 Chat。K2.5、Moonshot V1 和其他已下线模型不列入预设或当前路由。

首版公布文本、思考及工具调用能力，不开放图片／视频／文件／音频。模型标称窗口和真实长输入验证保持区别；K3 的标称最大生成长度来自 Chat／Responses 参考，Messages 沿用网关统一校验边界。

## 思考、工具和流式

- K3 始终思考且保留历史推理；支持 `low`／`high`／`max`，默认 max。Chat 使用 `reasoning_effort`，Responses 使用 `reasoning.effort`，Messages 使用 `output_config.effort`。不把 medium／xhigh 静默改成其他档位。
- K2.7 Code 及高速版始终思考并保留历史推理，不能关闭；K2.6 支持原生 `thinking` 开关及 `keep=all`。这些模型不声明多档推理强度。
- 原生 HTTP／SSE 保留 `reasoning_content`、思考块／签名和工具调用 ID，不经过跨协议转换。K3 Responses 支持 function、namespace、官方 `apply_patch` custom grammar 及 web_search 类型；本地工具仍由客户端执行，网关不会执行模型产生的代码。
- Responses 官方为无状态接口，需回传完整历史；拒绝服务端 ID 续接、后台任务、`store=true` 或未支持的上下文管理。明确为 false 的状态字段在验证后移除。
- Chat 将弃用的 max_tokens 转为 max_completion_tokens，二者同时出现时明确报错；保持固定采样参数及各模型思考约束。流式请求开启最终 usage，缺失／截断不伪造零值或完整统计。
- Messages 的总输入包括未命中输入＋缓存读取＋缓存写入；Chat／Responses 输入已经是总数，不重复加入缓存或思考分项。复用既有用量观察和累计差值算法。

## 目录与地区路由

账号同步使用对应地区的 `/v1/models`，有界读取并校验 ID、数量和 context_length，只保留当前已支持模型。官方上下文元数据优先，但不标为真实长窗口已验证。同步失败保留旧快照；首次同步前提供标记为 preset、没有发现时间的官方目录。

账号页设置可分别保存「Kimi Code 订阅／普通 API」默认顺序和「国内／海外」默认顺序，初始订阅优先、国内优先。先遵循模型服务商优先级，每个服务商内先选择有合格账号的首选计费方式，再选择首选地区，最后按账号创建时间／ID；无合格首选候选时选择下一通道，发出请求后的失败不换通道重试。设置选择后自动持久保存，关闭重开保持；保存中的两个选项互斥，失败保留旧显示。

公开目录、网关发现及受管客户端列表只发布一次同款逻辑模型，能力来自实际选中账号。旧 `kimi-cn/kimi-k3`、`kimi-global/kimi-k3` 保持普通 API 的明确地区语义，订阅可显式使用 `kimi-code-cn/kimi-k3`／`kimi-code-global/kimi-k3`；手工规则／固定账号优先。旧 Claude 上下文和桌面编码 ID 继续在能力校验后解析，兼容目录不重新发布副本。普通 K2 系列没有原生 Responses 的账号仍不加入 Codex 目标，不因同款订阅支持而放宽。

迁移 023 保存 Kimi 平台选项与版本，扩展目录版本视图并保持 Step 版本语义。旧空地址按原海外默认保留，显式 `.cn` 保留国内；未知自定义地址不猜测，提示确认。修改平台失效模型目录，同步和转发再次检查版本；更换 Key 保持既有凭据版本隔离，重命名不改变平台身份。请求历史保留当次地区。

迁移 028 为旧账号补普通按量计费字段，保留凭据、启用状态、地区、配置版本及未变更目录；增加独立的计费／地区偏好表。切换计费方式与切换地区一样递增配置版本、清除旧目录并阻止旧在途选择使用新通道。请求记录保存实际计费及地区快照。保存设置沿用受管目录刷新，外部客户端可能需要重新加载目录。

## 验证与边界

契约测试使用伪 Key 和独立数据库，验证两地区原生调用、工具／推理历史、SSE、参数、安全边界、缓存及迁移。真实 API Key 调用、真实 Codex／Claude 工具任务与长窗口仍待账号验收。

普通 API 余额已接入，见下文。Kimi Code 订阅 Key 已接入，官方剩余额度、OAuth 登录、文件／Batch、托管智能体和独立搜索接口没有接入；不能将模型名称或预设当成订阅认证或套餐额度证明。本次验证使用伪凭据、受控上游及临时数据库，没有发送真实付费请求。

## 普通 API 余额

账号卡片／列表与配额页查询所选地区的 `GET /v1/users/me/balance`：国内固定 `api.moonshot.cn`、CNY／¥，海外固定 `api.moonshot.ai`、USD／US$。显示官方可用余额、代金券余额和现金余额，保留十进制精度；现金可为负数，可用余额小于等于零标记不足，不按现金加代金券重新计算。

复用受保护 Key 回调、10 秒超时、64 KiB 响应上限和稳定错误码，禁止重定向和跨地区回退，不使用提供商自定义地址。查询结束复核账号版本与地区，变更 Key 或地区后不接受旧响应。前端与配额页共享 60 秒缓存、10 分钟保留和最多两个并发；刷新及失败保留同一版本旧快照，首次失败不显示零。仪表盘只读缓存，主动检查才查询官方接口。此处是平台账户余额，多把 Key 不代表多个独立余额池，不做加总。

契约与页面测试使用伪凭据；真实 Key 查询及桌面显示待实际账号验收。实施记录：TASK-20261015-006。

## 官方参考

- [Kimi Code 端点／模型／客户端身份](https://www.kimi.com/code/docs/)、[订阅模型配置](https://www.kimi.com/code/docs/kimi-code/models.html)、[会员权益及加油包](https://www.kimi.com/code/docs/kimi-code/membership.html)、[订阅控制台](https://www.kimi.com/code/console)（2026-10-16 核查）

- [用户指定快速开始](https://platform.kimi.com/docs/get-api-key)
- [模型列表及下线说明](https://platform.kimi.com/docs/models)、[模型参数](https://platform.kimi.com/docs/api/models-overview)
- [国内 API 概述](https://platform.kimi.com/docs/api/overview)、[海外 API 概述](https://platform.kimi.ai/docs/api/overview)
- [Chat](https://platform.kimi.com/docs/api/chat)、[Responses](https://platform.kimi.com/docs/api/responses)、[Messages](https://platform.kimi.com/docs/api/messages)
- [目录](https://platform.kimi.com/docs/api/list-models)、[思考历史](https://platform.kimi.com/docs/guide/use-thinking-models)
- [国内余额](https://platform.kimi.com/docs/api/balance)、[海外余额](https://platform.kimi.ai/docs/api/balance)（2026-10-15 核对）
- [Codex](https://platform.kimi.com/docs/guide/codex-kimi)、[Claude Code](https://platform.kimi.com/docs/guide/claude-code-kimi)

核对日期：普通模型协议 2026-10-14，余额协议 2026-10-15，订阅 Key 及默认顺序 2026-10-16。实施记录：TASK-20261014-017、TASK-20261015-006、TASK-20261016-005。相关决策：ADR-012。
