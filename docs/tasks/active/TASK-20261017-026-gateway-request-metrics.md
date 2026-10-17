# TASK-20261017-026: 网关运行概览与首字耗时统计

## Status

In Progress

## Created

2026-10-17（Asia/Singapore）

## Request

网关主页面记录请求首字耗时和其他信息，参考用户提供的统计卡片、请求趋势、上游分布、最近请求和接入信息布局。

## Goal

网关运行概览展示真实请求指标与可操作信息，补齐可验证的首字耗时采集、记录和统计；明确口径、样本及未知值，不使用示意图中的虚构数值。

## Analysis

- 请求已有 Trace、持久明细、独立 15 分钟结果／Token 桶；网关概览目前仅配置和资源卡。复用 DashboardRead 及变更信号，不另建统计体系或引入图表依赖。
- 首字定义为从网关受理到首次成功写出并刷新有效流式正文／推理／工具内容，HTTP 等完整 SSE 事件分隔后计时，WebSocket 等消息写出后计时；跳过心跳、空 delta、创建元数据。非流式和历史未知为 null，平均仅取成功且实际测得的流式样本，单独显示样本数。
- 计时采用单调时钟，不保存正文；在结束记账事务保存明细与累计，追加迁移 032。现有独立桶保留首字统计，明细淘汰不减少累计。
- 上游分布复用独立 provider/model 累计，明确标为累计，避免冒充时段分布；请求指标／趋势／最近记录共用今日／7／30 天边界。当前进行中按全部在途记录统计。
- 流式格式核对：[OpenAI Responses 事件](https://developers.openai.com/api/reference/resources/responses/streaming-events)、[OpenAI 流式指南](https://developers.openai.com/api/docs/guides/streaming-responses)、[Messages 流式](https://platform.claude.com/docs/en/build-with-claude/streaming)、[DeepSeek Chat](https://api-docs.deepseek.com/api/create-chat-completion/)。

## Implementation Plan

1. 评估首字耗时采集位置与定义，复用现有请求统计，确认最小兼容迁移及历史未知值处理。
2. 实现真实计时与持久／查询链路，完善网关概览的指标卡、趋势、上游分布、最近请求与运行信息。
3. 覆盖流式／非流式、失败／取消、聚合口径和未知值，完成必要集成、原生 UI 验证、质量门及文档提交归档。

## Implementation Result

- 网关概览改为六项真实指标、可悬浮／聚焦的成功失败折线、最近请求的本地模型／提供商及状态筛选、累计上游分布和接入运行卡；今日／7／30 天共用已有时区边界，背景更新不驱动手动刷新动画。手动刷新复用既有柔和反馈，所有查询仅访问本地数据。
- Core 按首次完整下游内容写出测量，覆盖 Chat／Responses／Messages SSE 和 Responses WebSocket，区分正文、推理和函数工具与元数据。新计时使用单调时钟，流式帧与错误行为不变，不保存或打印内容。
- 迁移 032 增加可空明细和首字桶总和／样本数，结束事务原子记账，成功样本才进入均值；失败／取消保留已测单次值，重复结束、明细淘汰不重复／减少累计，保护范围及整数溢出。历史与非流式保持未知，真实零有效。
- DashboardRead 可选字段扩展保留兼容；网关／首页复用统计 hook 和缓存，当前未结束请求按全局独立桶计算，不限所选时段，避免扫描大量保留明细。提供商分布使用已有独立累计并明确标注，保留删除资源后的历史；请求详情新增首字和总耗时。
- 日志仅增加白名单数字字段；无新增依赖、供应商请求、配置写入或大范围重构。新业务源码与测试保持小文件。

## Verification

- `go test ./internal/core/proxy ./internal/application/activity ./internal/infrastructure/logger`：PASS。覆盖 29 组内容／元数据识别、完整 SSE 分隔／刷新失败、首次不覆盖、HTTP 三协议与非流式、实际 WebSocket 写出；覆盖真实 0／未知、成功均值口径、取消失败、范围外未结束计数、独立累计、淘汰及幂等、非法值／溢出原子回滚与历史迁移。
- 原有网关实际本机上游转发／协议转换集成增加 032 并核对真实 usage 与首字，定向执行 PASS；未调用真实厂商模型。
- 前端新增 8 项概览／模型行为测试，覆盖真实计数、样本、有效 0、请求 ID 无精度丢失跳转、时间切换、筛选、未知／加载／错误恢复、后台刷新独立反馈、隐藏面板停读、英文和有效坐标。既有首页与网关控制保留回归。
- 完整质量门最终 PASS：`task --force verify BIN_DIR=tmp/gateway-request-metrics-20261017/verify-bin`，格式／lint／类型／全部 Go／100 个文件 559 项前端测试／前端生产包及 Windows 生产构建通过。全局未结束计数优化后已重新完整通过。日志 `tmp/gateway-request-metrics-20261017/verify-final.log`；不覆盖运行中的开发产物。
- 正常开发窗口真实历史验收：六组指标，未知首字显示「—」，累计与时段数量明确区分；1920 深浅主题、960 窄窗口、7 天切换后恢复今日均通过，未调用模型。原生最近请求链接实际打开详情，并显示首字／总耗时字段；恢复原窗口尺寸和主题。截图和脚本在 `tmp/gateway-request-metrics-20261017/native-*`。
- 初始检查发现一个新夹具需要补做格式化、旧记账集成夹具漏用新迁移、旧事件测试依赖订阅次序。均修正后重跑；按事件名验证网关回调及清理，未删除或跳过合理测试。请求 ID 链接改为验证真实导航往返，而非固定序列化外观。既有 bundle 大小与 JS DOM scrollTo 提示不影响质量门。

## Documentation

- 更新 GATEWAY／ACTIVITY／PRD、架构概览和 CURRENT；新增 ADR-014 记录下游计时、空值、样本及累计边界。Wails 模型绑定重新生成。

## Deviations

无。沿用现有今日日历与累计定义，布局参考用户示意，所有展示值来自实际记录。

## Git

待提交。

## Result

网关请求概览与首字计时已实现并通过完整验证，待提交归档。新请求采集首字，旧记录保持未知；原生验收未发送收费推理请求。
