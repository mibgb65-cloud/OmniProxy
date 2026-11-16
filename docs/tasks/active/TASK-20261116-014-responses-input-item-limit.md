# TASK-20261116-014: 放开 Responses 输入条数上限，修复长会话 Codex 被本地拒成 unsupported_content

## Status

In Progress

## Created

2026-11-16 16:30（Asia/Singapore）

## Request

> 又有报错：e862bb6d8da78f7df89c8f765c834934，你看看该怎么修复下。unsupported_content

## Goal

让 Codex 的长会话（历史输入项超过 512 条）能够继续发往 River 等第三方服务商，不再被网关
本地拒成 `400 unsupported_content`。

## Background

请求 `e862bb6d8da78f7df89c8f765c834934`：`DeepSeek-V4.1-Flash-1017`、`responses`、`codex`、
`400 unsupported_content`、耗时 1479 ms（本地拒绝，未到上游）。

同一时刻日志确认适配器工作正常：`codex namespace tools dropped count=11`、
`codex orphan tool outputs dropped count=2`、`codex unsupported includes dropped count=1`；
随后才被本地校验拒绝。线程 `01a1e338-648e-7763-bdb0-ea6c785955c9` 的 rollout 已有 514 条
`response_item`，加上 Codex 每回合动态注入的 developer 项后超过 512 条。

复现（已安装构建，用该线程 514 条输入项 + 3 条动态 developer 项，正文 13,907,104 字节）：

| 输入条数 | 结果 |
| --- | --- |
| 512 | `200` + `response.completed` |
| 513 | `400 unsupported_content`，91 ms（本地拒绝） |
| 517（真实长会话形状） | `400 unsupported_content`，1571 ms，与线上 1479 ms 一致 |

## Root Cause

`internal/provider/river/validation.go` 的 `textMessages` 把 Responses 输入数组硬限制为
512 条（`len(items) > 512 → false`），atria 与 mimo 的同类校验也沿用同一个数字。
`docs/product/GATEWAY.md` 早已声明 Responses「不套用消息数量限制」，组件里
`proxy.MaxConversationMessages = 100000` 也是给 Messages 用的既有安全上限——只有
Responses 校验还停在 512，长会话必然撞上。

## Scope

- `internal/provider/river/validation.go`、`internal/provider/atria/validation.go`、
  `internal/provider/mimo/content.go`：Responses 输入条数上限改用
  `proxy.MaxConversationMessages`（100,000），与 Messages／Chat 既有口径一致；
  单条消息的 content 块数上限保持不变。
- 复核 River 对 `custom_tool_call`／`custom_tool_call_output` 历史项的态度（Codex 用
  apply_patch 的线程会带这些项，atria／mimo 已放行而 River 目前本地拒绝）；按真实 Key
  实验结果决定放行或由 Codex 入口适配器移除。
- 测试：513 条以上输入放行、超过 100,000 条仍拒绝；`custom_tool_call` 相关按最终方案覆盖。
- 文档与版本：`GATEWAY.md`、`RIVER.md`、`v2.3.1-beta.14` 发行说明、`CURRENT.md`、
  `version.go`、`frontend/package.json`；随后构建 `v2.3.1-beta.14`。
- 真实 Key 验证：用上面这份 517 条正文复放，确认正常完成。

## Out of Scope

- 不改 16 MiB Responses 请求体上限（长会话最终仍可能撞上，属独立容量问题）。
- 不引入历史压缩／截断：不静默删条目。
- 不改 Messages／Chat 的既有上限。

## Existing Behavior

长会话输入超过 512 条时，网关在本地返回 `400 unsupported_content`，请求未到达上游。

## Expected Behavior

- Responses 输入 1 至 100,000 条正常校验并转发；超过 100,000 条仍明确拒绝。
- 真实长会话正文（517 条、13.9 MB）在真实 River Key 上正常完成。

---

# Analysis

## Related Modules

- `internal/provider/river/validation.go`、`internal/provider/atria/validation.go`、
  `internal/provider/mimo/content.go`
- `internal/core/proxy/types.go`（`MaxConversationMessages`）
- 测试：`internal/provider/river/tool_output_validation_test.go` 等

## Dependencies

N/A

## Risks

- 放开条数后单请求解析量增大：真实边界仍是 16 MiB 请求体与 JSON 深度，100,000 条上限
  只为避免退化数组无界迭代；与 Messages 同口径。
- River 对 `custom_tool_call` 的处理需要真实 Key 验证，不能只按本地校验放行。

---

# Implementation Plan

1. 记录本任务与复现数据。
2. 三处 Responses 条数上限改用 `proxy.MaxConversationMessages`。
3. 真实 Key 实验 `custom_tool_call` 历史项；按结果放行或移除。
4. 测试、文档、版本、`task verify`、真实正文复放、打包 `v2.3.1-beta.14`。

---

# Progress

- 2026-11-16 16:24：由请求记录与 rollout 定位到 512 条上限，并在已安装构建上复现。
- 2026-11-16 16:30：建立本记录。

