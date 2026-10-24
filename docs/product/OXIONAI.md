# Oxion AI

Oxion AI 是内置服务商，使用 OpenAI 兼容的 Chat Completions 接口。适配范围以官方文档
<https://oxionai.com/docs> 为准（读取日期 2026-10-24）；页面内容只作为接口事实来源。

## 接入参数

| 项目 | 值 |
| --- | --- |
| 服务商 ID | `builtin_oxionai`（迁移 043 写入，内置、默认启用） |
| kind | `oxionai` |
| Base URL | `https://oxionai.com/v1`（为空时使用该默认值，路径不再追加 `/v1`） |
| 认证 | `Authorization: Bearer <API Key>`，官方示例键前缀 `cc_` |
| 协议 | 仅 `POST /v1/chat/completions`；`GET /v1/models` 用于目录同步 |

## 模型目录

目录按账号发放，没有内置预设：添加账号后在账号或路由页执行「同步官方模型」，
网关调用 `GET /v1/models` 并只保留 `type=chat` 的条目。

- `context_window` 在 1–10,000,000 之间时作为上下文窗口保存；缺失或越界保持未知。
- `capabilities` 含 `vision` 时声明 `text` + `image` 输入，含 `tools` 时声明函数工具；
  没有 `capabilities` 字段时能力保持未知，不默认开启。
- 目录上限 1000 条、正文上限 1 MiB；解析失败按 `invalid_model_catalog` 处理，
  不静默降级为「无模型」。
- `type` 不是 `chat` 的条目（例如嵌入模型）不进入对话目录。

## 协议与转换

- Chat Completions：原样转发客户端正文，响应（JSON／SSE）原样回传，Token 用量按上游
  `usage` 记账。
- Responses／Messages：网关先用既有 `agentprotocol` 转换成 Chat，再按调用方原协议编码回包。
  这与 DeepSeek 通道一致；转换路径的 Token 用量保持未知（不推测数值）。
- 不注入 `stream_options`、`reasoning_effort` 等官方未声明的参数。上游只公布
  `reasoning_content` 形式的思考内容与 `max_tokens` 预算，因此不发布推理档位选择。

## 边界

- 未接入：Embeddings（`POST /v1/embeddings`）、`web_search` 能力、余额／额度面板。
- 错误按官方 OpenAI 形状的 `error{message,type,code}` 原样回传；`402 insufficient_funds`、
  `403 insufficient_scope`、`429 rate_limit_exceeded` 不重写为本地自定义码。
- 不注入官方未声明的请求头；客户端本地头部不外传，重定向不自动跟随。
- 真实 Key 的端到端调用与长上下文验收由用户执行，本仓库验证使用 loopback 假上游。
