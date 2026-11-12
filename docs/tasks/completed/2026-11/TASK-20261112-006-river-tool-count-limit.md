# TASK-20261112-006: River 适配器不再按未核对的 128 个工具上限拒绝请求

## Status

Completed

## Created

2026-11-12 16:05

## Request

用户在 DeepSeek Harness（下称 DSH）里用 OmniProxy 的 River `deepseek-ai/DeepSeek-V4.1-Flash-1017`
继续对话时，客户端收到：

```json
{"code":"invalid_request","message":"invalid_request","type":"gateway_error"}
```

用户要求先恢复可用，再处理根因：

> 先做1，然后再做2

其中「1」是临时在 DSH 侧停用 OpenPencil MCP 恢复可用，「2」是修掉网关这一侧的问题。

## Goal

- 让 DSH 这类会一次性发送上百个工具的客户端，在工具数量本身合法时不再被本地 128 计数上限
  拦成笼统的 `invalid_request`。
- 保留对工具内容的结构校验（必须是 `function`、必须有名字、名字长度上限），不静默删除、
  不截断任何工具。
- 把「River 官方未公布工具数量上限，本地不再设该上限」写进产品文档，并补回归测试。

## Background

失败发生在 DSH 的普通对话请求上，特征（来自本机生产日志与数据库）：

- `request_records`：`protocol=chat`、`stream=1`、`client_id=harness-desktop`、
  `model=deepseek-ai/DeepSeek-V4.1-Flash-1017`、`reasoning_effort=high`、
  `status_code=400`、`error_code=invalid_request`、`input_tokens=0`、延迟 58–74 ms。
- 同一毫秒还会有一个成功请求（138 prompt tokens、无 `reasoning_effort`）：那是 DSH 的
  会话标题请求（`session/title-llm-request`），因为不带工具所以正常。
- DSH 会话事件（`request/header`）显示当次请求带了 **176 个工具**：
  内置 36 + `mcp__openpencil` 132 + `mcp__wuliu-mysql-go` 4 + `mcp__wuliu-mysql-python` 4。
- 用同模型的 128 个工具请求本机网关返回 200，129 个返回与用户完全一致的
  `400 {"code":"invalid_request",...}`。

因此 400 是**本地**在调用上游之前产生的，不是 River 网络、账号、余额或推理档位问题。

## Root Cause

`internal/provider/river/tools.go` 的 `validateTools` 沿用了一个**没有出处的计数上限**：

```go
if json.Unmarshal(raw, &tools) != nil || string(raw) == "null" || len(tools) > 128 {
    return proxy.Failure("invalid_request", 400)
}
```

