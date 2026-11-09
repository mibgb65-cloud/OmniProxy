# TASK-20261107-001: 自动同步 OpenAI 模型目录并避免目录失效破坏账号平衡

## Status

In Progress

## Created

2026-11-07 11:55（Asia/Singapore）

## Request

用户询问：ChatGPT／Codex 的模型同步能否自动处理；靠界面提示让用户手动同步很难覆盖到。

用户确认先不修改代码，按「兜底也轮换 + 目录失效不再破坏平衡（B）」与「事件驱动自动同步（A）」的方向写 Task Record，审阅确认后再实施。

## Goal

- 目录同步不再依赖用户手动点击：应用启动、重新授权、账号更新后，后台自动补齐 OpenAI 订阅账号的当前版本目录。
- 账号平衡不再以「目录必须新鲜」为前提：目录过期、为空或同步失败时，ChatGPT／Codex 透明入口仍按同池候选轮换，而不是固定压在第一个账号。
- 自动同步不阻塞请求、不并发启动多个官方组件、失败保留旧快照；失败可在模型目录页被动看到，但不弹窗要求用户处理。
- 不新增数据库迁移、依赖或凭据访问面。

## Background

2026-11-07 实机排查结论（只读）：

- `codex_settings.account_rotation_mode = balanced`，三个 OpenAI 订阅账号已启用且已认证；实际流量自 2026-11-06 01:23（UTC）起几乎全部落在 `acct_…3e3fbf75`。
- 最近一次额度快照：该账号 5h 剩余 40%，另两个账号 5h 剩余 100%；因此不是「其他账号不可用」，而是平衡选号没有运行。
- 原因链：
  1. 2026-11-04／11-06 重新授权后 `credential_ref` 变化，`account_catalog_versions` 视图版本更新；`account_models` 中旧版本的目录行不再参与 JOIN，当前版本没有模型行；同时存在「目录同步成功但官方返回空目录」的既有情况，不能把空目录直接当成永久失败。
  2. 路由 `snapshot()`／`discovered()`／`targetSupports()` 找不到 OpenAI 模型，`evaluate()` 返回 `unsupported_model`。
  3. Codex 走 `/codex/v1` 透明入口；`OpenPassthrough` 把 `unsupported_model` 交给 `openFallbackSession`。
  4. `openFallbackSession` 只按 `accounts.List()` 顺序（`created_at DESC`）取第一个账号，**不调用 `rotateAccounts`**，于是所有请求固定到同一账号。
- 现有自动同步只有 Google：`modelcatalog.StartGoogleAutoSync` 在启动时和授权变化后串行刷新，失败保留旧目录；OpenAI 只有 `ModelSync` 手动入口。
- 相关记录：[ADR-018](../../decisions/ADR-018-codex-account-rotation.md)、[ADR-023](../../decisions/ADR-023-codex-v1-transparent-passthrough.md)、[TASK-20261019-017](../completed/2026-10/TASK-20261019-017-openai-account-rotation-settings.md)、[TASK-20261024-016](../completed/2026-10/TASK-20261024-016-activation-failure-code-and-websocket-toggle.md)（空目录记录）。

## Scope

本任务预计包含：

1. **透明入口兜底参与轮换（B）**
   - `openFallbackSession` 在构造同池候选后复用 `rotateAccounts`，让目录不可用时仍按 `balanced`／`queue` 设置选号；`queue` 保持原顺序。
   - 兜底路径对选号保持容错：设置读取失败或候选复核失败时保留原顺序，不把「平衡不可用」升级成「请求失败」。
   - 保持固定账号、冷却／额度阻断、同池边界、最多尝试次数和「结果不确定不重放」不变。
   - 兜底场景没有可用模型名时，权重自然退化为等权轮询，不额外请求上游。

2. **OpenAI 目录事件驱动自动同步（A）**
   - 启动时扫描「当前目录版本没有任何模型行」的已启用、已认证 OpenAI 订阅账号，后台补齐。
   - 订阅授权状态变化（登录完成、重新登录、刷新）、账号更新／启用时请求一次扫描；请求合并，避免重复同步。
   - 单 worker 串行、每账号 30 秒上限、每次应用运行中每账号每凭据版本最多自动尝试一次；空目录视为该版本的终态，不做无界重试。
   - 失败保留数据库内旧快照，只记录固定错误码；成功后刷新受管客户端目录并发出 `modelcatalog:updated`。

