# TASK-20261023-001: Claude 推理强度改为单模型选择而非多模型条目

## Status

Completed（2026-10-27 随 v2.0.0 正式发布归档）

## Created

2026-10-23（Asia/Singapore）

## Request

用户在 Claude Code Desktop 中使用网关模型时发现：推理强度没有按「一个模型 + 单独选择强度」的方式呈现，而是在模型菜单里出现了一堆不同强度的模型名（例如 `DeepSeek-V4.1-Flash · 高强度推理`、`DeepSeek-V4.1-Flash[1m] · 最高强度推理` 各占一行）。要求按「单一模型 + 单独档位选择」的方式处理。

## Goal

Claude 客户端菜单每个模型只出现一行，推理强度回到客户端原生的 effort 选择（`/effort` 或 `/model` 里的 effort 滑杆），而不是用派生模型名冒充档位；网关继续把用户选定的档位翻译成上游真实参数，绝不伪造未确认的档位能力。

## Background

- TASK-20261022-005 为 River 实现了「显式档位选项」：`internal/core/modelcatalog/claude_effort.go` 为每个已确认档位派生 `omniproxy-effort-<digest>` 条目，名称形如 `模型 · 高强度推理`，并同步派生 `[1m]` 变体。
- 该方案把 N 个模型 × M 个档位展开成 N×M 行，模型菜单被档位条目污染；同一模型被用户看成多个不同模型。

## Scope

- Claude CLI／Desktop 目录不再派生殖假档位条目。
- 客户端原生 effort 控件作用于网关模型，网关按档位落地到上游参数。
- 相关回归测试、产品文档与任务记录。

## Out of Scope

- 不伪造上游未确认的推理档位（例如 Nemotron 仍保持未知）。
- 不修改 Codex 侧的 `supported_reasoning_levels` 行为。
- 不改 River 之外的适配器 effort 契约。

## Existing Behavior

Claude 模型菜单列出 `模型`、`模型[1m]` 以及每个档位派生出的 `模型 · <档位>`、`模型[1m] · <档位>`；用户必须通过「换模型」来表达推理强度。

## Expected Behavior

Claude 模型菜单每个模型只有一行（普通 + 可选 `[1m]`）；推理强度由客户端 effort 控件单独选择，网关收到该模型 + effort 后只接受该模型元数据中已确认的档位。

---

# Analysis

## Root Cause

`claudeEffortEntries` 在 `ClaudeEntries` 中被无条件追加，把档位编码进模型 ID。这与客户端本身的 effort 分辨率是重复的：Claude 已按模型解析 effort（`modelSettings.<model>.effortLevel`、顶层 `effortLevel`、`/effort`、`--effort`、`CLAUDE_CODE_EFFORT_LEVEL`），不需要也不应该用模型 ID 区分档位。

## Related Modules

- `internal/core/modelcatalog/claude.go`、`claude_effort.go`、`claude_desktop.go`
- `internal/application/agentconfig/claude_picker.go`、`catalog_patch.go`、`refresh.go`
- `internal/application/gateway/routing.go`、`reasoning.go`
- `internal/provider/river/catalog.go`（档位与默认值来源）

## Dependencies

`modelPicker[].behavesAs` 需要 Claude Code v2.1.257 或更高：把网关路由名映射到客户端已知模型，从而获得 effort 能力与默认档位。低于该版本时客户端把路由名当未知模型，effort 控件不会出现。

## Risks

- 归档 `omniproxy-effort-*` 可能导致仍保存着旧档位选择的会话／设置指向失效 ID，需要回退到基础模型而不是报错。
- 基础模型在 `behavesAs` 下默认继承已知模型的默认档位（通常是 `high`），与 River 的默认档位不一致时会改变默认行为，需要在请求侧明确保持 River 元数据默认值。

---

# Implementation Plan

1. 用失败回归先固定目标：Claude 目录不再出现 `omniproxy-effort-*` 条目，且档位只能来自元数据枚举。
2. 删除档位派生条目，保留只读解析：旧档位 ID 在网关侧继续识别并还原为「基础模型 + 档位」，保证已保存的旧选择可用。
3. 用 `behavesAs` 让客户端对网关模型显示原生 effort 控件与档位范围；仅对已确认档位能力的模型设置。
4. 网关侧对 effort 做严格校验：不属于该模型枚举的值返回明确错误，不静默丢弃、不猜测。
5. 执行定向测试、完整质量门，更新文档并审查 diff。

