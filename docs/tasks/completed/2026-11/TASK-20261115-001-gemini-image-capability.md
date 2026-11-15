# TASK-20261115-001: Google 订阅 Gemini 声明图片输入并接受工具结果图片

## Status

Completed

## Created

2026-11-15 09:30（Asia/Singapore）

## Request

> 谷歌的这个模型应该是支持输入图片的啊，你看看是什么情况，为什么它提示不行
>
> 做出修复吧，你做

用户截图（Antigravity 桌面端，模型 `Gemini 4 Hydrogen Extra` 即 `gemini-4-hydrogen-xhigh`）
显示 agent 汇报：Mobbin／OpenPencil MCP 不可用之外，「主代理与子代理（含 pro／flash）通过
`read_file` 读取 PNG 均只收到占位提示而无图像像素」，因此无法目视核对 `before.png` 等参考图。

## Goal

Antigravity 桌面端（以及其它读同一份目录的客户端）不再把 Google 订阅的 Gemini 当纯文本模型：
模型目录发布 `supportsImages: true`，客户端可以附图；agent 工具结果里的内联图片也能转成上游
`inlineData` 正常到达 Gemini。

## Background

- Antigravity 桌面端的模型目录来自网关 `GET /v1/models?format=antigravity`
  （`models.listSource`），字段包含 `supportsImages`；客户端按该字段决定是否允许附图、
  以及 agent 是否把读到的图片当作像素发给模型。
- 网关当前把 Google 订阅模型一律声明为纯文本：`internal/provider/antigravity/discovery_models.go`
  的 `entry.Metadata = modelcatalog.Metadata{InputModalities: []string{"text"}, ToolUse: &tools}`。
  本地数据库中 `gemini-4-hydrogen-*` 的 `metadata_json` 因此都是
  `{"inputModalities":["text"],"toolUse":true}`。
- 图片通路本身已实现并验证过：`chat_image.go` 会把用户消息里的 `image_url`（data URL）转成
  `inlineData`；TASK-20261113-007 在隔离实例上用真实 1×1 PNG 验证返回 200，模型把像素描述成
  “solid coral-pink image”。v2.2.3-beta.18 发行说明也写明：桌面端隐藏附件按钮的原因就是
  `supportsImages`，可以再调整。
- 仍未覆盖的一环：`chat_request.go` 只对 `role: user` 调 `userParts`，`role: tool` 走
  `textContent`，工具结果里的图片会直接 400。agent 读图（`read_file`、截图工具）正是通过
  工具结果回到模型，且网关自己的 Responses／Messages 转换（`agentprotocol.toolResultContent`）
  本来就会产出「工具结果 content 数组内含 `image_url`」的 Chat 形态。

## Scope

- Google 订阅的 Gemini 模型在模型目录声明 `text` + `image` 输入（Claude／GPT-OSS 通道未验证，
  保持纯文本）。
- Antigravity 的 Chat 整形接受工具结果里的内联图片：转成同一 user 回合
  `functionResponse` 之后的 `inlineData` part；system／assistant 回合里的图片仍明确拒绝。
- 更新受影响单元测试与文档。

## Out of Scope

- 不处理音频、视频、文件类 part。
- 不下载远程 `http(s)` 图片。
- 不为 Claude／GPT-OSS 通道声明图片能力（上游未验证）。
- 不改客户端（Antigravity）本体，也不改任何已生成到本机的客户端配置；用户侧需要重新同步
  官方模型目录并重启桌面端才会拿到新标记。

## Existing Behavior

- 目录：Google 订阅模型 `supportsImages: false`；Antigravity 不给附图入口，agent 读图只得到占位文本。
- Chat：`role: user` 的内联图片转 `inlineData`；`role: tool` 内容带图片时返回
  `400 / unsupported_content`。

## Expected Behavior

- 目录：Gemini 订阅模型 `supportsImages: true`；重新同步后客户端可以附图。
- Chat：用户消息与工具结果的 data URL 图片都转 `inlineData`；不可转换的图片形式仍明确失败。

---

# Analysis

## Related Modules

- `internal/provider/antigravity/discovery_models.go`（模型目录能力声明）
- `internal/provider/antigravity/chat_image.go`、`chat_request.go`（Chat 整形）
- `internal/core/modelcatalog/antigravity.go`（`supportsImages` 输出）
- 受影响测试：`agent_models_test.go`、`chat_image_test.go`
- 文档：`docs/product/GOOGLE_SUBSCRIPTION.md`、`docs/plans/CURRENT.md`

## Dependencies

无新增依赖。

## Risks

- `supportsImages: true` 是能力广告：若某个 Gemini 变体实际不收图，客户端仍会失败。TASK-007 已在
  真实账号上确认 `gemini-4-hydrogen-high` 能收到并识别图片；其余 Gemini 变体同一通道，未逐档实测。
- 工具结果图片转换依赖上游接受同一 user 回合里的 `functionResponse` + `inlineData`；需隔离实例
  真实请求确认，不能只靠形状推断。
