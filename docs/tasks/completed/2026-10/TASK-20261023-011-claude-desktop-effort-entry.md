# TASK-20261023-011: Claude Desktop 思考强度仍无入口的根因定位与桌面目录能力声明

## Status

Completed（2026-10-27 随 v2.0.0 正式发布归档）

## Created

2026-10-23（Asia/Singapore）

## Request

用户第三次反馈：Claude Desktop（Code 标签页）里仍然没有调整思考强度（effort）的入口，模型选择器只能换模型。要求查清客户端到底依据什么决定是否显示 effort 控件，并给出可落地的修复。

## Goal

用客户端实现而不是文档措辞确定 effort 控件的显示条件；把 OmniProxy 在客户端侧可控制的档位信息按客户端真实接受的字段写入；对客户端本身无法为第三方路由名渲染 effort 控件这一事实给出明确结论与可用的调整路径。

## Out of Scope

- 不改写上游未确认的推理档位。
- 不把第三方模型路由名伪装成 Claude 官方模型 ID 以骗取控件。
- 不改动 CLI 侧的 Codex `supported_reasoning_levels` 与 Harness `reasoningEfforts`。

---

# Analysis

## 客户端实测与实现取证（只读）

本机 Claude Desktop `Claude 2.42304.2`（3p 部署，Code 标签页使用自带 `claude-code` 2.1.304）；3p 配置实体为
`%LOCALAPPDATA%\Claude-3p\configLibrary\<DesktopProfileID>.json`，已应用 `00000000-0000-4000-8000-000000309006`。

阅读客户端打包代码（`app\resources\app.asar` 主进程与 `ion-dist` 渲染层）后确认 effort 菜单的唯一来源：

1. `w8t(modelId)`：内置模型能力表 `d8t` 只包含 8 个键
   （`claude-haiku-4-5`、`claude-sonnet-4-5`、`claude-sonnet-4-6`、`claude-sonnet-5`、`claude-opus-4-6`、`claude-opus-4-7`、`claude-opus-4-8`、`claude-opus-5`），
   另有 `/^(?:claude-)?(?:fable|mythos)(?:-|$)/` 正则分支；命中才返回 `effort_options`。
   键的归一化只去掉 `[1m]`、`-YYYYMMDD`、`-vN`、Bedrock/`anthropic.` 前缀，不做模糊匹配。
2. 否则查签名模型目录（`downloads.claude.ai/model-catalog/v1/catalog.json`，本机缓存 `published.json`）中该 ID 或 `provider_ids` 的行，
   行内 `thinking` 或 `behaves_as` 才会给出 `effort_options`。
3. `H7t()` 的第三层回退仍然只是 `w8t`；自定义 `inferenceModels` 条目无法注入 `thinking`、`runtime`、`provider_ids`、`behaves_as`。

因此 `claude-omniproxy-<digits>` 这类不透明路由名永远拿不到 `effort_options`：
渲染层 `effortOptionsFor(model)` 返回空数组，Code 输入框与 `/model` 弹层都不渲染 effort 行
（诊断串即 “no effort menu for this model; the app sends its general default”）。
`modelCatalogUrl` 必须提供签名文档，`inferenceModels[].maxEffort` 只在该模型已有档位列表时裁剪档位，均不能凭空生成控件。

## Root Cause

TASK-20261023-001/007 两轮都把「控件是否出现」当成 settings 取值问题处理；实际是客户端的模型识别问题：
Desktop 只为自己认识的模型 ID 渲染 effort 控件，而 OmniProxy 为了不伪造厂商身份使用了不透明路由名。
`maxEffortLevel`／`behavesAs` 只能影响已被客户端识别的模型的档位取值与起点，无法让不透明 ID 获得控件。

同时发现两个真实缺陷（可在客户端侧修）：

- Desktop 目录从未写入 `inferenceModels[].maxEffort`：客户端无法按模型真实能力裁剪档位，也无法把上限同步给 Code 会话。
- 从未写入 3p 顶层 `defaultModelEffort`：Chat／Cowork／Code 的默认起手档位仍按 Anthropic 推荐值而非上游公布默认值。

---

# Implementation Plan

1. `core/modelcatalog` 增加 Desktop 档位解析：只用客户端档位序（`low/medium/high/xhigh/max`）与已确认枚举求交，未确认返回空。
2. `DesktopModels` 为每个条目写入 `maxEffort`（已确认最高档位），未确认不写。
3. Desktop 配置同时写入 `defaultModelEffort`＝默认模型已确认且客户端接受的默认档位；未确认时清除旧值。
4. 应用与目录刷新两条路径共用同一实现，避免只改一处。
5. 回归测试 + 完整质量门 + 文档与任务记录。

---

# Progress

- 完成客户端取证：确认 effort 菜单由 `d8t` 内置表与签名目录决定，不透明 ID 无入口。
- 完成本轮实现与定向回归；完整质量门 results 见下。

---

# Implementation Result

## Changes

