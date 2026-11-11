# TASK-20261111-001: 单实例运行并在重复启动时唤起已有窗口

## Status

Completed

## Created

2026-11-11 21:50

## Request

用户发现本机同时出现了两个「生产端」OmniProxy，不符合预期：

> 我发现了一个问题，我现在怎么启动了两个生产端的 OmniProxy，不应该是这样的，如果已经启动了应该是直接启动已经启动的窗口吧

期望：应用已经运行时再次启动，只把已经运行的窗口带到前台，不再产生第二个进程。

## Goal

同一个运行环境（profile）同一时刻只允许一个 OmniProxy 进程：

- 重复启动只唤起已有窗口并立即退出，不打开数据库、不执行迁移、不改写配置、不占用端口、不产生第二个托盘图标。
- 生产／开发／测试三个 profile 互相独立，仍可同时运行。
- 更新安装流程不变：安装器先停止旧进程，再启动新版本。

## Background

用户日志（`%APPDATA%\OmniProxy\v2\logs\omniproxy.log`）记录了 2026-11-11 20:31:25 的重复启动：

```text
20:31:25.671 INFO  starting OmniProxy  profile=production  app_id=com.omniproxy.app  proxy_port=3000  control_port=3890
20:31:25.937 WARN  installer quit listener unavailable  error="Cannot create a file when that file already exists."
20:31:25.944 ERROR control API failed to start; external clients cannot connect
                   error="listen tcp 127.0.0.1:3890: bind: Only one usage of each socket address ..."
```

`CURRENT.md` 已知缺口条目原文：`Single-instance protection is not wired yet; a manual launch after autostart can start a second process.` 本机注册表自启项为 `HKCU\...\Run\com.omniproxy.app = E:\codexproxy\OmniProxy\OmniProxy.exe`，开机自启后再次手动启动即复现。

## Scope

- 新增 `internal/platform/singleinstance`：Windows 命名互斥量 + 激活事件；非 Windows 空实现（产品只交付 Windows）。
- `main.go`：在 `profile.Detect()` 之后、`resolveDatabaseProfile`／打开数据库之前取得守卫。
- 桌面壳 `internal/transport/wails`：窗口创建后注册「唤起窗口」回调。
- 单元测试与回归测试。
- 文档：`docs/architecture/OVERVIEW.md`、`docs/plans/CURRENT.md`。

## Out of Scope