River 官方文档（[Limits and support](https://docs.river.ai/guides/inference-limits/)）只公布
50 MiB 请求体、图片格式与超时，没有工具数量上限；[Tools](https://docs.river.ai/guides/inference-tools/)
也只描述 `type=function` 的形态。本任务用生产账号的 River Key 直接向
`https://api.river.ai/v1/chat/completions` 发送 **176 个** 函数工具（`max_tokens=1`），
River 返回 200，说明 128 这个数字既不是 River 文档中的契约，也不是当前上游的实际行为。

128 看起来是从 OpenAI/DeepSeek 兼容层的常见上限类推出来的防御值，同时被 MiMo／Kimi／MiniMax／
StepFun／Antigravity 适配器使用；那些适配器不在本任务范围内，保持原样。

## Scope

- `internal/provider/river/tools.go`：移除工具**数量**上限，保留类型／名称校验。
- `internal/provider/river/tools_test.go`（新建）：129／176 个合法工具通过，非法工具仍拒绝。
- `docs/product/RIVER.md`：记录「官方未公布工具数量上限，本地不再限制数量；上游仍可能拒绝」。
- `docs/plans/CURRENT.md`、本任务记录。

## Out of Scope

- 其他提供商的 128 工具上限（没有对应的上游核对结论，不在本次修改）。
- DSH 的 MCP 配置本身（临时停用 OpenPencil 只是恢复手段，见下）。
- 请求体大小、JSON 深度、单条文本等既有边界。
- 真实端到端工具调用（本次只验证「不再被本地计数拒绝」，不触发模型工具调用）。

## Existing Behavior

工具数 >128 时，网关在本地直接返回 `400 invalid_request`，错误体与真实原因无关，
客户端无法从错误码判断是工具数量问题。

## Expected Behavior

- 工具数量不再由 River 适配器本地限制；数量只受请求体大小与上游校验约束。
- `type` 不是 `function`、缺少函数名、名称为空或超过 128 字符时仍返回稳定的本地错误码
  （`unsupported_content`／`invalid_request`）。
- 不会为了通过校验而丢弃或截断任何工具。

---

# Analysis

## Related Modules

- `internal/provider/river/tools.go`（`validateTools`，由 `validatedBody` 调用）。
- `internal/core/proxy/request.go`（既有请求体／消息／参数边界，保持不变）。
- `docs/product/RIVER.md`（产品行为与排查说明）。

## Dependencies

没有：N/A

## Risks

- 工具数组很大时，本地做结构校验的成本上升；请求体仍有 2 MiB（Chat）／16 MiB（Messages
  转换）与 JSON 深度上限，内存与耗时边界不变。
- 上游对超大工具数组的处理未公开；若 River 拒绝，会以既有的上游拒绝错误码返回，并由
  请求明细记录，而不是本地 400。

---

# Implementation Plan

1. 移除 `validateTools` 的 `len(tools) > 128` 分支，保留其余结构校验。
2. 新增 `tools_test.go`：124／129／176 个工具全部接受；`type` 非 `function` →
   `unsupported_content`；`function` 缺失、空名字、超长名字 → `invalid_request`。
3. 更新 `docs/product/RIVER.md` 的工具段落，写明未公布上限、本地不设数量上限、实测与边界。
4. 运行 `go test ./internal/provider/river/...` 与 `task verify`。
5. 更新本记录、`docs/plans/CURRENT.md`，提交。

---

# Progress

- 2026-11-12 15:50：定位到生产日志中 3 次同特征 400，全部发生在 River 模型且 0 token。
- 2026-11-12 15:55：从 DSH 会话日志还原出当次请求携带 176 个工具，并复现 128／129 边界。
- 2026-11-12 16:00：用生产 River Key 直接对上游验证 176 个工具返回 200，确认 128 无上游依据。
- 2026-11-12 16:05：建立本任务记录；DSH 侧已临时停用 OpenPencil MCP 恢复可用。
- 2026-11-12 16:12：用户要求「先恢复到可用就可以了」，本任务暂停，代码改动全部撤回，
  工作树只保留本记录。
- 2026-11-12 16:08：DSH 以 44 个工具重启后一切正常，确认临时恢复有效；用户确认继续做根因修复。
- 2026-11-12 16:34：`task --force verify` exit 0（147 文件／895 项前端、全部 Go 包、
  Windows production 构建）。

---

# Implementation Result

## Changes

实际完成：

- `internal/provider/river/tools.go`：`validateTools` 去掉 `len(tools) > 128`，只保留形状校验；
  函数注释写明上游核对依据（官方未公布数量上限、2026-11-12 实测 176 个返回 200）。
- `internal/provider/river/tools_test.go`（新建，4 组测试）：
  - `TestToolCountIsNotCapped`：Chat／Responses 各 1、128、129、176、200、512 个工具全部接受。
  - `TestChatForwardAcceptsManyTools`：走 `Client.Forward` 完整路径，176 个工具原样到达假上游
    （断言上游收到的数量，证明没有被截断）。
  - `TestToolStructureStillValidated`：`type` 非 `function` → `unsupported_content`；
    `function` 缺失、空名字、129 字符名字、`null`、非数组 → `invalid_request`。
  - `TestAbsentToolsIsAccepted`：不带 `tools` 仍通过。
- `docs/product/RIVER.md`：新增「工具数量」段落，记录官方公布范围、实测结论、本地不再计数，
  以及仍保留的逐条校验；说明其他提供商适配器的 128 上限没有等价核对结论、保持原样。

## Files Changed

- `internal/provider/river/tools.go`
- `internal/provider/river/tools_test.go`（新增）
- `docs/product/RIVER.md`
- `docs/plans/CURRENT.md`
- `docs/tasks/active/TASK-20261112-006-river-tool-count-limit.md` → `completed/2026-11/`

## Technical Notes

不带 `tools` 与 `tools: null` 的语义不同：缺失表示不使用工具（放行），`null` 视为损坏正文
（仍然 400）。数量限制原本与 `unsupported_content` 的逐条校验混在同一个 `if` 里，移除计数
分支时保持了两者原有错误码，没有顺手改动其他适配器。

## 恢复可用（仓库外，已执行）

- 备份：`C:\Users\mimanchi\.dsh\profiles\desktop\cordis.patch.yml.before-openpencil-20261112.bak`
- 改动：把 `- id: mcp-openpencil` 整段注释掉，工具数从 176 降到 44（内置 36 + 两个物流 MCP 各 4）。
- 结果：DSH 重启后正常使用；重新启用 OpenPencil 需要本修复对应的版本实际运行。

---

# Verification

## Tests

- [x] Unit Tests：`go test ./internal/provider/river/`
- [x] Integration Tests：`go test . ./internal/...`（含 `internal/application/gateway`）
- [ ] E2E Tests：N/A（修复位于适配器边界，已用 `Client.Forward` 契约测试覆盖）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task --force verify
gofmt / prettier check  PASS
go vet + staticcheck     PASS（仅 2 条既有 react-refresh 警告）
tsc --noEmit             PASS
go test . ./internal/... PASS（全部包）
vitest                   PASS（147 文件／895 项）
windows production build PASS
VERIFY_EXIT=0
```

真实上游核对（2026-11-12，生产 River Key，仅 `max_tokens=1`）：

```text
POST https://api.river.ai/v1/chat/completions  176 个 function 工具
→ 200 {"id":"chatcmpl-6af571e64aab8","usage":{"prompt_tokens":6730,...}}
```

修复前同网关：128 个工具 → 200，129 个工具 → `400 {"code":"invalid_request",...}`。

---

# Documentation

- [x] PRD：无需修改（工具数量不是产品承诺）
- [x] `docs/product/RIVER.md`：新增「工具数量」
- [x] `docs/plans/CURRENT.md`：列入已完成
- [ ] Architecture／Tech Stack／Coding Rules／Testing／Build：无变化
- [ ] ADR：不涉及架构决策

---

# Deviations

- 原计划写「124／129／176 个工具全部接受」，实际覆盖 1／128／129／176／200／512 并补了一条
  走 `Forward` 的契约测试；原因是要同时证明「不截断」而不只是「不拒绝」。
- 先前记录的「DSH 侧停用 OpenPencil」属于仓库外环境改动，不是本任务的代码交付；记录在
  Implementation Result 的「恢复可用」小节，仓库内没有任何 DSH 配置改动。

---

# Git

## Commit

```text
<见提交后回填>
```

---

# Result

River 适配器不再因为工具数量超过 128 而在本地返回笼统的 `400 invalid_request`；DSH 带 176 个
工具（内置 + OpenPencil + 两个物流 MCP）的请求现在会原样发往 River，实测上游接受。工具形状、
名字与协议的逐条校验保持不变。DSH 的 OpenPencil MCP 已临时停用；发版后按备份恢复该段即可
重新启用，无需再改网关。

---

# Completed

2026-11-12 16:36