3. **被动可观测性**
   - 模型目录页显示「后台同步中／自动同步失败，可手动重试」状态，不再只显示空列表；不新增必须用户确认的弹窗。
   - 日志只记录固定错误码，不输出账号身份、路径或上游正文。

4. **测试与文档**
   - 覆盖兜底轮换、自动同步、并发合并、失败保留、空目录、版本竞态和关闭清理。
   - 同步更新 ROUTING、LOCAL_AGENTS、SUBSCRIPTION、OVERVIEW 与 CURRENT。

## Out of Scope

本任务明确不处理：

- 不改变 `account_catalog_versions` 的版本语义（不把 OpenAI 订阅目录改成按身份而非 `credential_ref` 锚定）；该方案影响既有「凭据版本隔离目录」边界，留作后续独立评估。
- 不让普通 `/v1/...` 路由在目录过期时接受未经确认的模型；本次只修复透明入口的兜底轮换，普通入口仍由自动同步补齐目录后恢复。
- 不做定时高频轮询、每请求同步、无限自动重试，也不新增后台探测模型或上游请求。
- 不复制用户缓存、不跨账号伪造目录、不借用其他账号的上下文窗口。
- 不修改官方登录／授权流程、凭据存储、余额查询或发布流程。
- 不包含安装包构建、发布或真实客户端人工验收。

## Existing Behavior

- OpenAI 目录只能在「路由 → 模型目录」手动选择账号并点击同步；应用启动和重新登录不会自动刷新。
- 重新授权后旧目录立即从路由 JOIN 中失效；当前版本没有模型行时，Codex 透明入口走 `openFallbackSession`。
- `openFallbackSession` 按 `accounts.List()` 的 `created_at DESC` 顺序取第一个账号，不读取 `queue`／`balanced` 设置，也不参与轮询。
- 全局 `balanced` 设置仍然保存为 `balanced`，界面看起来已开启，但实际请求不经过选号逻辑。
- 日志只有额度自动激活的 `models_unavailable` 等间接信号，没有「目录过期导致平衡旁路」的明确记录。

## Expected Behavior

- 重新授权或应用启动后，过期目录在后台自动恢复；用户无需点击「同步」。
- 即使目录尚未恢复、同步失败或官方返回空目录，`/codex/v1` 的 HTTP／WebSocket 请求仍从完整同池候选中按 `queue`／`balanced` 选号；成功请求继续按既有规则刷新额度观察。
- 固定账号、关闭故障切换、额度阻断和冷却仍优先；不会为了平衡重放已开始或结果不确定的请求。
- 自动同步失败时请求路径不受影响，旧快照保留；模型目录页显示可重试状态，日志有稳定错误码。
- `queue`、Google 自动同步和其他提供商目录行为不变。

---

# Analysis

## Related Modules

预计涉及：

- `internal/application/modelcatalog/`：现有 `Sync`、`List`、`GoogleAutoSync`；新增或泛化后台自动同步。
- `internal/application/gateway/passthrough.go`：`OpenPassthrough`、`openFallbackSession`、`passthroughFallbackTargets`。
- `internal/application/gateway/account_rotation.go`：复用 `rotateAccounts` 与轮询状态。
- `internal/application/routing/`：`snapshot`、`discovered`、`evaluate`、`targetSupports`（只读确认，不改版本语义）。
- `internal/transport/wails/`：启动装配、订阅 `OnChange`、账号更新事件、受管目录刷新、模型目录状态接口／事件。
- `frontend/src/features/routing/`：模型目录页的自动同步状态展示与测试。
- `docs/product/ROUTING.md`、`LOCAL_AGENTS.md`、`SUBSCRIPTION.md`、`docs/architecture/OVERVIEW.md`、`docs/plans/CURRENT.md`。

## Dependencies

