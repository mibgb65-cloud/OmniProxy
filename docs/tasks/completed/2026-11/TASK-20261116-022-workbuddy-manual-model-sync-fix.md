# TASK-20261116-022: 修复 WorkBuddy AI 手动同步模型失败（Wails 目录服务未接线）

## Status

Completed

## Created

2026-11-16 22:20（Asia/Singapore）

## Request

`v2.3.1-beta.17` 上在「路由 → 模型目录」选中 WorkBuddy AI 账号点「同步官方模型」，
出现红色「同步失败，保留上次成功的目录。请检查账号授权与网络。」；而账号后台自动同步
其实已经写入 25 条模型，`GET /v3/config` 用当前令牌也是 200。

## Goal

让手动同步与自动同步使用同一套已接线的目录服务：WorkBuddy AI 账号点「同步」应成功并返回
`/v3/config` 目录；同时让这类失败不再只显示通用文案（日志保留真实错误码）。

## Background

- 数据库证据：`account_models` 有 25 行，`account_version` 与
  `account_catalog_versions.version` 一致（目录有效）；`model_routes` 为 0（还没建规则）。
- 上游证据：用账号当前令牌 `GET https://www.workbuddy.ai/v3/config` → `200`，21,047 字节。
- 日志：只有 `WorkBuddy AI model auto sync incomplete`（beta.17 起带码，但仍是回退码），
  手动同步失败没有任何日志。

## Analysis

### 根因：`AppService.modelCatalog()` 只接入了 Google，没有接入 WorkBuddy

`internal/transport/wails/modelcatalog.go` 的 `modelCatalog()`（`ModelCatalog()` 与
`ModelSync()` 共用）构造 `modelcatalog.Service` 后只调用了 `SetGoogleSubscription`。
`modelcatalog.Sync` 的 WorkBuddy 分支第一句是：

```go
} else if kind == workbuddyai.Kind && current.AuthMethod == account.AuthWorkBuddyLogin {
    if s.workbuddyAuth == nil {
        return nil, ErrUnavailable
    }
```

因此**手动同步对 WorkBuddy 账号 100% 失败**，且 `Sync` 把底层错误统一替换成
`ErrUnavailable`，`ModelSync` 也没有日志，界面只能显示「同步失败，请检查账号授权与网络」。
自动同步不受影响：`startWorkBuddyModelSync()` 自己构造 source 并调用了
`SetWorkBuddySubscription`——两条路径各建一套服务，接线只有一处，漏掉就静默失效。

### 同类隐患

`startGoogleModelSync()`／`startWorkBuddyModelSync()`／`modelCatalog()` 各自
`modelcatalog.New(...)` + 手动 `Set*`，新增订阅通道时容易再次漏接。

## Scope

- `internal/transport/wails/modelcatalog.go`：`modelCatalog()` 补
  `SetWorkBuddySubscription`，并把 Codex／Google／WorkBuddy 的接线收敛到一个构造函数，
  三处调用点共用。
- `internal/application/modelcatalog/sync.go`：失败时保留底层错误（`errors.Join`），
  不再只返回 `ErrUnavailable`，让调用方能记录真实错误码。
- `internal/transport/wails/modelcatalog.go`：`ModelSync` 失败记 WARN（稳定码）。
- 回归测试：Wails 层「手动同步 WorkBuddy 账号」端到端（假上游 `/v3/config`），
  以及「目录服务接线齐全」的断言。
- 文档：`docs/product/WORKBUDDY.md` 说明手动/自动同步共用同一服务。

## Out of Scope

- 不改自动同步策略（stale-only、重试时机）。
- 不改其他服务商的同步行为；统一构造函数只收敛接线，不改变语义。
- 不改前端文案（仍是通用提示，真实原因进日志）。

---

# Implementation Result

## Changes

- **修复接线**（根因）：`internal/transport/wails/modelcatalog.go` 新增
  `wireModelCatalog`，统一给目录服务接上 Google 与 WorkBuddy 订阅；`modelCatalog()`
  （`ModelCatalog`／`ModelSync` 共用）现在调用它，WorkBuddy 手动同步不再落进
  `s.workbuddyAuth == nil` 的 `unavailable`。
- **自动同步共用同一接线**：`startGoogleModelSync`／`startWorkBuddyModelSync` 也改用
  `wireModelCatalog`，去掉各自 `Set*`，以后新增订阅通道不会再只接一条路径。
