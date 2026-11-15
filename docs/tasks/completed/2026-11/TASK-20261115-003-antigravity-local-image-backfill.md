# TASK-20261115-003: Google 订阅通道回填被客户端丢弃的工具结果图片

## Status

Completed

## Created

2026-11-15 12:40（Asia/Singapore）

## Request

> 还是不行，他说：`read_file` 本身没有返回报错（调用状态是成功的），之所以看不到图片，
> 是因为工具返回给模型上下文的内容里只有纯文本占位符，没有注入任何图像多模态数据。
> 你看看能不能辅助你进行修复，如果有问题，你也可以自己去问 gemini。

用户截图：Antigravity 桌面端（`Gemini 4 Hydrogen Extra` = `gemini-4-hydrogen-xhigh`，
`wireProtocol: openai`）里 agent 用 `read_file` 读 `blind_v4.png`，只拿到
`Showing binary file of type image/png`，随后明确回答「看不到图片像素」。

## Goal

Antigravity 在网关模式下读图时，即使客户端把工具结果里的图片换成了占位文本，模型也能
真正看到像素：网关按同一会话里 `read_file` 的路径回读本地文件并补成 `inlineData`。

## Background

排查结论（2026-11-15 12:20–12:36）：

- 客户端**确实产出了图片**：会话 `b06b83bb-201c-4403-ae6b-7dcf8765e51d` 的
  `transcript.jsonl` step 2 里既有占位文本，也有
  `"media":[{"mime_type":"image/png","uri":"file:///…/logs/media/step_2_media_0.png"}]`
  （图片本体 7598 字节已落盘）。
- 但发往网关的请求里没有图片：同一步下一次请求 `input_tokens` 14137 → 14466（+329）；
  同一张图经网关实测占 1100+ tokens（`prompt_tokens` 1173，且模型答对了底色、`CODE:`
  后的字符和三个图形）。
- 网关目录声明正确：用客户端自己的 key 拉
  `GET /v1/models?format=antigravity`，Hydrogen 四个档位 `supportsImages` 全为 `true`；
  `account_models.metadata_json` 也是 `{"inputModalities":["text","image"],...}`。
- 因此丢图点在客户端：`wireProtocol: openai` 下它把 tool result 的 media 剥成了占位文本。
  客户端闭源改不了；但占位文本旁边的 tool call 参数里带着文件路径，网关能自己回读。

## Scope

- Google 订阅通道（`internal/provider/antigravity`）的 Chat 整形：工具结果只剩
  `Showing binary file of type image/…` 占位文本时，用同一 `tool_call_id` 的调用参数里的
  绝对路径回读文件，嗅探确认为允许的图片格式后补成 `inlineData`。
- 边界：文件必须是绝对路径、普通文件、≤12 MiB、内容嗅探为 png/jpeg/gif/webp；任何一步
  不满足就保持原样（纯文本占位），不报错、不改变已有语义。

## Out of Scope

- 不改客户端、不改客户端配置；不做 OCR／图片转文字。
- 不回填非图片文件、不读网络地址、不递归解析目录。
- 不处理浏览器截图走别的消息形态（当前客户端把截图也并入 tool result，同一条路径覆盖）。

## Existing Behavior

- 工具结果里的 `image_url`／`input_image` 块会转成 `inlineData`（TASK-20261115-001）。
- 工具结果只有占位文本时，模型只能看到那句话。

## Expected Behavior

- 占位文本 + 同一轮 `read_file(Path=…)` 指向存在的图片时，请求里出现该图片的 `inlineData`，
  模型能看到像素。
- 路径不存在、不是图片、超限或没有对应调用参数时，行为与现在完全一致。

---

# Analysis

## Related Modules

- `internal/provider/antigravity/chat_request.go`（工具轮组装、tool_call 索引）
- `internal/provider/antigravity/chat_image.go`（`inlineImage`／`toolParts`，图片白名单）
- 新增 `internal/provider/antigravity/chat_backfill.go`

## Risks

- 网关读本机文件：只读「同一会话里 agent 自己刚 `read_file` 过的绝对路径」，且必须嗅探为
  图片才内联，非图片内容不会被读出。
- 占位文本可能来自别的工具：回填仍要求存在同 id 的调用参数与可解析路径，否则不动。
- 图片体积：文件上限 12 MiB，base64 后仍低于 `chatImageLimit`（16 MiB）。

---

# Implementation Plan

