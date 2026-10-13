# TASK-20261013-012: 统一局部面板悬停滚动条

## Status

Completed

## Created

2026-10-13（Asia/Singapore）

## Request

优化重置卡右侧滚动条，并同样调整 OpenAI 账号设置：保留占位但默认不显示，仅鼠标移入后动画显示，移开后动画消失。

## Goal

两个局部面板使用细圆角、主题一致、布局稳定的悬停滚动条。

## Analysis

两处面板目前直接使用原生 overflow-y-auto，显示系统轨道及箭头。项目已有 `ScrollRegion`：12px 固定轨道、6px 圆角滑块、180ms 透明度过渡，仅悬停轨道／拖动时显示，支持滚轮、键盘、轨道点击、拖拽和减少动态效果。复用该组件，避免新增滚动实现或改变应用其他区域。

## Implementation Plan

1. 将重置卡明细及 OpenAI 设置内容接入 ScrollRegion，保持原有内边距和内容间距。
2. 运行已有滚动／两个面板交互测试及 `task verify`。
3. 原生验证默认隐藏、悬停显隐、占位稳定及实际滚动，更新文档并提交。

## Implementation Result

- 重置卡明细和 OpenAI 设置内容改用已有 `ScrollRegion`，移除面板系统滚动条的轨道底色与箭头。
- 复用 12px 固定轨道和 6px 细圆角滑块，默认及滚动时透明，悬停轨道／拖动时以 180ms 淡入，移开后淡出，深浅主题沿用语义色。
- 保留内容间距、滚轮／键盘／轨道点击／滑块拖拽及减少动态效果支持；仅调整两处组件接入，没有改写共享滚动逻辑或其他页面，没有新增依赖。

## Verification

- PASS：已有滚动组件、重置卡及设置面板四份测试文件共 17 项，覆盖独立滚动、隐藏轨道点击／拖拽、面板关闭和业务操作回归。
- PASS：`VITEST_MAX_WORKERS=4 task verify BIN_DIR=E:/go/new/wails3-v/OmniProxy/tmp/popover-scrollbars/quality-bin` 退出码 0；格式、Go vet／staticcheck、ESLint、TypeScript、全部 Go 测试、69 个前端文件／366 项测试及 Windows 生产构建通过。
- PASS：按路径识别开发窗口 PID 42068，深浅主题下两个面板默认隐藏、悬停显现、移开隐藏，拖动均达到 100% 滚动位置。重置卡视口宽 314px、设置视口宽 374px，显隐前后宽度和位置一致，12px 轨道固定占位。
- PASS：原生动画中间帧与端点截图比较，暗色滑块采样亮度从背景 23，经淡入中间帧 86／92 到 119；移开后从 119 经 45 回到 23，证实淡入／淡出过渡。深浅主题截图人工核对正常，验收后恢复原主题及鼠标位置。
- PASS：`git diff --check`、本次文件 UTF-8 无 BOM 核对。未操作设置开关或兑换真实重置卡。

截图、脚本、动画与布局结果和质量日志位于忽略目录 `tmp/popover-scrollbars/`。既有工具探测、jsdom scrollTo 和打包体积提示不影响质量门通过。

## Documentation

已同步 SUBSCRIPTION 的统一滚动交互说明和 CURRENT。没有产品范围、接口或架构变更。

## Deviations

None.

## Git

实现提交待记录。

## Result

两个局部面板统一使用细圆角、固定占位、悬停淡入淡出的滚动条，拖拽可用，完整质量门和深浅主题原生验收通过。

## Remaining Risks

没有新增业务风险；减少动态效果行为沿用已有共享组件 CSS，未更改。

## Completed

2026-10-13（Asia/Singapore）
