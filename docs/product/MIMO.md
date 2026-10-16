# 小米 MiMo 普通 API 与 Token Plan 接入

## 使用

提供商总览卡片同时展示 API Key 和可展开的 Token Plan 标识。套餐浮层列出中国／新加坡／欧洲的 OpenAI 与 Anthropic 兼容基础地址、个人／团队 Key 说明及套餐模型范围；打开明细不会撑高卡片。卡片上的常驻地址明确标为「按量入口」，连通性按钮为「测试按量入口」；实际套餐转发仍由账号所选集群决定。

账号详情页在「卡片／列表」切换左侧提供设置按钮，与 OpenAI 使用同款入口和局部浮层。面板中的「默认使用顺序」可选 `Token Plan → API Key（按量）` 或 `API Key（按量） → Token Plan`，初始 Token Plan 优先，选择后自动保存并对下一次请求生效。支持关闭按钮、再次点击入口、Escape 和外部点击关闭；读取失败可重试，保存失败保留原顺序。

1. 在账号页选择已有的 Xiaomi MiMo 提供商，选择「普通按量 API」或「Token Plan」。套餐账号选择控制台指定的中国／新加坡／欧洲集群，填入个人 `tp-` 或团队 `ttp-` 专属 Key；普通按量使用原普通 Key。单个、批量与编辑均支持，批量共用计费／集群选项。凭证沿用受保护存储，不写入数据库或日志。
2. 自动显示明确标注「官方预设 · 账号访问权限待验证」的对话模型。可在路由页的模型目录选择该账号，点击「同步官方模型」，查询实际 `/v1/models`。
3. 默认映射自动使用同名模型，也可创建精确别名、指定账号和服务商顺序。
4. 启动网关，通过现有客户端配置功能接入 Codex 或 Claude。Claude 的 `[1m]` 选项通过已确认上下文目录解析后，网关恢复真实模型 ID。

官方模型目录可见不等于账号具备推理权限或余额。请求记录及用量页显示实际调用结果与官方 usage；预设不伪造发现时间或真实调用验证。

## 官方对话模型

- `mimo-v2.6-pro`
- `mimo-v2.6-flash`
- `mimo-v2.6-pro-ultraspeed`

Token Plan 当前仅列入 `mimo-v2.6-pro`、`mimo-v2.6-flash`；高速版不属于已确认的套餐权益。个人和团队 Key 使用同一集群及原生协议适配，团队席位的共享／余额不由 Key 数量推算。

保留大小写准确的 API 模型 ID，使用官方 1M 上下文及 131072 输出上限。按用户确认的可用模型范围，V2.5 系列已从内置预设和发现过滤白名单移除；历史精确规则和客户端配置保留，实际可用性以账号同步和调用结果为准。未核对的新模型、ASR 和 TTS 不进入对话目录。首版网关只公布文本输入能力，不宣称已接入图像／音频／视频。

## 原生协议

| 本机协议 | MiMo 上游请求路径 |
| --- | --- |
| Chat Completions | `/v1/chat/completions` |
| Responses | `/v1/responses` |
| Anthropic Messages | `/anthropic/v1/messages` |
| 账号模型同步 | `/v1/models` |

普通入口为 `https://api.xiaomimimo.com/v1`，认证使用 `api-key`。Messages 是独立前缀，不能追加成 `/v1/messages`。只允许官方按量 HTTPS 域名或明确配置的回环测试／本机代理地址，拒绝 URL 账号、查询串、片段、异常端口与重定向；不透传客户端认证头。

### Token Plan 集群

| 集群 | OpenAI 兼容基础地址 | Messages 完整路径 |
| --- | --- | --- |
| 中国 cn | `https://token-plan-cn.xiaomimimo.com/v1` | `https://token-plan-cn.xiaomimimo.com/anthropic/v1/messages` |
| 新加坡 sgp | `https://token-plan-sgp.xiaomimimo.com/v1` | `https://token-plan-sgp.xiaomimimo.com/anthropic/v1/messages` |
| 欧洲 ams | `https://token-plan-ams.xiaomimimo.com/v1` | `https://token-plan-ams.xiaomimimo.com/anthropic/v1/messages` |

Chat Completions／Responses／模型发现分别使用基础地址后的 `/chat/completions`、`/responses`、`/models`；Messages 使用独立 `/anthropic/v1/messages`。套餐由账号结构化选项解析精确官方主机，不使用提供商的按量默认地址，也不自动探测或在请求失败后跨集群／计费通道重试。普通通道拒绝套餐 Key，套餐拒绝普通 Key。实际套餐集群以官方控制台显示为准。

JSON／SSE 原样返回，思考内容、工具调用关联和增量事件不经跨协议转换；官方用量由既有观察器记录。Chat 流式请求合并 `stream_options.include_usage=true`，兼容 `max_tokens` 到官方 `max_completion_tokens`。输出参数、思考开关、消息／工具类型及整体请求大小均受校验。

