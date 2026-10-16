# Architecture Decision Records

This directory records significant technical decisions as ADRs.

An ADR is required when a decision:

- changes the database or the core framework,
- introduces important infrastructure,
- changes how core components communicate,
- designs an important abstraction,
- modifies the overall architecture,
- is hard to reverse.

Ordinary code adjustments do not need an ADR.

## Naming

```text
ADR-001-short-description.md
ADR-002-short-description.md
```

## Template

```md
# ADR-XXX: Title

## Status

Proposed / Accepted / Deprecated / Superseded

## Context

## Decision

## Alternatives Considered

## Reasoning

## Consequences
```

ADRs are never fabricated for decisions that were not actually made. When a
decision replaces an older one, mark the old ADR as `Superseded by ADR-XXX`
rather than editing history.

## Index

MiMo 默认通道选择与目录去重见 [ADR-010](ADR-010-mimo-default-billing-order.md)。

| ADR                                                              | Status  | Summary                                   |
| ---------------------------------------------------------------- | ------- | ----------------------------------------- |
| [ADR-001](ADR-001-wails3-go-react-stack.md)                      | Accepted| Wails 3 + Go core + React frontend stack  |
| [ADR-002](ADR-002-dpapi-credential-files.md)                     | Accepted| Windows 用户范围 DPAPI 凭证文件           |
| [ADR-003](ADR-003-official-codex-subscription-auth.md)            | Accepted| 官方 Codex 隔离订阅授权与系统凭证库       |
| [ADR-004](ADR-004-reset-credit-idempotency.md)                   | Accepted| 重置卡手动兑换与持久化幂等操作记录       |
| [ADR-007](ADR-007-dashboard-time-buckets.md)                    | Accepted| 仪表盘时间桶、统计保留与事件更新         |
| [ADR-008](ADR-008-client-model-catalog.md)                    | Accepted| 统一模型目录、客户端格式与安全自动配置   |
