# TASK-20261024-005: 关闭思考映射为 low、Google 名称去重、River 模型声明图片

## Status

In Progress（实现与本地验证完成，Beta.31 打包中）

## Created

2026-10-24 04:20（Asia/Singapore）

## Request

用户三点反馈：

1. `DeepSeek-V4.1-Flash-1017` 的 `none` 档位改为映射到 `low`。
2. Google 的 Gemini 模型在菜单里还是显示一大堆（重名多行、档位变体行）。
3. `DeepSeek-V4.1-Flash-1017` 支持图像输入，但 Codex 提示「此模型不支持图像输入」。

## Goal

- Codex 档位映射表改为 `none → low`、`max → xhigh`，请求期反向还原。
- 目录构建时折叠档位变体，并让重名条目改用模型 ID 显示。
- River 条目在 Codex 目录声明 `text + image`，客户端不再拦截图片。

## Background

- Beta.30 已实现档位词汇映射与折叠，但折叠只在**模型同步**时发生；未同步时菜单仍是旧样子。
- 实测目录里 4 个条目共用显示名 `Gemini 3.5 Flash Lite`，但 ID 不同
  （`gemini-2.5-flash`、`gemini-2.5-flash-lite`、`gemini-2.5-flash-thinking`、`gemini-3.5-flash-lite`），
  客户端菜单出现四行完全相同的选项。
- `codexInputModalities` 之前只对 `openai` 放行图片，River 条目一律写 `text`，
  Codex 因此在本地拦截图片；River 适配器已按官方契约转发 `image_url` / `input_image`。

## Scope

- `internal/core/modelcatalog/codex_effort.go`：`none → low`。
- `internal/core/modelcatalog/names.go`（新增）：`UniqueNames` 重名回退到模型 ID。
- `internal/application/routing/model_entries.go`：目录构建时折叠变体 + 去重名称。
- `internal/core/modelcatalog/codex.go`：River 条目按已确认能力声明图片。
- 相关测试与文档；新增 Beta.31 发布说明。

## Out of Scope

- 不改上游协议，不新增图片格式支持。
- 不合并显示名相同但 ID 不同的型号（它们是不同路由目标）。

---

# Implementation Result

- `codexEffortAlias`：`none → low`、`max → xhigh`；`CodexUpstreamEffort` 反向还原 `low → none`、`xhigh → max`，
  未确认档位仍明确报错。
- `UniqueNames`：重名条目（大小写不敏感）改用模型 ID 作为显示名，不修改入参切片。
- `routing.catalog`：`Normalize(UniqueNames(FoldEffortVariants(entries)))`，因此折叠与去重不再依赖模型同步。
- `codexInputModalities`：`openai` 与 `river` 在目录已声明 `image` 时输出 `text + image`；
  其他适配器保持 `text`。
- 测试更新：档位映射与默认档位、River 图片声明（含 text-only 与未验证适配器边界）、重名回退。

# Verification

```text
gofmt -l internal cmd main.go       空
go test . ./internal/...            PASS（全部包）
task --force verify                 （Beta.31 打包前执行，见 TASK-20261024-006）
```

未执行：实机 Codex 菜单与真实图片请求（需用户本地账号与客户端）。

# Remaining Risks

- 客户端菜单仍取决于客户端自身是否渲染 effort 控件；本任务只保证目录词汇与请求链路正确。
- River 四款模型的图片能力来自官方能力表，未逐张实测；`system`／`assistant` 消息中的图片仍由上游按官方限制拒绝。
- 显示名改用 ID 后菜单更技术化，但每一行都可区分。

# Git

## Commit

```text
94985cb feat(gateway): Codex 透明转发、网关自有登录与 WebSocket 握手预算
e8f0756 feat(frontend): 激活失败原因、Codex 运行时标签与请求页字段
90681e0 docs: 同步 Codex 透明转发决策、产品文档与 2026-10 任务记录
```

代码随 94985cb 落库；记录随 90681e0 归档。

---