沿用网关正文上限：Chat／Responses 为 2 MiB，Messages 为 16 MiB。目录中的 1M 是官方模型规格，不是本次账号的长输入实测结果；实际可传 Token 数受正文编码和工具定义大小影响。

MiMo 当前只区分关闭／开启思考，目录提供 `none`／`medium`。Responses 其他合法推理级别行为相同；Claude Messages 的 `adaptive` 归一为 `enabled`，不把预算映射为不存在的强度档位。历史 `reasoning_content`、`thinking` 和函数调用输出必须由客户端保留，网关不会虚构或缓存缺失的历史。

Responses 不支持上游持久响应引用、后台任务和 context_management。`store:false`／`background:false` 从转发正文移除；非空 `previous_response_id`、启用存储／后台或 context_management 明确拒绝。仅支持函数工具，未接入托管搜索工具与自定义自由格式工具。

## 目录与安全边界

模型发现有 45 秒期限、128 KiB 正文和 1000 个模型限制，去重并筛选已核对的对话模型。成功发现后按该账号结果限制路由，不以预设补回已消失模型。失败保留当前凭据版本的上次快照；更换 Key 后旧快照失效，恢复为未验证预设。

### 配置版本与通道路由

迁移 025 保存账号级 MiMo 计费方式、套餐集群和版本。旧官方与可识别回环按量地址保持按量，未知地址提示确认；不读取 Key 猜套餐。修改集群或计费方式清除旧目录并递增通道版本；替换 Key 使原凭据快照失效，重命名保留通道版本。目录同步和网关转发通过账号版本受控读取 Key，同步写回前复核版本。

同时存在多个计费通道／集群时，裸模型名按默认顺序选择：先遵循服务商优先级，每个服务商内优先选择首选计费方式的已启用且支持该模型账号；没有候选时选择另一计费方式。同一通道及多个套餐集群沿用账号创建时间、ID 的稳定顺序；例如高速版仅普通按量支持，即使 Token Plan 优先也选择普通按量。额度未知不会被推断为满额或耗尽，请求发出后的失败不换账号／通道重试。

公共模型列表、网关 `/v1/models` 及受管客户端目录仅发布一次原模型名，不再自动列出各个 MiMo 通道副本。旧 `mimo-api/mimo-v2.6-pro`、`mimo-cn-plan/mimo-v2.6-pro`、`mimo-sgp-plan/mimo-v2.6-pro`、`mimo-ams-plan/mimo-v2.6-pro` 继续兼容解析，包括旧 Claude `[1m]`／桌面编码 ID；用户显式创建的别名继续发布。固定账号及显式通道优先于默认顺序，通道名只在本机解析，上游仍接收原模型 ID。设置保存沿用已有受管目录刷新，客户端可能需要重新加载目录；不修改凭据或集群，也不改变已选定目标的在途请求。请求历史记录当次 `billingMode` 和区域，不因账号后续编辑而改变。

### 套餐资源与验证边界

账号卡片／列表、配额页显示所选通道和集群。Token Plan 官方剩余 Credits 及有效期保持未知，提供固定官方套餐页面入口；不会伪造零值、满额、周期，或把本机 Token 用量当作官方剩余。配额概况将其列为待确认，不发起未验证的额度 RPC。普通按量余额同样未接入，现金余额和套餐额度不互通。

未接入额外网页登录／Cookie 导入、官方余额／配额查询、ASR／TTS、上游 WebSocket、请求失败重试或自动切换。无新增 SDK；客户端配置仍通过现有预览／应用流程接入。契约与回归测试使用伪 Key 和临时数据库，真实套餐 Key、真实地区权限、真实长输入和原生桌面显示仍待验收。

## 官方来源

- [Chat Completions](https://mimo.mi.com/docs/zh-CN/api/chat/openai-api)
- [Responses 与兼容限制](https://mimo.mi.com/docs/zh-CN/api/chat/responses)
- [Anthropic Messages](https://mimo.mi.com/docs/zh-CN/api/chat/anthropic-api)
- [模型发现](https://mimo.mi.com/docs/zh-CN/api/model/list-models)
- [思考与采样参数](https://mimo.mi.com/docs/zh-CN/api/guidance/model-hyperparameters)
- [MiMo V2.6 Pro 规格](https://mimo.mi.com/models/zh-CN/mimo-v2.6-pro)
- [Key 类型与基本入口](https://mimo.mi.com/docs/zh-CN/quick-start/summary/first-api-call)
- [团队版及三个 Token Plan 集群](https://mimo.mi.com/docs/zh-CN/tokenplan/Token%20Plan/team)
- [官方 OpenCode 区域配置](https://github.com/XiaomiMiMo/awesome-mimo-agent/blob/main/docs/opencode.md)
- [API Key 额度接口功能请求](https://github.com/XiaomiMiMo/MiMo-Code/issues/2495)（不能作为接口已发布的依据）

相关任务：TASK-20261014-011、TASK-20261015-007、TASK-20261016-003；默认顺序见 ADR-010。Token Plan 协议核对：2026-10-15。