- `internal/core/modelcatalog/claude_desktop.go`：新增 `DesktopEffortLevels`、`DesktopMaxEffort`、`DesktopDefaultEffort`，`DesktopModel` 增加 `maxEffort`。
- `internal/application/agentconfig/catalog_patch.go`、`refresh.go`：Desktop 配置写入 `defaultModelEffort`（仅已确认档位）。

## Files Changed

- `internal/core/modelcatalog/claude_desktop.go`
- `internal/application/agentconfig/catalog_patch.go`、`refresh.go`、`desktop_effort.go`（新增）
- 测试：`internal/core/modelcatalog/claude_desktop_effort_test.go`、`internal/application/agentconfig/desktop_effort_test.go`（新增）
- 文档：`docs/product/LOCAL_AGENTS.md`、`docs/plans/CURRENT.md`

## Technical Notes

`maxEffort` 与 Claude Code `modelSettings.maxEffortLevel` 取值域不同：前者接受 `max`，后者不接受（TASK-20261023-007 已实测 `max` 被压回 `high`）。
`defaultModelEffort` 是 3p 配置的顶层键，只作用于默认模型，且不覆盖用户自己保存的档位。

---

# Verification

## Tests

- [x] Unit Tests
- [ ] Integration Tests（本轮无协议改动）
- [ ] E2E Tests（需真实客户端验收：控件是否出现仍取决于客户端识别）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task --force verify   → exit 0
  format check（gofmt -l main.go internal：无输出）
  lint / typecheck 通过
  Go 全部包 ok（含 modelcatalog、agentconfig）
  前端 123 个文件 / 731 项测试通过
  Windows production 构建通过（bin/omniproxy.exe）
```

- PASS：`go test ./internal/core/modelcatalog/... ./internal/application/agentconfig/... -count=1`。
- PASS：新增 `TestDesktopEffortOnlyUsesConfirmedLevels`（含只支持 `none`、未确认、默认档位不在客户端取值域三个分支）。
- PASS：新增 `TestDesktopModelsDeclareConfirmedCeilingOnly`（未确认模型不得写入 `maxEffort`）。
- PASS：新增 `TestDesktopConfigDeclaresConfirmedEffort`（应用与改默认模型两条路径，含清除陈旧 `defaultModelEffort`）。
- 说明：首次门禁以 exit 1 结束且无失败任务名（`windows:generate:syso` 尾部竞争，与 TASK-20261023-001／TASK-20261022-005 记录同类）；随后串行 `task --force verify` exit 0，日志留存 `tmp/verify-20261023-011.log`。
- 未执行：真实 Claude Desktop 客户端界面验收（控件是否出现取决于客户端识别，本轮不宣称已出现）。

---

# Result

用客户端打包代码确证 effort 控件的显示条件，并补齐 Desktop 目录可如实声明的两处档位信息（`inferenceModels[].maxEffort`、顶层 `defaultModelEffort`）。
不透明路由名在 Desktop 侧不会有控件入口，这一点已在文档和任务记录中写明，不再用配置改动冒充修复。

---

# Git

`bc258ba` — fix(claude): 确证 Desktop effort 控件条件并补齐目录档位声明。工作区其余在办任务的改动未混入；本任务改动为
`internal/core/modelcatalog/claude_desktop.go`、`internal/core/modelcatalog/claude_desktop_effort_test.go`、
`internal/application/agentconfig/desktop_effort.go`、`desktop_effort_test.go`、`catalog_patch.go`、`refresh.go`、
`docs/product/LOCAL_AGENTS.md`、`docs/plans/CURRENT.md` 及本记录。

---

# Remaining Risks

- 只要路由名是不透明的 `claude-omniproxy-*`，Desktop 不会渲染 effort 控件；这是客户端侧限制，不能靠配置注入绕过。
- 可用的调整路径只有：客户端可以识别的模型 ID（会伪装身份，不做）、Claude Code 侧 `modelSettings.<模型>.effortLevel`／`CLAUDE_CODE_EFFORT_LEVEL`，以及 Code 会话内的 `/effort`。
- 若产品要求 OmniProxy 内提供档位选择入口，需要新增受管档位字段与前端控件（建议独立任务，含迁移、绑定与测试）。
- `maxEffort`／`defaultModelEffort` 的真实生效仍需用户在重新应用配置并完全重启客户端后核对。

---

# Archival (v2.0.0)

- 原状态：In Progress
- 2026-10-27：v2.0.0 正式版已公开（标签源码 `c9c48b4a31988a6cc3fa2f4b1d423e61bd10f71b`，
  Actions 完整质量门与 Windows 构建通过），本任务交付内容已随该版本交付。
- 归档依据：归档时工作区无未提交代码改动，本记录引用的实现提交均为上述发布提交的祖先提交。
- 未执行项（实机客户端／原生窗口视觉／真实覆盖安装验收等）不伪造为已验收，转为发布后
  跟进事项，见 `docs/plans/CURRENT.md` 的 Post-release Follow-ups。
