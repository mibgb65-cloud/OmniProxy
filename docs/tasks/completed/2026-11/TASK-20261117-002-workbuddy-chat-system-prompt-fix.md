# TASK-20261117-002: 修复 WorkBuddy AI 对话被 11128 拒绝（首条必须是 system）

## Status

Completed

## Created

2026-11-17 08:25（Asia/Singapore）

## Request

用户授权真实对话验证（TASK-20261117-001）：用 Free 计划账号发最小请求，结果上游返回
`400 code 11128 "first message is not system prompt"`，并提示「没通过渠道校验」。

## Goal

让 WorkBuddy AI 通道能接受任何对话形态的客户端请求：出站正文首条消息必须是 `system`，
并在上游按渠道校验拒绝（11128）时用「系统提示词下移为用户消息」的形态自动重试一次。

## Background

真实证据（Free 账号，2026-11-17）：

| 请求 | 结果 |
| --- | --- |
| `messages` 只有一条 user | `400` + `code 11128 msg "first message is not system prompt"`，未扣积分 |
| 首条加 `{"role":"system","content":"You are a helpful assistant."}` | `200` + SSE，回答 `OK`，`usage` 26/3/29 |
| 同样带 system、再发一次（43 tokens） | `200` + SSE，回答正常 |

社区插件 0.2.36 里有对应实现（`systemOn` / `systemAsUserOn`）：

- `systemOn`：`developer` 角色改 `system`；`max_completion_tokens` 改名 `max_tokens`；
  首条不是 `system` 时在开头插入 `"You are a helpful assistant."`。
- `systemAsUserOn`：把开头的 system 文本移到首条 user 消息里，用
  `<instructions>…</instructions>` 包裹，开头换成默认 system；聊天中其它 system 就地变成 user。
  插件注释说明：渠道校验只读 system 与 assistant 消息，把系统提示词读成用户消息即可通过；
  这是它在 11128 之后的第二次尝试形态。

我们的适配器 `Forward` 原样转发客户端正文，所以任何首条不是 system 的客户端（含网关把
Responses／Messages 转成 Chat 时的部分形态）都会直接 11128。

## Scope

- `internal/provider/workbuddyai/chatbody.go`（新）：
  `prepareChatBody`（首条 system／developer 归一／`max_completion_tokens`）与
  `systemAsUserBody`（11128 重试形态）。
- `client.go` 的 `Forward`：出站前做 `prepareChatBody`；遇到业务码 11128 时用
  `systemAsUserBody` 重试一次（仍失败才返回 `channel_rejected`）。
- 测试：正文归一（无 system／developer／字段改名／已有 system 不变）、系统提示词下移、
  以及「上游先 11128 后 200」的契约测试。
- 文档：`docs/product/WORKBUDDY.md` 协议边界与错误码说明。

## Out of Scope

- 不实现插件里更激进的「给 agent 开场白插入逗号以躲开整行指纹匹配」（`opened` / `commaPaused`）：
  那属于规避厂商的渠道识别，首版不做，遇到再单独评估。
- 不改网关的协议转换与聚合逻辑。

---

# Implementation Result

## Changes

- 新增 `internal/provider/workbuddyai/chatbody.go`：
  - `prepareChatBody`：`developer` 角色归一为 `system`、`max_completion_tokens` 改名
    `max_tokens`（仅在缺 `max_tokens` 时）、首条不是 `system` 时在开头插入
    `"You are a helpful assistant."`。结构不符或本就合规时**原样返回**，不改写未知字段。
  - `systemAsUserBody`：把开头的系统提示词下移为首条用户消息（`<instructions>…</instructions>`
    包裹），开头换成默认 system；没有可移动内容时返回 nil。
- `client.go` 的 `Forward` 重构为「`sendChat` + 一次重试」：出站前 `prepareChatBody`；
  非 2xx 且业务码为 11128 时用 `systemAsUserBody` 重试一次；仍被拒才返回
  `channel_rejected`。401／429／其它状态的既有语义不变，错误前缀仍会回放给网关。
- 新增 `chatbody_test.go`（7 条）：补齐 system、developer／字段归一、合规正文不变、
  系统提示词下移、无 system 时不重试、11128 后重试成功、重试两次仍拒时返回
  `channel_rejected`（不无限重试）。
