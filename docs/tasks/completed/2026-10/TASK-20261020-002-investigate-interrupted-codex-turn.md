# TASK-20261020-002: 排查 Codex 工作中断与 invalid_request

## Status

Completed

## Created

2026-10-20

## Request

用户提供 Codex Desktop 截图，要求检查为什么工作途中出现 `invalid_request`，点击继续仍然报错。

## Goal / Scope

只读关联指定聊天的错误、OmniProxy 生产请求与相关实现，解释中断原因和可验证的恢复方向。

不修改产品代码、客户端配置、账号、凭据或运行数据库，不重启服务、不重放模型请求，不执行截图中的开发需求。

## Analysis

- 截图对应聊天有两次失败：02:14:47 与 02:15:53（新加坡时间），错误均为网关返回的 400 / invalid_request。
- 首次失败前客户端出现 503 / credential_unavailable 后重试；需关联生产元数据，不能仅凭截图认定为上下文过长或程序编译失败。

## Implementation Plan

1. 只读核对客户端聊天错误、生产健康和对应请求／尝试元数据。
2. 对照错误分类和会话路由实现，区分已确认原因与证据不足的推断。
3. 记录结论、验证边界及后续建议，不干扰原未完成任务。

## Implementation Result

### 现场链路

以下时间均为新加坡时间（UTC+08:00）。

- 生产健康接口确认 `production / 2.0.0-beta.10 / ok`。
- 指定聊天 02:05:32 开始本轮工作，02:14:47 失败；02:15:53 用户发送“继续”，约 264 毫秒后再次失败。两轮最终错误均为网关事件 `400 / invalid_request`。
- 02:13:09–02:14:47 的请求 `ca1d133f878917c7eee4b01dd774d188` 是 WebSocket，单次上游尝试真实状态为 200，已经产生部分输出，最终网关记为 `503 / credential_unavailable`，耗时约 98 秒。
- 客户端收到上述 503 后，以 `connection_closed` 原因重连并发送非增量请求，随后才收到 400。用户“继续”时同样重连并发送非增量请求。
- 对应 400 未形成新的生产请求记录，与 WebSocket 在 `beginTrace` 前校验失败的实现一致。

### 关键边界与复现

- `internal/core/proxy/agent_request.go:56` 对 Responses `input` 固定限制为 512 项，超过时返回 `invalid_messages / 400`。
- `internal/core/proxy/websocket.go:84-87` 把协议解析错误统一改写成 `invalid_request`，掩盖了输入条目超限的具体原因；该分支位于请求记账之前。
- 只按本地 rollout 的条目类型与时间统计，无压缩事件：02:13:09 请求前有 510 个 `response_item`；流中又收到 5 个 reasoning 和 1 个 assistant message，重试前增至 516 项；用户继续后为 517 项。
- 用原历史条目在内存中离线重建请求，510 项（2,420,866 字节）通过解析，516 项（2,462,469 字节）和 517 项（2,462,765 字节）均触发 `invalid_messages`。大小均低于 Responses 的 16 MiB 请求上限。
- 合成输入边界验证：512 项通过，513 项被解析器拒绝；513 项 WebSocket 请求得到与截图相同的 `400 / invalid_request`，且上游调用计数为 0。
- 因而，“长会话在流中断后全量回放触及本地 512 项上限”是与现场最吻合、且已离线复现的直接失败机制。历史重建不是抓获的实际网络帧，客户端可能另做输入规范化；不能把离线重建宣称为逐字节还原原始失败请求。

### 首次中断的边界

- `internal/application/gateway/failover_policy.go:85` 会把未分类的普通错误统一映射为 `credential_unavailable / 503`；流扫描或 socket 写入的底层错误也能到达这里。
- 该请求上游已返回 200 并流出内容，因此不能仅凭错误名称断言登录过期或凭据读取失败。
- 当前生产元数据没有保存底层错误分类，不能确认最初是读流、写连接、响应行大小还是其他底层错误。没有重放真实请求或修改生产诊断。

### 建议

- 临时绕过：使用新聊天读取原任务记录继续，避免携带整段长历史；不会自动丢失已经落盘的项目文件。未替用户创建聊天或执行截图中的额度持久化需求。
- 后续修复应分别处理 Responses 长会话条目边界、WebSocket 具体错误透传与校验失败可观测性、流错误分类。不能靠删除上下文、无限放宽限制或打印原始请求正文解决。
- 原额度持久化 Task Record 仍为 Planned，且说明尚未修改业务代码；本轮未改其内容。

## Progress

- 当前生产为 `2.0.0-beta.10`，健康接口正常。首次中断请求真实上游为 200，流中已有输出，最终记录为 503 / credential_unavailable；该错误也可能来自未分类底层错误，不能直接认定账号失效。
- 客户端在流中断后以 `connection_closed` 原因重新建立 WebSocket 并发送非增量请求，随后在请求记账前收到 400。
- 现有 Responses 输入有固定 512 项上限；本地会话记录从中断请求前的 510 项增至重试前的 516 项，用户继续后为 517 项。已完成无上游调用的边界验证。

## Verification

- PASS：Codex 状态／聊天历史／日志 SQLite 和生产请求／尝试 SQLite 均以 `mode=ro`、`PRAGMA query_only=ON` 查询；生产请求与尝试在同一只读事务核对。
- PASS：生产 `3890/healthz`、生产安全日志、指定聊天重试日志与历史条目数量交叉核对。
- PASS：通过临时 Go overlay 执行 `go test -overlay <temporary-overlay.json> ./internal/core/proxy -run '^TestDiagnosticInputLimitAndHistoricalReplay$' -count=1 -v`，覆盖 512／513 项边界、WebSocket 通用错误与零上游调用、510／516／517 项历史离线重建。
- 测试源码仅位于系统临时目录，通过 overlay 加入测试构建；未向仓库增加测试或修改产品源码。历史正文仅在内存中用于解析，没有打印、另存或发送到上游。
- 本轮仅新增诊断记录，不修改代码／配置，不运行全量 `task verify`；不将离线复现声称为线上修复。
- PASS：`git diff --check`；工作区已有未跟踪任务记录保持不动。

## Documentation

仅新增本诊断记录。无产品行为、技术栈、架构或当前实施任务变更，不修改 PRD、ADR 或 CURRENT。

## Git

Not committed by Agent. 本轮为只读排查及记录，未提交或推送，未纳入其他未跟踪文件。

## Result

已定位并离线复现长会话全量回放超过 512 项后被 WebSocket 改写为通用 `invalid_request` 的失败机制；最初流中断原因仍受现有日志限制。未修改生产或产品代码，原开发任务未被擅自继续。

## Completed

2026-10-20
