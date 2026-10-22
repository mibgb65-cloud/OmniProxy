# TASK-20261023-007: Claude 与 Harness 推理强度选项修复

## Status

In Progress

## Created

2026-10-23（Asia/Singapore）

## Request

用户反馈两件事：

1. Claude Code Desktop 里仍然没有选择推理强度的地方，上一版（TASK-20261023-001 / Beta.21）的修复没有成功；要求查官方文档确认正确做法。
2. DeepSeek Harness 也没有模型思考程度的选择项，需要一并优化。

## Goal

让两个客户端都出现可用的推理强度选择，且选中值真正到达上游；不伪造模型未确认的档位。

---

# Analysis

## 本机实测（只读探测，未改真实配置）

用本机 Claude Code 2.1.292 指向一个本地捕获服务，观察实际请求体：

- 网关模型作为 `--model` 时，客户端报告 `unrecognized_model`，但请求体 **仍然携带** `"output_config":{"effort":"high"}` 与 `"thinking":{"type":"adaptive"}`。
- 加 `behavesAs: claude-opus-4-8` 后，请求体形态相同，只少了那条识别警告。
- 依次设置 `modelSettings.<模型>.effortLevel` 为 `low`／`medium`／`xhigh`／`max`：
  | 设定值 | 实际发送 |
  | --- | --- |
  | low | low |
  | medium | medium |
  | xhigh | xhigh |
  | **max** | **high** |

## Root Cause 1：`maxEffortLevel` 被写成了模型真实档位值

`claudeEffortSettings`（TASK-20261023-001）把 `maxEffortLevel` 设为该模型最高档位，River 模型是 `max`。官方文档的 `modelSettings.maxEffortLevel` 类型是 `low`/`medium`/`high`/`xhigh`，**不接受 `max`**（"`max` isn't accepted as a level in either key"）。

后果：`maxEffortLevel: "max"` 不是合法上限，客户端的档位范围与上限解析异常，强度控件不可用；即便可见，选 `max` 也会被压回 `high`——实测已确认。

## Root Cause 2：文档中 `behavesAs` 的用途与档位显示无关

官方文档对 `modelPicker[].behavesAs` 的说明是：让客户端对更新的模型 ID 应用「已知模型的能力**和 effort 默认值**」，用于上下文窗口与身份解析。`model-config` 关于强度的唯一陈述是「The effort slider appears in `/model` when a supported model is selected」——「supported」指模型自身被识别，`behavesAs` 并不声明 effort 能力。

因此 TASK-20261023-001 的注释「客户端据此显示 effort 控件」是错误推断，需要修正为准确描述。

## Root Cause 3：Harness 的 `reasoningEfforts` 被写死为 false

`internal/application/agentconfig/harness.go` 对每个模型固定输出 `"reasoningEfforts": false`。Harness 官方文档明确：模型只有在**声明了 reasoning levels** 时，模型选择器才出现 **Effort** 菜单；自定义 model API 提供商需要在模型上写 `reasoningEfforts`。

正确字段形状（官方示例）：

```yaml
models:
  - id: my-reasoner
    reasoningEfforts:
      off:
      high: high
      max: max
```

键是菜单档位，值是线路上发送的 `reasoning_effort` 拼写；只有 `off` 可以留空。Ollama 之外，思考默认开启的模型（如 DeepSeek V4 走 OpenAI 兼容网关）需要 `compat.thinkingFormat: deepseek`，让 `off` 发送 `thinking: {type: disabled}`。

---

# Implementation Plan

1. 修正 `maxEffortLevel` 取值：只用客户端接受的档位序，不写 `max`；同时明确不覆盖用户已存值。
2. 修正 `behavesAs` 与 effort 相关注释，避免把未验证的因果关系写进代码。
3. Harness：按上游已确认档位输出 `reasoningEfforts` 映射；未确认档位的模型不声明，保持菜单不出现。
4. 对 River 这类思考默认开启的模型补 `compat.thinkingFormat: deepseek`，使 `off` 真正关闭思考。
5. 补回归测试，执行完整质量门，更新产品文档。

---

# Progress

- 用本机 Claude Code 2.1.292 + 本地捕获服务完成只读实测，确认 effort 字段一直在发送，问题在档位取值与 UI。
- 建立失败回归：`TestClaudePickerGrantsNativeEffortWithinConfirmedRange` 在旧实现下因 `maxEffortLevel: "max"` 失败。
- `TestHarnessReasoningEffortsFollowConfirmedMetadata` 暴露延续缺陷：`harnessReasoningEfforts` 未校验 `ProviderKind`，会给非 River 模型声明档位；已修为只接受 River。
- 实测确认 `max` 档位在 Harness 侧是上游真实支持的，因此菜单映射保留 `max`，而 Claude 的 `modelSettings` 不能写 `max`——两者取值域不同，不可混用。

---

# Implementation Result

