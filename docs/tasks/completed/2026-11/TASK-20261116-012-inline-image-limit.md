# TASK-20261116-012: 放行内联 data 图片到组件上限，修复 Codex 工具结果图片被本地拒成 unsupported_content

## Status

Completed

## Created

2026-11-16 15:26（Asia/Singapore）

## Request

> 又报错了，图片这个模型是支持的呀，请求id：d91090a0b2e8567ff9697c1329edcc6c

用户附 Codex 截图：消息里出现「已查看 1 张图像」提示，随后报 `unsupported_content`
（请求 `d91090a0b2e8567ff9697c1329edcc6c`）。

## Goal

让 Codex 的图片工具结果（内联 base64 data URL）能够通过网关到达 River，
不再被本地校验误判成 `unsupported_content`。

## Background

请求记录：`DeepSeek-V4.1-Flash-1017`、`responses`、`codex`、`400 unsupported_content`、
耗时 161 ms（本地拒绝，未发往上游）。

Codex 线程 `01a1e338-648e-7763-bdb0-ea6c785955c9` 的 rollout 显示该回合先调用了
`view_image`，工具结果为：

```json
{"type":"function_call_output","call_id":"call_6afaaf0d2186f_0",
 "output":[{"type":"input_image","image_url":"data:image/png;base64,iVBORw0KG…","detail":"high"}]}
```

该条目 1,014,497 字节，其中 `image_url` 是约 1 MB 的内联 PNG data URL。

River 适配器的 `imageBlockValid` 对所有图片地址统一套用 `len(url) > 8192` 上限（原本用于
远程 URL 的反滥用边界），因此把这条合法的内联图片判成无法表达 → `unsupported_content`。
`docs/product/RIVER.md` 与组件其它图片路径（`agentprotocol` 的 `imageLimit`）都以
**16 MiB** 为单图上限，本次校验与该口径不一致。

## Root Cause

`internal/provider/river/validation.go` 的 `imageBlockValid` 只有 8 KiB 的 URL 上限，
没有区分内联 `data:` 图片与远程 URL；Codex 0.169 的 `view_image` 结果使用 1 MB 级内联
base64，因此任何带截图工具结果的请求都会在本地被拒。

## Scope

- `internal/provider/river/validation.go`：图片地址校验按形式分档——`data:image/` 内联图片
  放宽到组件上限 16 MiB 的字符长度，其它地址保持 8 KiB；仍拒绝空地址、控制字符与非图片
  data URL。
- 测试：内联大图（1 MiB）在用户消息与工具结果两种位置均放行；超过 16 MiB 的内联图片、
  超长远程 URL、非图片 data URL 仍明确失败。
- 文档与版本：`GATEWAY.md`／`RIVER.md`（如需要）、`v2.3.1-beta.13` 发行说明、
  `CURRENT.md`、`version.go`、`frontend/package.json`；随后构建 `v2.3.1-beta.13`。
- 真实 Key 验证：用合成的大尺寸 PNG（约 1 MB）复放工具结果图片，确认上游正常完成。

## Out of Scope

- 不放宽远程 URL 的 8 KiB 边界，不放宽请求体 16 MiB 上限。
- 不改 Codex 入口以外的图片处理策略，不改 Chat／Messages 转换路径的既有上限。
- 不把聊天历史里的图片改写或压缩；不做静默删图。

## Existing Behavior

Codex 的 `view_image` 工具结果带 1 MB 级内联图片时，网关在 161 ms 内本地返回
`400 unsupported_content`，请求没有到达 River。

## Expected Behavior

- 内联 `data:image/...` 图片在 16 MiB 字符以内原样转发到上游，用户消息与工具结果都适用。
- 空地址、控制字符、非图片 data URL、超过 16 MiB 的内联图片、超过 8 KiB 的远程 URL
  仍返回 `400 unsupported_content`，不静默删内容。

---

# Analysis

## Related Modules

- `internal/provider/river/validation.go`（`supportedContent`、`imageBlockValid`）
- 参照口径：`internal/core/agentprotocol/images.go`（`imageLimit = 16 << 20`）
- 测试：`internal/provider/river/image_boundary_test.go`

## Dependencies

N/A

## Risks

- 放宽后单张内联图片最多占 16 MiB 请求体额度；Responses 入口本就是 16 MiB 上限，
  不会被放大。
- 仍需真实 Key 验证 River 对 1 MB 级内联图片的行为；若上游拒绝，需要按上游口径收窄。

---

# Implementation Plan

1. 记录本任务与证据。
2. `imageBlockValid` 拆出 `imageURLValid`：`data:image/` 与远程 URL 分档校验。
3. 测试：内联大图放行、各类越界仍失败；工具结果位置覆盖。
4. 真实 Key 复放合成大图，确认上游完成。
5. 文档、版本、`task verify`、打包 `v2.3.1-beta.13`。