- 新增真实账号集成测试 `real_integration_test.go`（构建标签 `workbuddyreal`，默认不运行）：
  用 `WORKBUDDY_REAL_CREDENTIAL` 指向的本机凭据跑一次最小对话，覆盖 `Forward` 全链路。
- `docs/product/WORKBUDDY.md`：协议边界表新增「会话形态」「渠道校验（11128）」两行；
  `channel_rejected` 说明更新；真实验证进度补第 4／5 条。

## Files Changed

- `internal/provider/workbuddyai/chatbody.go`（新）、`chatbody_test.go`（新）、
  `real_integration_test.go`（新，`workbuddyreal` 标签）、`client.go`
- `docs/product/WORKBUDDY.md`

## Technical Notes

- 真实证据：user-only 正文 → `400 code 11128 "first message is not system prompt"`；
  补 system 后 → `200` + SSE，`usage` 26/3/29。
- 重试发生在推理之前（11128 是渠道校验，不产生计费），所以重试成本只有一次请求。
- 未实现插件的「给 agent 开场白插入逗号以躲开整行指纹匹配」（`opened`／`commaPaused`）：
  那属于规避厂商的渠道识别，首版不做，若真实客户端的系统提示词被拒再单独评估。

---

# Verification

## 质量门（`task verify`，2026-11-17 08:17–08:31 Asia/Singapore）

```text
task verify → exit 0
```

| 阶段 | 结果 |
| --- | --- |
| format | gofmt 无输出；prettier 通过 |
| lint | `go vet`／`staticcheck` 无告警；`eslint` 0 error |
| typecheck | `tsc --noEmit` exit 0 |
| test | Go 全包通过；前端 157 files / 965 tests |
| build | 绑定 + vite + Windows production 构建通过 |

## 真实账号验证（本仓库适配器，`workbuddyreal` 标签）

```text
$env:WORKBUDDY_REAL_CREDENTIAL = <本机 DPAPI 解出的凭据 JSON>
go test -tags workbuddyreal ./internal/provider/workbuddyai -run TestRealChatAgainstWorkBuddy -v

=== RUN   TestRealChatAgainstWorkBuddy
    real_integration_test.go: usage: prompt=26 completion=3 total=29
    real_integration_test.go: answer: "OK"
--- PASS: TestRealChatAgainstWorkBuddy (7.31s)
```

入参正文**不含 system**，由 `prepareChatBody` 补齐后上游 `200`，说明修复在真实链路上生效。
凭据临时文件用后立即删除，未写入仓库或日志。

## 未验证

- 真实客户端的系统提示词（如 Codex／Claude Code 的开场白）是否命中上游整行指纹而需要
  11128 重试——重试路径由契约测试覆盖，真实命中情况需要对应客户端实测。
- 付费套餐的积分扣减（见 TASK-20261117-001）。

---

# Documentation

- [x] `docs/product/WORKBUDDY.md`（会话形态、11128 重试、真实验证进度）

---

# Deviations

- 计划只写「补齐首条 system + 11128 重试」，实际还把 `Forward` 拆出 `sendChat`：重试需要
  重新 gzip、重签会话 ID，并保留第一次响应的错误前缀，否则会破坏既有的「非 2xx 原样交给
  网关」约定。
- 额外新增真实账号集成测试（`workbuddyreal` 标签、默认跳过）：没有它就无法证明修复在真实
  上游生效；它不提交任何凭据，只读环境变量指向的文件。
- 未实现插件的「逗号改写」规避手段（见 Out of Scope），保持与 ADR-026 的风险边界一致。

---

# Git

## Commit

b8c5b086 fix(provider): 修复 WorkBuddy AI 对话被 11128 拒绝（首条必须是 system）

---

# Result

上游要求首条消息是 `system`，缺了直接 `400 code 11128 "first message is not system prompt"`
（真实账号实测，未计费）。适配器现在出站前补齐会话形态（system 首条、developer→system、
`max_completion_tokens`→`max_tokens`），并在 11128 时把系统提示词下移为用户消息重试一次，
仍被拒才报 `channel_rejected`。

真实链路已验证：入参不含 system 的最小对话经 `Client.Forward` 得到 `200` + SSE，
回答 `OK`、`usage` 26/3/29。积分扣减仍未在 Free 账号上观测到。

---

# Completed

2026-11-17 08:32（Asia/Singapore）