# TASK-20261118-005: Google 订阅通道记录缓存命中（cachedContentTokenCount）

## Status

Completed

## Created

2026-11-18 13:20（Asia/Singapore）

## Request

用户贴出另一款工具对同一 Antigravity 订阅流量的记录截图：`gemini-4-carbon-high` 的请求显示
缓存命中 94.9%、96.1%、91.6%……，要求我们查清「我们这个记录怎么没有」缓存命中。

## Goal

Antigravity（Google 订阅）通道把上游 `usageMetadata.cachedContentTokenCount` 映射进合成
Chat 响应的 `usage.prompt_tokens_details.cached_tokens`，使新请求的缓存命中量、命中率、
成本估算与累计用量都能显示真实值；上游没给该字段时仍然保持未知（`—`），不伪造 0。

## Background

本机 `request_records`（只读抽查，近 7 天）：

| 提供商 | 协议 | 请求数 | 缓存已知 | 缓存 > 0 |
| --- | --- | --- | --- | --- |
| Google | chat | 4,510 | 0 | 0 |
| River | chat | 3,421 | 3,379 | 3,355 |
| OpenAI | responses | 1,536 | 1,493 | 1,438 |

即：其他通道的缓存明细正常，Google 订阅通道一条都没有；今天 `gemini-4-hydrogen-xhigh`
2,195 条请求，`cached_input_tokens` 全为 `NULL`。

官方语义（Google 文档）：`promptTokenCount` 是**包含**缓存内容的完整提示词长度，
`cachedContentTokenCount` 是其中被缓存命中的子集 —— 与 OpenAI Chat 的
`prompt_tokens` / `prompt_tokens_details.cached_tokens` 语义一一对应。

## Scope

- `internal/provider/antigravity/chat_response.go`：读取并校验 `cachedContentTokenCount`，
  在合成 usage 中写 `prompt_tokens_details.cached_tokens`。
- Antigravity 适配器测试：非流式、流式、缺字段、真实零命中、越界值。
- 相关文档与 `docs/plans/CURRENT.md` 同步。

## Out of Scope

- 不回填历史记录：上游用量没有落盘，旧记录继续显示 `—`（与迁移 037 之前的口径一致）。
- 不改 Google API Key 通道：那条链路是透明转发，是否有缓存明细取决于上游 OpenAI 兼容
  响应本身，本次不动。
- 不改前端展示逻辑：缓存列、命中率、成本区间已经就绪，只等后端给出已知值。
- 不重新构建安装包（如需发版另开构建任务）。

## Existing Behavior

Antigravity 适配器只读取 `promptTokenCount`、`candidatesTokenCount`、
`thoughtsTokenCount`、`totalTokenCount` 四个字段，合成 Chat usage 时只写
`prompt_tokens` / `completion_tokens` / `total_tokens`。`core/proxy` 的
`cachedInputUsage` 在 Chat 协议下只认 `prompt_tokens_details.cached_tokens` 与
`prompt_cache_hit_tokens`，两者都取不到 → `CachedInput = nil` → 请求库写 `NULL` →
列表与用量页显示「—」、缓存命中率显示下界或「还没有请求报告缓存明细」。

## Expected Behavior

- 上游返回 `cachedContentTokenCount`（含 0）时：请求详情显示真实缓存量与命中率
  （如 `缓存 50,560 / 输入 53,300`、`94.9%`），成本按「未缓存部分全价 + 缓存部分折扣价」估算。
- 上游缺少该字段时：仍显示「—」，不补零。
- 缓存量大于提示词总数（上游自相矛盾）时：按未知处理，不写入错误比例。
- 负数或异常大值：按 `invalid_upstream_response` 拒绝，与既有用量校验一致。

---

# Analysis

## Related Modules

- `internal/provider/antigravity/chat_response.go`（合成 usage 的唯一出口）
- `internal/core/proxy/usage_cache.go`、`usage.go`（Chat 协议解析与子集校验）
- `internal/application/gateway/google_agent.go`（Responses／Messages 也在这条转换后观察用量）
- `internal/application/activity/*`（入库、累计、成本估算，无需改动）

## Dependencies

无新增依赖。

## Risks

- 上游 `usageMetadata` 在 Gemini 4 上存在已知的间歇性字段缺失／掉 0 行为（社区报告），
  因此「上游没给」必须继续表示未知，不能当成 0 写死。
- Claude／GPT-OSS 经 Antigravity 也走同一信封；本任务只增加一个子集字段并加
  `cached <= prompt` 保护，不改变已有的输入／输出计数。

---

# Root Cause

Google 订阅通道的响应是我们自己转换出来的：`nativeResponse.Usage` 结构体没有
`cachedContentTokenCount` 字段，`s.usage` 也只拼三个计数，于是上游明明返回了缓存命中量，
网关在写库前就把它丢掉了；解析侧没有任何可读字段，只能记为未知。

## Fix

在 `chat_response.go` 中把 `cachedContentTokenCount` 读成可空整数，校验非负且不超过
提示词总数后写入 `prompt_tokens_details.cached_tokens`；缺失时保持不写，由解析侧记为未知。

