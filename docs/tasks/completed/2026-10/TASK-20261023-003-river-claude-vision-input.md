# TASK-20261023-003: River 视觉输入接入（Claude Code / Desktop 与原生 Responses）

## Status

Completed

## Created

2026-10-23 10:00

## Request

用户反馈：`deepseek-v4.1-flash-1017` 官方支持视觉识别（DeepSeek 文档 `图像理解` 页
与 `模型 & 价格` 页均标注 Flash 支持图像理解），但 Claude Code Desktop 经过 OmniProxy
使用 River 版本时发送图片报错。用户询问是否为链路未接入，并要求参照 River 文档修复。

原始需求要点：

- 核对 DeepSeek 官方与 River 官方的图片输入契约。
- 定位 OmniProxy 当前在哪个环节拒绝图片。
- 按文档修复该链路。

## Goal

让 River 已确认支持图片的模型在 OmniProxy 的 Messages 与 Responses 两条链路上
真正把图片传给上游，而不是本地 400 拒绝；目录元数据如实反映图片能力，使客户端
与用户都能看到该路由支持图片。

## Background

River 官方文档（`/guides/inference-models/`）声明四款模型均支持文本与图片输入；
`/guides/inference-images/` 明确使用 Chat Completions 的
`{"type":"image_url","image_url":{"url":"data:image/jpeg;base64,..."}}` 块。
本项目 PRD／RIVER 此前明确写为「首版仅文本」，属于已知未接入能力。

## Scope

- River 模型元数据 `InputModalities` 增加 `image`。
- Messages 协议转换：`image` 块 → Chat `image_url` 块（含工具结果中的图片）。
- River Responses 原生校验：允许 `input_image`，拒绝无法验证的文件／音频块。
- 目录对 Claude Desktop／Claude CLI 公布图片能力（Codex 维持原策略）。
- 定向回归测试与产品文档同步。

## Out of Scope

- 直连 DeepSeek 适配器（keeper 模型没有图像理解能力，且其 Responses／Anthropic
  端点为原样透传，不需本地解析）。
- 音频、文件（`file_id`、`input_file`）输入。
- Codex `input_modalities` 对 River 放开（保持「只公布原生订阅图片」的既有结论）。
- 真实 River Key 的端到端计费与图片 Token 实测。

## Existing Behavior

- `internal/provider/river/catalog.go`：`InputModalities: []string{"text"}`。
- `internal/core/agentprotocol/request.go:68` 与 `types.go:35`：遇到
  `image`／`input_image`／`image_url` 直接返回 `ErrUnsupportedImage`。
- `internal/application/gateway/content_failure.go` 将其转成
  `400 / unsupported_content` + 固定文案。
- `internal/provider/river/validation.go:textContent` 拒绝 Responses 中的任何图片块。
- 因此 Claude Code／Desktop 发出的截图或图片历史在本地即被拒绝，根本不会到达 River。

## Expected Behavior

- River 目标模型在 Messages／Responses 中收到图片时，把图片原样（`detail` 与 URL
  不变）转成上游 Chat `image_url` 块，工具结果中的图片同样保留。
- 图片块结构非法、或为未验证类型时返回明确的 400，不静默删除内容。
- 目录 `inputModalities` 对 River 包含 `image`；Claude Desktop 模型菜单能据此识别视觉能力。

---

# Analysis

## Related Modules

- `internal/provider/river`（元数据、Responses 校验）
- `internal/core/agentprotocol`（Messages／Responses → Chat 转换）
- `internal/core/modelcatalog`（Desktop 与 Codex 目录输出）
- `internal/application/gateway`（错误分类）

## Dependencies

无新增依赖。

## Risks

图片请求体显著变大（base64）。Responses 有 16 MiB 容量；Messages 有 16 MiB 容量。
Chat 外部入口仍为 2 MiB，不放宽。需要确认转换后仍做容量校验。

---

# Implementation Plan

1. River 元数据标记图片能力，并补充对应回归。
2. `agentprotocol` 增加图片块转换：Messages `image`（base64／url），
   Responses `input_image`，工具结果内的图片；保留 `detail`；越界时明确失败。
3. River Responses 校验允许 `input_image`，拒绝 `input_file`／`input_audio`。
4. `modelcatalog` 为 Claude 目录（CLI 与 Desktop）公布图片能力，Codex 策略不变。
5. 定向测试 + `task verify`，同步 RIVER／PRD／GATEWAY／CURRENT。

---

# Progress

- 已定位根因：本地协议转换与 Responses 校验双重拒绝图片，与 River 官方能力不一致。
- 网关集成测试又暴露 `core/proxy` 纯文本前置判定，补齐后链路完整。
- 已完成实现、定向回归、完整质量门与产品文档同步。

---

# Implementation Result

## Changes

- `river.ModelMetadata` 对四款已核对模型标注 `text` + `image`；DeepSeek 两版与
  Nemotron 官方文档明确为视觉模型，GLM-5.3-Flash 依 models 页能力表标注。