---

# Implementation Result

## Changes

- `internal/provider/river/validation.go`、`internal/provider/atria/validation.go`、
  `internal/provider/mimo/content.go`：Responses 输入项上限由硬编码的 512 改为
  `proxy.MaxConversationMessages`（100,000，与 Messages／Chat 既有口径一致）。
- `internal/provider/river/validation.go`：`custom_tool_call` 与
  `custom_tool_call_output` 按 atria／mimo 的既有做法放行（后者仍要求非空 `call_id` 与
  可表达的输出内容）；真实 Key 已验证 River 原样接受这些历史项。
- 测试：
  - `internal/provider/river/conversation_limits_test.go`（新增）：513 条放行、达到
    100,000 条放行、超过 100,000 条拒绝；`custom_tool_call` 历史放行、缺 `call_id`
    的自定义工具结果拒绝。
  - `internal/provider/atria/validation_test.go`、`internal/provider/mimo/validation_test.go`：
    同一条数边界用例。
- 文档：`docs/product/GATEWAY.md`（Responses 条数口径）、`docs/product/RIVER.md`
  （100,000 条上限与 custom 工具历史）、`docs/releases/v2.3.1-beta.14.md`、
  `docs/plans/CURRENT.md`、版本号 `2.3.1-beta.14`。

## Files Changed

- `internal/provider/river/validation.go`
- `internal/provider/atria/validation.go`
- `internal/provider/mimo/content.go`
- `internal/provider/river/conversation_limits_test.go`（新增）
- `internal/provider/atria/validation_test.go`、`internal/provider/mimo/validation_test.go`
- `internal/version/version.go`、`frontend/package.json`
- `docs/product/GATEWAY.md`、`docs/product/RIVER.md`、`docs/plans/CURRENT.md`
- `docs/releases/v2.3.1-beta.14.md`（新增）

## Technical Notes

- 条数上限只防退化数组的无界迭代；真实容量边界仍是 16 MiB 请求体与 JSON 深度，
  与 Messages 完全同口径。
- 单条消息的 content 块数上限（512）保持不变：它不随会话长度增长，与本次问题无关。
- `custom_tool_call` 的放行不是猜测：先用真实 River Key 在隔离实例上验证
  `200` + `response.completed` 后才保留。

---

# Verification

## Tests

- [x] Unit Tests：`TestResponsesInputItemsUseConversationLimit`（River／Atria／MiMo）、
  `TestCustomToolHistoryIsAccepted` 通过；三个包全部测试通过。
- [x] Integration Tests：`validatedBody` 完整校验链路覆盖条数与 custom 工具历史。
- [x] E2E Tests：真实 River Key + 生产数据隔离实例，复放失败线程 517 条／13,907,104 字节
  正文：`200` + `response.completed`（40.6 s，模型正常作答）。

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task verify → EXIT=0（155 文件／948 项前端测试、全部 Go 包、Windows production 构建）
```

复放对照（失败线程自身输入项 + 3 条动态 developer 项）：

| 场景 | 已安装构建 | 新构建隔离实例 |
| --- | --- | --- |
| 512 条输入 | 200 | 200 |
| 513 条输入 | 400 unsupported_content（91 ms） | 200 |
| 517 条／13.9 MB（真实形状） | 400 unsupported_content（1571 ms，线上 1479 ms） | 200 + `response.completed`（40.6 s） |
| `custom_tool_call` 历史（真实 Key） | 本地 400 | 200 + `response.completed` |

未验证：安装构建后由真实 Codex 桌面端在同一线程重试。

---

# Documentation

- [x] `docs/product/GATEWAY.md`：Responses 输入条数口径
- [x] `docs/product/RIVER.md`：100,000 条上限与 custom 工具历史
- [x] `docs/releases/v2.3.1-beta.14.md`：根因、修复与验证边界
- [x] `docs/plans/CURRENT.md`：当前状态
- [x] 其它文档无需修改

---

# Deviations

- 原计划只放开条数上限。排查中发现 Codex 的 apply_patch 历史会带 `custom_tool_call`，
  而 River 是唯一仍本地拒绝它的 Responses 适配器（atria／mimo 早已放行）。用真实 Key
  验证上游接受后一并放行，避免下一轮再报同类 `unsupported_content`。
- 正文 13.9 MB 已接近 Responses 16 MiB 上限：本任务不扩大该上限（属独立容量问题），
  如继续增长会按既有规则返回 `request_too_large`。

---

# Git

## Commit

```text
Not committed by Agent.
```

---

# Result

长会话 Codex 请求不再因 Responses 输入超过 512 条被本地拒成 `unsupported_content`：
River／Atria／MiMo 统一改用与 Messages 相同的 100,000 条安全上限，并放行真实 Key 已验证的
`custom_tool_call` 历史。失败线程的 517 条／13.9 MB 正文在新构建上 `200` +
`response.completed`，`task verify` EXIT=0。

---

# Completed

2026-11-16 16:45（Asia/Singapore）
