# TASK-20261112-013: Antigravity 家族模型名解析与推理档位保留

## Status

In Progress

## Created

2026-11-12 21:30（Asia/Singapore）

## Request

用户连接 Antigravity 后发送消息失败，桌面端显示 “Agent terminated due to error”。
需要定位并修复，让桌面端能用刚发布的模型目录正常对话。

## Goal

- 桌面端按自己的模型注册表发送的家族名（例如 `gemini-4-argon`）能被网关解析到目录里
  实际存在的档位变体（`gemini-4-argon-high`），并按客户端选择的档位选型。
- 推理档位在选择变体前不被丢弃，选择完成后才从发往上游的正文里移除。
- 不改变其它客户端、其它提供商的请求语义。

## Background

用户连接后，网关请求明细里只有两条 Antigravity 记录，均为失败：

```text
2026-11-12T13:20:55.885Z  input_model=gemini-4-argon  model=gemini-4-argon  status=400  code=unsupported_model  client=antigravity
2026-11-12T13:21:10.435Z  input_model=gemini-4-argon  model=gemini-4-argon  status=400  code=unsupported_model  client=antigravity
```

- 目录按 Google 账号模型原样发布档位变体：`gemini-4-argon-high`、`-medium`、`-low`、
  `gemini-3.8-flash-*`、`gpt-oss-120b-medium`、`claude-opus-4-6-thinking` 等 17 个。
- 桌面端把 “Gemini 4 Argon (High)” 折叠成家族 `gemini-4-argon` 加档位后再发请求
  （请求里的模型名没有档位后缀），网关按 ID 精确匹配，家族名不在目录里，
  于是直接 400，请求没有到达上游。
- 同时 `NormalizeClientBody` 在路由之前就删掉 `reasoning_effort`，即使目录里有家族行，
  也拿不到客户端选择的档位。

## Scope

- `internal/core/proxy/antigravity_client.go`：不再在路由前丢弃 `reasoning_effort`；
  新增路由完成后的剥离函数。
- `internal/core/proxy/handler.go`：Antigravity 请求在路由完成后剥离推理档位字段。
- `internal/application/gateway/`：新增家族名 → 档位变体的解析（只在精确匹配失败后触发）。
- 相应单元测试。

## Out of Scope

- 不改模型目录的发布形状（仍发布档位变体，桌面端自己折叠）。
- 不改 Claude／Codex／其它客户端的行为。
- 不重新打包、不安装；由后续构建任务发布。

## Existing Behavior

`gemini-4-argon` 不在路由目录中，`RouteRequest` 返回 `unsupported_model` 400。

## Expected Behavior

- `gemini-4-argon` + 档位 `medium` → 上游收到 `gemini-4-argon-medium`。
- `gemini-4-argon` 无档位 → 选目录默认档位（优先 `high`）。
- 精确 ID（`gemini-4-argon-high`）行为不变。
- 发往 Google 订阅适配器的正文仍不含 `reasoning_effort`，避免白名单拒绝。

---

# Analysis

## Related Modules

- `internal/core/proxy/antigravity_client.go`、`handler.go`、`request.go`
- `internal/application/gateway/routing.go`、`effort_variant.go`
- `internal/core/modelcatalog/effort_variants.go`（只读参考）

## Dependencies

无新增依赖。

## Risks

- 家族名展开只应在精确匹配失败后触发，否则可能把真实存在的模型名改写错。
- 只有单一变体的家族（`gpt-oss-120b-medium`、`claude-opus-4-6-thinking`）也可能收到
  家族名，需要按唯一后缀回退，但不能跨提供商猜测。
- 档位字段若在上游仍存在，会让 Google 订阅 Chat 适配器的生成参数白名单拒绝请求；
  必须在路由后剥离。

---

# Implementation Plan

1. 创建本记录。
2. 调整 `NormalizeClientBody`：保留 `reasoning_effort`，仍剥离 `service_tier`／
   `metadata`／`store`；新增 `StripClientEffort`。
3. `handler.go` 在 `RouteRequest` 之后、转发之前调用 `StripClientEffort`。
4. 新增 `gateway.applyFamilyVariant`：按 `模型 + "-"` 前缀在目录里找同提供商变体，
   优先匹配客户端档位，其次 `high`／`medium`／`low`，再退到唯一变体。