---

# Progress

- 2026-11-16 15:24：由请求记录与 rollout 定位到 `view_image` 工具结果的 1 MB 内联图片。
- 2026-11-16 15:26：建立本记录。

---

# Implementation Result

## Changes

- `internal/provider/river/validation.go`：新增 `imageURLValid` 与两个上限常量
  （`imageRetrievableURLLimit = 8192`、`imageInlineLimit = 16 << 20`），
  `imageBlockValid` 的字符串与对象两种 `image_url` 形式都改用它：内联 `data:image`
  图片放宽到 16 MiB 字符，远程地址保持 8 KiB；空地址、控制字符、非图片 data URL 仍拒绝。
- 测试：`image_boundary_test.go` 新增 `TestResponsesInlineToolResultImageIsAccepted`
  （1 MiB 内联图片作为 `function_call_output` 数组块，经完整适配器链路放行且正文不变）
  与 `TestImageURLValidBounds`（内联／远程边界、非图片 data URL、空地址、控制字符）。
- 文档：`docs/product/GATEWAY.md`、`docs/product/RIVER.md`、
  `docs/releases/v2.3.1-beta.13.md`、`docs/plans/CURRENT.md`、版本号 `2.3.1-beta.13`。

## Files Changed

- `internal/provider/river/validation.go`
- `internal/provider/river/image_boundary_test.go`
- `internal/version/version.go`、`frontend/package.json`
- `docs/product/GATEWAY.md`、`docs/product/RIVER.md`、`docs/plans/CURRENT.md`
- `docs/releases/v2.3.1-beta.13.md`（新增）

## Technical Notes

- 上限口径与组件其它图片路径（`internal/core/agentprotocol` 的 `imageLimit`）保持 16 MiB；
  Responses 请求体上限本来就是 16 MiB，放宽图片地址不会放大请求。
- 远程 URL 的 8 KiB 边界保留：签名 URL 远小于该值，避免把任意正文伪装成长地址。
- 没有引入新的错误码：越界与非法形式继续返回 `unsupported_content`。

---

# Verification

## Tests

- [x] Unit Tests：`TestImageURLValidBounds`、
  `TestResponsesInlineToolResultImageIsAccepted`、
  `TestUnsupportedContentStillFailsExplicitly`、
  `TestResponsesImageInputReachesUpstreamUnchanged` 全部通过。
- [x] Integration Tests：完整 `Client.Forward` 链路，断言内联图片原样到达上游。
- [x] E2E Tests：真实 River Key + 生产数据隔离实例复放合成 1 MB PNG 的
  `view_image` 工具结果：两次 `200` + `response.completed`，模型答出图片颜色（`灰色`）。

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

对照（同一份 1 MB 图片工具结果正文）：

| 场景 | 已安装构建 | 新构建隔离实例 |
| --- | --- | --- |
| 合成 600×600 PNG（1,080,993 字节 → 1,441,346 字符 data URL） | `400 unsupported_content`／352 ms，未到上游 | `200` + `response.completed`（两次），模型答「灰色」 |
| 小图（现有用例） | 200 | 200 |
| 超 16 MiB 内联 / 超 8 KiB 远程 / 非图片 data URL | 400 | 400（未放宽） |

未验证：安装构建后由真实 Codex 桌面端在同一截图会话重试；需要安装并重启 Codex 确认。

---

# Documentation

- [x] `docs/product/GATEWAY.md`：Codex 内联图片的放行范围
- [x] `docs/product/RIVER.md`：Responses 图片地址按形式分档
- [x] `docs/releases/v2.3.1-beta.13.md`：根因、修复与验证边界
- [x] `docs/plans/CURRENT.md`：当前状态
- [x] 其它文档无需修改

---

# Deviations

- 原计划只“放宽上限”。实际把校验拆成 `imageURLValid` 并按形式分档：远程 URL 保持
  8 KiB（避免放宽后把任意长正文当图片地址），只有 `data:image/` 内联图片用 16 MiB。
- 验证未使用用户截图，而是生成高熵合成 PNG，既能复现 1 MB 级内联图片，又避免把用户
  屏幕内容再次上传。

---

# Git

## Commit

```text
fd6859d7 fix(river): 放行内联 data 图片到组件上限，修复 Codex 工具结果图片被误拒
```

---

# Result

Codex `view_image` 工具结果里的内联图片不再被本地误拒：River 的 Responses 图片校验改为
按形式分档（内联 `data:image` 16 MiB、远程 URL 8 KiB），合成 1 MB PNG 复放从
`400 unsupported_content` 变为两次 `200` + `response.completed`，模型能答出图片颜色；
`task verify` EXIT=0。

---

# Completed

2026-11-16 15:40（Asia/Singapore）



