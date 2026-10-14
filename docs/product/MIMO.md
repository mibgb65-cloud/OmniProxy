# 小米 MiMo 普通 API Key 接入

## 使用

1. 在账号页选择已有的 Xiaomi MiMo 提供商，添加普通按量 API Key。凭证沿用受保护存储，不写入数据库或日志。
2. 自动显示明确标注「官方预设 · 账号访问权限待验证」的对话模型。可在路由页的模型目录选择该账号，点击「同步官方模型」，查询实际 `/v1/models`。
3. 默认映射自动使用同名模型，也可创建精确别名、指定账号和服务商顺序。
4. 启动网关，通过现有客户端配置功能接入 Codex 或 Claude。Claude 的 `[1m]` 选项通过已确认上下文目录解析后，网关恢复真实模型 ID。

官方模型目录可见不等于账号具备推理权限或余额。请求记录及用量页显示实际调用结果与官方 usage；预设不伪造发现时间或真实调用验证。

## 官方对话模型

- `mimo-v2.6-pro`
- `mimo-v2.6-flash`
- `mimo-v2.6-pro-ultraspeed`
- `mimo-v2.5-pro`
- `mimo-v2.5`

保留大小写准确的 API 模型 ID，使用官方 1M 上下文及 131072 输出上限。V2.5 系列在官方超参文档标注即将下线，实际可用性以账号同步和调用结果为准。未核对的新模型、ASR 和 TTS 不进入对话目录。首版网关只公布文本输入能力，不宣称已接入图像／音频／视频。

## 原生协议

| 本机协议 | MiMo 上游请求路径 |
| --- | --- |
| Chat Completions | `/v1/chat/completions` |
| Responses | `/v1/responses` |
| Anthropic Messages | `/anthropic/v1/messages` |
| 账号模型同步 | `/v1/models` |

普通入口为 `https://api.xiaomimimo.com/v1`，认证使用 `api-key`。Messages 是独立前缀，不能追加成 `/v1/messages`。只允许官方按量 HTTPS 域名或明确配置的回环测试／本机代理地址，拒绝 URL 账号、查询串、片段、异常端口与重定向；不透传客户端认证头。

JSON／SSE 原样返回，思考内容、工具调用关联和增量事件不经跨协议转换；官方用量由既有观察器记录。Chat 流式请求合并 `stream_options.include_usage=true`，兼容 `max_tokens` 到官方 `max_completion_tokens`。输出参数、思考开关、消息／工具类型及整体请求大小均受校验。

沿用网关正文上限：Chat／Responses 为 2 MiB，Messages 为 16 MiB。目录中的 1M 是官方模型规格，不是本次账号的长输入实测结果；实际可传 Token 数受正文编码和工具定义大小影响。

MiMo 当前只区分关闭／开启思考，目录提供 `none`／`medium`。Responses 其他合法推理级别行为相同；Claude Messages 的 `adaptive` 归一为 `enabled`，不把预算映射为不存在的强度档位。历史 `reasoning_content`、`thinking` 和函数调用输出必须由客户端保留，网关不会虚构或缓存缺失的历史。

Responses 不支持上游持久响应引用、后台任务和 context_management。`store:false`／`background:false` 从转发正文移除；非空 `previous_response_id`、启用存储／后台或 context_management 明确拒绝。仅支持函数工具，未接入托管搜索工具与自定义自由格式工具。

## 目录与安全边界

模型发现有 45 秒期限、128 KiB 正文和 1000 个模型限制，去重并筛选已核对的对话模型。成功发现后按该账号结果限制路由，不以预设补回已消失模型。失败保留当前凭据版本的上次快照；更换 Key 后旧快照失效，恢复为未验证预设。

本次不接入 Token Plan、余额／配额查询、ASR／TTS、上游 WebSocket、请求失败重试或自动切换。没有新增 SDK 或数据库迁移；不改写既有客户端配置，用户仍通过预览／应用流程授权接入。

## 官方来源

- [Chat Completions](https://mimo.mi.com/docs/zh-CN/api/chat/openai-api)
- [Responses 与兼容限制](https://mimo.mi.com/docs/zh-CN/api/chat/responses)
- [Anthropic Messages](https://mimo.mi.com/docs/zh-CN/api/chat/anthropic-api)
- [模型发现](https://mimo.mi.com/docs/zh-CN/api/model/list-models)
- [思考与采样参数](https://mimo.mi.com/docs/zh-CN/api/guidance/model-hyperparameters)
- [MiMo V2.6 Pro 规格](https://mimo.mi.com/models/zh-CN/mimo-v2.6-pro)

相关任务：TASK-20261014-011。
