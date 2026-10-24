# TASK-20261023-005: 本地 Agent 客户端目录公布图片能力

## Status

Completed

## Created

2026-10-23 19:10（Asia/Singapore）

## Request

用户反馈：DeepSeek Harness 里相同模型的图片能力没有接上，需要处理。

## Goal

让接入 OmniProxy 的本地 Agent 客户端（Harness Web／Desktop、OpenCode、Pi、Goose 及兼容客户端）在模型目录中按上游已确认的能力声明图片输入，使客户端允许发送图片；不虚报未知能力。

## Background

TASK-20261023-003 已修复网关层的图片转发（Messages `image`、Responses `input_image`、工具结果截图都会转成 River 的 `image_url`）。但客户端配置层仍有四处硬编码纯文本，客户端据此在本地禁止附图。

Harness 官方文档明确：`input` 省略或为空时继承安装目录，再回落到路由 `defaultInput`（默认 `[text]`）；**显式非空即优先**。当前实现显式写 `[text]`，等于主动声明该模型不支持图片。

## Scope

- `internal/application/agentconfig/harness.go`、`opencode.go`、`pi.go`、`goose.go`：按 `Entry.Metadata.InputModalities` 输出各客户端字段。
- 未确认能力的模型保持原纯文本默认。
- 定向回归与产品文档同步。

## Out of Scope

- 网关图片转发链路（已由 TASK-20261023-003 完成）。
- Codex 的 `input_modalities` 策略（仍只认原生订阅直通）。
- 新增音频／文件输入。
- 真实客户端发送图片的端到端验收。

## Existing Behavior

- Harness：`"input": []string{"text"}`。
- OpenCode 及兼容客户端：`modalities.input: [text]`。
- Pi：`input: []string{"text"}`。
- Goose：`supports_vision: false`。

## Expected Behavior

- 上游已确认 `image` 的模型：Harness／Pi 输出 `input: [text, image]`，OpenCode 输出 `modalities.input: [text, image]`，Goose 输出 `supports_vision: true`。
- 未确认图片能力的模型维持原输出，不因其他模型支持图片而放宽。

---

# Analysis

## Related Modules

- `internal/application/agentconfig`（四个客户端的构建函数）
- `internal/core/modelcatalog`（提供 `InputModalities`）

## Risks

各客户端字段名不同，写错会让配置被拒绝。已按官方文档逐一核对：
Harness `input`、OpenCode `modalities.input`、Goose `supports_vision`；Pi 与 Harness 同为 pi-ai `input`。

---

# Implementation Plan

1. 增加统一的模态判定辅助，只认 `InputModalities` 中已确认的 `image`。
2. 四个构建函数改用该辅助输出对应字段。
3. 增加回归：已确认图片 → 各客户端输出图片能力；未确认 → 保持纯文本。
4. `task verify` 与文档同步。

---

# Progress

- 已按官方文档核对四个客户端的字段名；OpenCode 的 `modalities` 经上游 issue #8875 确认已支持。

---

# Implementation Result

## Changes

- 新增 `internal/application/agentconfig/client_modalities.go`：`clientInputModalities` 只在上游已确认
  `image` 时返回 `[text, image]`，否则维持 `[text]`；`supportsClientImage` 供 Goose 布尔字段使用。
- Harness Web／Desktop 与 Pi（同为 pi-ai）：`input` 改用已确认模态。原先写死 `[text]` 等于主动
  声明不支持图片，压掉了安装目录的继承。
- OpenCode 及 Kilo／Qwen／MiMo Code：`modalities.input` 改用已确认模态。
- Goose：`supports_vision` 按已确认能力输出，不再固定 `false`。

## Files Changed

- `internal/application/agentconfig/client_modalities.go`（新增）、`client_modalities_test.go`（新增）
- `internal/application/agentconfig/harness.go`、`opencode.go`、`pi.go`、`goose.go`
- `docs/product/HARNESS_OPENCODE.md`、`LOCAL_AGENTS.md`、`docs/plans/CURRENT.md`

## Technical Notes

Harness 官方文档明确 pi-ai 的 `input`「显式非空即优先」，省略或为空才继承安装目录，再回落到
路由 `defaultInput`（默认 `[text]`）。因此显式写 `[text]` 是最差选择：既放弃继承，又明确否认图片。
本轮改为如实输出，并保持未确认能力不推断。

---

# Verification

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

- PASS：`go test ./internal/application/agentconfig -run TestLocalAgentCatalogsPublishOnlyConfirmedImageModalities -v`
  覆盖 Harness Web／Desktop、OpenCode、Pi、Goose 各两例（已确认图片／纯文本），并断言同目录
  纯文本模型不会获得图片能力。
- PASS：`go test ./internal/... . -count=1` 全部通过。
- PASS：`task --force verify BIN_DIR=bin/verify-task-20261023-005` 退出码 0；格式、lint、类型检查、
  Go 全套、121 文件／724 项前端测试与 Windows production 构建通过。日志 `tmp/verify-task-20261023-005.log`。
- PASS：`gofmt -l main.go internal` 为空；`git diff --check` 无输出。
- 观察：首次全量 `go test ./internal/... .` 出现过一次 `internal/core/proxy` 的
  `TestLifecycleAndConcurrentStart` 失败；该包与本任务无关，单跑 5 次、`-count=3` 及复跑全量均通过，
  判定为既有时序／端口竞争，未在本任务修改。

---

# Deviations

原计划只改四个构建函数。实际新增独立辅助文件承载模态判定，避免四处重复同一判断；
测试中发现 Goose 的模型落在 `custom_providers/omniproxy.json` 而非 `config.yaml`，
以及 bundle 客户端需先 `snapshotClient` 取 revision，均按既有实现调整测试夹具，未改生产逻辑。

---

# Git

## Commit

Not committed by Agent.

---

# Result

Harness Web／Desktop、OpenCode（含 Kilo／Qwen／MiMo Code）、Pi、Goose 的模型目录现在按上游已确认能力
声明图片输入，客户端不再在本地拦截附图；未确认能力的模型保持纯文本。真实客户端发送图片与上游识别
效果未在本轮验收。
