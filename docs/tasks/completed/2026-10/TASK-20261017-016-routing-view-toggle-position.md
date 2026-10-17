# TASK-20261017-016: 默认映射视图切换移到右上角

## Status

Completed

## Created

2026-10-17（Asia/Singapore）

## Request

用户要求将默认映射的“按模型／按服务商”切换按钮移到右上角模型数量下方。

## Goal

标题和说明在左，模型数量与视图切换在右；窄容器自然换行，保留既有视图切换与搜索行为。

## Analysis

只调整默认映射卡片的头部结构，将现有切换组件移入模型数量下方；不修改共用切换组件或其他页面。头部使用可换行 Flex，说明可缩放，右侧控件保持靠右。

## Implementation Plan

1. 调整头部左右布局并移动既有切换组件。
2. 执行现有分组交互测试、格式／类型／静态检查，原生核对宽窄窗口、深浅主题与放大字号；按仓库规范运行质量门。
3. 更新 CURRENT 与实际验证结果，审查差异后提交并归档。

## Implementation Result

默认映射头部改为可换行的左右两组：左侧标题／说明，右侧模型数量／既有视图切换。右组靠右对齐，在窄容器或放大字号时自然换行；移除原内容区顶部的切换组件占位，搜索框随之上移。只修改一个组件，无新依赖、翻译或后端变更。

## Verification

- 原生项目开发窗口已通过 HMR 显示数量下方的切换按钮。
- UIAutomation 几何断言：2560／960 深色、960 浅色及 140% 字号共 4 个场景，切换组位于数量下方且右边缘对齐，不越过内容区，也不与说明／搜索框重叠。字号与主题已恢复。
- 截图、几何记录、质量门日志位于忽略目录 `tmp/routing-view-toggle-20261017/`。
- `VITEST_MAX_WORKERS=4 task --force verify BIN_DIR=tmp/routing-view-toggle-20261017/verify-bin`：PASS，退出码 0。格式、Go vet／staticcheck、ESLint、类型检查、全部 Go 测试、91 文件／514 项前端测试、生产前端与 Windows 构建全部通过，包含既有键盘切换和视图／搜索保留回归。
- `git diff --check` 通过，生成绑定与依赖无变化。沿用既有主 JS 包超过 500kB 的构建提示。

## Git

实现提交：`ab0dadf2c648f7adff4caebf635dbefcb5577796`（`fix(routing): 将视图切换移到模型数量下方`）。

分支 `codex/river-builtin-provider`，未推送。任务归档及 CURRENT 由后续文档提交保存；五份既有无关未跟踪记录保留原样。

## Result

按钮已位于右上角模型数量下方并右对齐，小窗口及字号放大时可自然换行，开发窗口已更新；没有新增功能风险或改变路由行为。

## Completed

2026-10-17（Asia/Singapore）
