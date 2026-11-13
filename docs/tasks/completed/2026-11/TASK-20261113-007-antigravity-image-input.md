# TASK-20261113-007: Antigravity 发图片被本地 400 unsupported_content 拒绝

## Status

Completed

## Created

2026-11-13 10:05（Asia/Singapore）

## Request

> 我给 gemini 发了一个图片，返回了错误：
> Error: request failed (code 400): {"error":{"code":"unsupported_content","message":"unsupported_content","type":"gateway_error"}}
> 它应该是支持图片的，不应该报这个错

## Goal

- Antigravity 桌面端（Chat Completions）里的内联图片（`data:` base64）能转换到上游
  Gemini `inlineData` 并正常返回。
- 无法转换的图片形式（远程 URL、非白名单格式、超限、位于 system／assistant／工具结果）
  仍然明确失败，不静默丢弃、不猜测。
- 模型目录按模型真实能力发布 `supportsImages`，界面不再允许发送必然失败的图片。

## Background

- Antigravity 请求走 `internal/provider/antigravity/chat_request.go` 的 `chatPayload`，
  它把每条消息的 `content` 交给 `textContent`（`chat_message.go`）：只接受字符串或
  `type=text` 的 part，其余一律 `400 unsupported_content`。
- 桌面端发送的图片是 OpenAI 形态的内联块，例如
  `{"type":"image_url","image_url":{"url":"data:image/png;base64,..."}}`；
  其它协议（Responses／Messages）在 `internal/core/agentprotocol/images.go` 已经转成同样的
  `image_url` 块，所以缺口只在 Antigravity 的整形层。
- 上游 Code Assist 的 `request.contents[].parts[]` 与 Gemini `Part` 同构，支持
  `{"inlineData":{"mimeType":"image/png","data":"<base64>"}}`。

## Scope

- 新增图片 part 解析（白名单 MIME、base64、单图上限）。
- `chatPayload` 的用户消息同时收集文本与图片 parts；其它角色仍拒绝图片。
- `supportsImages` 目录字段按账号模型元数据修正（如与实际不符）。
- 单元测试 + 隔离实例真实请求验证。

## Out of Scope

- 不下载远程图片 URL（不引入额外出网请求）。
- 不改变其它提供商的图片策略。
- 不处理音频、视频、文件类 part。

## Existing Behavior

`{"type":"image_url",...}` → 400 `unsupported_content`，请求未到上游。

## Expected Behavior

内联 base64 图片转为 `inlineData` 发往上游；文本与图片同一条消息时都保留；
不可转换的图片返回带说明的 `unsupported_content`。

---

# Analysis

## Related Modules

- `internal/provider/antigravity/chat_request.go`、`chat_message.go`
- `internal/core/agentprotocol/images.go`（上游协议已产出的 `image_url` 形态）
- `internal/core/modelcatalog/antigravity.go`（`supportsImages`）

## Dependencies

无新增依赖。

## Risks

- 图片体积：请求正文上限已由 `MaxResponsesRequestBytes` 约束；单张图片再按 16 MiB base64
  上限拒绝，避免构造超大上游请求。
- 上游是否真的接受 `inlineData`（Code Assist 包封）需要真实请求确认，不能只靠形状推断。

---

# Implementation Plan

1. 创建本记录。
2. 在 antigravity 包新增 `userParts`：解析文本 + `image_url`（data URL）→ `inlineData`。
3. `chatPayload` 使用它；system／assistant／tool 仍只接受文本。
4. 补单元测试（成功、无文本仅图片、非白名单 MIME、远程 URL、超限、system 图片）。
5. 隔离实例发一张真实小图片确认上游返回 200，并核对 `supportsImages`。
6. `task verify`，提交并构建本地 Beta。

---

# Progress

- 2026-11-13 10:05：创建记录；确认拒绝点在 `textContent`。

---

# Implementation Result

## Changes

- 新增 `internal/provider/antigravity/chat_image.go`：`userParts` 解析用户消息内容，
  文本与 `image_url`／`input_image` 块分别转成 Gemini `text` 与 `inlineData` part；
  `inlineImage` 只接受 `data:` URL（白名单 MIME、base64 校验、单图 16 MiB 上限），
  远程 `http(s)` 图片按图片能力边界返回带说明的 `unsupported_content`。
- `chat_request.go`：用户消息改用 `userParts`，其它角色仍走 `textContent`，因此
  system／assistant／工具结果里的图片依旧明确失败；只有图片、没有文本的消息不再被判空。
- 测试 `chat_image_test.go`：内联图片转 `inlineData`、纯图片消息、六类不可用形式、
  非用户回合图片、未知内容块、历史图片保留。

## Files Changed

- `internal/provider/antigravity/chat_image.go`（新增）
- `internal/provider/antigravity/chat_image_test.go`（新增）
- `internal/provider/antigravity/chat_request.go`
- `docs/product/GOOGLE_SUBSCRIPTION.md`

## Technical Notes

- 上游 `request.contents[].parts[]` 与 Gemini `Part` 同构，`inlineData` 直接可用；
  `detail` 字段只服务本地视觉模型，不转发给 Gemini。
- `supportsImages` 仍按账号模型元数据发布（当前 Google 侧快照没有模态字段，因此为 false）；
  不猜测能力，避免把图片发给不支持的通道。

---

# Verification

## Tests

- [x] Unit Tests：`go test ./internal/provider/antigravity/` 通过（含 6 项图片用例）。
- [x] 隔离实例真实图片请求（生产数据副本、端口 3011、未触碰运行中的应用）：
      非流式 200 且回答 `OK`；流式 200，同一 1x1 PNG 被上游描述为
      “solid coral-pink image”，证明 `inlineData` 真的到达模型。
- [x] Go 全量：`go test . ./internal/...` 通过。
- [x] Verify：见下。

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
隔离实例（OMNIPROXY_PROFILE=development，生产数据副本，端口 3011）：
POST /v1/chat/completions  model=gemini-4-hydrogen-high  stream=false  内含 1x1 PNG → 200 "OK"
POST /v1/chat/completions  model=gemini-4-hydrogen-high  stream=true   内含 1x1 PNG → 200（3 个 data 分片）
```

---

# Documentation

- [x] `docs/product/GOOGLE_SUBSCRIPTION.md`：改写图片能力边界
- [x] CURRENT
- [ ] `docs/releases/`：随下一版本发行说明

---

# Git

## Commit

```text
task --force verify
VERIFY_EXIT=0（148 文件 / 902 项前端测试，全部 Go 包与 Windows production 构建）
```

---

# Result

Antigravity 桌面端发送的内联图片不再被本地 400 拒绝：用户消息里的图片转成上游
`inlineData` 正常返回；无法内联的图片形式仍然明确失败，不会静默丢弃。

---

# Completed

2026-11-13 09:40
