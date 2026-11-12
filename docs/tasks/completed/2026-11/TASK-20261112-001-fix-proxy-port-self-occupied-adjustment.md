# TASK-20261112-001: 保存当前正在使用的网关端口不再被误判为占用

## Status

Completed

## Created

2026-11-12 08:45

## Request

用户发现「设置与关于 → 通用设置 → 本地网关端口」的端口检测有 bug：

> 在这个端口检测时，如果本来软件就处于打开的状态，它本身会自动占用3000端口，此时再点击保存端口，
> 它会自动切换到3001再保存，这个不对，它本身就是自己软件占用的端口，不应该显示占用并自动切换。
> 看看该怎么修复这个bug比较好

界面此时已经显示「当前端口: 3000　默认端口: 3000　端口可用　网关正在监听」，即状态栏认知正确，
只有保存动作的探测把本进程自己监听的端口当成了被其它程序占用。

## Goal

- 网关正在运行且用户保存／恢复的端口就是它当前监听的端口时，按「无变化」处理：不避让、不换端口、
  不打断在途请求、不改写受管客户端地址，并如实返回 `Adjusted=false`。
- 端口确实被其它程序占用时，避让行为保持不变（仍在同区间内寻找可用端口）。
- 端口状态与保存结果使用同一套「自己持有的端口可用」判定，不再出现两处结论互相矛盾。

## Background

端口可配置与自动避让由 TASK-20261028-008 引入。`internal/core/netport.Available` 的判定标准是
「能否在 127.0.0.1 上再监听一次」，因此本进程已持有的端口必然探测失败。代码在一处做了例外，
在另一处没有做，于是产生本 bug。

## Scope

- `internal/transport/wails/proxy_port.go`：端口可用性判定与会话保存流程。
- `internal/transport/wails/proxy_port_test.go`：回归测试。
- `docs/product/GATEWAY.md`、`docs/plans/CURRENT.md`：行为说明。

## Out of Scope

- `internal/core/netport` 的探测语义（`Available` 表示「能否再监听」本身是正确的，不改）。
- 前端 `proxy-port-setting.tsx` 与文案（状态栏展示本就正确，无需改动）。
- 避让区间宽度（20）、环境变量锁定语义、受管客户端改写规则。
- 打包与发布。

## Existing Behavior

- `proxyPortState()`（`ProxyPort` 绑定，界面状态栏）：网关 `running` 时直接返回
  `Running=true, Available=true`，不看探测结果 —— 正确。
- `applyProxyPort()`（`SetProxyPort` 绑定，保存动作）：无条件调用 `netport.Available(target)`。
  目标就是本进程正在监听的端口时探测失败，于是 `NextAvailable` 跳到下一个端口并 `Adjusted=true`，
  随后停止监听、落盘新端口、改写受管客户端、重新监听。
- 结果：软件开着时「保存端口」填当前端口 → 3000 被换成 3001；开着时点「恢复默认端口」且默认端口
  就是当前端口 → 同样被换成 3001。状态栏仍显示「端口可用」，与保存结果自相矛盾。

## Expected Behavior

- 网关在自己监听的端口上运行时保存该端口：`Adjusted=false`、`State.Port` 仍为该端口、监听器不重启
  （`StartedAt` 不变、在途请求不受影响）、不产生受管客户端改写。
- 保存其它可用端口：按原逻辑切换并重启监听。
- 保存被其它程序占用的端口：按原逻辑在同区间内避让并落盘。
- 网关未运行时保存端口：仍以真实探测结果判断（含「被其它程序占用」）。

---

# Analysis

## Related Modules

- `internal/transport/wails/proxy_port.go`：`ProxyPortState`／`ProxyPortUpdate`、
  `proxyPortState()`、`applyProxyPort()`。
- `internal/core/netport`：`Available`／`NextAvailable`（只读依赖，不改）。
- `internal/application/gateway`：`Info().Runtime`（`proxy.Status` 快照，`State`／`Port`）。
- `internal/core/proxy.Server`：`Start` 记录真实监听端口，`serve` 失败置 `error`，
  `Stop` 置 `stopped` —— 只有 `running` 表示端口由本进程持有。

## Dependencies

无新增依赖。

## Risks

- 若把「自己持有」的判定放宽成「配置端口等于运行端口」（不看 `State`），
  服务在 `stopping`／`error` 时可能漏掉真实冲突并让重启失败。判定必须以
  `Runtime.State == "running"` 为前提。
- 过度跳过探测：只跳过 `target` 恰等于本进程监听端口的情形；其它端口一律照常探测。
- `Info()` 会读取模型目录（一次数据库查询）。保存流程原本已调用一次，本次保持调用次数不变
  （在 `applyProxyPort` 内只取一次运行时快照用于探测与 `running` 判断）。

