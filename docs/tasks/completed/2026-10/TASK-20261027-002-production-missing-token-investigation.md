# TASK-20261027-002: 生产请求日志缺失 Token 用量排查

## Status

Completed

## Created

2026-10-27（Asia/Singapore）

## Request

查看生产环境日志，解释为什么出现大量未记录 Token 的请求日志。

## Goal

只读核对本机生产运行日志与请求历史，区分正常无用量请求、上游未返回用量和代理统计缺陷，给出可验证的根因及影响范围。

## Scope

- 确认生产数据目录、运行版本及请求历史来源。
- 对缺失用量的状态、接口、传输协议、提供商和时间分布做脱敏聚合。
- 对照请求生命周期、用量提取和前端展示逻辑，必要时做定向回归验证。

## Out of Scope

- 不修改生产数据、凭据、客户端配置，不重启或部署生产程序。
- 不输出请求正文、密钥、令牌、账号身份等敏感信息。
- 未经证据确认，不用估算值冒充实际 Token 用量。

## Analysis

- 已确认运行进程为正式 profile，生产数据来自 `%APPDATA%/OmniProxy/v2`，不是旧版 `.omniproxy` 或开发数据库。
- 2026-10-27 09:33:39（Asia/Singapore）只读快照：留存 10,000 条请求中 703 条用量未知，其中 313 条状态为成功；313 条对应日志中 309 条同时为 `usage_state=complete`、`accounting_failed=true`，另 4 条较早记录的运行日志也没有用量，不能仅凭现有证据判定其具体原因。
- HTTP 透明转发在读取响应正文前调用 `EndAttempt`；用量入库要求同一尝试仍为 `pending`。后续 usage 更新因影响行数为零而回滚，运行日志保留内存用量，请求明细及汇总则漏记。
- 首条可见的相同记账失败为 2026-10-26 20:17:34（Asia/Singapore）。生产实时请求仍在增加，以上数字仅代表该快照。
- 使用临时 Go overlay 和独立测试数据库完成根因与候选修复验证，未修改正式源码、生产数据库、账号或已安装程序。

## Implementation Plan

1. 阅读现有文档和相关任务记录，确认日志与用量存储路径。
2. 只读分析生产日志和 SQLite 请求历史，抽取不含敏感信息的证据。
3. 对照实现验证根因，记录结果及剩余限制，向用户说明。

## Implementation Result

### Root Cause

`internal/core/proxy/passthrough.go:100` 在收到上游响应头后、读取正文之前调用 `EndAttempt`，将尝试提前由 `pending` 改为 `success`。
稍后的 `response.completed.usage` 虽被成功解析，但 `internal/application/activity/attempts.go:85` 更新尝试时要求 `state='pending'`，因此影响行数为零并返回错误。
`Service.Update` 会回滚同一个事务中的请求用量、账号／提供商／仪表盘／累计用量更新。
`Service.Finish` 只保存请求结果，不补写失败的用量事务，所以最终形成“请求成功、用量未知”。

运行日志使用 `Trace` 的内存快照，因此能显示完整 Token 和 `accounting_failed=true`，与数据库及 UI 不一致；这不是前端把已保存的值隐藏了。
该调用顺序可追溯到 `1173f20` 的 HTTP 透明转发尝试级记账；WebSocket 透明转发的 `Report → EndAttempt` 顺序不同，不能把本次根因泛化到所有协议。

### Evidence

- 生产日志：`%APPDATA%/OmniProxy/v2/logs/omniproxy.log` 及轮转文件。
- 请求数据：`%APPDATA%/OmniProxy/v2/omniproxy.db`，通过 SQLite `mode=ro`／`query_only` 查询。
- 脱敏样例：2026-10-27 09:31:59，`gpt-6.1-sol`，HTTP 200，运行日志输入 149,931、输出 78、`usage_state=complete`、`accounting_failed=true`；同一请求及尝试在库中均为 `success`、Token 为 0 且 known 标志均为 false。
- 09:33:39 快照中，309 条明确漏记记录的日志合计输入 108,425,853、输出 349,199；这是请求级上游计数之和（可能反复包含上下文／缓存），不是去重内容、实际收费量或金额。
- 同一快照中，其余 390 条未知用量记录为失败／取消；它们不能与成功漏记混为一谈，也不能据此断言上游没有消费。
- 当日截至快照有 82 条请求，均成功但均未入库用量，因此用户近期会连续看到“未报告”。

### Changes

- 仅新增本排查记录并同步 CURRENT 状态；正式业务代码无变更。
- 在被 Git 忽略的 `tmp/missing-token-investigation/` 中保留诊断用例及 baseline／candidate overlay，全部上游由本机 `httptest` 模拟，全部测试数据库由 `t.TempDir()` 创建。
- 候选方案仅将成功响应的 `EndAttempt` 延后至响应正文转发结束；隔离验证成立，不代表已正式修复或部署。

### Remaining Work

- 正式修复应补充 HTTP 透明转发与真实 activity 持久化的回归测试，覆盖 SSE／JSON、取消／中断、换号和记账失败，再执行 `task verify`。
- 该缺陷影响请求详情及累计统计；正式修复只能阻止后续漏记，历史不会自动恢复。
- 可评估按 `request_id` 从现有日志补回明确的输入／输出计数，但必须先备份、设计幂等且原子同步明细／尝试／各级汇总，不能直接改一个表或把未知值填为零。现有日志不含完整缓存明细，不能承诺恢复全部统计。
- 未进行真实推理、生产数据回填、软件重启、安装或发布；未核验安装程序的精确产品版本，仅确认运行位置、正式编译 profile 和实际数据目录。

## Verification

1. baseline overlay：新增 `TestInvestigationPassthroughPersistsReportedUsage`，对未声明类型 SSE、声明 SSE、JSON 三种响应返回确定的输入 12／输出 4。
   三个子用例均按预期失败，实际持久化为 0／0 且 known=false；此失败是问题复现，不是通过的质量门。
2. candidate overlay：同一测试的三个子用例全部通过，同时断言请求明细、尝试、累计用量和透明响应字节。
3. 候选 overlay 的相关包完整测试：

   ```text
   go test -overlay=<candidate.json> ./internal/core/proxy ./internal/application/activity ./internal/application/gateway -count=1
   PASS（proxy 2.778s；activity 15.689s；gateway 44.512s）
   ```

4. 只读跨日志／数据库关联及聚合通过；原始账号身份、凭据、请求内容均未输出或写入任务记录。
5. 未执行 `task verify`：本次只排查与隔离验证，没有修改正式业务代码、配置、依赖或生产数据，候选 overlay 不视为可交付补丁。
6. Git 状态与文档差异已检查；临时诊断文件不入库。

## Git

Not committed by Agent.

## Result

已定位并在隔离环境复现 HTTP 透明转发提前结束尝试导致的用量事务回滚，确认近期大量成功请求缺失 Token 是本地记账缺陷，而非上游统一不返回用量。
修复方向已通过相关包测试，但本任务仅完成排查，生产问题尚未实施修复，历史数据尚未回填。

## Completed

2026-10-27（Asia/Singapore）
