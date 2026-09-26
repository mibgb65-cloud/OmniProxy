# TASK-20260926-008: 开机启动与关闭按钮行为

## Status

Completed

## Created

2026-09-26 11:09

## Request

用户希望设置页面增加是否开机自启动，以及点击关闭按钮时选择保留在系统托盘还是直接退出。

## Goal

设置页能读取并更改 Windows 登录自启动状态与关闭按钮行为；关闭到托盘后可以从托盘恢复窗口或真正退出应用。

## Existing Behavior

自绘标题栏的关闭按钮直接调用 `Window.Close()`，应用没有托盘图标；设置页没有桌面行为选项。

## Expected Behavior

- 默认保持直接退出。选择“关闭到托盘”后，标题栏关闭、系统窗口关闭均隐藏窗口并保持后台运行。
- 托盘图标可重新打开窗口，托盘菜单提供真正退出入口，退出时仍执行正常关机与数据迁移。
- 正式版的开机启动开关读取系统注册状态，启用/停用通过 Wails 自带 Autostart API 完成；开发版说明原因并禁用开关。
- 关闭行为持久化到本地配置，修改后立即生效；失败时界面显示错误，原状态保持可见。

# Analysis

## Related Modules

- `internal/platform/profile/config.go`
- `internal/transport/wails/app.go` 与新增桌面设置绑定
- `frontend/src/features/settings`、`frontend/src/services/omniproxy.ts`、`frontend/src/i18n/locales`
- `frontend/bindings` 生成代码

## Dependencies

使用已锁定的 Wails 3 Autostart、SystemTray 与 WindowClosing Hook，无新第三方依赖。

## Risks

关闭到托盘必须保留明确的退出入口，否则后台进程和退出时迁移都无法由用户触发。开机启动注册是系统状态，测试不能改写用户真实启动项。开发与正式构建可能覆盖同一个输出路径，不能让开发版注册该路径。配置文件的多个写入操作需避免相互覆盖。

# Implementation Plan

1. 在 profile 配置中增加可校验的关闭行为，并将 Wails 设置写入串行化；仅正式配置允许注册开机启动。
2. 接入 Wails Autostart、托盘菜单、关闭事件钩子及设置绑定。
3. 设置页增加开机启动开关和关闭行为选项，生成绑定并补充中英文案。
4. 通过隔离配置与伪造 Autostart 控制器测试关键逻辑，运行 `task verify`，检查差异并提交。

# Progress

已确认 Wails 3 beta.25 提供 Autostart、SystemTray 和可取消的 WindowClosing Hook；系统托盘菜单必须提供独立的退出入口，关闭行为默认维持现状。

确认开发与正式构建共用 `bin/omniproxy.exe` 输出路径。Wails 在 Windows 上按可执行文件路径识别启动项，因此开发配置禁用开机启动，避免错误注册或覆盖正式版启动项。

# Implementation Result

## Changes

- 在 `app.json` 增加经校验的关闭行为字段，默认值保持直接退出；设置更改后立即作用于窗口关闭钩子。
- Wails 主窗口加入可取消的关闭事件钩子和系统托盘。托盘左击、菜单“显示”恢复窗口，菜单“退出”执行正常关机与待迁移数据处理。
- 新增桌面设置绑定：正式版读取和控制当前用户的 Windows 登录自启动状态；开发版显示不可用且后端拒绝注册。
- 设置页新增开机启动开关与“关闭到托盘 / 直接退出”选项，错误在控件旁反馈；迁移提示说明托盘模式下需从托盘菜单真正退出。
- `SetDataDir` 与关闭行为配置写入在 Wails 服务中串行化，避免并发覆盖。

## Files Changed

- `internal/platform/profile/config.go`、`desktop.go`、`desktop_test.go`
- `internal/transport/wails/app.go`、`desktop_settings.go`、`desktop_settings_test.go`、`storage.go`
- `frontend/src/features/settings/desktop-behavior.tsx`、`desktop-behavior.test.tsx`、`use-desktop-settings.ts`
- `frontend/src/app/routes/settings.tsx`、`frontend/src/services/omniproxy.ts`、`frontend/src/i18n/locales/{zh,en}.ts`
- `frontend/bindings/**`（Wails 生成）
- `docs/architecture/OVERVIEW.md`、`docs/development/BUILD.md`

## Technical Notes

Wails 自带 Autostart 在 Windows 使用当前用户的登录启动项，启用时传入生产配置的应用标识。关闭行为读自 profile 配置，不依赖前端窗口保持打开。托盘菜单使用中英双语标签，保证两种界面语言都能找到退出入口。

现有单实例保护尚未接入，用户在开机启动后手动再次启动程序仍可能产生第二个进程；已记入 `CURRENT.md` 已知问题。本任务未修改该独立功能。

# Verification

## Tests

- [x] Profile：默认值、非法值、持久化及保留原有配置。
- [x] Wails 设置绑定：OS 状态、生产版启动标识、启用/停用失败、开发版保护及关闭行为立即生效。
- [x] 前端：开关、关闭行为、错误与开发版禁用态；全套 50 个测试通过。

## Quality Checks

- [x] `task verify`：格式、Go vet、staticcheck、ESLint、TypeScript、Go/前端测试和 Windows 生产构建通过。
- [x] 深色、浅色静态预览核对。
- [x] `git diff --check`。

## Verification Result

2026-09-26：`task verify` PASS（50 个前端测试）；本机真实启动项未在测试中写入。Vite 原有 500 KB 包体积提醒仍存在，构建通过。原生托盘操作未在自动化 GUI 环境中端到端演练。

# Documentation

- [x] Architecture：说明托盘、关闭钩子与自启动边界。
- [x] Build：说明正式版自启动限制和托盘退出与迁移。
- [x] CURRENT：完成时更新任务看板及单实例已知限制。
- PRD 与技术栈无需修改，相关目标及 Wails 基线已存在。

# Deviations

初始预期开发/正式配置各自注册启动标识；实际开发版不允许注册。原因是两种构建可能覆盖同一个可执行文件路径，而 Wails 的 Windows 状态/停用操作按该路径定位注册项，仅凭不同标识无法安全隔离。

# Git

## Commit

```text
18c8f34 feat(settings): 增加开机启动与关闭到托盘设置
```

（归档记录本身另行提交，无法在同一提交中记录自身哈希。）

# Result

设置页具备可靠的 Windows 登录启动与关闭行为控制；托盘模式保留恢复和真正退出的完整路径。

# Completed

2026-09-26 15:19