5. 补测试；跑 `task verify` 相关部分（Go 全量 + 前端不受影响）。
6. 填写结果并提交。

---

# Progress

- 2026-11-12 21:30：创建记录；两个失败请求确认为 `gemini-4-argon` 400。
- 2026-11-12 21:38：核心改动与测试完成；`go test . ./internal/...` 通过。

---

# Implementation Result

## Changes

- `internal/core/proxy/antigravity_client.go`
  - `antigravityDroppedChatFields` 不再包含 `reasoning_effort`：客户端档位是选型的唯一
    依据，必须留到路由完成；仍剥离 `service_tier`／`metadata`／`store`。
  - 新增 `StripClientEffort(client, protocol, body)`：路由完成后移除档位字段（Chat 删
    `reasoning_effort`；Messages／Responses 只删 `output_config.effort`／
    `reasoning.effort`，容器其余字段保留）。仅对 `antigravity` 生效。
- `internal/core/proxy/handler.go`：`RouteRequest` 之后、转发之前调用 `StripClientEffort`。
- `internal/application/gateway/family_variant.go`（新增）
  - `applyFamilyVariant`：精确匹配失败且未命中别名时，按「家族名 + `-`」在同提供商
    目录条目中挑选变体；优先客户端档位，其次 `high`／`medium`／`low`／…，最后接受唯一
    候选。只改写正文 `model`，其余字段原样保留。
- `internal/application/gateway/routing.go`：在既有的历史档位变体回退之后追加家族名回退，
  命中后重新解析路由。

## Not Changed

- 模型目录发布形状（仍发布 `gemini-4-argon-high` 等变体，桌面端自己折叠）。
- Claude／Codex／其它客户端与服务商的请求语义。
- 未给桌面端补发档位字段：档位缺失时按目录默认（`high`）选型。

---

# Verification

## Tests

- [x] Unit Tests：`go test ./internal/application/gateway/... ./internal/core/proxy/...` 通过。
- [x] Go 全量：`go test . ./internal/...` exit 0。
- [ ] Integration Tests：N/A
- [ ] E2E Tests：真实桌面端重试由用户完成

## Quality Checks

- [x] Format：`gofmt -l` 无输出。
- [x] Lint：`go vet . ./internal/...` exit 0。
- [x] Type Check：`task verify` 内含前端 typecheck 与 Go 编译，exit 0。
- [x] Test：见上。
- [x] Build：`task --force verify` exit 0（含 Windows production 构建）。

## Verification Result

```text
go build . ./internal/...                                  → exit 0
go test ./internal/application/gateway/... ./internal/core/proxy/...
  ok omniproxy/internal/application/gateway 49.3s
  ok omniproxy/internal/core/proxy 3.1s
go test . ./internal/...                                   → exit 0
gofmt -l internal/core/proxy internal/application/gateway   → 无输出
go vet . ./internal/...                                    → exit 0
task --force verify                                       → exit 0（VERIFY_EXIT=0）
```

新增测试覆盖：家族名 + 档位选中 `gemini-4-argon-medium`；无档位落到 `-high`；只有单一变体
的家族（`gpt-oss-120b` → `gpt-oss-120b-medium`）；精确 ID 不经过回退；未知家族仍 400；
正文里的档位在转发前被剥离且保留其余字段；非 Antigravity 客户端正文不变。

---

# Documentation

- [x] PRIMARY：无需修改产品文档（行为修正，不改变对外契约）
- [x] PRD／Architecture／Tech Stack／Coding Rules／Testing／Build／ADR：无需修改

---

# Git

## Commit

（实现提交与归档提交待回填）

---

# Result

Antigravity 桌面端发送的家族模型名（`gemini-4-argon`）现在会按客户端档位解析到目录里
实际存在的上游变体（`gemini-4-argon-high／-medium／-low`），档位在选型后才从发往上游的
正文中移除；`gpt-oss-120b`、`claude-opus-4-6` 这类只有单一变体的家族同样可解析，未知家族
仍然返回 `unsupported_model`。

---

# Completed

2026-11-12 21:40
