# TASK-20261116-001: HowToFix 官方模型自动同步与手动同步入口

## Status

In Progress

## Created

2026-11-16 08:25（Asia/Singapore）

## Request

用户装上 `v2.3.1-beta.7` 后在 HowToFix 账号页截图反馈：

> 我看这个没有同步官方模型啊，能不能自动同步官方模型？

截图同时暴露出账号页顶部仍显示「余额查询和密钥有效性检测尚未接入，下方仅显示已保存的账号信息」，
而四张 HowToFix 账号卡已经显示了 `US$50.00` 余额。

## Goal

- HowToFix 账号的目录同步不需要用户每次手动点：应用启动与账号／服务商变更后在后台补齐缺失目录。
- 路由页「同步官方模型」能选到 HowToFix 账号（当前可选清单里没有它）。
- 账号页在 HowToFix（已支持余额）时不再显示「余额尚未接入」的过期提示。

## Background

- `v2.3.1-beta.7` 只实现了 `modelcatalog.Sync` 对 `howtofix` 的分支，没有接任何入口：
  路由页的「可同步账号」过滤器（`frontend/src/features/routing/model-catalog.tsx`）按 kind
  白名单列出 stepfun／deepseek／river／google／mimo／openai／kimi／minimax，
  **不含 howtofix**，因此界面上没有任何地方能触发 HowToFix 的目录同步。
- 后台自动同步已存在（`internal/application/modelcatalog/auto.go` 的 `StartAutoSync`），
  但只有 OpenAI 与 Google 各有一个实例（`StartOpenAIAutoSync`／`StartGoogleAutoSync`），
  其余 API Key 服务商都依赖手动同步。
- 账号页的「余额尚未接入」提示按 kind 逐项排除，`howtofix` 未加入排除链，
  所以在余额已经能显示的情况下仍显示过期文案。

## Scope

- 前端：路由页可同步账号清单加入 `howtofix`；账号页余额提示排除 `howtofix`。
- 后端：新增 HowToFix 的后台目录自动同步（只处理当前版本没有目录的账号），
  在启动／账号增删改／服务商启停／手动同步成功后触发，并在应用退出时关闭。
- `ModelCatalogSyncStatus` 合并多个自动同步实例的只读状态（运行中、待同步数量、错误码）。
- 测试：自动同步只处理过期账号且不碰其他服务商；状态合并；前端可同步清单与提示。

## Out of Scope

- 不为其他 API Key 服务商（答题：Oxion AI 等）新增自动同步或修正同样的入口缺口。
- 不改变同步失败时保留上次成功目录的既有语义，不改目录内容与能力声明。
- 不新增迁移、不改网关转发、不改余额读取逻辑。

## Existing Behavior

- HowToFix 账号添加后没有任何同步入口，路由页下拉里看不到它，模型目录为空。
- 只有 OpenAI／Google 会在启动后后台同步；其他服务商需要手动点「同步官方模型」。

## Expected Behavior

- 添加 HowToFix 账号后无需手动操作：后台在启动或配置变更后补齐该账号的官方模型目录，
  完成后刷新受管客户端目录；失败时保留原快照并允许手动重试。
- 路由页的下拉能选中 HowToFix 账号并手动重新同步。
- 账号页不再对 HowToFix 显示「余额尚未接入」。

---

# Analysis

## Related Modules

- `internal/application/modelcatalog/auto.go`：通用 `StartAutoSync`（kind + staleOnly），无需改动。
- `internal/transport/wails/openai_model_sync.go`：单实例自动同步的模板（启动／请求／关闭／状态）。
- `internal/transport/wails/app.go`：`AppService` 字段与启动／退出钩子。
- `internal/transport/wails/{accounts,providers,modelcatalog,legacy_import}.go`：变更后请求同步的调用点。
- `frontend/src/features/routing/model-catalog.tsx`：可同步账号过滤。
- `frontend/src/features/accounts/provider-accounts-view.tsx`：余额提示的 kind 排除链。

