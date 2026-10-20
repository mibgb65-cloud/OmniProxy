# TASK-20261019-004: 排查生产环境 Codex Desktop 请求失败

## Status

Completed

## Created

2026-10-19 11:33

## Request

你查看我的生产环境，我之前在 Codex Desktop 的请求全部都失败了，你看看为什么会出现这种情况。

## Goal / Scope

只读核对当前生产版本、历史 Desktop 请求、上游尝试和安全日志，结合实现解释失败链路与证据边界。

不重启生产、不修改客户端配置、账号或数据库，不读取凭据或对话正文，不重放模型请求。

## Analysis

已有 Beta.2、Beta.3、Beta.5 排查记录涉及不同故障阶段，必须以本次生产快照校验，不能将旧结论直接套用。

## Implementation Plan

1. 核对生产健康、数据目录、日志和请求表结构。
2. 用只读事务聚合请求及尝试，区分客户端、时间、模型和传输。
3. 对照响应处理实现，记录根因定位、无法确认的细节及建议。

## Implementation Result

### 生产状态

- 本次只读健康检查确认 production / 2.0.0-beta.5 / ok。PID 48072 同时监听 127.0.0.1:3000 与 3890；日志记录本次启动时间为 2026-10-19 10:50:33（UTC+08:00）。
- 全部时间均为新加坡时间。请求数据库使用 sqlite3 -readonly、PRAGMA query_only=ON 与只读事务查询；只选择非敏感诊断字段，账号仅匿名聚合。

### 本次新增证据

| 请求开始时间 | 模型 | 传输 | 结果 |
| --- | --- | --- | --- |
| 11:01:35–11:04:56 | gpt-6-luna | HTTP | 9 次 502 / invalid_upstream_response |
| 11:05:17 | gpt-6-luna | HTTP | 1 次 499 / request_cancelled |
| 11:20:30–11:23:09 | gpt-6.1-sol | WebSocket | 6 次 502 / invalid_upstream_response |
| 11:23:31–11:25:23 | gpt-6.1-sol | HTTP | 6 次 502 / invalid_upstream_response |
| 11:25:48 | gpt-6.1-sol | HTTP | 1 次取消，11:26:04 结束 |

- 最新一批共 12 次 502，均仅有 attempt_index=1，真实上游状态均为 200；无一次网关请求内的多账号重试。连续请求先 WebSocket 后 HTTP，符合客户端重试并回落的时序；请求表没有客户端／会话标识，不能仅凭此证明具体 Desktop 会话身份或重试决策。
- 所有 21 次 Beta.5 的 502，结束日志均为 upstream_media_type=missing、upstream_content_type_inferred=false。取消应单独计算，不能混为同一种格式错误。
- 11:20:30 同账号 gpt-6-luna WebSocket 成功。11:27:29 起同账号 gpt-6.1-sol WebSocket 持续成功；截至 11:38:31 的结束日志，Beta.5 有 gpt-6.1-sol 14 次、gpt-5.6-luna 2 次、gpt-6-luna 1 次成功，均为 missing / inferred=true 并有用量。成功不能直接归属于 Desktop，也不能证明其原失败会话已恢复。
- 失败与成功路由到同一个匿名账号，近期没有新增 401／403／429 证据。因此不能把故障归为账号失效、模型完全不可用、网关未接入或单纯 WebSocket 故障。
- 生产自检查前后没有重启。本 Agent 未发送模型请求；后续成功是已有客户端自然产生的记录。

### 代码关联与根因边界

- internal/provider/chatgpt/client.go 对固定订阅 Responses 端点接收真实上游响应；internal/provider/chatgpt/response_type.go 只在 200、缺失类型且有 Body 时有界预读首个事件。首事件校验成功才补 text/event-stream。
- 本批 missing / inferred=false 表明此响应没有被识别为可补头的合法首事件。internal/core/proxy/response.go 与 websocket_response.go 的前置媒体类型检查随后返回 502 / invalid_upstream_response，尚未进入正常流式转发。
- “上游 HTTP 200”只证明响应状态，不能证明模型成功；“缺失 Content-Type”也不独立解释所有失败，因为同版本成功流同样缺头，但首事件校验通过。
- Beta.4 的补头兼容已随 Beta.5 生效，本次不能继续归因于尚未升级。已定位到补头识别未通过的响应分支，但现有布尔日志不区分 EOF、读取失败、前缀上限、非 SSE、JSON／事件／ID 校验；历史响应正文没有保存，无法从当前记录还原具体形态。
- managed client catalog refresh failed 是独立警告；已有真实上游尝试证明请求能进入转发路径，不把目录刷新警告当作本次 502 根因。

### 后续修复方向

先在首事件判定增加不含正文的固定失败原因枚举，在隔离环境对相应客户端形态复现，再按实际协议处理。合法 SSE 变体应补兼容；错误响应应保留安全错误分类。不能无条件信任缺失类型的 200，也不能将本轮定向模拟测试通过宣称为真实失败已复现／修复。

本轮仅写此记录，未修改产品代码、运行配置、账号、数据库或发布版本；未读取凭据、真实请求正文和回复正文。

## Verification

- PASS：healthz、端口、启动日志、请求及尝试表、匿名账号聚合和安全日志字段交叉核对。
- PASS：go test ./internal/provider/chatgpt ./internal/core/proxy -run 'Test(ResponseTypeInference|Inference|ResponseHeader)' -count=1。
- PASS：任务记录 UTF-8 无 BOM 与空白检查；既有未跟踪记录保持不变。
- N/A：本轮仅只读诊断和文档，不运行完整 task verify，不进行真实上游重放或线上插桩。

## Git

Not committed by Agent.

本轮仅排查，诊断记录保留工作区，与仓库既有生产排查记录一致；未提交、推送或更改发布资产。

## Result

确认最新 12 次 502 同时覆盖 WebSocket 与 HTTP，失败在上游 200 后的缺失类型／首事件识别与流式响应前置校验。补充同账号、同模型随后成功的证据，排除全局不可用的直接解释。具体首事件失败形态仍缺日志证据，未宣称最终根因或修复完成。

## Completed

2026-10-19 11:40