- 现有官方 Codex app-server 模型读取路径（`subscription.Models`）和 `modelcatalog.Sync` 的版本复核。
- 现有 `account_catalog_versions` 视图与 SQLite 事务；不新增迁移。
- Wails 事件与查询缓存；若新增状态查询，仅使用内存状态。
- 现有额度观察来源（成功请求／额度页）作为 `balanced` 权重，不新增探测。

## Root Cause

平衡功能的正确性被绑定在「目录缓存必须由人工保持新鲜」上：

- 目录版本随 `credential_ref` 变化，重新授权即失效；
- 路由把「目录没有当前版本模型」直接判成 `unsupported_model`；
- 透明入口的兜底路径只解决「能发出去」，不执行账号选择策略。

因此即使 `balanced` 设置正确，请求也会绕过选号；这是控制流断点，而不是权重计算错误。

## Alternatives Considered

- **定时全量轮询（例如每小时）**：能兜底，但对官方组件和网络是持续成本，且大多数时间没有变化；不采用。
- **每请求懒同步**：首请求延迟不可控，且容易在目录持续为空时形成重试风暴；不采用。
- **路由接受任意过期目录**：会扩大旧快照的权限，重新登录到不同账号时可能把旧模型带过去；超出本任务边界，不采用。
- **把目录版本锚定到身份（`auth_account_id`）而非凭据引用**：更接近根因，但改变既有隔离语义，需要单独评估；列为后续可选项。

## Risks

- 启动时自动同步会启动官方组件，可能与登录、额度查询竞争；需要单 worker、超时、跳过忙碌账号和合并请求。
- 官方目录可能「同步成功但为空」；必须按版本记录一次性尝试，避免每个事件都重启组件。
- 兜底轮换改变透明入口的选号顺序；需要确认固定账号、冷却、额度阻断、同池校验和最多尝试次数不变，且不新增上游请求。
- 兜底路径复用设置读取器后，设置损坏／数据库暂时不可用可能让原本可发送的请求失败；必须降级为原顺序并记录固定错误码，不能放大故障。
- 自动同步失败时的 UI 状态可能与实际请求路径不同步；以「目录状态」表述，不承诺平衡一定已恢复。
- 启动时无网络或官方组件暂不可用会失败；本任务不引入定时重试，由下一次授权／账号事件或手动重试恢复，期间透明入口仍按兜底轮换工作。
- 并发测试容易受时序影响；使用可控时钟／注入同步函数，避免依赖真实网络和真实账号。

---

# Implementation Plan

1. **先复现并加回归**
   - 用隔离 SQLite 和假订阅构造 3 个 OpenAI 订阅账号，令当前版本目录为空；断言透明入口连续请求只落到第一个账号（现有缺陷）。
   - 增加固定账号、冷却账号、重复身份和并发请求的边界用例。

2. **兜底轮换（B）**
   - 在 `openFallbackSession` 构造候选后调用共享选号逻辑；`queue` 保留原顺序，`balanced` 按现有权重／等权规则。
   - 对兜底路径单独处理选号错误：设置读取失败、候选版本变化或规则不可用时保留原顺序并允许按既有流程发送；只有明确无可用账号时才返回冷却错误。
   - 兜底候选仍逐次复核账号启用、认证状态、配置版本、同池和额度阻断；不新增请求。
   - 让 `passthroughSession` 的首次 `Next` 使用重排后的候选；失败切换语义不变。

3. **自动同步组件（A）**
   - 优先把 `GoogleAutoSync` 泛化为按提供商过滤的订阅目录自动同步，保留 Google 现有行为；若耦合过大，则新增 OpenAI 专用实现并在 Deviations 说明。
   - 陈旧判定：`accounts` 中启用的 OpenAI 订阅账号在当前 `account_catalog_versions.version` 下没有任何 `account_models` 行。
   - 触发：应用启动、订阅 `OnChange`、账号更新／启用；请求合并，单 worker 串行，30 秒超时。
   - 每次应用运行中每账号每版本最多尝试一次（内存记录）；空结果视为该版本终态；失败保留旧快照，记录固定错误码。
   - 成功后调用受管客户端目录刷新并发出 `modelcatalog:updated`。

