# TASK-20261014-006: 优化一键配置弹窗滚动条

> 原分支任务编号为 TASK-20261014-002；合入 main 时该编号已用于 Gemini 规划，故统一改为 TASK-20261014-006，原实现和验收记录保持不变。合并过程见 TASK-20261014-007。

## Status

Completed

## Created

2026-10-14 10:52（Asia/Singapore）

## Request

用户反馈「这个页面的右侧的滚动条好丑啊，你看看能不能优化一下呢」，附 Claude Code Desktop 一键配置弹窗截图。

## Goal

弹窗正文与模型目录统一采用项目现有细圆角、悬停渐显的滚动条，去掉系统箭头和粗灰滑块，保持滚动和配置操作可用。

## Scope

仅修改一键配置弹窗和模型目录预览的滚动容器，不改变配置业务或共享滚动组件行为，不新增依赖。

## Analysis

`AgentConfigDialog` 正文使用 `overflow-y-auto`，`AgentCatalogPreview` 模型列表使用 `overflow-auto`，均显示 Windows 原生滚动条。已有 `ScrollRegion` 提供 12px 固定占位轨道、6px 圆角滑块、180ms 悬停淡入淡出、主题语义色、滚轮／键盘／点击轨道／拖拽及减少动态效果支持。复用它比新增局部 CSS 或滚动逻辑更一致且维护成本更低。

## Implementation Plan

1. 正文和模型列表接入 `ScrollRegion`，保留正文间距、192px 列表上限及固定标题／按钮区。
2. 运行现有滚动组件、Agent 配置交互测试和 `task verify`，核对原生窗口深浅主题及滚动行为。
3. 更新相关文档，审查本任务差异，提交并归档记录。

## Implementation Result

- 弹窗正文和模型目录接入现有 `ScrollRegion`，去掉 Windows 系统箭头与粗灰滑块，复用 12px 固定轨道、6px 圆角滑块、180ms 悬停淡入淡出及深浅主题语义色。
- 正文继续保持 20px 段落间距，模型目录保留 192px 高度上限及列表语义；标题和底部按钮固定可见。滚轮、键盘、点击轨道、滑块拖动及减少动态效果沿用已有组件。
- 两个滚动区域带有现有翻译生成的可访问名称。没有修改共享滚动组件或配置业务，没有新增依赖。

主要修改文件：`agent-config-dialog.tsx`、`agent-catalog-preview.tsx`。

## Verification

- PASS：现有滚动组件、滚动恢复、配置弹窗及 Agent 接入四份测试文件共 19 项，覆盖独立滚动、隐藏轨道点击／拖动、配置预览、应用和重新读取行为。
- PASS：`VITEST_MAX_WORKERS=4 task verify BIN_DIR=E:/go/new/wails3-v/OmniProxy/tmp/agent-config-scrollbars/quality-bin` 最终退出码 0。格式检查、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、69 个前端测试文件／379 项测试及 Windows 生产构建通过。
- PASS：使用隔离测试数据和独立端口运行本项目 `task dev`，日志确认连接 `http://localhost:9257`；按绝对路径定位 `bin/omniproxy.exe`，原生验收 PID 38548，未操作其他同名应用。
- PASS：深浅主题中正文和模型目录默认隐藏，悬停显示细圆角滑块，移开隐藏；两处滑块拖动均达到 100% 滚动位置。正文视口宽 738px、高 564px，模型目录宽 688px、高 191px；显隐／滚动前后宽度及位置稳定。
- PASS：原生可访问性滚动操作另行验证两处区域独立滚动，均到达 100%，另一处保持顶部；截图人工核对原段落间距、固定标题／按钮、系统箭头移除和主题显示正常。
- PASS：`git diff --check`。仅使用无凭据测试账号和测试模型，未应用真实客户端配置。

截图、原生验收结果、脚本和最终质量门日志保存在忽略目录 `tmp/agent-config-scrollbars/`。既有工具探测、jsdom `scrollTo` 和打包体积提示未影响质量门。

## Documentation

已更新 `docs/product/LOCAL_AGENTS.md` 的弹窗滚动说明和 `docs/plans/CURRENT.md`。没有产品范围、接口或架构变更。

## Deviations

原生检查发现初次正文容器未保留段落间距，已改用普通 `space-y-5` 内层容器并重新运行相关测试和完整质量门。原生指针验收临时置顶隔离窗口，避免其他同名窗口遮挡，并等待滚动复位后再测下一条滑块；验收后取消置顶并恢复主题、鼠标位置。

## Git

`b078db01f69d01f49376364fbad3a42fc60bbc71 fix(gateway): 优化一键配置弹窗和模型目录滚动条`

仅提交本任务代码和文档。保留既有未跟踪任务记录 `TASK-20261013-017-atria-builtin-provider.md`，不纳入本任务提交。

## Result

一键配置弹窗及模型目录统一使用细圆角悬停滚动条，默认隐藏、固定占位，完整质量门和深浅主题原生滚动／拖动验收通过。

## Remaining Risks

无新增业务风险。减少动态效果支持沿用共享组件 CSS，未单独切换系统偏好验证。

## Completed

2026-10-14（Asia/Singapore）
