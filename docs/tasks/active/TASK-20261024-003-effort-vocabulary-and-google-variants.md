# TASK-20261024-003: 推理档位按客户端词汇发布并折叠 Google 变体型号

## Status

In Progress（实现与本地验证完成，等待实机客户端验收）

## Created

2026-10-24 03:10（Asia/Singapore）

## Request

用户反馈两个问题：

1. River 的 `DeepSeek-V4.1-Flash-1017` 在客户端无法选择推理强度。
2. 客户端模型列表里 Google 的 `(High)`／`(Low)`／`(Medium)` 其实是推理档位，应在网关处理；Codex 里选了档位就能传到网关。

## Goal

- 目录向客户端发布的档位使用客户端自己的词汇，档位选择在 Codex／Claude 客户端可用。
- Google 系按档位拆分的型号折叠成一行模型，档位由客户端 effort 选择，网关请求时换成实际变体 ID。
- 旧选择与新映射都不产生 `unsupported_model`／`unsupported_reasoning_effort` 回归。

## Background

- 实机证据（`%USERPROFILE%\.codex\omniproxy-models.json`）：`DeepSeek-V4.1-Flash-1017` 发布
  `supported_reasoning_levels = [none, high, max]`，`default = high`。Codex 只认
  official catalog 的 `minimal…xhigh`，`none`／`max` 不可选，选择器只剩「高」。
- `%APPDATA%\OmniProxy-dev\omniproxy.db` 的 `account_models` 显示 Google 目录用独立型号表达档位：
  `gemini-3.8-flash-high/low/medium`（显示名 `Gemini 3.8 Flash (High/Low/Medium)`）、
  `gemini-3-flash-agent`（`Gemini 3.5 Flash (High)`）、`gemini-3.5-flash-low`（`Gemini 3.5 Flash (Medium)`），
  另有 4 个不同 ID 但显示名同为 `Gemini 3.5 Flash Lite` 的条目。
- 请求记录显示用户当前直接用变体型号发起请求（`input_model = gemini-3.8-flash-low`，`protocol=messages`）。
- 订阅适配器 `antigravity` 的生成参数是白名单校验，`reasoning_effort` 会被判为 `unsupported_content`，
  这就是档位此前只能靠"换型号"表达的原因。

## Scope

- `internal/core/modelcatalog`：Codex 档位映射（`codex_effort.go`）、变体折叠与反向解析
  （`effort_variants.go`）、`Metadata.EffortModels` 及校验。
- `internal/provider/antigravity`、`internal/provider/google`：发现目录后折叠变体。
- `internal/application/gateway`：Responses 档位还原（River）、档位 → 变体型号改写、
  旧变体 ID 回退解析；路由决策提取为 `resolvePreview` 以便失败后重试一次。
- `internal/application/agentconfig/claude_picker.go`：折叠后的非 River 模型也声明 effort 能力。
- 新增 ADR-022 与发布说明。

## Out of Scope

- 不改上游协议本身，不新增托管工具或多模态能力。
- 不为 Google 增加新的上游字段（继续用已存在的变体型号表达档位）。
- 不改动客户端 UI 代码；档位的显示仍取决于客户端是否渲染 effort 控件。

---

# Analysis

## Related Modules

- `internal/core/modelcatalog/{codex,codex_effort,effort_variants,model,claude_picker}.go`
- `internal/provider/antigravity/discovery.go`、`internal/provider/google/models.go`
- `internal/application/gateway/{routing,reasoning,river,effort_variant}.go`

## 关键判断

- 档位词汇不匹配是 `1017` 无法选择的根因：`none`／`max` 不在 Codex 词汇表内。
- Google 变体是可验证的上游事实（目录里真实存在这些型号），把它们折叠成档位映射
  比假设「基础型号接受 `reasoning_effort`」更可靠。
- 反向解析必须保留：用户已保存 `gemini-3.8-flash-low` 这类选择，折叠后菜单里没有该行。

## Risks

- 档位就近映射不是语义等价（`none` → `minimal`、`max` → `xhigh`），文案必须如实说明。
- 折叠在下次同步模型后生效；同步前客户端仍看到变体行。
- 同一显示名对应多个不同 ID 的条目（`Gemini 3.5 Flash Lite` ×4）不会合并，
  它们确实是不同型号，合并会改变路由目标。
- 实机客户端 UI 是否渲染 effort 控件仍取决于客户端自身；本任务只保证目录词汇与请求链路正确。

---

# Implementation Result

- `internal/core/modelcatalog/codex_effort.go`：`CodexEfforts`／`CodexUpstreamEffort`／`UpstreamEffort`
  统一映射与还原；未确认档位不发布、不猜测。
- `codex.go`：目录档位与默认档位改用映射后的值。
- `internal/core/modelcatalog/effort_variants.go`：`FoldEffortVariants` 折叠 `(High)／(Low)／(Medium)／(Minimal)`
  及中文变体后缀；`EffortModel`／`EffortLevels`／`EffortVariantBase` 提供请求期与历史兼容解析。
- `model.go`：新增 `Metadata.EffortModels` 及边界校验（档位名与变体 ID）。
- `antigravity/discovery.go`、`google/models.go`：发现后折叠，只发布基础模型。
- `gateway/reasoning.go`：Messages 档位改用 `UpstreamEffort`（同时接受上游原生值与 Codex 档位）；
  新增 `withResponsesEffort` 还原 River Responses 档位。
- `gateway/river.go`：Responses 请求按模型元数据还原档位。
- `gateway/effort_variant.go`：`carriedEffort` 读取三种协议的档位，`applyEffortVariant`
  换成变体型号并移除档位字段，`applyLegacyVariant` 在路由失败时还原旧变体 ID。
- `gateway/routing.go`：抽出 `resolvePreview`，未知模型时尝试一次旧 ID 还原后重新解析。
- `agentconfig/claude_picker.go`：折叠后的模型（有 `EffortModels`）也写入 `effortLevel`／`maxEffortLevel`。

# Verification

```text
gofmt -l internal cmd main.go          空
go build ./internal/... .              PASS
go test . ./internal/...               PASS（全部包）
```

新增用例：Codex 档位映射与默认档位、折叠与孤立型号保留、同档位保留首个映射、
`EffortModels` 边界校验、旧变体 ID 反向解析、三协议档位读取与改写、变体型号替换后移除档位字段。

未执行：实机 Codex／Claude 客户端档位菜单复核；Google 真实请求（需要真实账号额度）。

---

# Remaining Risks

- 需要在实机确认：Codex 选择器是否出现 `minimal／low／medium／high／xhigh`；
  Google 型号是否只剩一行；档位选择后上游收到的型号是否为对应变体。
- 折叠要等一次模型同步才体现在客户端菜单。

# Git

## Commit

```text
94985cb feat(gateway): Codex 透明转发、网关自有登录与 WebSocket 握手预算
e8f0756 feat(frontend): 激活失败原因、Codex 运行时标签与请求页字段
90681e0 docs: 同步 Codex 透明转发决策、产品文档与 2026-10 任务记录
```

代码随 94985cb 落库；记录随 90681e0 归档。

---
