# TASK-20261114-003: DuMate 模型声明图片输入能力

## Status

Completed

## Created

2026-11-14 20:20（Asia/Singapore）

## Request

用户问：「另外 Dumate 的模型应该是支持图片的吧？」——此前 DuMate 在 OmniProxy 里只被当成纯文本模型。

## Goal

- 让 DuMate 的两个预设别名在模型目录与管理客户端配置里声明 `image` 输入。
- 只改能力声明，不发布未经确认的上下文／输出容量，也不新增模型别名。

## Background

- `internal/provider/dumate/dumate.go` 的 `ModelMetadata` 目前只返回 `InputModalities: ["text"]`，注释写明「只发布可以确认的文本能力」。
- 该元数据会流向受管客户端目录（`agentconfig/client_modalities.go` → harness／opencode／pi／codex／claude），决定客户端是否允许发图。
- 搭子桌面端自己的模型目录（`%APPDATA%\qianfan-desktop-app\qianfan_desk_xdg\<账号>\config\opencode\opencode.json`）对 `model-text`、`dm-auto-model/text.L0` 以及 `model-mm`、`model-mm-fallback` 等别名都声明了 `image` 输入；`model-text` 为 `text`+`image`。
- 网关的 Chat 请求整形（`internal/core/proxy/chat_text.go`）明确保留 `image_url` 部分；Responses／Messages 走 `agentprotocol.ToChat`，也支持 `data:`／URL 图片。因此阻断点只有能力声明。

## Scope

- `internal/provider/dumate/dumate.go`：`ModelMetadata` 声明 `text` + `image`。
- 更新 `internal/application/routing/dumate_test.go` 的模态断言。
- 任务记录与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不新增 `model-mm`／`model-mm-fallback`／`dm-auto-model/*.L1` 等别名。
- 不发布上下文窗口、输出上限等未验证容量。
- 不改上游请求形状、协议转换、路由或前端。
- 不使用真实 DuMate 账号发请求（需用户另行授权）。

## Existing Behavior

DuMate 模型在目录与客户端配置里只有 `text` 输入，客户端不会把图片发给它。

## Expected Behavior

DuMate 两个别名声明 `text` + `image` 输入；发图请求按既有路径原样转发或转换到上游，容量仍保持未知。

---

# Analysis

## Related Modules

- `internal/provider/dumate/dumate.go`
- `internal/application/routing/dumate_test.go`
- `internal/application/agentconfig/client_modalities.go`（消费方）

## Dependencies

无新增依赖。

## Risks

- 能力声明来自桌面端自带目录而非真实调用；若上游拒绝图片，客户端会收到上游错误而不是本地 `unsupported_content`。
- 只加 `image` 不改容量，避免客户端据此截断或超发。

---

# Implementation Plan

1. `ModelMetadata` 返回 `[]string{"text", "image"}`，注释说明依据。
2. 更新路由测试断言为 `text` + `image`。
3. 运行 dumate／routing 相关测试与全仓 Go 测试。
4. 更新 CURRENT，提交。

---

# Progress

- 2026-11-14 20:20：确认阻断点是能力声明；上游请求路径本身不拦截图片。
- 2026-11-14 20:28：改完元数据与路由断言，目标包与全仓 Go 测试通过。
- 2026-11-14 20:30：format／lint／build 通过，完成记录。

---

# Implementation Result

## Changes

- `internal/provider/dumate/dumate.go`：`ModelMetadata` 的输入模态从 `["text"]` 改为 `["text", "image"]`，注释记录依据（桌面端自带模型目录）与仍然不发布容量的原因。
- `internal/application/routing/dumate_test.go`：目录断言改为 `text` + `image`，容量仍必须未知。

## Files Changed

- `internal/provider/dumate/dumate.go`
- `internal/application/routing/dumate_test.go`
- `docs/tasks/…/TASK-20261114-003-dumate-image-modality.md`
- `docs/plans/CURRENT.md`

## Technical Notes

- 受管客户端目录（harness／opencode／pi／codex／claude）通过 `agentconfig.clientInputModalities` 读取该元数据，改完后这些客户端才会把 DuMate 模型当作可发图模型。
- Chat 请求的 `image_url` 部分在 `normalizeChatText` 中本来就是「含图片的数组原样保留」，Responses／Messages 也会转换图片；无需改转发路径。

---

# Verification

## Tests

- [x] Unit Tests：`go test ./internal/application/routing/ ./internal/provider/dumate/ ./internal/application/agentconfig/ ./internal/application/modelcatalog/ ./internal/core/modelcatalog/` 全部 ok。
- [x] Integration Tests：全仓 `go test . ./internal/...` 通过。
- [ ] E2E Tests：真实 DuMate 账号发图未执行（需要用户授权，会产生一次真实积分消耗）。

## Quality Checks

- [x] Format：`task format:check` exit 0
- [x] Lint：`task lint` exit 0（仅 2 条既有 warning）
- [x] Type Check：`task typecheck` 之前一轮已通过，本轮未改前端
- [x] Test：前端 154 文件／935 项在上一任务已通过，本轮只改 Go
- [x] Build：`task build` exit 0

## Verification Result

```text
go test ./internal/application/routing/ ./internal/provider/dumate/ \
        ./internal/application/agentconfig/ ./internal/application/modelcatalog/ \
        ./internal/core/modelcatalog/                  → 全部 ok
go test . ./internal/... -count=1                      → exit 0
task format:check                                      → exit 0
task lint                                              → exit 0
task build                                             → exit 0
```

---

# Documentation

- [x] CURRENT：已追加本轮变更摘要
- [x] 其余文档无需修改

---

# Deviations

None.

---

# Git

## Commit

见完成记录。

---

# Result

DuMate 的两个预设别名在模型目录与受管客户端配置里声明 `text` + `image` 输入，客户端可以把图片发给它们；容量仍未公布。上游是否真的接受图片需要一次真实账号请求确认。

---

# Completed

2026-11-14 20:30（Asia/Singapore）

