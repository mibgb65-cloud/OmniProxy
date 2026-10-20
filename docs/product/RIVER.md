# River

River 是内置 API Key 服务商，默认启用；已有安装通过迁移 029 追加，不重置其他服务商和账号配置。默认入口为 `https://api.river.ai/v1`，使用 `Authorization: Bearer`，密钥沿用 DPAPI 保护及凭证回调。

在「账号」选择 River 并添加 API Key 后，路由页和客户端目录自动提供下列官方预设。可在路由页选择 River 账号「同步官方模型」，从 `GET /v1/models` 获取账号目录；只开放已核对的模型，未知模型不会自动进入网关。预设没有查询时间，不代表账号已认证或有额度。成功发现替换预设，失败保留上次成功结果；换 Key 后旧发现失效。

| 精确模型 ID | 上下文（输入＋输出） | 最大输出 | 已确认推理档位 |
| --- | --- | --- | --- |
| `deepseek-ai/DeepSeek-V4.1-Flash` | 1048576 | 384000 | none／high／max |
| `deepseek-ai/DeepSeek-V4.1-Flash-1017` | 1048576 | 384000 | 官方控制参数表未单列，使用上游默认 |
| `nvidia/NVIDIA-Nemotron-3.5-Super-VL-120B-A12B-NVFP4` | 1048576 | 131072（官方 128K） | 未公布，使用上游默认 |
| `zai-org/GLM-5.3-Flash` | 1048576 | 131072（官方 128K） | low／high／max，不能关闭思考 |

公共目录使用 `deepseek-flash`、`NVIDIA-Nemotron-3.5` 和 `GLM-5.3-Flash`，其中 Flash 与直连 DeepSeek 合并为同款模型，在路由页保存服务商优先级。选择 River 后才还原上表的完整官方 ID；选择直连后发送 `deepseek-flash`。优先级只选择配置可用的候选，不新增请求失败重试。智谱直连尚未接入，不发布虚构 GLM 候选。旧 River 完整 ID、精确规则、固定账号、Claude `[1m]` 和 Desktop 摘要仍兼容解析，公共菜单不重复发布旧 ID；旧优先级通过迁移 030 归入逻辑模型，已有逻辑配置优先。也可配置自定义别名或固定账号。停用服务商／账号后，不在可调用目录中发布对应模型。

River 原版 Flash 显示为 `DeepSeek-V4.1-Flash`，保留 `deepseek-flash` 公共 ID 和既有优先级。1017 版本单独使用 `DeepSeek-V4.1-Flash-1017` 公共 ID／显示名，只映射到其完整官方 ID，不合并原版或回退到直连 DeepSeek。Nemotron 和 GLM 的显示与映射不变。已同步过的账号仍以最近成功的官方目录为准；需要再次「同步官方模型」才能纳入新版本。

Chat Completions 与 Responses 使用原生 HTTP JSON／SSE，保留思考、函数工具调用、工具结果及官方 usage。Chat 流式请求添加 `stream_options.include_usage`。Responses 无状态，客户端必须发送完整历史；拒绝 `store: true`、`background: true`、`previous_response_id`、`conversation` 和上下文管理引用，不静默丢失历史。仅支持函数工具，不开放托管搜索、计算机等专用工具。

Messages 使用现有文本／函数工具协议转换，经 Chat 接入。工具调用 ID 和结果保持关联；跨协议先收完整响应，再生成下游 JSON／SSE，不提供原生首 Token 增量输出，也不保证保留跨协议的思考正文。不能将其当作 River 官方 Messages API。

首版沿用本项目文本输入链路，目录只标记文本。River 官方四款模型具有图片能力，但本次未接入图片、音频或文件输入。现有本地请求体限制、超时与并发继续生效；1M 官方上下文元数据不代表完成长输入或最大输出实测。没有余额／套餐查询，不显示推算剩余额度；不增加自动重试、故障切换或官方上游 WebSocket。

## `unsupported_content` 排查

该错误不一定是账号额度不足。当前 River Responses 适配器会在本地拒绝图片、非函数
工具及不支持的服务端会话引用。尤其是在同一聊天里从 GPT 切换到 River 时，客户端可能
继续发送之前上传的图片；即使最新一条消息只有文字，历史中的 `input_image` 仍会触发
400 / `unsupported_content`。这是当前 OmniProxy 适配范围，不代表 River 官方模型没有视觉能力。

可先新建不含图片的纯文本聊天，使用受支持的函数工具和完整历史配置。仅重发文字或继续
原聊天不能保证移除旧图片；如仍失败，需要继续核对工具和会话参数，而不是反复换账号。
网关不会通过静默删除图片、工具或历史来伪装请求成功。

核对日期：初版 2026-10-17，四款模型及 1017 独立版本于 2026-10-20 再次核对。依据 [官方模型与限制](https://docs.river.ai/guides/inference-models/)、[推理接入](https://docs.river.ai/guides/inference/)、[API 控制参数](https://docs.river.ai/guides/inference-controls/)、[函数工具](https://docs.river.ai/guides/inference-tools/) 与 [超时及错误](https://docs.river.ai/guides/inference-limits/)。尚未使用真实 River Key 验证账号权限、计费、客户端或长窗口。