## Root Cause

可用性探测只回答「还能不能再监听一次」，无法区分「被别的程序占用」与「被本进程自己占用」。
`proxyPortState()` 手工加了例外（`running` 时视为可用），`applyProxyPort()` 漏加同一例外，
于是保存自己正在使用的端口时被自动避让。

## Fix

在桌面适配层集中表达「本进程持有的端口」这一事实：新增 `listeningPort()`（仅在
`Runtime.State == "running"` 时返回 `Runtime.Port`，否则返回 0）。`proxyPortState()` 与
`applyProxyPort()` 都基于同一判定：目标端口等于本进程监听端口时跳过探测；不等时才走
`netport.Available`／`NextAvailable`。

## Regression Prevention

`internal/transport/wails/proxy_port_test.go` 新增：

- 运行中保存当前端口：`Adjusted=false`、端口不变、`StartedAt` 不变（证明监听器未重启）。
- 运行中 `ProxyPort()` 状态：`Running=true` 且 `Available=true`。
- 运行中保存被其它程序占用的端口：仍然避让（防止把「跳过探测」放宽成「永不避让」）。

---

# Implementation Plan

1. `proxy_port.go` 新增 `listeningPort()`；`proxyPortState()` 改用它表达「自己持有的端口可用」。
2. `applyProxyPort()` 先取一次运行时快照，`target` 等于该端口时跳过 `netport.Available` 探测。
3. 补上述 3 项回归测试，并确认它们在修复前会失败。
4. 运行 `gofmt`、`go vet`、`staticcheck`、目标包测试与 `task verify`。
5. 更新 `docs/product/GATEWAY.md`（保存当前端口按无变化处理）与 `docs/plans/CURRENT.md`。

---

# Progress

- 已定位根因：`proxyPortState()` 与 `applyProxyPort()` 对「自己持有的端口」判定不一致。
- 方案取舍：不改 `netport.Available` 语义（它正确表示「能否再监听」），在桌面层用运行时状态补上
  「本进程已持有」的例外，避免探测结果污染业务判断。
- 实现与 3 项回归测试完成；`go test ./internal/transport/wails/`、`go vet`、`staticcheck` 通过。

---

# Implementation Result

## Changes

实际完成：

- 新增 `AppService.listeningPort()`：只在 `Runtime.State == "running"` 时返回 `Runtime.Port`，
  否则返回 0，集中表达「本进程当前持有哪个端口」。
- `proxyPortState()` 改用它判断：运行中且配置端口就是该端口时 `Available=true`（初版等价逻辑保留）；
  配置端口与运行端口不一致时仍走真实探测，不再无条件报可用。
- `applyProxyPort()` 在避让前先取一次运行时快照：目标端口等于本进程监听端口时跳过
  `netport.Available` 探测；`running` 也复用同一快照，避免重复读取网关信息。
- 结果：软件运行中保存当前端口（或「恢复默认端口」正好是当前端口）按无变化处理，
  `Adjusted=false`，不停止/重启监听器、不取消在途请求、不改写受管客户端、不落盘新端口。
- 新增 3 项回归测试：运行中保存当前端口、运行中状态栏报可用、运行中保存被其它程序占用的端口仍避让。

## Files Changed

- `internal/transport/wails/proxy_port.go`
- `internal/transport/wails/proxy_port_test.go`
- `docs/product/GATEWAY.md`
- `docs/plans/CURRENT.md`

## Technical Notes

- 不改 `netport.Available` 语义：「能否在 127.0.0.1 上再监听一次」是正确回答，
  只是它无法区分占用者是谁。调用方必须自己排除本进程持有的端口。
- 判定前提是 `Runtime.State == "running"` 而非「`Port` 非零」：`serve` 失败后状态为 `error`
  但仍保留旧 `Port`，若只看端口号会把真实冲突当成自己持有而漏掉避让。
- 回归测试用 `Runtime.StartedAt` 未变化来证明监听器没有重启，这是「不打断在途请求」的可观测代理。

---

# Verification

## Tests

- [x] Unit Tests：`internal/transport/wails` 新增 3 项端口回归测试；修复前
  `TestSetProxyPortKeepsCurrentRunningPort` 失败（`saving port 55021 was adjusted away to 55022`），
  修复后全部通过。
- [x] Integration Tests：`go test ./internal/transport/wails/ ./internal/core/netport/ -count=1` 通过；
  `task verify` 的 `go test . ./internal/...` 覆盖全部包。
- [ ] E2E Tests：N/A（本改为后端判定，界面状态栏文案与绑定结构未变；原生窗口未自动覆盖）。

## Quality Checks

