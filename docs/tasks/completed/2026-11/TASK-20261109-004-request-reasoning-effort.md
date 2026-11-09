# TASK-20261109-004: 请求明细记录并展示推理等级

## Status

Completed

## Created

2026-11-09 14:15

## Request

用户看到「请求」页只有模型列（截图：`gpt-6.1-sol`／`gpt-6-luna`），提出：

> 这个可以增加一个显示使用的推理等级吗？这个会记录吗？目前只知道选择的模型

## Goal

请求明细能够记录并展示每次请求实际携带的推理等级（reasoning effort）：

- 新产生的请求在列表和详情中显示档位（如 `high`／`medium`／`low`）。
- 未指定档位的请求显示 `—`，不猜测上游默认值。
- 历史记录（升级前）无法回溯，同样显示 `—`，不伪造。

## Background

请求明细目前只保存模型（`input_model`／`model`）、协议、传输、用量、耗时等字段
（`request_records`，迁移 008／032／033／040）。推理等级只存在于请求正文的
协议字段里，正文按隐私规则不入库：

- Messages：`output_config.effort`
- Responses：`reasoning.effort`
- Chat：`reasoning_effort`

网关在路由阶段会用「按模型固定档位」（迁移 041）覆盖客户端档位，并把 Google 系
折叠模型的档位换回具体型号 ID（`applyEffortVariant`），这些改写都发生在正文上，
请求记录里没有任何字段留下结果。因此用户只能看到模型，看不到档位。

## Scope

本任务预计包含：

- 迁移 046：`request_records` 增加 `reasoning_effort`。
- Core：从三种协议的请求正文提取档位，随 `RequestEvent` 写入请求记录；
  覆盖 HTTP 协议入口、Codex 透明转发（HTTP）与 `/v1/responses`／Codex WebSocket 逐轮记录。
- 网关：路由改写（模型选择、固定档位、档位变体）后以实际发送的档位为准。
- 请求列表新增「推理等级」列，请求详情新增对应字段，中英文案与说明提示。
- Go／前端回归测试与文档更新。

## Out of Scope

本任务明确不处理：

- 不保存请求正文、提示词或上游原始档位枚举之外的推理内容。
- 不推断未指定档位时上游实际使用的默认值。
- 不回填历史请求记录。
- 不按档位做聚合统计、筛选或成本差异化。
- 不改变任何适配器已实现的档位映射逻辑。

## Existing Behavior

请求明细没有任何推理等级字段；列表与详情只显示模型。

## Expected Behavior

- 客户端在正文里选择档位：记录并显示该档位（经网关映射后的取值）。
- 用户在应用内固定了模型档位：记录并显示固定档位（它覆盖客户端选择）。
- 模型的档位以变体型号表达（Google `…-low`）：记录并显示对应档位。
- 客户端未指定档位：记录空串，界面显示 `—`。
- 历史记录：显示 `—`。

---

# Analysis

## Related Modules

- `internal/core/proxy`：请求解析、`RequestEvent`／`Trace`、透明转发与 WebSocket 逐轮记账。
- `internal/application/gateway`：路由改写、按模型固定档位、档位变体、透明转发会话。
- `internal/application/activity`：请求明细写入与查询。
- `migrations/`：新增迁移 046。
- `frontend/src/features/activity`、`frontend/src/i18n/locales/activity-*.ts`、`frontend/bindings`。

## Dependencies

`wails3 generate bindings`（已安装 v3.1.0）用于刷新前端绑定类型。

## Risks

- 档位字段来自客户端正文，必须限长并过滤控制字符，避免把任意字符串写进明细。
- 透明转发必须保持字节原样，只允许旁路解析，解析失败只丢统计。
- 表宽增加一列，需要保持现有横向滚动与吸顶表头行为。

---

# Implementation Plan