## Changes

- `claude_picker.go`：`maxEffortLevel` 改为只写客户端合法档位（`low`／`medium`／`high`／`xhigh`），不再写 `max`；新增 `effortDefaultLevel` 把上游默认档位 `max` 映射为 `xhigh`，模型只支持 `max` 时不写上限。修正 `behavesAs` 注释，去掉「客户端据此显示 effort 控件」这一未验证断言。
- 新增 `harness_effort.go`：`harnessReasoningEfforts` 按已确认枚举生成映射，`off` 显式写 `none`；`harnessThinkingFormat` 只为可关闭思考的 River 模型返回 `deepseek`。
- `harness.go`：模型条目改为按确认能力输出 `reasoningEfforts` 映射与模型级 `compat.thinkingFormat`，未确认档位仍写 `false`。
- 回归：`claude_picker_test.go` 增补 max 默认值、仅 max 模型两个分支；新增 `harness_effort_test.go`（枚举映射、不伪造档位、真实配置形状）。

## Files Changed

- `internal/application/agentconfig/claude_picker.go`
- `internal/application/agentconfig/harness.go`
- `internal/application/agentconfig/harness_effort.go`（新增）
- `internal/application/agentconfig/claude_picker_test.go`
- `internal/application/agentconfig/harness_effort_test.go`（新增）
- `docs/product/LOCAL_AGENTS.md`

## Technical Notes

Claude 与 Harness 的档位取值域不同，这是本次的核心教训：

| 位置 | 合法值 |
| --- | --- |
| Claude `modelSettings.effortLevel`／`maxEffortLevel` | `low`／`medium`／`high`／`xhigh`（**无 max**） |
| Claude 线路 `output_config.effort` | 由网关按模型枚举校验，可含 `max` |
| Harness `reasoningEfforts` 键与值 | 可含 `max`，`off` 映射 `none` |

把上游真实档位直接抄进 Claude 的 settings 键，正是上一版失败的根因。

---

# Verification

## Tests

- [x] Unit Tests
- [x] Integration Tests
- [ ] E2E Tests（需真实客户端验收）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

- PASS：`go test ./internal/application/agentconfig -count=1` 通过（含新增 `TestHarnessReasoningEffortsFollowConfirmedMetadata`、`TestHarnessConfigDeclaresReasoningEffortsMapping`、`TestClaudePickerGrantsNativeEffortWithinConfirmedRange`）。
- PASS：`go test ./internal/... -count=1` 全部包通过。
- PASS：`task --force verify` exit 0 —— 格式 → lint → 类型 → Go 全套 → 121 文件／724 项前端测试 → Windows production 构建。
- PASS：手工导出生成结果核对——DeepSeek 系输出 `reasoningEfforts: {off: none, high: high, max: max}` + `compat.thinkingFormat: deepseek`；GLM 输出 `{low, high, max}` 无 `off`、无 thinkingFormat；Nemotron 保持 `reasoningEfforts: false`。
- 实测证据（本机 Claude Code 2.1.292，只读捕获请求体）：`effortLevel` 为 `low`／`medium`／`xhigh` 时原样发送，`max` 被压为 `high`。
- 未执行：真实 Claude 菜单与 Harness Effort 菜单的界面验收，需用户应用配置并重启客户端后确认。

---

# Documentation

- [x] `docs/product/LOCAL_AGENTS.md`：修正 Claude 小节（合法档位、max 的实测退化行为），新增「推理强度（DeepSeek Harness／Pi）」小节
- [x] CURRENT

---

# Deviations

- 原计划假定 `behavesAs` 能让 effort 控件出现，实测与文档均不支持该结论。本轮改为「修正会导致档位失效的 settings 取值」并如实记录控件出现取决于客户端对模型的识别，未再宣称该因果。
- 原计划未包含 `ProviderKind` 校验，由新增测试暴露后补上。

---

# Git

## Commit

Not committed by Agent.

---

# Result

修正了让 Claude 强度设置失效的根因（把上游 `max` 写进只接受 `low/medium/high/xhigh` 的 settings 键），并为 Harness 按上游已确认枚举声明 `reasoningEfforts`，使 Effort 菜单得以出现且「关闭思考」真实生效。真实界面表现待用户验收。

---

# Remaining Risks

- Claude 是否显示强度控件取决于客户端对模型 ID 的识别；`behavesAs` 只声明能力继承，不能保证控件出现。若客户端仍不显示，需要按其未识别模型路径另行处理（如 `modelOverrides` 或改用 `claude-` 前缀 ID），不能靠伪造身份绕过。
- Harness 的 `reasoningEfforts` 与 `compat.thinkingFormat` 只在本地生成层面验证；真实 Effort 菜单与思考开关行为需在实际 Harness 客户端确认。
- 未在本机改动真实 Claude／Harness 配置，验证使用的是独立临时配置目录。
