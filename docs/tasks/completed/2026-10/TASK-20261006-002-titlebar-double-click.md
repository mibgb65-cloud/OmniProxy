# TASK-20261006-002: 标题栏双击最大化与还原

## Status

Completed

## Created

2026-10-06 20:43

## Request

操作栏的中间部分，即亮暗色模式的左侧部分，需要能够实现鼠标双击最大化和窗口化的功能。

## Goal

在主题切换按钮左侧的标题栏区域双击，切换窗口最大化与还原；保留窗口拖动和现有按钮操作。

## Analysis

- `AppTitlebar` 已有最大化按钮、窗口状态同步与错误提示，但左侧弹性区域没有双击处理。
- 已安装的 Wails Runtime 仅为 macOS 显式处理拖动区域双击；Windows 的 CSS 拖动标记不能满足本次需求。
- 左侧区域当前未撑满标题栏高度，需让双击命中区域覆盖完整高度。
- 采用前端事件复用已有窗口切换 API，影响限于标题栏；无需调整 Go 窗口配置或增加依赖。

## Implementation Plan

1. 将左侧标题栏区域撑满高度，绑定双击事件，复用最大化按钮的动作及状态／错误处理。
2. 验证双击最大化与还原、右侧按钮不误触发、失败提示，以及既有窗口控制。
3. 运行 `task verify`，将构建产物输出到独立目录；在实际开发窗口验证交互。
4. 更新任务记录与状态看板，审查差异，提交本任务涉及的文件。

## Risks

双击与原生拖动处理可能相互影响，需实际窗口验证。`task verify` 的生产构建默认使用同一二进制路径，使用 `BIN_DIR=bin/verify-titlebar-double-click` 避免覆盖正在运行的开发程序。

## Implementation Result

- 主题切换按钮左侧的标题栏弹性区域增加 `h-full`，双击命中区域覆盖完整高度。
- 增加双击事件，提取并复用现有 `toggleMaximise` 动作，窗口最大化图标与错误提示沿用现有处理。
- 双击区域与右侧主题／窗口控制按钮分属不同容器，避免按钮双击触发额外的窗口切换。
- 修改文件：`frontend/src/components/common/app-titlebar.tsx`、同目录测试、状态看板及本任务记录。
- 无新增依赖，无 Go 代码或窗口配置变更。

## Verification

- 标题栏针对性测试：6 项通过，新增最大化／还原、右侧按钮隔离、失败提示与重试回归测试。
- 真实 Windows 开发窗口实测通过：鼠标双击标题栏中间区域最大化、再次双击还原、原有拖动仍可移动窗口。验证后已恢复原窗口位置、尺寸、状态、前台焦点和鼠标位置。
- 全量 Go 测试通过；前端 17 个测试文件、81 项测试通过。
- `task verify BIN_DIR=bin/verify-titlebar-double-click` 完整通过：格式、lint、类型检查、Go／前端测试、Windows 生产构建均成功，命令退出码为 0。
- 生产程序输出到 `bin/verify-titlebar-double-click/omniproxy.exe`，正在运行的开发二进制未被覆盖。
- `git diff --check` 通过。

## Documentation

更新 `docs/plans/CURRENT.md`。本次沿用现有窗口控制和前端技术栈，PRD、架构、技术栈、构建说明和 ADR 无需修改。

## Deviations

使用 `BIN_DIR` 覆盖参数为质量门禁指定独立构建产物目录，开发环境保持运行；实际验证流程和检查内容未改变。

## Remaining Risks

已在当前 Windows / WebView2 开发环境实测；其他平台不属于当前项目支持范围。

## Git

```text
a61dc52ee28d6ea7fad9223bb74c8a01f01fd214 fix(frontend): 支持标题栏双击最大化与还原
```

本记录由后续文档提交回填功能提交哈希并归档。上一任务的未提交启动记录未包含在本任务提交中。

## Result

标题栏主题按钮左侧支持双击最大化／还原，既有拖动与窗口按钮保持正常；开发窗口已热更新生效。

## Completed

2026-10-06 20:50