- [x] Format（`gofmt -l internal/transport/wails` 无输出；`prettier --check .` 通过）
- [x] Lint（`go vet`、`staticcheck` 通过；ESLint 0 error，仅 2 条未改动文件 `reset-credit-list.tsx` 的既有 `react-refresh` warning）
- [x] Type Check（`tsc --noEmit` 通过）
- [x] Test（Go 全部包通过；前端 146 文件／893 项在限 4 worker 下通过，默认并发见下）
- [x] Build（`task build` exit 0，含 Wails syso 生成、前端生产构建与 `bin/omniproxy.exe` 生产编译）
- [x] Verify（已运行，结果见下）

## Verification Result

```text
task verify
FAIL (exit 201) —— 失败点仅为本机 Vitest 默认并发下的既有超时，与本次改动无关
```

逐项结果：

- format:check PASS（`gofmt -l main.go internal` 无输出；prettier 通过）
- lint PASS（`go vet . ./internal/...`、`staticcheck . ./internal/...` 通过；ESLint 0 error，
  仅 2 条未改动文件 `reset-credit-list.tsx` 的既有 `react-refresh` warning）
- typecheck PASS（`tsc --noEmit`）
- Go 测试 PASS（`go test . ./internal/...` 全部包，含本次 `internal/transport/wails` 27s 全量通过）
- **前端测试：默认并发下不稳定**。两次 `task verify` 分别有 3 项／4 项 5 秒超时
  （第一次 `stepfun.test.tsx`×2、`kimi-code.test.tsx`；第二次 `mimo.test.tsx`、
  `agent-installation.test.tsx`、`default-model-routes.test.tsx`、`routing-grouping.test.tsx`），
  失败文件每次不同，且伴随 Vitest forks worker 启动超时的 unhandled error。
  限 4 worker 全量重跑 **146 文件／893 项全部通过**；第二次失败涉及的 4 个文件单独重跑
  **27 项全部通过**。这与 `CURRENT.md` 已记录的同类抖动一致（「`task verify` 在默认并发下因
  无关既有文件的 5 秒超时中断，限并发重跑通过」），未修改测试配置，也未放宽任何断言。
- build PASS（`task build` exit 0）

结论：本次改动涉及的 Go 包测试、3 项端口回归测试、format／lint／typecheck／build 全部通过；
`task verify` 的失败点是前端默认并发下的既有超时抖动，不在本次改动范围内，也未通过修改
测试配置掩盖。

---

# Documentation

- [x] PRD：无需修改（PRD 只描述「端口被占用时自动避让」，未定义「自己占用」这一细节）
- [x] Architecture：无需修改（未改变模块边界，探测语义仍在 `netport`）
- [x] Tech Stack：无需修改
- [x] Coding Rules：无需修改
- [x] Testing：无需修改
- [x] Build：无需修改
- [x] ADR：无需新增（不是架构决策，只是补齐同一判定的一致用法）
- [x] CURRENT：已加入本次完成项

---

# Deviations

- 原计划只修 `applyProxyPort`；实际同时收紧了 `proxyPortState` 的「运行即可用」捷径，
  改为「运行且配置端口就是该端口才可用」。原因：同一事实出现在两处就应该由同一个判定表达，
  否则「配置端口与运行端口不一致」时会谎报可用。
- 原计划测试里断言配置文件未被改写；实际未加。原因是该断言依赖测试直接改写 `prof.ProxyPort`
  而未落盘的细节，表达意图不如 `Adjusted=false` + `StartedAt` 不变直接。
- 原计划「`task verify` 一次通过」；实际两次都在前端默认并发下超时中断（失败文件每次不同），
  以限 4 worker 全量重跑与失败文件单独重跑补证。未修改 Vitest 配置，避免用配置掩盖既有抖动。

---

# Git

## Commit

```text
15842462 fix(gateway): 保存当前正在监听的端口不再被误判为占用
```

本次提交包含代码、测试与 `docs/product/GATEWAY.md`；任务记录与 `docs/plans/CURRENT.md`
更新单独提交。工作区中的未跟踪文件 `docs/tasks/completed/2026-10/TASK-20261029-010-subq-provider.md`
与 `docs/tasks/active/TASK-20261112-002-google-subscription-agent-chain-parity.md` 与本任务无关，
保持原样未提交。未推送远端。

---

# Result

软件开着时，在「本地网关端口」点「保存端口」填入当前端口（例如生产环境 3000）不再被自动避让：
该端口由本应用自己持有，按无变化处理，监听器不重启、在途请求不取消、受管客户端地址不改写、
配置不落盘新端口，界面也不再出现「已自动改用 3001」的提示。「恢复默认端口」得到的正是当前端口时
同样按无变化处理。被其它程序占用的端口仍然照常在同区间内避让，行为不变。

---

# Completed

2026-11-12 09:30