1. 创建本记录。
2. `chat_request.go`：把 tool_call 索引从「id→名字」扩成「id→名字+参数」。
3. 新增 `chat_backfill.go`：占位识别、参数取路径、文件嗅探、`inlineData` 组装。
4. 工具轮组装时：内容里没有图片且命中占位文本才尝试回填。
5. 单元测试：命中回填、非图片不回填、缺文件不回填、无调用参数不回填。
6. `go test ./internal/provider/antigravity/`，再跑 `task verify`。
7. 更新文档与看板，提交并回填本记录。

---

# Progress

- 2026-11-15 12:40：创建记录，完成证据链与方案确认。
- 2026-11-15 12:52：实现回填链路与单元测试，`go test ./internal/provider/antigravity/` 通过。

---

# Implementation Result

## Changes

- `chat_request.go`：tool_call 索引由「id→名字」扩成「id→`pendingCall{name, arguments}`」；
  tool 结果装配完成后，若回合里没有图片且模型属于 Gemini 族，按调用参数尝试回填。
- 新增 `chat_backfill.go`：
  - `backfillInlineImage`：只认 `Showing binary file of type image/…` 占位文本；从参数取绝对
    路径，`os.Stat` 普通文件、≤12 MiB，`http.DetectContentType` 嗅探为 PNG／JPEG／GIF／WebP
    后才读文件并转成 `inlineData`；任何一步不满足返回 `nil`，请求保持原样。
  - `callPathCandidates`/`pathCandidates`：大小写不敏感地匹配 `path`／`file_path`／`file` 等
    参数名，兼容客户端把路径再包一层引号或再转义一次反斜杠的编码层次。
- 新增 `chat_backfill_test.go`：5 组用例覆盖命中回填、单层参数、非图片文件、缺文件／相对
  路径／无路径参数／文本结果／其它二进制占位，以及 Claude／GPT-OSS 不回填。

## Files Changed

- `internal/provider/antigravity/chat_request.go`
- `internal/provider/antigravity/chat_backfill.go`（新增）
- `internal/provider/antigravity/chat_backfill_test.go`（新增）
- `docs/product/GOOGLE_SUBSCRIPTION.md`
- `docs/plans/CURRENT.md`

## Technical Notes

- 回填的图片与 `toolParts` 产出的图片走同一个 `inlineData` 形状，落在同一 user 回合
  `functionResponse` 之后；`mergeTurns` 已保证带 `functionResponse` 的回合独立。
- 只在 Gemini 族生效，与目录声明（`supportsImages: true` 仅 Gemini）保持一致。
- 网关只重读「同一会话里 agent 自己刚 `read_file` 过的绝对路径」，且必须嗅探为图片才内联；
  非图片内容不会被读出，因此没有引入额外的本机文件访问能力。
- 历史轮次每次都会重新带上同一张图（与客户端原生行为一致），Token 成本按图片张数线性增长。


---

# Verification

## Tests

- [x] Unit Tests：`go test ./internal/provider/antigravity/` 通过（新增 5 组回填用例）。
- [x] Go 全量：`go test . ./internal/...` 全部包通过。
- [x] 网关侧图片通路复核（生产实例、客户端自己的 key）：同一张 `blind_v4.png` 经
      `/v1/chat/completions` 直发 `gemini-4-hydrogen-xhigh` 返回 200，`prompt_tokens` 1173，
      模型答出「浅绿色 / Z3ZVB9 / 深蓝正方形、红色菱形、绿色圆形」，与图片内容一致；
      对照客户端同轮请求只多 329 Token，证明丢图点在客户端而非网关。
- [ ] Integration Tests：隔离实例真实请求（本任务只改整形层，未新增上游形态）。
- [ ] E2E Tests：Antigravity 真机复测待用户安装新构建后执行。

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task verify  →  EXIT=0
format check / lint（0 errors）/ typecheck 通过
go test . ./internal/...    全部包通过
frontend                     154 files / 935 tests passed
windows production build     bin/omniproxy.exe（24,187,392 bytes）
```

---

# Documentation

- [x] `docs/product/GOOGLE_SUBSCRIPTION.md`：客户端缺口与回填边界
- [x] `docs/plans/CURRENT.md`
- [ ] `docs/releases/`：随下一版本发行说明

---

# Git

## Commit

```text
（见下方回填）
```

---

# Result

Antigravity 网关模式丢图这一环现在由网关兜底：工具结果只剩二进制占位文本时，网关按同一
`tool_call_id` 的参数路径回读本地图片，嗅探确认为 PNG／JPEG／GIF／WebP 且 ≤12 MiB 后补成
同一回合的 `inlineData`；仅对声明了图片输入的 Gemini 族生效，其余情况保持原样。客户端侧
根因（`wireProtocol: openai` 下剥掉 tool result 的 media）仍未修复，需要客户端更新后本兜底
才会自然失效。

---

# Completed

2026-11-15 13:06（Asia/Singapore）