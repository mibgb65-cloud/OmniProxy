# ADR-004: 重置卡手动兑换与持久化幂等记录

## Status

Accepted — 2026-10-09

## Context

重置卡是有限的账号权益。超时、IPC 响应丢失、关闭弹窗或应用重启都可能发生在官方已执行之后，仅靠前端 loading 或内存去重不能避免再次消耗。

## Decision

- 通过官方 account/rateLimits/read 读取数量与详情，数量以 availableCount 为准，保留未返回与零数量的区别。
- 用户选择并确认后生成一个安全随机 UUID；同一次核对和重试复用它，调用 account/rateLimitResetCredit/consume。
- 在迁移 006 的 subscription_reset_attempts 表先保存账号、官方公开身份标识、卡片 ID、幂等键与 pending 状态，再调用官方接口。
- 每账号最多一个 pending 操作；重复同键直接恢复记录，其他新请求先返回待核对记录，不覆盖它。
- 确认的官方结果保存为 completed。保留最近完成回执，以支持服务器完成但 IPC 丢失后的恢复。记录不含凭据、授权 URL、令牌、邮箱或原始响应。
- 查询和兑换复用账号操作互斥。指定卡片须为最新快照中的可用、未过期且支持的类型；仅数量时可省略卡片 ID，由官方选择。
- 兑换后的额度和数量由官方重新读取，不自行减数量或恢复额度；刷新失败与兑换结果分开展示。

## Alternatives Considered

- 只在按钮上禁用：不能处理重启或丢失响应。
- 只在内存保存 UUID：应用退出后失去核对依据。
- 每次失败生成新 UUID：不能排除上次已执行，会重复消耗。

## Consequences

增加小型操作表和明确的 pending 核对交互。有限权益只响应界面确认，不自动兑换。删除账号通过外键级联清理记录；历史记录只保留必要的公开标识与稳定状态。

协议依据：[官方 app-server 文档](https://learn.chatgpt.com/docs/app-server)。