## Dependencies

无新增依赖。

## Risks

- 自动同步会并发读取账号 Key：复用既有 `UseAPIKey` 受控回调，日志只写固定错误码。
- 同步触发过频：`staleOnly` 按「账号 + 凭据版本」在单次运行内去重，且只在目录为空时发起。
- 状态合并若把两个实例混淆，会让用户看到错误的「同步中／失败」提示；用纯函数合并并加单测。

---

# Implementation Plan

1. 新增 `internal/transport/wails/howtofix_model_sync.go`：单实例自动同步（kind `howtofix`、
   `staleOnly`、刷新受管目录、失败写固定错误码）。
2. `app.go`：`AppService` 增加字段；启动与退出钩子各加一行。
3. 变更请求统一走 `requestModelSync()`（OpenAI + HowToFix），替换现有 `requestOpenAIModelSync()` 调用点。
4. `ModelCatalogSyncStatus()` 合并两个实例的状态。
5. 前端：`model-catalog.tsx` 加入 howtofix；`provider-accounts-view.tsx` 排除 howtofix 余额提示。
6. 测试：`modelcatalog` 自动同步（只同步过期账号、不碰其他服务商）；状态合并；
   前端可同步清单与余额提示；`task verify`。
7. 文档：`docs/product/HOWTOFIX.md` 补自动同步说明；CURRENT 更新；构建 `v2.3.1-beta.8`。

---

# Progress

- 2026-11-16 08:25：定位根因——后端已支持 howtofix 目录同步，但前端可同步清单缺该 kind、
  后台自动同步只有 OpenAI／Google 两个实例；建立本记录。

---

# Implementation Result

## Changes

- `internal/application/modelcatalog/auto.go`：`AutoSync` 增加 `apiKey` 标记与内部构造
  `startAutoSync`；`StartAutoSync` 保持原签名（订阅类服务商语义不变），新增
  `StartAPIKeyAutoSync`（`staleOnly` + 只处理启用的 `api_key` 账号）；扫描条件由
  写死 `kind == "openai"` 改为按该标记判断。
- `internal/application/modelcatalog/google_auto.go`：`StartOpenAIAutoSync` 改用
  `startAutoSync(..., apiKey=true)`，与之前行为等价。
- `internal/transport/wails/howtofix_model_sync.go`（新）：HowToFix 后台目录同步单实例
  （启动、请求、关闭、只读状态、固定错误码日志），kind `howtofix`。
- `internal/transport/wails/model_catalog_sync.go`（新）：`requestModelSync()` 统一通知
  OpenAI 与 HowToFix 两个后台任务；`ModelCatalogSyncStatus()` 合并两个实例的状态；
  `mergeSyncStatus` 纯函数。
- `internal/transport/wails/openai_model_sync.go`：原 `ModelCatalogSyncStatus` 改为
  `openAIStatus()`，状态合并移到新文件。
- `internal/transport/wails/app.go`：`AppService` 增加 `howtofixSync` 字段，
  启动钩子加 `startHowToFixModelSync()`、退出钩子加 `shutdownHowToFixModelSync()`。
- `internal/transport/wails/google_model_sync.go`：`bindModelCatalogEvents` 同时绑定
  HowToFix 的目录变更事件。
- `internal/transport/wails/{accounts,providers,modelcatalog,legacy_import,subscription}.go`：
  12 处 `requestOpenAIModelSync()` 改为 `requestModelSync()`，账号／服务商／授权变化后
  两个后台任务都能收到通知。
- 前端 `frontend/src/features/routing/model-catalog.tsx`：可同步账号清单加入 `howtofix`。
- 前端 `frontend/src/features/accounts/provider-accounts-view.tsx`：
  「余额查询和密钥有效性检测尚未接入」提示的 kind 排除链加入 `howtofix`。
