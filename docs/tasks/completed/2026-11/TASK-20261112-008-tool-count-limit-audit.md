# TASK-20261112-008: 核对 MiMo／Kimi／MiniMax／StepFun／Antigravity 的 128 工具上限是否有上游依据

## Status

Completed

## Created

2026-11-12 16:50（Asia/Singapore）

## Request

River 修复（TASK-20261112-006）完成后，用户同意继续处理同类问题：

> 查一下吧

即：核实其余五个适配器（MiMo、Kimi、MiniMax、StepFun、Antigravity）里同样的
`len(tools) > 128` 本地计数是否有官方依据或上游实测支撑，再决定是否统一移除。

## Goal

- 对每个适配器：确认计数位置、官方文档是否公布工具数量上限、用真实账号向上游发一次
  129 个函数工具的请求（`max_tokens=1`），记录接受／拒绝结果。
- 有实测支撑的结论写入任务记录；是否统一移除由后续实现任务（若需要）承担。
- 本轮只做只读核对与一次最小计费探测，不改代码。

## Background

- `internal/provider/river/tools.go` 的 128 上限已在 `4a62be80` 移除（官方未公布，实测 176 个 200）。
- 其余适配器仍有相同常量：`internal/provider/mimo/content.go`、`kimi/content.go`、
  `minimax/validation.go`、`stepfun/validation.go`、`antigravity/chat_request.go`。
- 这些数字看起来是从 OpenAI 兼容层的常见上限类推，但没有记录核对来源。

## Scope

- 读取五个适配器的端点、认证方式与计数分支。
- 从本机数据库取已有账号，按 DPAPI 解密凭据，向上游各发一次 129 工具探测。
- 记录结果、失败原因与未覆盖的边界。

## Out of Scope

- 不修改任何适配器代码（结论成立时另建实现任务）。
- 不测试图片、文件、长上下文等其他限制。
- 不探测没有可用账号的提供商（无法核对即如实记录）。

---

# Analysis

## Related Modules

- `internal/provider/mimo/`、`kimi/`、`minimax/`、`stepfun/`、`antigravity/`
- 生产数据库 `accounts`／`mimo_account_options` 等（只读取账号与区域，不导出个人信息）

## Dependencies

各提供商本机已保存的有效 API Key／订阅凭据。

## Risks

- 探测会产生少量计费（每次 `max_tokens=1`，129 个函数工具）。
- 上游可能对超大工具数组直接断开或返回非 JSON，需按原样记录，不重试。
- Antigravity 是订阅通道且请求形态特殊（包封 + 会话 ID），单次手工构造可能触发与工具数量无关的拒绝；
  若无法把变量隔离到工具数量，则结论标记为「未验证」。

---

# Implementation Plan

1. 读取五个适配器的端点、请求路径与工具校验位置。
2. 从数据库取每个提供商的一个可用账号（记录名称前若干字符，不导出凭据）。
3. 解密凭据，构造 129 个函数工具的最小请求，直连上游，记录状态码与错误码。
4. 对照官方文档确认是否公布数量上限。
5. 汇总结论，决定是否需要实现任务。

---

# Progress

- 已建立记录并开始读取适配器实现。
- 已确认五个适配器的计数位置与协议覆盖范围（Chat／Responses／Messages）。
- 已核对官方文档：StepFun 与 Google（Gemini）明确公布 128；Kimi／MiniMax 未公布数量上限。
- 本机数据库只有 Google 订阅账号，没有 MiMo／Kimi／MiniMax／StepFun 的 API Key，
  这四家无法用真实账号向上游实测；结论按「文档 + 可否验证」分级记录。

---

# Implementation Result

本轮只做只读核对与文档查证，未修改任何代码。

## 五个上限的实际位置