- 安装器与自动更新判定流程（现有 `stopInstalled` 已先等待旧进程退出）。
- 控制端口／网关端口占用后的重试或自动换端口（既有告警行为不变）。
- 跨登录会话共享同一实例（使用 `Local\` 命名空间，按 Windows 会话隔离，与按用户的数据目录一致）。
- 托盘图标、窗口布局等界面改动。

## Existing Behavior

- 无任何单实例保护。第二次启动完整走启动流程：读配置、可能隔离旧库并改写 `app.json`、打开 SQLite、应用迁移、初始化日志、创建第二个窗口与托盘图标；控制端口 3890 与网关端口 3000 已被占用时只记录错误。
- 两次启动写同一份日志与同一份配置，第二个进程即使立刻不可用也会触碰共享状态。

## Expected Behavior

- 第一个进程持有命名互斥量 `Local\OmniProxy.Instance.<app id>`。
- 第二次启动在读写任何共享状态之前发现互斥量已被持有：向已有进程发送激活事件（`Local\OmniProxy.Activate.<app id>`）后以 0 退出。
- 已有进程收到激活事件后显示并聚焦主窗口；窗口在托盘隐藏或最小化时都能恢复。
- 已有进程无响应时不降级启动第二个实例（标准单实例语义）。

---

# Analysis

## Related Modules

- `internal/platform/singleinstance`（新增）：进程级锁与激活通知。
- `main.go`：启动顺序与守卫的取得位置。
- `internal/transport/wails/app.go`：`App` 结构、窗口创建与激活回调绑定。
- `internal/platform/profile`：`AppID` 作为 profile 身份（`com.omniproxy.app` / `.dev` / `.test`）。

## Dependencies

无新增依赖：只用既有的 `golang.org/x/sys/windows`（`go.mod` 已直接依赖 v0.48.0）与 Wails 3.1.0。

## Risks

- Low：已有进程卡死时重复启动唤不起窗口；这是单实例的标准取舍，不降级为第二个进程。
- 激活回调可能在 Wails 主循环启动前到达。Wails 的 `InvokeSync` 会阻塞等待主线程，在主线程上调用会自锁，因此回调必须交给独立协程。
- 更新后首次启动若旧进程尚未完全退出，新进程会退出而不启动。安装器 `stopInstalled` 已等待进程消失，行为不变；需在验证中确认更新流程未被破坏。
- 开发模式同时运行两份源码（例如两个 `task dev`）时第二个只会唤起第一个窗口；这是本需求的预期语义。

## Root Cause

产品要求单实例（`PRD.md` Goals／Core Features），但桌面壳只创建窗口与托盘，从未申请进程级锁；Wails 3.1.0 自带 `SingleInstance` 选项，其检查点在 `application.New`，晚于 `profile.Detect()`、`resolveDatabaseProfile()`、数据库打开与迁移，无法阻止第二个进程触碰共享状态（日志中的第二次启动确实走到了控制端口绑定失败才暴露）。

## Fix

在 `main()` 取得 profile 之后、任何共享状态之前取 Windows 命名互斥量；失败即发送激活事件并以 0 返回。互斥量持有者创建激活事件，桌面壳在窗口创建后注册「显示 + 聚焦」回调。

## Regression Prevention

`internal/platform/singleinstance` 新增 Windows 测试：

- 同一 ID 第二次 `Acquire` 返回 `ErrAlreadyRunning`，且第一次实例收到激活请求；
- 注册回调前到达的请求在注册时补发；
- 不同 ID 互不干扰；
- `Close` 后可以重新取得锁。

---

# Implementation Plan

1. 新增 `internal/platform/singleinstance`：`Guard` 接口、`ErrAlreadyRunning`、平台实现（Windows 互斥量 + 自动重置激活事件 + 停止事件；其他平台空实现）。
2. `main.go`：`Detect()` 之后调用 `Acquire(prof.AppID)`；`ErrAlreadyRunning` 直接返回，其他错误只告警不阻断启动；守卫传给桌面壳。
3. `internal/transport/wails/app.go`：`App` 增加守卫字段，窗口创建后注册回调，协程内 `Show()` + `Focus()`。
4. 补测试并运行 `go test`、`go vet`、`staticcheck`、`task verify`。
5. 更新 `OVERVIEW.md`（桌面壳）与 `CURRENT.md`（移除已知缺口、加入本次完成项）。

---

# Progress

- 已确认根因：无单实例守卫；生产日志显示两个进程在同一秒启动，第二个程序走到控制端口绑定失败。
- 方案取舍：不使用 Wails 内建 `SingleInstance`（检查点太晚，第二个进程仍会打开数据库、写配置），改为在 `main()` 早期自行加锁，并沿用 `appinstall` 既有「命名事件跨进程通知」的模式做窗口唤起。
- 实现与 6 项 Windows 单元测试完成；`go test . ./internal/...`（53 包）、`go vet`、`staticcheck`、前端 146 文件／893 项、`task build` 通过。
- 进程级验证：占位进程持有同一 ID 的锁后启动真实 exe，259 ms 内 exit 0，未创建数据目录与日志，持有者收到激活请求。
- `task verify` 首次与并行运行的前端构建／测试竞争 syso 清理步骤返回 1；`task build` 与重跑 `task verify` 均 exit 0。

---

# Implementation Result

## Changes

- 新增 `internal/platform/singleinstance`：`Guard` 接口与 `ErrAlreadyRunning`；Windows 实现用 `Local\OmniProxy.Instance.<app id>` 命名互斥量占位、`Local\OmniProxy.Activate.<app id>` 自动重置事件接收激活、一个手动重置的停止事件让等待协程干净退出；非 Windows 为空实现（产品只交付 Windows）。
- 第二个启动者：互斥量已存在时先把前台权限让给已有实例（`AllowSetForegroundWindow(ASFW_ANY)`），置位激活事件后返回 `ErrAlreadyRunning`；`main()` 直接以 0 退出。
- `main.go`：在 `profile.Detect()` 之后、`resolveDatabaseProfile`／打开数据库／迁移／初始化日志之前 `Acquire(prof.AppID)`；守卫本身出错只告警不阻断启动；守卫随 `App` 传入桌面壳。
- `internal/transport/wails`：`App.Instance` 字段；窗口创建后注册激活回调，记录一条 `another launch requested the existing window` 日志，并在独立协程里 `Show()` + `Focus()`。
- 新增 6 项 Windows 单元测试：第二次取得返回 `ErrAlreadyRunning` 且触发首次实例、注册前到达的请求在注册时补发、profile 之间互不干扰、`Close` 后可重新取得、空 ID 拒绝、名称按 profile 区分。
- 文档：`OVERVIEW.md` 桌面壳补充单实例段落；`CURRENT.md` 移除「单实例未接线」的已知缺口。

## Files Changed

- `internal/platform/singleinstance/instance.go`
- `internal/platform/singleinstance/instance_windows.go`
- `internal/platform/singleinstance/instance_windows_test.go`
- `internal/platform/singleinstance/instance_other.go`
- `main.go`
- `internal/transport/wails/app.go`
- `docs/architecture/OVERVIEW.md`
- `docs/plans/CURRENT.md`

## Technical Notes

- 不使用 Wails 3.1.0 的 `SingleInstance`：其检查点在 `application.New`，晚于 profile 探测、`resolveDatabaseProfile`（会改写 `app.json`）、数据库打开与迁移；只有自己更早加锁，重复启动才真正不触碰共享状态。
- 激活事件先于互斥量创建，因此任何看到互斥量已存在的进程一定找得到事件，不会丢掉激活请求；`requestActivation` 仍保留 2 秒、25ms 间隔的重试，只覆盖「持有者刚取得锁、尚未开始监听」的瞬间。
- 互斥量只做存在性判断（`CreateMutex` 的 `ERROR_ALREADY_EXISTS`），不申请所有权：进程退出由内核释放，卸载／更新时无需清理。
- 名称使用 `Local\` 命名空间，按登录会话隔离，与按用户的数据目录一致，快速用户切换时各自保留实例；命名方式沿用 `appinstall` 的 `Local\OmniProxy.Quit.<hash>` 模式。
- 激活回调统一放进协程：Wails 的 `Show`／`Focus` 内部走 `InvokeSync` 同步等待主循环，若在注册时的补发路径（主线程）直接调用会自锁。

---

# Verification

## Tests

- [x] Unit Tests：`internal/platform/singleinstance` 6 项 Windows 测试。
- [x] Integration Tests：`go test . ./internal/...`（53 个包全部通过）。
- [ ] E2E Tests：原生窗口显示／置前未自动覆盖（本会话 WebView2 报 `800700aa`，GUI 控制器无法创建）；已完成进程级验证，见下。

## Quality Checks

- [x] Format（`gofmt -l main.go internal` 无输出；`prettier --check .` 通过）
- [x] Lint（`go vet`、`staticcheck` 通过；ESLint 0 error，仅 2 条未改动文件 `reset-credit-list.tsx` 的既有 `react-refresh` warning）
- [x] Type Check（`tsc --noEmit` 通过）
- [x] Test（Go 53 个包；前端 146 文件／893 项在 `task verify` 中一次通过）
- [x] Build（`task build` exit 0，含 Wails 绑定生成、前端生产构建与 `bin/omniproxy.exe` 生产编译）
- [x] Verify（`task verify` exit 0）

## Verification Result

```text
task verify
PASS (exit 0)
```

覆盖 format:check、lint、frontend typecheck、Go 全部测试包、前端 146 文件／893 项、Wails 绑定生成、前端生产构建与 Windows production 编译。

首次 `task verify` 返回 1：当时代码已判定但进程级验证的前端构建正在并行运行，最后一次 `powershell Remove-item *.syso` 清理步骤竞争失败；单独执行 `task build`（exit 0）与重跑 `task verify`（exit 0）均正常，未改动 Taskfile。

进程级验证（隔离 test profile、独立数据目录、端口 3911／3912，避免触碰生产数据）：

- 无其它实例时启动：应用正常写入 `starting OmniProxy` 并创建数据目录。
- 占位进程持有同一 `com.omniproxy.app.test` 锁后启动真实 exe：259 ms 内以退出码 0 结束，**未创建数据目录、未写日志**；占位进程收到 `activation`。
- 该验证覆盖了「不触碰共享状态 + 激活通知」两件事，但未覆盖原生窗口真实的显示与置前，需用户在实机确认（`800700aa` 使本会话无法创建 WebView2 控制器）。

---

# Documentation

- [x] PRD：无需修改（PRD 已把 single instance 列为产品行为，本次只是补齐实现）
- [x] Architecture：`docs/architecture/OVERVIEW.md` 桌面壳补充单实例段落
- [x] Tech Stack：无需修改
- [x] Coding Rules：无需修改
- [x] Testing：无需修改
- [x] Build：无需修改
- [x] ADR：无需新增（ADR-001 已把 single instance 列为 Wails 3 能力，本次未改变选型）
- [x] CURRENT：已更新（移除已知缺口，加入本次完成项）

---

# Deviations

- 原计划只写单元测试；实际追加了一次进程级验证（临时占位进程 + 真实 exe），验证后删除占位代码与临时目录。
- 原计划 `task verify` 一次通过；实际首次因与并行前端构建竞争 syso 清理步骤返回 1，重跑通过，未修改 Taskfile。
- 计划中未提到日志：实施时在激活回调里补了一条 `another launch requested the existing window`，便于用户与日志排查重复启动。

---

# Git

## Commit

```text
5d6fef11 fix(app): 单实例运行并在重复启动时唤起已有窗口
```

本次提交包含代码、测试与 `docs/architecture/OVERVIEW.md`；任务记录、`CURRENT.md` 更新单独提交。工作区中原有未跟踪文件 `docs/tasks/completed/2026-10/TASK-20261029-010-subq-provider.md` 与本任务无关，保持原样未提交。未推送远端。

---

# Result

同一 profile 现在只允许一个 OmniProxy 进程：重复启动在读取配置、打开数据库、执行迁移与初始化日志之前被命名互斥量拦下，只向已有实例发送激活请求并以 0 退出；已有实例把主窗口从托盘隐藏或最小化状态恢复并聚焦。生产／开发／测试 profile 各自独立，更新安装流程（先停旧进程再启新版）不受影响。

---

# Completed

2026-11-11 22:25