---

# Progress

- 已确认根因：档位被编码进模型 ID，导致模型菜单按档位展开，与客户端原生 effort 控件重复。
- 失败回归先行：`TestClaudeDirectoryKeepsOneEntryPerModel` 在旧实现下得到 8 条目而非 2，`TestArchivedEffortChoiceStillResolvesToBaseModel` 在旧实现下旧档位 ID 解析失败，两者都先红后绿。
- 移除派生条目后，客户端原生 effort 需要可控的「封顶」手段，否则 `behavesAs` 会继承来源模型的完整档位序；因此新增 `effortLevel`／`maxEffortLevel` 写入。
- 质量门第一次以 exit 1 结束且未输出失败任务名；串行 `task --force verify` exit 0，判定为与 TASK-20261022-005 记录过的 `windows:generate:syso` 尾部清理竞争同类，未修改质量门。

---

# Implementation Result

## Changes

- 删除 `claudeEffortEntries`：Claude 目录不再为每个档位派生 `omniproxy-effort-<digest>` 条目，因此菜单里每个模型只有「普通」和可选 `[1m]` 两行，档位不再伪装成不同模型。
- 新增 `ClaudeEffort(entry, effort)`：只按该模型元数据中已确认的档位枚举校验，不推断。
- 新增 `ResolveClaudeSelection(entries, model, effort)`：模型 ID 只解析模型本身，档位来自客户端原生 `output_config.effort`；未确认档位解析为空档位（由 River 侧明确拒绝），缺省不影响路由。
- 保留 `ResolveClaudeEffort` 作为归档兼容：历史 `omniproxy-effort-*`（含 `[1m]` 与 Desktop 摘要形式）继续解码回「基础模型 + 档位」，旧会话不会变成 `unsupported_model`；该 ID 不再进入任何菜单。
- `claudePickerSettings` 对 `ProviderKind == river` 且有已确认档位枚举 + 默认档位的模型写入 `behavesAs`，让客户端显示原生 effort 控件并送出 `output_config.effort`；标签与请求模型 ID 不变，仍显示真实模型名。
- 新增 `claudeEffortSettings`：写入 `modelSettings.<模型或摘要>.effortLevel`（上游公布默认）与 `maxEffortLevel`（该模型真实支持的最高档位），保留用户已显式保存的值和既有 `autoCompactWindow`。
- 网关 `RouteRequest` 改为「原生 effort + 单模型」解析；`claudeRequestedEffort` 只读 `output_config.effort`，不解析其他字段、不猜默认值。
- River Messages → Chat 在请求未带档位时补上该模型公布的上游默认档位，避免客户端在原生控件下沿用旧模型档位或空值改变默认行为。

## Files Changed

- `internal/core/modelcatalog/claude_effort.go`（重写为校验 + 归档解码）
- `internal/core/modelcatalog/claude.go`（`ResolveClaudeSelection`）
- `internal/application/agentconfig/claude_picker.go`（`behavesAs`、`effortLevel`、`maxEffortLevel`）
- `internal/application/agentconfig/catalog_patch.go`、`refresh.go`（应用与刷新同路径）
- `internal/application/gateway/routing.go`、`reasoning.go`、`river.go`
- 测试：`claude_effort_test.go`、`claude_picker_test.go`、`reasoning_test.go`、`river_test.go`
- 文档：`docs/product/LOCAL_AGENTS.md`、`docs/product/RIVER.md`

## Technical Notes

`behavesAs` 只在客户端侧借用「该已知模型支持 effort 及默认档位」这一能力识别，菜单标签、请求模型 ID、计费与服务商归属都不改变，不构成把第三方模型伪装成 Claude。`maxEffortLevel` 是必需的收窄手段：客户端会为 `behavesAs` 模型提供来源模型的完整档位序，不封顶就会显示该模型实际不支持的档位。

---

# Verification

## Tests