4. **可观测性与前端**
   - 新增只读内存状态（同步中／最近失败／待同步数量）或等价事件；模型目录页展示被动状态与手动重试入口。
   - 不新增数据库表；不改变现有 `ModelSync` 手动行为。

5. **文档与验证**
   - 更新 ROUTING、LOCAL_AGENTS、SUBSCRIPTION、OVERVIEW、CURRENT；评估是否需要 ADR（预计作为 ADR-018／023 的补充，不新增）。
   - 运行定向测试与 `task verify`；在隔离环境验证自动同步和请求分布，不消耗真实账号额度。

---

# Progress

- 2026-11-07 11:55：创建任务记录，Status = Planned；未修改任何代码或配置，等待用户确认方案。
- 2026-11-07 12:05：用户确认方案，Status → In Progress；按 B → A → 状态展示 → 文档顺序实施。

---

# Implementation Result

已完成实现：

- 透明入口兜底轮换：`openFallbackSession` 构造同池候选后复用 `rotateAccounts`；`queue` 保持原顺序，`balanced` 按既有权重／等权规则。选号设置读取失败或候选复核异常时退回原顺序，仅全部候选不可用时保留既有 `account_pool_cooling` 错误，避免把平衡配置问题放大成请求失败。
- 自动同步：把 `GoogleAutoSync` 泛化为通用 `AutoSync`（Google 行为与测试保持），新增 `OpenAIAutoSync`。启动、订阅授权状态变化、刷新／注销、账号更新／启用、提供商更新／启用及手动同步后触发；只扫描当前凭据版本没有任何模型行的已启用、已认证 OpenAI 订阅账号。
- 同步约束：单 worker 串行、每账号 30 秒上限、每次应用运行每账号每凭据版本最多一次；官方空目录按该版本终态处理，不做无界重试；失败保留数据库旧快照，客户端目录刷新失败与账号目录失败分开上报。
- 可观测性：新增只读 `SyncStatus`（running／stale／lastError／updatedAt），经 Wails 薄绑定返回；模型目录页展示同步中、自动同步失败与未同步数量，`modelcatalog:updated` 事件同时失效目录与状态查询。
- 测试：新增网关兜底轮换 4 项、OpenAI 自动同步 4 项；更新 Google 自动同步超时常量引用、前端目录事件与模型目录测试；重新生成 Wails 绑定。

## Files Changed

主要修改文件：

- `internal/application/gateway/account_rotation.go`、`passthrough.go`、`passthrough_rotation_test.go`
- `internal/application/modelcatalog/auto.go`、`google_auto.go`、`openai_auto_test.go`、`google_auto_lifecycle_test.go`
- `internal/transport/wails/openai_model_sync.go`、`app.go`、`subscription.go`、`accounts.go`、`providers.go`、`modelcatalog.go`、`google_model_sync.go`
- `frontend/src/services/models.ts`、`frontend/src/features/routing/model-catalog.tsx`／`model-catalog.test.tsx`／`routing-test-helpers.tsx`、`frontend/src/features/gateway/use-model-catalog-events.ts`／`.test.tsx`、`frontend/src/i18n/locales/agents-zh.ts`／`agents-en.ts`
- `frontend/bindings/omniproxy/internal/application/modelcatalog/{index,models}.ts`、`frontend/bindings/omniproxy/internal/transport/wails/appservice.ts`（生成）
- `docs/product/ROUTING.md`、`FAILOVER.md`、`LOCAL_AGENTS.md`、`SUBSCRIPTION.md`、`docs/architecture/OVERVIEW.md`、`docs/decisions/ADR-018-codex-account-rotation.md`、`docs/plans/CURRENT.md`

## Technical Notes

- 陈旧判定复用 `account_catalog_versions` 视图：当前 `version` 在 `account_models` 下没有任何行即视为待同步；空目录同步成功也保留“有版本、无模型”状态，由本次运行内的账号＋版本去重避免反复启动官方组件。
- 兜底轮换仅对 OpenAI 订阅账号生效；固定账号、冷却、额度阻断、同池与最多三次尝试语义不变，不新增上游请求。
- 目录版本语义、普通 `/v1` 路由和数据库结构均未改变；无新增依赖或迁移。

