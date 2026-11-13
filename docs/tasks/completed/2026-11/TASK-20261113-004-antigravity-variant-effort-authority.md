# TASK-20261113-004: 型号自带档位时被客户端档位字段降级（推理等级显示 low）

## Status

Completed

## Created

2026-11-13 09:20（Asia/Singapore）

## Request

> 我用的是 gemini-4-hydrogen-high，但是推理等级显示的是 low

## Goal

- 客户端直接使用带档位后缀的上游型号（`gemini-4-hydrogen-high`）时，型号里的档位是用户的
  实际选择：请求明细显示 high，发给上游的 `thinkingLevel` 也是 high。
- 客户端同时携带的档位字段（实测 Antigravity 会带 `reasoning_effort: "low"`）不得把型号降级。
- 家族名 + 档位字段的既有解析（`gemini-4-argon` + high → `-high`）保持不变。

## Background

请求明细（2026-11-13 08:57:45，`client=antigravity`）：

```text
input_model     gemini-4-hydrogen-high
model           gemini-4-hydrogen-high
reasoning_effort low          ← 界面「推理等级」列显示的值
status          200
```

`input_model` 与 `model` 相同，说明目录里存在该变体 ID、路由没有改写型号；而
`reasoning_effort` 来自客户端正文。桌面端在发送变体型号的同时还带了一个档位字段，于是
`request.Effort`（以及上游 `thinkingConfig.thinkingLevel`）用的是字段值 low，与型号本身矛盾。

## Scope

- `internal/core/modelcatalog/effort_variants.go`：新增识别型号档位后缀的只读辅助函数。
- `internal/application/gateway/effort_variant.go`：型号已是目录确认的档位变体时，以型号为准。
- 单元测试。

## Out of Scope

- 不改其它客户端、协议、目录发布与额度逻辑。
- 不改变家族名解析与 `-xhigh` 透传（TASK-20261113-001）。

## Existing Behavior

`gemini-4-hydrogen-high` + `reasoning_effort: "low"` → 记录与下发都是 low。

## Expected Behavior

型号带后缀且目录确认该变体时，档位取型号后缀（high）；型号仍原样转发。

---

# Analysis

## Related Modules

- `internal/application/gateway/effort_variant.go`
- `internal/core/modelcatalog/effort_variants.go`
- `internal/provider/antigravity/chat_request.go`（读取 `request.Effort`）

## Dependencies

无新增依赖。

## Risks

- 目录读取次数：只在型号长得像档位变体（`-low`／`-medium`／`-high`／`-xhigh` 等结尾）时
  才多读一次目录，普通请求不受影响。
- 若某个真实型号恰好以 `-high` 结尾但不是变体，目录确认（`EffortModels` 映射）会把它排除。

---

# Implementation Plan

1. 创建本记录。
2. 新增 `catalog.EffortVariantSuffix` 判断型号是否以上游档位后缀结尾。
3. `applyEffortVariant` 先按型号判定：目录确认是变体时 `request.Effort = 后缀档位` 并直接返回。
4. 补测试：折叠目录下 `gemini-4-hydrogen-high` + low → effort high；家族名 + low 仍解析到 `-low`。
5. 跑 `task verify`，填结果、提交。

---

# Progress

- 2026-11-13 09:20：创建记录。
- 定位根因：`RouteRequest` 用客户端正文里的档位覆盖 `request.Effort`，型号变体未参与判定。

---

# Implementation Result

## Changes

- `internal/core/modelcatalog/effort_variants.go`：新增 `EffortVariantSuffix`，判断型号是否
  以上游档位后缀（`-minimal`／`-low`／`-medium`／`-high`／`-xhigh`／`-ultra`）结尾。
- `internal/application/gateway/effort_variant.go`：`applyEffortVariant` 先看型号——目录确认它是
  档位变体时，`request.Effort` 取该档位，并从正文移除矛盾的档位字段；目录不可用或未确认时
  保持原行为。
- 测试：`internal/application/gateway/effort_variant_test.go` 新增折叠目录用例
  （`gemini-4-hydrogen-high` + `reasoning_effort: low` → effort `high`、正文不再带档位字段），
  并保留家族名 + 档位、以及未指定档位时不猜测的断言。

## Files Changed

- `internal/core/modelcatalog/effort_variants.go`
- `internal/application/gateway/effort_variant.go`
- `internal/application/gateway/effort_variant_test.go`

## Technical Notes

- 只有型号看起来像变体时才多读一次目录；目录确认（`Metadata.EffortModels`）是唯一判定依据，
  恰好以 `-high` 结尾的普通型号不会被误认。
- 上游 `thinkingConfig.thinkingLevel` 由 `request.Effort` 决定，因此显示与下发同时修正。

---

# Verification

## Tests

- [x] Unit Tests：`go test ./internal/application/gateway/`（含新增折叠目录用例）通过。
- [x] Go 全量：`go test . ./internal/...` 通过。
- [x] Verify：`task --force verify` exit 0。

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build

## Verification Result

```text
task --force verify
VERIFY_EXIT=0 (148 文件 / 902 项前端测试，全部 Go 包与 Windows production 构建)
```

---

# Documentation

- [x] CURRENT：记录本修复
- [x] docs/product/GOOGLE_SUBSCRIPTION.md：说明型号档位优先

---

# Git

## Commit

```text
97d15cdb fix(gateway): 型号自带档位时不再被客户端档位字段降级
```

未打标签、未推送。

---

# Result

`gemini-4-hydrogen-high` 之类变体型号的档位不再被客户端自带的档位字段降级：请求明细显示
high，上游 `thinkingLevel` 也是 high；家族名 + 档位与未指定档位的行为保持不变。

---

# Completed

2026-11-13 09:44