- 已有账号的目录元数据存在本地数据库里，改代码不会自动更新；必须重新「同步官方模型」，否则
  客户端仍读到 `supportsImages: false`。

---

# Implementation Plan

1. 创建本记录。
2. `discovery_models.go`：Gemini 模型声明 `text` + `image`，补注释说明依据。
3. `chat_image.go`：新增工具结果内容解析（文本 + 内联图片），复用 `inlineImage`。
4. `chat_request.go`：`role: tool` 使用新解析，`functionResponse` 后追加 `inlineData`。
5. 更新／新增单元测试（目录能力、工具结果图片、system／assistant 仍拒绝）。
6. 构建隔离实例（生产数据副本、独立端口），用真实 1×1 PNG 验证工具结果图片请求。
7. `task verify`。
8. 更新文档与看板，提交并回填本记录。

---

# Progress

- 2026-11-15 09:30：创建记录，确认根因与图片通路现状。
- 2026-11-15 09:40：完成目录能力声明与工具结果图片转换，补齐单元测试；`go test . ./internal/...` 通过。
- 2026-11-15 09:46：隔离实例（生产数据副本、端口 3011）真实验证三组请求全部 200。

---

# Implementation Result

## Changes

- `discovery_models.go`：Google 订阅发现的 Gemini 型号声明 `InputModalities: ["text","image"]`
  （Claude／GPT-OSS 仍为 `["text"]`），模型目录随之输出 `supportsImages: true`。
- `chat_image.go`：新增 `toolParts`，按用户消息同一套白名单／容量边界解析工具结果的文本与
  `image_url`／`input_image` 块。
- `chat_request.go`：`role: tool` 改用 `toolParts`，`functionResponse` 之后追加 `inlineData`
  part；system／assistant 回合仍只接受文本。
- 测试：`agent_models_test.go` 按 Gemini／第三方分别断言能力；`chat_image_test.go` 新增
  工具结果带图、纯图片工具结果、工具结果不可用图片与未知块仍失败三项。

## Files Changed

- `internal/provider/antigravity/discovery_models.go`
- `internal/provider/antigravity/chat_image.go`
- `internal/provider/antigravity/chat_request.go`
- `internal/provider/antigravity/agent_models_test.go`
- `internal/provider/antigravity/chat_image_test.go`
- `docs/product/GOOGLE_SUBSCRIPTION.md`
- `docs/plans/CURRENT.md`

## Technical Notes

- 工具结果映射到上游的 user 回合，`functionResponse` 与 `inlineData` 同回合共存；
  `mergeTurns` 本来就让带 `functionResponse` 的回合保持独立，因此图片不会漂到相邻回合。
- 网关自己的 Responses／Messages 转换（`agentprotocol.toolResultContent`）早就会产出
  「工具结果 content 数组内含 `image_url`」的 Chat 形态；本次让 Antigravity 的 Chat 整形
  接受同一形态，两条路径不再分叉。
- 能力声明只表示「允许发送」：目录更新需要重新「同步官方模型」并重连桌面端才生效，
  实际识别仍由上游决定。

---

# Verification

## Tests

- [x] Unit Tests：`go test ./internal/provider/antigravity/` 通过（含新增 3 项工具结果图片用例）。
- [x] Go 全量：`go test . ./internal/...` 通过。
- [x] 隔离实例真实请求（生产数据副本、`OMNIPROXY_DATA_DIR` 指向副本、端口 3011、
      未触碰运行中的应用；账号 `acct_58e1f41a…`）：

```text
user-image         HTTP 200 content=Pink
tool-text-control  HTTP 200 content=I cannot determine the dominant color because ...
tool-image         HTTP 200 content=Pink
```

  对照组说明：同一工具轮里只有文本时模型明确说拿不到图片；加上 `inlineData` 后直接读出
  像素颜色（Pink），证明工具结果的图片确实到达 Gemini，而不是被丢弃。

- [ ] Integration Tests：N/A
- [ ] E2E Tests：Antigravity 桌面端复测待用户安装新构建后执行。

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify（结果见下）

## Verification Result

```text
task --force verify  →  EXIT=0
format check / lint / typecheck 通过
go test . ./internal/...         全部包通过
frontend                          154 files / 935 tests passed
windows production build          bin/omniproxy.exe（24,176,128 bytes）
```

---

# Documentation

- [x] `docs/product/GOOGLE_SUBSCRIPTION.md`：能力边界与目录字段
- [x] `docs/plans/CURRENT.md`
- [ ] `docs/releases/`：随下一版本发行说明

---

# Git

## Commit

```text
1d7120c4 feat(antigravity): 声明 Gemini 图片输入并接受工具结果图片
```

---

# Result

Google 订阅的 Gemini 不再被网关声明成纯文本模型：模型目录发布 `supportsImages: true`，
Antigravity 可以附图，agent 工具结果里的图片也会转成上游 `inlineData` 并真实到达模型。
已有账号需要重新同步官方模型并重连桌面端才会看到新能力。

---

# Completed

2026-11-15 10:02（Asia/Singapore）