- `agentprotocol` 新增图片转换：Messages `image` 块与 Responses `input_image` 块统一
  输出为 Chat `image_url`（`{"url":...}`；存在 `detail` 时一并保留）。工具结果中的
  图片转成 `content` 数组，保持 `tool_call_id` 关联。
- 图片源只接受已确认形式：base64（需要 `media_type` 与 `data`）、`url`、
  Responses `image_url` 字符串或对象。其它形式返回 `ErrUnsupportedImage`，不猜测。
- River Responses 原生校验改用 `supportedContent`：允许 `input_image`，
  仍然拒绝 `input_file`、`input_audio`、专用工具等未接入内容。
- `modelcatalog` 为 Messages 协议（Claude CLI／Desktop）公布上游已确认的 `image`
  能力；Codex 继续只承认原生订阅线路的图片直通。

## Files Changed

- `internal/provider/river/catalog.go`、`validation.go`、`image_boundary_test.go`
- `internal/core/agentprotocol/images.go`（新增）、`request.go`、`types.go`、
  `image_conversion_test.go`（新增）、`protocol_test.go`
- `internal/core/proxy/chat_text.go`、`request.go`、`chat_image_test.go`（新增）、`chat_text_test.go`
- `internal/core/modelcatalog/claude.go`、`claude_image_test.go`（新增）
- `internal/application/routing/river_test.go`、`internal/application/gateway/river_image_test.go`（新增）
- `docs/product/RIVER.md`、`PRD.md`、`GATEWAY.md`、`LOCAL_AGENTS.md`、`docs/plans/CURRENT.md`

## Technical Notes

- 不改变 Chat 外部入口 2 MiB 上限；Messages／Responses 继续使用既有 16 MiB 有界容量。
- 不静默删除图片，也不把图片降级成文本描述。

---

# Verification

## Tests

- [x] Unit Tests
- [ ] Integration Tests
- [ ] E2E Tests

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task verify BIN_DIR=bin/verify-task-20261023-003
PASS（退出码 0）
```

格式检查（gofmt、前端 format）、`go vet`、`staticcheck`、TypeScript、全部 Go 测试、
121 文件／724 项前端测试、Wails 绑定生成与 Windows production 构建全部通过。
日志：`tmp/verify-task-20261023-003.log`（忽略，不提交）。

定向验证：

- PASS：`go test ./internal/core/agentprotocol ./internal/core/proxy ./internal/core/modelcatalog ./internal/provider/river ./internal/application/routing ./internal/application/gateway -count=1`。
- PASS：`go test ./internal/... . -count=1`（无失败包）。
- PASS：`TestRiverForwardsClaudeImagesToUpstream` 覆盖 Messages／Responses × 两版 Flash，
  经真实网关服务（临时数据库 + 合成上游）确认图片以 `image_url`／`input_image` 到达上游，
  且凭据字节已清零。
- PASS：`git diff --check`。

## Verification Notes

- 门禁存在既有 `jsdom scrollTo` 未实现提示和大 chunk 警告，均未导致失败，不在本轮扩展修复。
- 未使用真实 River Key，未发送真实上游推理请求；图片计费与真实客户端验收不在本轮结论内。
- 原生客户端（Claude Code Desktop）未重启验证；目录需重新同步或重新应用配置后才显示图片能力。
- 过程记录：初次实现遗漏了 `core/proxy` 的 `normalizeChatText` 与 `validateMessages`
  纯文本判定，网关集成测试以 `invalid_messages` 暴露后补齐；`river.imageBlockValid`
  首次写成了字符串 `image_url` 提前返回，导致 `detail` 校验被跳过，已由定向用例修正。

---

# Documentation

- [x] PRD
- [x] Architecture（无需变更）
- [x] Tech Stack（无需变更）
- [ ] Coding Rules
- [ ] Testing
- [ ] Build
- [ ] ADR
- [x] CURRENT

---

# Deviations

原计划只改 `agentprotocol` 与 River 适配器。实际发现 `internal/core/proxy` 的
`normalizeChatText`／`validateMessages` 仍以「content 必须是字符串」为前提，会在适配器
之前把合法图片请求判为 `invalid_messages`，因此一并最小放宽该层（仍拒绝未知块、
超量与破坏字段）。这是链路完整性所必需，不是顺带重构。

---

# Git

## Commit

```text
94985cb feat(gateway): Codex 透明转发、网关自有登录与 WebSocket 握手预算
e8f0756 feat(frontend): 激活失败原因、Codex 运行时标签与请求页字段
90681e0 docs: 同步 Codex 透明转发决策、产品文档与 2026-10 任务记录
```

代码随 94985cb 落库；记录随 90681e0 归档。

---

# Result

River 图片输入链路已接通：Claude Code／Desktop 的 Messages 与 Responses 图片（含截图
工具结果）不再被本地 400 拒绝，而是按 River 官方 `image_url` 契约转发给上游；目录把
四款模型标记为 `text` + `image`。音频、文件（Files API）与 Codex 的 `input_modalities`
保持原策略。真实 Key 计费与原生客户端验收未做，不宣称已通过。

---

# Completed

2026-10-23 18:25