---

# Verification

## Tests

- [x] Unit Tests
- [x] Integration Tests
- [ ] E2E Tests

覆盖结果：

- `go test ./internal/application/gateway -run 'TestFallbackSession' -count=1`：PASS，覆盖 queue 顺序、balanced 分布、额度阻断、设置读取失败回退原顺序、全部冷却错误。
- `go test ./internal/application/modelcatalog -count=1`：PASS，覆盖启动补齐、空目录终态、每版本一次、失败保留旧行与稳定码、禁用／登出／API Key／其他提供商跳过、运行状态与超时。
- `go test . ./internal/...`：PASS，全部 Go 包通过（gateway 54s、modelcatalog 7s、transport/wails 18–23s）。
- 前端：`vitest run --maxWorkers=4` 146 文件／891 项 PASS；新增模型目录 7 项与目录事件断言通过。
- 未做真实上游模型同步、真实账号重新授权或原生窗口验收；全部使用隔离 SQLite 与假订阅源。

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify（测试段通过；构建依赖见说明）

## Verification Result

```text
gofmt -l main.go internal          PASS（无输出）
pnpm format:check                  PASS
go vet . ./internal/...            PASS
staticcheck . ./internal/...       PASS
pnpm lint                          PASS（0 error，2 个既有 react-refresh warning）
pnpm typecheck                     PASS
go test . ./internal/...           PASS
vitest run --maxWorkers=4          PASS（146 文件／891 项）
pnpm build                         PASS
go build -tags production ...      PASS（bin/omniproxy-verify.exe）
```

`VITEST_MAX_WORKERS=4 task verify`（Git Bash）：格式、lint、类型检查、Go 测试与前端 146 文件／891 项测试均通过；进入 `windows:common:go:mod:tidy` 后长时间无输出、无子进程，判定为环境中的 Task 构建依赖卡住，未完成该次 verify 的 build 段。已用等价的 `pnpm build` 与生产标签 `go build` 验证通过，`go.mod`／`go.sum` 未改变。

首次默认并发全量前端运行出现过 4 项既有账号页 5 秒超时和 1 项目录事件断言差异；前者单独重跑通过，后者按新增状态查询更新断言。受限并发全量运行无超时。

真实客户端／真实账号模型同步未执行，不能将隔离契约等同于真实官方目录返回。

---

# Documentation

已更新：

- [x] PRD（用户可见的目录自动补齐与兜底轮换）
- [x] Architecture（OVERVIEW 新增自动同步与兜底轮换小节）
- [ ] Tech Stack（无技术栈变化）
- [ ] Coding Rules（无规则变化）
- [ ] Testing（复用既有测试结构）
- [ ] Build（无构建流程变化）
- [x] ADR（补充 ADR-018，不新增）
- [x] CURRENT

- [x] ROUTING、FAILOVER、LOCAL_AGENTS、SUBSCRIPTION（产品行为说明）

---

# Deviations

- 原计划新增 OpenAI 专用自动同步；实际把 `GoogleAutoSync` 泛化为通用 `AutoSync`，再用 `OpenAIAutoSync` 配置 OpenAI 的「仅陈旧账号」策略。Google 行为与既有测试保持不变，避免复制约百行后台逻辑。
- 原计划状态展示可选；实际新增只读 `ModelCatalogSyncStatus` Wails 方法（无数据库表、无凭据），用于被动显示同步中／失败／未同步数量。
- 原计划不区分同步错误与客户端目录刷新错误；实现把两者分开上报，客户端刷新失败只记录日志，不让模型目录页误报账号目录同步失败。
- `task verify` 的 build 段因 Task 的 `go mod tidy` 卡住未完成；以等价的 `pnpm build` 和生产标签 `go build` 补足，未改变模块依赖。

---

# Git

## Commit

```text
Not committed by Agent.
```

---

# Result

待实现。当前仅完成任务记录，等待用户审阅确认后进入 In Progress。

---

# Completed

Pending
