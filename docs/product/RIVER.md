# River

River 是内置 API Key 服务商，默认启用；已有安装通过迁移 029 追加，不重置其他服务商和账号配置。默认入口为 `https://api.river.ai/v1`，使用 `Authorization: Bearer`，密钥沿用 DPAPI 保护及凭证回调。

在「账号」选择 River 并添加 API Key 后，路由页和客户端目录自动提供下列官方预设。可在路由页选择 River 账号「同步官方模型」，从 `GET /v1/models` 获取账号目录；只开放已核对的模型，未知模型不会自动进入网关。预设没有查询时间，不代表账号已认证或有额度。成功发现替换预设，失败保留上次成功结果；换 Key 后旧发现失效。

| 精确模型 ID | 上下文（输入＋输出） | 最大输出 | 已确认推理档位 |
| --- | --- | --- | --- |
| `deepseek-ai/DeepSeek-V4.1-Flash` | 1048576 | 384000 | none／high／max |
| `nvidia/NVIDIA-Nemotron-3.5-Super-VL-120B-A12B-NVFP4` | 1048576 | 131072（官方 128K） | 未公布，使用上游默认 |
| `zai-org/GLM-5.3-Flash` | 1048576 | 131072（官方 128K） | low／high／max，不能关闭思考 |

公共目录使用 `deepseek-flash`、`NVIDIA-Nemotron-3.5` 和 `GLM-5.3-Flash`，其中 Flash 与直连 DeepSeek 合并为同款模型，在路由页保存服务商优先级。选择 River 后才还原上表的完整官方 ID；选择直连后发送 `deepseek-flash`。优先级只选择配置可用的候选，不新增请求失败重试。智谱直连尚未接入，不发布虚构 GLM 候选。旧 River 完整 ID、精确规则、固定账号、Claude `[1m]` 和 Desktop 摘要仍兼容解析，公共菜单不重复发布旧 ID；旧优先级通过迁移 030 归入逻辑模型，已有逻辑配置优先。也可配置自定义别名或固定账号。停用服务商／账号后，不在可调用目录中发布对应模型。

Chat Completions 与 Responses 使用原生 HTTP JSON／SSE，保留思考、函数工具调用、工具结果及官方 usage。Chat 流式请求添加 `stream_options.include_usage`。Responses 无状态，客户端必须发送完整历史；拒绝 `store: true`、`background: true`、`previous_response_id`、`conversation` 和上下文管理引用，不静默丢失历史。仅支持函数工具，不开放托管搜索、计算机等专用工具。

Messages 使用现有文本／函数工具协议转换，经 Chat 接入。工具调用 ID 和结果保持关联；跨协议先收完整响应，再生成下游 JSON／SSE，不提供原生首 Token 增量输出，也不保证保留跨协议的思考正文。不能将其当作 River 官方 Messages API。

首版沿用本项目文本输入链路，目录只标记文本。River 官方三款模型具有图片能力，但本次未接入图片、音频或文件输入。现有本地请求体限制、超时与并发继续生效；1M 官方上下文元数据不代表完成长输入或最大输出实测。没有余额／套餐查询，不显示推算剩余额度；不增加自动重试、故障切换或官方上游 WebSocket。

核对日期：2026-10-17。依据 [官方模型与限制](https://docs.river.ai/guides/inference-models/)、[推理接入](https://docs.river.ai/guides/inference/)、[API 控制参数](https://docs.river.ai/guides/inference-controls/)、[函数工具](https://docs.river.ai/guides/inference-tools/) 与 [超时及错误](https://docs.river.ai/guides/inference-limits/)。尚未使用真实 River Key 验证账号权限、计费、客户端或长窗口。