1. 新增迁移 046：`request_records.reasoning_effort`（`TEXT NOT NULL DEFAULT ''`，长度 1–32 或空）。
2. `internal/core/proxy` 新增档位提取（三种协议字段），并在协议解析时写入 `Request.Effort`。
3. `RequestEvent` 增加 `Effort`，`Trace.Update`／`Begin` 落库；透明转发 HTTP 在开始记账时即带上档位，
   `/v1/responses` 与 Codex WebSocket 逐轮从帧正文提取。
4. 网关在路由改写完成后（固定档位、变体型号之后）统一写回 `request.Effort`；
   透明转发会话把正文档位带进每次尝试。
5. `activity`：`RequestItem.Effort`、`requestColumns` 与 `Begin`／`Update` 读写。
6. 前端：绑定类型、请求表新增列、详情新增字段、中英文案与提示。
7. 测试：Core 提取用例、activity 存取用例、网关固定档位用例、前端列渲染用例。
8. `task verify`，更新 `docs/product/ACTIVITY.md` 与 `docs/plans/CURRENT.md`。

---

# Progress

记录重要节点：

- 2026-11-09 14:15 现状核对：请求明细无档位字段；正文按隐私规则不入库，需在请求时旁路提取。
- 2026-11-09 14:20 Core／网关／activity 落库链路完成，`go test` 三个受影响包通过。
- 2026-11-09 14:40 前端绑定重新生成，列表列、详情字段与中英文案完成，活动页测试通过。

---

# Implementation Result

## Changes

实际完成：

- 迁移 046：`request_records.reasoning_effort`（`TEXT NOT NULL DEFAULT ''`，空或 1–32 字符）。
- Core 新增 `ReasoningEffort`／`NormalizeEffort`：按协议读取 Messages `output_config.effort`、
  Responses `reasoning.effort`、Chat `reasoning_effort`；只接受 32 字节内、字母数字与 `-_` 的取值，
  其他内容按未指定处理。协议解析（`ParseProtocolRequest`／`ParseRequest`）写入 `Request.Effort`。
- `RequestEvent` 增加 `Effort`；`beginTrace` 增加档位参数并在 `Begin` 落库，`Trace.Update`
  只在档位非空时覆盖，避免后到的用量快照把已记录的档位抹掉。
- HTTP 透明转发在开始记账时旁路解析正文档位（压缩或非 JSON 正文按未指定处理），
  `PassthroughAttempt` 携带档位；Codex WebSocket 每轮从 `response.create` 帧读取档位。
- 网关 `RouteRequest` 在模型选择、按模型固定档位、路由改写之后、档位变体换型号之前写回
  `request.Effort`，因此记录的是实际携带的档位；透明转发会话把正文档位带进每次尝试。
- `activity`：`RequestItem.Effort`（`json:"effort,omitempty"`）、`requestColumns`、
  `Begin` 写入与 `Update` 的非空覆盖。
- 前端：重新生成绑定；请求表在「模型」后新增「推理等级」列（带说明提示），
  请求详情新增同名字段；未指定显示「—」。中英文案同步。
- 测试：Core 协议／归一化／解析用例，activity 存取与非空覆盖、异常取值用例，
  网关固定档位写回用例，Codex WebSocket 逐轮档位用例，前端列表列渲染用例。

## Files Changed

主要修改文件：

- `migrations/046_request_reasoning_effort.sql`
- `internal/core/proxy/effort.go`（新增）、`effort_test.go`（新增）、`types.go`、`request.go`、
  `agent_request.go`、`trace.go`、`handler.go`、`websocket.go`、`passthrough.go`、
  `passthrough_types.go`、`passthrough_socket_accounting.go`
- `internal/application/gateway/routing.go`、`effort_variant.go`、`passthrough.go`、
  `passthrough_session.go` 及对应测试
- `internal/application/activity/types.go`、`queries.go`、`observer.go`、
  `request_effort_test.go`（新增）