- **诊断**：`modelcatalog.Sync` 对外仍只返回 `ErrUnavailable`（不泄漏上游细节），
  但把底层错误记录到 `Service.LastSyncError()`；`ModelSync` 失败时记 WARN
  （`module=modelcatalog,operation=manual,code=<稳定码>`），取不到码才回退 `model_sync_failed`。
- **测试**：
  - `internal/transport/wails/workbuddy_model_sync_windows_test.go`（新）：Wails 层端到端
    手动同步 WorkBuddy 账号（假 `/v3/config`），断言返回 2 条模型、DB 行数、目录版本绑定；
    去掉接线时该测试以 `models: unavailable` 失败（已验证）。
  - `internal/application/modelcatalog/workbuddy_test.go`（新）：同步失败时对外仍是
    `models: unavailable`，而底层 `unauthorized` 稳定码保留给日志。
- `docs/product/WORKBUDDY.md`：说明手动同步与自动同步共用同一接线，并记录这个回归点。

## Files Changed

- `internal/transport/wails/modelcatalog.go`、`google_model_sync.go`、`workbuddy_subscription.go`
- `internal/application/modelcatalog/service.go`、`sync.go`、`workbuddy_test.go`（新）
- `internal/transport/wails/workbuddy_model_sync_windows_test.go`（新）
- `docs/product/WORKBUDDY.md`

## Technical Notes

- 用户侧证据：`account_models` 已有 25 行且 `account_version` 与
  `account_catalog_versions.version` 一致；当前令牌 `GET /v3/config` → 200（21,047 字节）。
  说明上游与凭据都没问题，失败纯在本地接线。
- `LastSyncError` 只在内存里保存最近一次失败的错误对象（请求范围内），对外接口不变。
- 无数据库结构变化、无新增依赖。

---

# Verification

## 质量门（`task verify`，2026-11-16 22:12–22:26 Asia/Singapore）

```text
task verify → exit 0
```

| 阶段 | 结果 |
| --- | --- |
| format | gofmt 无输出；prettier 通过 |
| lint | `go vet`／`staticcheck` 无告警；`eslint` 0 error（2 条既有 warning） |
| typecheck | `tsc --noEmit` exit 0 |
| test | Go 全包通过；前端 157 files / 965 tests |
| build | 绑定 + vite + Windows production 构建通过 |

## 回归有效性

- 新增的 Wails 端到端测试在**保留原缺陷**（`wireModelCatalog` 不接 WorkBuddy）时失败：

  ```text
  --- FAIL: TestWorkBuddyManualModelSyncUsesWiredSubscription
      workbuddy_model_sync_windows_test.go: manual sync failed: models: unavailable
  ```

  恢复接线后通过。这与用户看到的现象一致（自动同步成功、手动同步失败）。
- 第一次加诊断时把底层错误 `errors.Join` 进对外错误，触发 6 个既有测试失败
  （它们要求 `err.Error() == "models: unavailable"`，即不向界面泄漏细节）；
  已改为带外记录 `LastSyncError()`，既有测试与语义都保持不变。

## 未验证

- 安装 `v2.3.1-beta.18` 后在界面点「同步官方模型」（需用户确认）。
- 真实对话与计费扣减（消耗积分）。

---

# Documentation

- [x] `docs/product/WORKBUDDY.md`（手动/自动同步共用接线与回归点）

---

# Deviations

- 计划只写「补接线 + 日志」，实际还收敛了自动同步的两处构造：否则同一个漏接缺陷可以在四条
  路径上复发；收敛后 `startGoogleModelSync`／`startWorkBuddyModelSync` 与手动同步完全一致。
- 诊断实现改了两次：`errors.Join` 会把底层文本带进对外错误（违反既有「不泄漏细节」约定，
  6 个测试立刻失败），最终改为 `LastSyncError()` 带外记录。
- 未改前端文案：界面仍是通用「同步失败」，真实原因进日志（与既有约定一致）。

---

# Git

## Commit

87bd9b05 fix(wails): 修复 WorkBuddy AI 手动同步模型失败（目录服务漏接订阅）

---

# Result

「点同步就失败、但模型其实已同步」是本地接线问题：手动同步在 Wails 层只接入了 Google 订阅，
WorkBuddy 分支直接返回 `unavailable`；自动同步用的是另一套已接线服务，所以 25 条模型在库里、
按钮却一直报「同步失败」。修复后两条路径共用 `wireModelCatalog`，手动同步返回真实目录；
失败时底层稳定码进日志，界面保持一句通用提示。

---

# Completed

2026-11-16 22:28（Asia/Singapore）