- 文档：`docs/product/HOWTOFIX.md` 补自动同步与手动同步说明；
  `docs/product/PRD.md` 同步措辞；新增发行说明 `docs/releases/v2.3.1-beta.8.md`。

## Files Changed

- `internal/application/modelcatalog/{auto.go,google_auto.go,howtofix_auto_test.go}`
- `internal/transport/wails/{howtofix_model_sync.go,model_catalog_sync.go,model_catalog_sync_test.go,openai_model_sync.go,app.go,google_model_sync.go,accounts.go,providers.go,modelcatalog.go,legacy_import.go,subscription.go}`
- `frontend/src/features/routing/model-catalog.tsx`、`model-catalog.test.tsx`
- `frontend/src/features/accounts/provider-accounts-view.tsx`、`provider-accounts-view.test.tsx`
- `docs/product/{HOWTOFIX.md,PRD.md}`、`docs/releases/v2.3.1-beta.8.md`

## Technical Notes

- 自动同步沿用既有「staleOnly + 账号／凭据版本去重」语义，只在目录为空时访问上游，
  不引入定时器或轮询；失败保留上次成功快照，状态只暴露固定错误码。
- 合并状态只合并 OpenAI 与 HowToFix：Google 订阅的后台任务状态不在原有展示范围内，
  本轮不改动，避免顺带改变用户已看到的提示语义。
- 扫描条件的通用化保留了 OpenAI 原有的「订阅账号 + API Key 账号」组合，
  Google 仍只处理已授权订阅账号（`apiKey=false`），行为不变。
- 发现的问题但未处理：Oxion AI 同样有「路由页可同步账号清单缺该 kind」的入口缺口，
  属另一服务商的独立问题，本轮不顺手改，需另行安排。

---

# Verification

## Tests

- `internal/application/modelcatalog`：`TestAPIKeyAutoSyncFillsOnlyStaleEnabledAccounts`
  （只同步启用的 HowToFix API Key 账号，停用账号／其他服务商／订阅账号都跳过；
  首次同步后目录只有该账号的模型；同一运行内再次请求不再访问上游）与
  `TestAPIKeyAutoSyncSkipsDisabledProvider`（停用服务商不访问上游、目录保持为空）。
- `internal/transport/wails`：`TestMergeSyncStatusCombinesProviderInstances`
  （运行状态取或、待同步数量求和、错误码取第一个、时间取最近）与
  `TestMergeSyncStatusKeepsEmptyStatusZero`。
- 前端：路由页 `HowToFix API Key 可手动同步官方模型`（下拉能选到并触发 `ModelSync`）；
  账号页 `hides the stale balance notice for HowToFix` 与
  `keeps the balance notice for providers without a balance query`。
- 全量：`go test . ./internal/...` 全部包通过；前端 155 文件／943 项测试通过。

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task verify（随 build-release.ps1 -Version v2.3.1-beta.8 执行）
EXIT=0
```

format check（gofmt＋prettier）→ lint（go vet＋staticcheck＋eslint）→ typecheck（tsc --noEmit）
→ test（Go 全包；前端 155 文件／943 项）→ build（Windows amd64 production）→ NSIS 打包 →
原生版本资源校验 → 发布资产校验，全部通过；安装器 21,085,494 bytes。

未验证：真实桌面窗口里用真实 HowToFix 账号的启动同步时序，以及 Claude Code／Codex
模型菜单中的实际出现时机；本机验证使用 loopback 假上游。

---

# Git

Not committed by Agent.

---

# Result

HowToFix 账号的官方模型目录不再需要手动触发：应用启动与账号／服务商变更后后台自动补齐
缺失目录，路由页仍可手动重新同步，账号页也不再显示过期的「余额尚未接入」提示。
自动同步沿用既有的「只在没有目录时发起 + 按账号与凭据版本去重 + 失败保留旧快照」语义，
不新增定时器或轮询。

---

# Completed

2026-11-16 08:35（Asia/Singapore）