| 适配器 | 位置 | 覆盖协议 |
| --- | --- | --- |
| MiMo | `internal/provider/mimo/content.go:102` | Chat／Responses／Messages |
| Kimi | `internal/provider/kimi/content.go:58` | Chat／Responses／Messages（另有 `depth > 1` 拒绝嵌套工具） |
| MiniMax | `internal/provider/minimax/validation.go:73` | Chat／Responses／Messages |
| StepFun | `internal/provider/stepfun/validation.go:84` | Chat／Responses／Messages |
| Google 订阅（Antigravity） | `internal/provider/antigravity/chat_request.go:50`（`len(input.Tools) > 128`） | 内部 `chatPayload` |

## 官方依据核对

| 适配器 | 官方是否公布工具数量上限 | 结论 |
| --- | --- | --- |
| StepFun | 公布：Tool Call 文档写明 “Up to 128 tools are allowed.”（[Tool Call](https://platform.stepfun.ai/docs/en/api-reference/tool-call)） | **有依据，保留** |
| Google 订阅 | 公布：Gemini 函数声明上限 128（[Firebase AI Logic function calling](https://firebase.google.com/docs/ai-logic/function-calling)：“The maximum number of function declarations you can provide in a request is 128.”） | **有依据，保留** |
| Kimi | 未公布数量上限；Tool Calls 文档对「几十或上百个工具」建议改用 [Dynamic Tool Loading](https://platform.kimi.ai/docs/guide/use-dynamic-tool-loading)，但未给出硬上限（[Tool Calls](https://platform.kimi.ai/docs/guide/use-kimi-api-to-complete-tool-calls)） | **无依据，未实测** |
| MiniMax | 未公布数量上限；只公布请求体 64 MB、`tools` 为函数定义列表（[OpenAI SDK](https://platform.minimaxi.com/docs/api-reference/text-openai-api)、[工具使用](https://platform.minimaxi.com/docs/guides/text-m3-function-call)） | **无依据，未实测** |
| MiMo | 文档站在核对时只能取到页面标题，模型／接口页未给出工具数量上限；与 River 情况相同，属于无出处常量（对比参考：MiMo 目录声明 1M 上下文与工具能力） | **无依据，未实测** |

## 上游实测

- River：见 TASK-20261112-006，176 个函数工具返回 200，上限已移除。
- StepFun／Google 订阅：文档已明确 128，未再消耗真实额度复测。
- MiMo／Kimi／MiniMax：本机没有任何可用账号（`accounts` 表只有 3 个 Google 订阅账号，
  无 MiMo／Kimi／MiniMax／StepFun 的 `api_key` 账号），**无法向上游实测**；不做猜测。

## 风险与影响

- 这三个未核对的上限会让超限请求在本地变成笼统的 `400 invalid_request`，与 River 修复前的
  症状完全一样；DSH 这类把内置工具与多个 MCP 合并的客户端是主要受害者。
- 移除它们后，若上游真的拒绝，请求会走到上游并以既有上游拒绝错误返回（仍会记录到请求明细），
  不会静默截断工具；代价是错误码来自上游，可能不如本地明确。

---

# Verification

## Tests

- [ ] Unit Tests：N/A（未改代码）
- [ ] Integration Tests：N/A
- [ ] E2E Tests：N/A

## Quality Checks

- [x] 改动仅为任务记录，未触发构建或测试。

## Verification Result

```text
本轮为只读核对：未运行 task verify（无代码变更）。
官方文档引用已逐一打开核对；本机账号清单来自生产数据库只读查询。
```

---

# Documentation

- [x] 本任务记录：结论与证据
- [ ] `docs/product/*`：若后续移除上限，再补对应提供商文档
- [ ] CURRENT：按审计结论登记

---

# Result

五个适配器里，**StepFun 与 Google 订阅的 128 有官方依据，保留**；**Kimi、MiniMax、MiMo 的 128
没有任何官方或实测依据**，与 River 修复前的性质相同，但因为本机没有这三家的可用 Key，本轮
无法像 River 那样用真实请求核对。

下一步建议（待用户确认）：按 River 的方式移除这三家的数量分支、保留形状校验与错误码，补
「129 个工具不被本地拒绝」的单元测试，并在 `docs/product/` 对应页面写明「官方未公布数量上限，
本地不再计数」；等将来有账号时再补一次真实上游探测。

---

# Completed

2026-11-12 17:05
