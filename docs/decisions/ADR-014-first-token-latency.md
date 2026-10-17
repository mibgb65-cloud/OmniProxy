# ADR 014: 按实际下游内容测量首字耗时

Status: Accepted

Date: 2026-10-17

## 背景

用户要求网关页面记录首字耗时并展示请求表现。响应头、SSE 心跳／创建事件可能早于模型输出；缓冲的协议转换还可能提前读完上游，因此上游首字节无法代表客户端等待。历史总耗时也不能恢复首字。

## 决策

- 起点沿用 Trace 受理时点，并用独立单调时钟计算持续时间；对外开始／结束时间仍为 UTC。
- 只测流式请求首次有效下游内容，包含正文、拒绝、思考或工具输出。HTTP 在有效 SSE 事件完整写出并刷新后计时，WebSocket 在消息成功写出后计时；跳过空值、心跳、创建元数据、用量与未知事件。首值不覆盖，不保存输出内容。
- 普通 JSON、旧请求和未观察到内容保持 null；0 毫秒是有效样本。失败／取消仍保留已经测得的单次值，平均首字只使用成功且已测得的流式样本，并展示数量。
- 在完成事务保存明细和独立 15 分钟桶的总和／样本数。不会增加首内容发出前的数据库等待；累计不依赖明细保留上限，完成幂等并保护溢出。
- 基于 ADR-006／007 的现有记账、查询、缓存和更新事件扩展，新增字段可选以兼容现有消费者；不引入新图表或模型 SDK。
- 网关时间筛选作用于请求指标、趋势及最近明细；提供商分布复用既有生命周期独立累计并显式标为累计，不从最近记录推算所选时段全量。

## 后果

缓冲转换的首字可能接近总耗时，这是实际下游表现。此计时不包含客户端界面渲染，也不是上游厂商的纯模型生成指标。崩溃前未完成记账的首字可能未知，不能补造历史；升级后逐步积累样本。请求体、输出内容和凭据不会进入记录或日志。

## 依据

- [OpenAI 流式事件](https://developers.openai.com/api/reference/resources/responses/streaming-events)
- [OpenAI 流式指南](https://developers.openai.com/api/docs/guides/streaming-responses)
- [Messages 流式](https://platform.claude.com/docs/en/build-with-claude/streaming)
- [DeepSeek Chat](https://api-docs.deepseek.com/api/create-chat-completion/)
