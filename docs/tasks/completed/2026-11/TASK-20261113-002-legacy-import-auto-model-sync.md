# TASK-20261113-002: 导入旧版数据后自动同步模型目录

## Status

Completed

## Created

2026-11-13 09:10（Asia/Singapore）

## Request

> 《重新同步到 Agent》这个按钮，我现在不想要我手动同步了，你看看能不能我导入后就自动同步，自己管理太累了

## Goal

- 从旧版本导入账号成功后，模型目录与受管客户端在后台自动补齐，不再需要用户逐个账号点「重新同步到 AI Agent」。
- 导入仍然只新增账号，不改变用户已手工调整的客户端配置语义。
- 导入失败（例如目标不可用）不触发任何同步。

## Background

- 自动同步基础设施已经存在：`modelcatalog.StartGoogleAutoSync`（每次启动都扫描）与
  `StartOpenAIAutoSync`（只补过期目录），在登录、授权变化、新建账号时由
  `requestGoogleModelSync`／`requestOpenAIModelSync` 触发；每轮结束后调用
  `RefreshCatalog` 重写受管客户端配置，Google 目录同步同时展开 Antigravity 目录。
- 旧版数据导入（`AppService.ImportLegacyData`）此前明确不触发同步，注释写的是
  「避免同时改写正在使用的客户端配置」。结果是导入后新账号没有模型行，客户端也没有对应
  目录，用户必须逐个账号点一次同步按钮。
- 本轮用户明确要求由产品自己管理，不再手动同步。

## Scope

- `internal/transport/wails/legacy_import.go`：导入成功后请求 Google／OpenAI 目录同步。
- `internal/transport/wails/legacy_import_windows_test.go`：导入触发同步的回归测试。
- 任务记录与 `docs/plans/CURRENT.md`。

## Out of Scope

- 不改同步实现本身（`modelcatalog.AutoSync`），不改同步触发点（登录／授权／新建账号）。
- 不改前端按钮：手动同步入口保留，供同步失败后重试。
- 不改旧版数据读取与凭据加密流程。

## Existing Behavior

导入完成后返回结果即结束，账号没有模型目录，受管客户端也没有对应模型；用户需要打开网关页
逐个账号点击「重新同步到 AI Agent」。

## Expected Behavior

导入返回后，后台自动扫描新增（以及已有但过期的）账号目录，完成后重写受管客户端配置并发出
`modelcatalog:updated` 事件；失败只记录固定错误码并保留旧快照，手动按钮仍可重试。

---

# Analysis

## Related Modules

- `internal/transport/wails/legacy_import.go`
- `internal/transport/wails/google_model_sync.go`、`openai_model_sync.go`
- `internal/application/modelcatalog/auto.go`

## Dependencies

无新增依赖。

## Risks

- 同步会在后台重写受管客户端配置。这是同一入口（手动按钮）已有的行为，且导入本身的目的就是让
  新账号立即可用；写入按既有原子替换，失败不影响导入结果。
- 导入后立即同步会发起上游请求；同步有 30 秒单账号上限与并发合并，不影响界面响应。

---

# Implementation Plan

1. 创建本记录。
2. `ImportLegacyData` 成功后调用 `requestGoogleModelSync()` 与 `requestOpenAIModelSync()`。
3. 补测试：先消费启动扫描，再导入，断言两个通道各收到一次同步完成通知。
4. 跑 `task verify`，填写结果并提交。

---

# Progress

- 2026-11-13 09:10：创建记录并完成实现（本轮记录在改动后补写，见 Deviations）。
- 导入后两个同步通道各收到一次通知的回归测试通过。

---

# Implementation Result

## Changes

- `internal/transport/wails/legacy_import.go`：导入成功后调用 `requestGoogleModelSync()` 与
  `requestOpenAIModelSync()`；导入失败仍直接返回错误，不触发任何同步。原「不触发目录同步」
  的注释同步更新。
- `internal/transport/wails/legacy_import_windows_test.go`：在隔离 profile 中启动
  `StartGoogleAutoSync`／`StartOpenAIAutoSync` 并注入 `AppService`，先消费启动扫描，
  再执行导入，断言两条通道都收到由本次导入触发的完成通知。

## Files Changed

- `internal/transport/wails/legacy_import.go`
- `internal/transport/wails/legacy_import_windows_test.go`

## Technical Notes

- 复用既有自动同步实例，不新建 goroutine 与定时器；`Request()` 自带合并，连续多次导入只会
  追加一轮扫描。
- 同步是后台任务，导入返回值与写库结果不受其影响。

---

# Verification

## Tests

- [x] Unit Tests：`go test ./internal/transport/wails/ -run TestLegacyImport -v` 通过。
- [x] Go 全量：`go test . ./internal/...` 通过。
- [ ] Integration Tests：需要在安装版上导入真实旧版数据后观察目录补齐（待用户实机）。
- [ ] E2E Tests：N/A

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task verify
PASS
```

---

# Documentation

- [x] CURRENT：记录导入后自动同步
- [x] PRD／Architecture／Tech Stack／Coding Rules／Testing／Build／ADR：无需修改

---

# Deviations

- 本轮沿用连续诊断，代码改动先于本记录写出，记录随后补齐（含实际改动与验证）。

---

# Git

## Commit

```text
c0c06f2f feat(gateway): 导入旧版数据后自动同步模型目录
```

归档提交见 `docs/tasks/completed/2026-11/`。未打标签、未推送。

---

# Result

导入旧版数据后模型目录与受管客户端由后台自动补齐，用户不再需要手动点击「重新同步到 AI Agent」。

---

# Completed

2026-11-13 08:52