## Regression Prevention

新增用例直接断言 `proxy.ParseUsage` 能从转换后的响应里读到 `CachedInput`，
覆盖非流式、流式终帧、缺字段、零命中与越界五种情况，避免以后再次丢掉该字段。

---

# Implementation Plan

1. 扩展 `nativeResponse` 的 usage 结构与校验，加入可空缓存计数。
2. 合成 usage 时按协议写 `prompt_tokens_details.cached_tokens`（仅在有值且 ≤ 提示词数时）。
3. 流式路径保留已报告的缓存量，终帧缺字段时不把已知值抹成未知。
4. 补适配器测试，并用 `proxy.ParseUsage` 验证端到端映射。
5. `task verify`，更新文档与 `CURRENT.md`，提交并回填。

---

# Progress

- 2026-11-18 13:30 定位根因：转换层未读取 `cachedContentTokenCount`；本机近 7 天 Google chat
  4,510 条记录缓存明细全为 `NULL`，其他通道正常。
- 2026-11-18 13:34 适配器改为读取并可空映射该字段，新增 6 项用例；`go test` 三个相关包通过。
- 2026-11-18 13:43 `task verify` EXIT=0（前端 159 文件／977 项测试、wails3 生产构建通过）。

---

# Implementation Result

## Changes

- `nativeResponse.Usage` 增加可空 `Cached *int64`（`cachedContentTokenCount`），与其它计数
  一起做负数／上限校验；负数按 `invalid_upstream_response` 拒绝。
- `responseState` 保存最近一次报告的缓存量：流式逐帧累计，终帧省略该字段时保留已报告值，
  不把已知命中抹成未知。
- 新增 `chatUsage` 合成器：缓存量 ≤ `promptTokenCount` 时写
  `prompt_tokens_details.cached_tokens`，缺失或大于提示词总数时不写，保持未知。
- 新增 6 项适配器用例，其中三项直接调用 `proxy.ParseUsage`，确保转换后的正文能被
  请求记账读到 `CachedInput`。

## Files Changed

- `internal/provider/antigravity/chat_response.go`
- `internal/provider/antigravity/chat_usage_test.go`（新增）
- `docs/product/ACTIVITY.md`、`docs/product/GOOGLE_SUBSCRIPTION.md`
- `docs/plans/CURRENT.md`

## Technical Notes

- Gemini 的 `promptTokenCount` 是包含缓存内容的完整提示词长度，`cachedContentTokenCount`
  是其中的子集（官方 `UsageMetadata` 文档），因此映射到 OpenAI Chat 的
  `prompt_tokens` / `prompt_tokens_details.cached_tokens` 语义一致，输入不需要再加一次缓存。
- 社区已报告 Gemini 4 会间歇性缺少 `cachedContentTokenCount`（含 0 值掉字段），所以
  「缺失 → 未知」必须保留；真实 0 仍显示 0%。
- 历史记录无法回填：上游用量当时就被丢弃，旧行继续显示 `—`，与迁移 037 之前的口径一致。

---

# Verification

## Tests

- [x] Unit Tests：`internal/provider/antigravity` 新增 6 项（非流式映射、缺字段、真实零命中、
  缓存大于提示词、负数拒绝、流式终帧缺字段保留），其中 3 项直接断言 `proxy.ParseUsage`
  能读到 `CachedInput`
- [x] Integration Tests：Go 全包 `go test . ./internal/...`（含 gateway、activity、core/proxy）
- [ ] E2E Tests：未做真实订阅调用（会消耗订阅额度，需用户授权）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task verify
PASS (EXIT=0)
format:check  gofmt 无待格式化文件；prettier 全部通过
lint          go vet / staticcheck 通过；eslint 0 errors（2 个既有 warnings）
typecheck     tsc --noEmit 通过
test          Go 全包通过；前端 159 files / 977 tests passed
build         wails3 生产构建通过（bin/omniproxy.exe）
```

说明：本会话的受限沙箱里前端测试无法启动（Vite 子进程 `spawn EPERM`），因此质量门在
完整权限的同一命令下重跑并以上述结果为准；Go 侧在受限沙箱中也已单独通过。

---

# Documentation

- [x] `docs/product/ACTIVITY.md`
- [x] `docs/product/GOOGLE_SUBSCRIPTION.md`
- [x] `docs/plans/CURRENT.md`

---

# Deviations

None.

---

# Git

## Commit

```text
Pending
```

---

# Result

Google 订阅通道不再丢缓存：`usageMetadata.cachedContentTokenCount` 现在写进合成 Chat
响应的 `usage.prompt_tokens_details.cached_tokens`，新请求会显示真实缓存命中率、
命中量与成本折扣；上游没给该字段、或缓存量大于提示词总数时保持未知，历史记录继续显示
`—`。本机已知受影响范围（近 7 天 4,473 条 Google 订阅请求）从下一版起恢复可见。

---

# Completed

2026-11-18 13:45（Asia/Singapore）
