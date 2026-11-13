# TASK-20261113-001: Gemini 4 Hydrogen 的 xhigh 档位被降级为 high

## Status

In Progress

## Created

2026-11-13 08:34（Asia/Singapore）

## Request

> Hydrogen 有 -xhigh 档，但我们发给上游的 thinkingLevel 只认 low / medium / high，这个也需要修复掉

## Goal

- 客户端选择 `xhigh` 档位时，发往上游的 `thinkingLevel` 不再被降级成 `high`。
- 上游不接受 `xhigh` 时回退到不发送档位，而不是发送一个错误的档位。

## Background

- 账号模型同步已包含 `gemini-4-hydrogen-low/medium/high/xhigh`（2026-11-13 08:23 同步）。
- `internal/provider/antigravity/chat_family.go` 的 `levelOf` 只映射
  `minimal`/`none`/`low` → `low`、`medium` → `medium`（Flash 或 4 代以上）否则 `high`，
  其余全部 → `high`。因此 `xhigh` 的请求虽然模型 ID 已换成 `-xhigh`，
  `thinkingConfig.thinkingLevel` 仍写 `high`。
- Gemini 2.5 路径用 `thinkingBudgets`，其中已包含 `xhigh: 32000`，不受影响。

## Scope

- `internal/provider/antigravity/chat_family.go`：`levelOf` 透传已确认的档位。
- 单元测试。
- 隔离实例上验证上游是否接受 `thinkingLevel: "xhigh"`。

## Out of Scope

- 不改模型目录、额度展示与其它客户端。
- 不改变 `max` 在 Gemini 2.5 上的预算行为。

## Existing Behavior

`xhigh` → `thinkingLevel: "high"`。

## Expected Behavior

`xhigh` → `thinkingLevel: "xhigh"`（上游接受时）；上游拒绝时回退为不发送档位。

---

# Analysis

## Related Modules

- `internal/provider/antigravity/chat_family.go`、`chat_request.go`

## Dependencies

无新增依赖。

## Risks

- 若上游枚举不接受 `xhigh`，请求会 400 `unsupported_content`；必须先用真实请求确认。

---

# Implementation Plan

1. 创建本记录。
2. 改 `levelOf` 透传 `xhigh`。
3. 用隔离实例发 `gemini-4-hydrogen-xhigh` + `xhigh` 真实请求确认。
4. 不通过则改为「档位为 xhigh 时不下发 thinkingLevel」。
5. 补测试、跑 `task verify`、提交。

---

# Progress

- 2026-11-13 08:34：创建记录。
- 2026-11-13 08:40：隔离实例真实请求确认上游接受 `thinkingLevel: "xhigh"`，改用透传方案。

---

# Implementation Result

## Changes

- `internal/provider/antigravity/chat_family.go`：`levelOf` 增加 `case "xhigh"` 直接返回
  `xhigh`；其余档位映射保持原样（`minimal`／`none`／`low` → `low`，`medium` → `medium`，
  其它 → `high`）。
- `internal/provider/antigravity/chat_envelope_test.go`：新增
  `gemini-4-hydrogen-xhigh` 用例，断言请求体 `thinkingConfig.thinkingLevel` 为 `xhigh`。

## Files Changed

- `internal/provider/antigravity/chat_family.go`
- `internal/provider/antigravity/chat_envelope_test.go`

## Technical Notes

- 上游接受 `xhigh`（真实请求 200），因此选择透传而不是「不下发档位」的兜底方案。
- Gemini 2.5 路径的 `thinkingBudgets`（其中 `xhigh: 32000`）不受影响。

---

# Verification

## Tests

- [x] Unit Tests：`go test ./internal/provider/antigravity/...` 通过。
- [x] Go 全量：`go test . ./internal/...` 通过。
- [x] Integration Tests：隔离实例上 `gemini-4-hydrogen-xhigh` 与
  `gemini-4-hydrogen` + `reasoning_effort: xhigh` 均返回 200，响应模型为对应变体。
- [ ] E2E Tests：桌面端选择 xhigh 档位待用户安装新构建后确认。

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
VERIFY_EXIT=0
```

- 前端 147 文件／898 项、全部 Go 包与 Windows production 构建通过。
- 隔离实例（端口 3011，生产数据副本）真实请求：

```text
gemini-4-hydrogen + reasoning_effort=medium  → 200  model=gemini-4-hydrogen-medium
gemini-4-hydrogen-xhigh                      → 200
gemini-4-hydrogen + reasoning_effort=xhigh   → 200
```

---

# Documentation

- [x] PRIMARY：无需修改产品文档（档位说明未在文档中固定枚举）
- [x] PRD／Architecture／Tech Stack／Coding Rules／Testing／Build／ADR：无需修改

---

# Deviations

- 计划中的「上游不接受时不下发档位」兜底未启用：真实请求证明上游接受 `xhigh`，透传即可，
  不再增加无依据的分支。
- 本轮沿用连续诊断，代码改动先于本记录写出，记录随后补齐（含实际改动与验证）。

---

# Git

## Commit

```text
<commit hash> 见归档提交
```

未打标签、未推送。

---

# Result

Antigravity 的 `xhigh` 档位不再被降级成 `high`：模型 ID 与 `thinkingLevel` 都按用户选择下发。

---

# Completed

（未完成）
