# TASK-20261013-004: 接入钴蓝汇聚应用图标

## Status

In Progress

## Created

2026-10-13（Asia/Singapore）

## Request

用户选定 C「钴蓝汇聚」后要求开始接入桌面、任务栏和托盘图标。

## Goal

使 Windows 应用资源、运行时托盘和应用内品牌入口统一使用选定图标，并验证实际原生显示。

## Background

设计及选择记录：TASK-20261013-003。

## Scope

- 替换构建源图与 Windows 多尺寸 ICO。
- 在 Wails 桌面适配层接入托盘图标。
- 同步前端标题栏、侧栏和浏览器图标。
- 检查格式与资源，运行 `task verify`，核对原生窗口与系统图标。

# Analysis

现有桌面资源仍为 Wails 默认图标，托盘未显式指定图像，前端 favicon 与侧栏品牌占位未统一。

已核对本机 Wails v3.0.0-beta.25：窗口读取可执行文件中的图标资源 ID 3，托盘 `SetIcon` 接受 PNG。采用现有 C 资产和标准库嵌入，不新增依赖；构建命令显式保留全部 10 个 ICO 尺寸，避免重新构建退回默认 6 个尺寸。

## Risks

- Windows 图标缓存及当前运行进程可能保留旧图标，需要重新构建与启动验证。
- 16／20 像素托盘尺寸需要保持透明外底与主体辨识度。

# Implementation Plan

1. 核对当前 Wails 版本的应用图标与托盘接口，以及现有构建链。
2. 沿用已选定 C 资产，统一构建、运行时和前端入口。
3. 进行最小资源验证、完整质量门及原生显示检查。
4. 更新构建说明、设计状态、CURRENT 与实际任务结果，检查并提交差异。

# Implementation Result

- 使用选定 C 原图替换 `build/appicon.png`，通过现有 Wails 构建任务生成 Windows ICO 和共享任务自带的 macOS ICNS。
- 构建命令显式保留 10 个 ICO 尺寸，重建后仍覆盖 16–256 像素资源。
- 在桌面适配层内嵌 256 像素透明 PNG，传入应用图标与托盘 `SetIcon`，不依赖运行目录中的外部文件。
- 标题栏、侧栏和 favicon 统一使用同一透明 PNG，删除旧占位 favicon。
- 更新构建说明和设计接入状态；未新增依赖、公共 API 或业务逻辑。

# Verification

- PASS：源图与选定 C 完全一致，前端和运行时 PNG 内容一致；重新生成的 ICO 含 10 个预期尺寸，帧数据边界有效。
- PASS：`task verify` 退出码 0；格式、Go vet／staticcheck／ESLint、TypeScript、全部 Go 测试、64 个前端测试文件／340 项测试及生产构建通过。
- PASS：生产 EXE 实际提取的大／小图标为 C；新开发版 EXE 与窗口 `WM_GETICON` 均为钴蓝图标，Windows Shell 从测试快捷方式提取的图标同样为 C。
- PASS：实际原生窗口标题栏和侧栏视觉核对、隐藏通知区展开后的 C 托盘图标视觉核对；托盘菜单可正常打开。
- PASS：`http://127.0.0.1:9245/app-icon.png` 返回 HTTP 200，内容哈希与前端源图完全一致；3891 `/healthz` 为 development／ok。
- PASS：普通 `task dev` 重新启动，任务 PID 2444，当前仓库应用 PID 4220，窗口可见并正常响应，日志包含连接前端开发服务成功信息。
- PASS：`git diff --check`；未出现无关源码、依赖或生成绑定差异。
- 原生资源提取、图标截图、构建和开发日志存放于 `tmp/app-icon-integration/`，均为本地验证产物，不纳入 Git。

## Remaining Risks

既有已安装版本和缓存中的旧快捷方式图标需要随新版安装包更新；本任务已验证新构建及其快捷方式资源。Wails 工具链 uname／tail、jsdom scrollTo 和既有 Vite 大包提示未影响质量门，本次没有扩大修改范围。

# Documentation

已更新 BUILD 的图标来源、尺寸、生成命令和运行时同步方式；已同步设计说明、预览接入状态与 CURRENT。

# Git

实现提交后补充提交哈希。

# Result

C「钴蓝汇聚」已接入桌面／任务栏、托盘、标题栏／侧栏和 favicon，完整质量门与原生验收通过，更新后的开发版保持运行。