- `frontend/bindings/omniproxy/internal/application/activity/models.ts`（重新生成）
- `frontend/src/features/activity/request-table.tsx`、`request-dialog.tsx`、
  `request-metrics.test.tsx`、`frontend/src/i18n/locales/activity-{zh,en}.ts`
- `docs/product/ACTIVITY.md`、`docs/product/PRD.md`、`docs/plans/CURRENT.md`

## Technical Notes

- 档位只来自请求正文的旁路解析，正文本身仍不入库；透明转发只读不改，解析失败只丢统计。
- 记录口径是「网关实际为这次请求携带的档位」：客户端选择或应用内固定档位都算，
  但适配器为未选择请求补的默认档位不算，界面保持「—」，不冒充用户选择。
- 上游 websocket 每轮 `response.create` 是独立请求记录，档位也逐轮记录。

---

# Verification

## Tests

- [x] Unit Tests
- [x] Integration Tests
- [x] E2E Tests（不适用：桌面应用未自动启动，界面由组件测试覆盖）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify（分步执行，见下）

## Verification Result

```text
gofmt -l main.go internal            → 空
go vet . ./internal/...              → 通过
staticcheck . ./internal/...         → 通过
go test . ./internal/...             → 51 个包全部 ok
pnpm exec prettier --check（改动文件）→ 通过
pnpm lint                            → 0 error（2 个既有的 reset-credit-list 警告）
pnpm typecheck                       → 通过
pnpm exec vitest run --maxWorkers=4  → 146 文件 / 892 项全部通过
pnpm build                           → 通过（dist 1,692.50 kB js / 105.42 kB css）
go build -tags production -trimpath -buildvcs=false -ldflags="-w -s" -o bin/omniproxy.exe .
                                     → 通过（23,571,456 bytes）
```

说明：

- 默认并发（无 `--maxWorkers`）运行时 `src/features/accounts/stepfun.test.tsx` 出现过 1 项
  5 秒超时；该文件单独运行 5/5 通过，按项目惯例限制到 4 worker 后全量 892 项通过，
  判定为与本次改动无关的并发抖动。
- 未整体执行 `task verify`：本机 `task` 环境缺少 `uname`／`tail`，且既有记录显示其 build
  依赖在此环境会卡住；已按该任务链逐条执行等价命令（format/lint/typecheck/test/build）。
- 未启动桌面应用做实机观感验收，避免与用户正在运行的实例争用配置与数据库。
```

---

# Documentation

- [x] PRD — `docs/product/PRD.md` 请求明细条目补充迁移 046 与推理等级口径。
- [ ] Architecture
- [ ] Tech Stack
- [ ] Coding Rules
- [ ] Testing
- [ ] Build
- [ ] ADR
- [x] CURRENT — 进行中条目与 In Progress 段。

`docs/product/ACTIVITY.md` 增加迁移 046 的档位来源、展示口径与「不进入聚合」边界。

---

# Deviations

- 原计划把档位读取函数放在网关并沿用既有 `carriedEffort`；实际把协议字段解析下沉到
  `internal/core/proxy`（`ReasoningEffort`），网关与 Core 共用一份，避免两处字段名漂移。
- 原计划只改 `Trace.Update`；实际额外给 `beginTrace` 增加档位参数，使透明转发在响应前就失败时
  也能留下客户端档位。
- `task verify` 未整体执行，改为等价分步命令（原因见 Verification）。

---

# Git

## Commit

```text
e8dfc47e feat(activity): record reasoning effort per request
```

---

# Result

请求明细现在会记录并展示推理等级：迁移 046 增加 `reasoning_effort`，HTTP 协议入口、
Codex 透明转发（HTTP 与 WebSocket 逐轮）都在请求时旁路提取档位，以路由改写后的实际档位为准。
请求列表新增「推理等级」列、详情新增同名字段，未指定与历史记录显示「—」，不推测上游默认值，
也不参与任何聚合、成本或配额统计。质量门分步执行通过。

---

# Completed

2026-11-09 14:52
