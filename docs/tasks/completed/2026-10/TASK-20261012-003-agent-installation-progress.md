# TASK-20261012-003: 客户端下载与安装进度

> Retroactive record：实现完成后按用户要求补录；此前未按规范事先创建记录。

## Status

Completed

## Created / Completed

2026-10-12（补录日期）

## Request / Goal

“这个下载安装中有进度提示吗？没有的话可以做吗？”在已有客户端安装流程中增加真实阶段、百分比、下载量和已用时间。

## Analysis / Root Cause

安装原来只有忙碌按钮。WinGet 在普通管道中可能不输出终端进度；不能用计时器伪造下载百分比。

## Implementation Plan

追溯实际步骤：结构化进度模型 → Windows ConPTY 输出读取 → 有界进度解析 → 后台任务快照 → 前端进度与轮询 → 验证。

## Implementation Result

- 展示准备、下载、安装和核对阶段；只有安装器报告数值时才展示阶段百分比与下载字节数。
- 未报告数值时显示动态进度及时间，下载 100% 不表示安装完成。
- Windows 后台 ConPTY 读取终端进度，解析 ANSI / OSC 指令和有界文本帧；前端只获得阶段、百分比和字节数。
- 只有尚未启动进程时才允许终端创建失败降级，避免重复安装；退出应用可中断任务并在下次启动重新检测。
- UI 支持 ARIA 进度状态和减少动态效果偏好；仅安装忙碌时轮询。

## Files Changed

- `internal/infrastructure/agentinstall/progress.go`、`terminal_windows.go`、平台降级实现及测试
- `internal/application/agentinstall/progress.go`、`job.go`
- `frontend/src/features/gateway/agent-installation-progress.tsx` 及测试
- 安装卡片、任务快照、中英文词条及生成绑定

## Verification

- 有界解析、非法数值、阶段状态、取消和原生终端输出测试通过。
- 原生终端测试使用合成进度和可取消的测试命令，没有执行真实安装。
- 前端进度测试通过；完整 `task verify` 通过，后续最近一次完整验证为 64 个前端测试文件、340 项测试及全部 Go 检查、构建。

## Documentation

已更新 `docs/product/LOCAL_AGENTS.md`；本次补录更新 CURRENT。

## Risks / Deviations

安装器不提供数值时无法显示准确百分比，仅显示阶段和时间。缺少事前 Task Record 属于流程遗漏，本记录不将其伪装成事前记录。

## Git

Not committed by Agent. 相关模块包含此前尚未提交的实现，未将其与工作区其他修改混合提交。

## Result

下载安装流程已有真实进度反馈及终态核对。