- [x] Unit Tests
- [x] Integration Tests（网关 Messages → Chat 档位透传与记账）
- [ ] E2E Tests（需真实 Claude 客户端原生验收）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task --force verify
PASS（exit 0）
```

- FAIL→PASS：`TestClaudeDirectoryKeepsOneEntryPerModel`（旧实现 8 条目）、`TestArchivedEffortChoiceStillResolvesToBaseModel`（旧实现旧 ID 失效）。
- PASS：新增 `TestEffortCapabilitiesComeOnlyFromMetadata`、`TestClaudeSelectionUsesNativeEffortOnSingleModel`、`TestClaudePickerGrantsNativeEffortWithinConfirmedRange`。
- PASS：`go test ./internal/... -count=1` 全部包通过，含 `gateway`、`agentconfig`、`modelcatalog`、`river`。
- PASS：`TestRiverNativeGatewayStreamingRoutingAndUsage` 改为「模型 ID 不含档位 + `output_config.effort=max`」形态，仍断言上游收到 `"reasoning_effort":"max"` 且请求体不含 `omniproxy-effort-`。
- PASS：`task --force verify` exit 0 —— 格式检查 → lint → 类型检查 → Go 全套 → 121 文件／724 项前端测试 → Windows production `go build -tags production`。
- 说明：同一门禁首次运行 exit 1 且未输出失败任务；串行 `task --force verify` 通过。与 TASK-20261022-005 记录的 `windows:generate:syso` 清理竞争为同类现象，未修改质量门、未跳过检查。
- 未执行：真实 Claude Code CLI／Desktop 原生菜单渲染与 effort 滑杆效果验证（需用户在实际客户端完全重启后验收）。

---

# Documentation

- [x] PRD（无需改动：未改变产品边界）
- [x] `docs/product/LOCAL_AGENTS.md`：新增「推理强度（Claude）」小节，并修正「不伪装模型身份」的旧表述为准确的能力借用说明
- [x] `docs/product/RIVER.md`：补充档位传递与默认档位行为
- [x] CURRENT

---

# Deviations

- 原计划「用 `behavesAs` 让客户端显示 effort 控件」未预计到客户端会继承来源模型的完整档位序。实际新增 `modelSettings.maxEffortLevel` 封顶与该模型真实枚举一致，避免展示非法档位。
- 原计划未包含 River 缺省档位补写。实际发现客户端在原生控件下可能不再依赖旧派生 ID 的「显式选择」语义，因此在请求未带档位时补写上游公布默认值，保持 TASK-20261022-005 已确认的默认行为。

---

# Git

## Commit

```text
e8e684e54d0146db9e2849b2ef53c07937cad516 fix(claude): 推理强度改用客户端原生 effort 控件
```

首次提交 `e99ed98` 的中文说明因 PowerShell 重定向写入 BOM 变成乱码，未推送，随即以 `--amend` 用 UTF-8 文件重写说明（内容与文件树不变）。

---

# Result

Claude 模型菜单回归「一个模型一行」，推理强度交回客户端原生 effort 控件；网关按模型真实枚举校验档位并翻译为上游参数，历史档位选择仍可路由，未新增依赖或伪造能力。

---

# Remaining Risks

- `behavesAs` 需要 Claude Code 2.1.257+；更旧版本不显示 effort 控件，只能使用模型的默认档位。
- 客户端原生 effort 控件展示的是「能力借用」模型的档位名，真实透传值由网关二次校验；若客户端未来虚构来源模型 ID，`behavesAs` 会被忽略，需按官方更新重新核对。
- 本机既有 `~/.claude/settings.json` 仍包含上一次写入的 `omniproxy-effort-*` 选项；需在应用中重新应用或目录刷新一次才会替换为单模型列表，且客户端可能缓存旧菜单，需完全重启。
- 未在真实 Claude 客户端验证 effort 滑杆、默认档位显示与上游实际表现。


---

# Archival (v2.0.0)

- 原状态：In Progress
- 2026-10-27：v2.0.0 正式版已公开（标签源码 `c9c48b4a31988a6cc3fa2f4b1d423e61bd10f71b`，
  Actions 完整质量门与 Windows 构建通过），本任务交付内容已随该版本交付。
- 归档依据：归档时工作区无未提交代码改动，本记录引用的实现提交均为上述发布提交的祖先提交。
- 未执行项（实机客户端／原生窗口视觉／真实覆盖安装验收等）不伪造为已验收，转为发布后
  跟进事项，见 `docs/plans/CURRENT.md` 的 Post-release Follow-ups。
