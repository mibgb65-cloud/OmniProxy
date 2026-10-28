# TASK-20261028-007: 修复账号 Bento 展开闪动并优化底部 Dock

## Status

Completed

## Created

2026-10-28（Asia/Singapore）

## Request

在已实现的账号概览 Bento 原位展开基础上，修复展开、切换、收起时的闪动；
保持 600ms FLIP / Layout 过渡、顶部全宽双栏详情和底部迷你提供商切换。
底部 Dock 不应拥挤截断或换行成卡片墙，避免「方框套方框」的视觉层级。
参考用户截图，但不复制其中的真实账号数据。

## Goal

保留既有账号管理、额度、独立详情路由和可访问性行为，提供连续、可打断的布局动画，
去除多余外框，使多提供商 Dock 在窄窗下仍可清楚辨认和切换。

## Background

延续 TASK-20261028-005，当前工作区开始时无未提交变更。

## Scope

- 检查并修复账号 Bento 的节点生命周期、FLIP 动画及焦点/滚动处理。
- 优化概览、内嵌详情及底部 Dock 的层级与响应式布局。
- 增加动画与交互回归测试，运行质量门禁并记录实际结果。

## Out of Scope

- 不改后端、凭证、账号业务规则及核心技术栈。
- 不增加动画框架或产品需求，不变更独立详情页公共路由。

# Analysis

- 旧实现把整个面板在飞行期间淡到 0.12，600ms 后才重新显现，因此出现明显「消失再出现」。
- 同一面板 DOM 被不同提供商复用，未清理的 WAAPI 动画会污染下一次终点测量，
  连续切换时残留变换和完成回调互相覆盖。
- 对整个详情做 scale 会压扁文字；读写交错也增加重复布局计算。
- `focus()` 默认滚动，以及长卡片墙变为短内容时的滚动锚定，会放大跳动感。
- Dock 的 auto-fit 网格导致多提供商换行且名称过度截断；详情外框、账号框、额度框叠加。
- 采用既有 WAAPI FLIP + 内容反向缩放，不引入动画库或改变技术栈；
  分批测量、按当前屏幕几何打断动画，保留 600ms 与减少动画偏好。

# Implementation Plan

1. 确认闪动根因，比较保留现有零依赖 FLIP 与引入动画库的取舍。
2. 在现有 React / CSS / Web Animations API 上做最小闭环修复。
3. 验证连续切换、反向收起、键盘、减少动画和多提供商布局。
4. 执行 `task verify`，更新任务记录与 CURRENT，并检查 Git diff。

# Progress

- 已阅读项目规则及上一版任务记录，开始检查现有动画与页面结构。
- 已确认全内容淡化、残留动画与默认焦点滚动问题；选择保留零依赖 FLIP，
  并将底部切换改为可横向滚动、可键盘操作的单行 Dock。
- 浏览器中间帧发现 Dock 横向容器会裁掉飞行中的卡片，因此补充只读过渡层，
  不通过临时放开 overflow 造成页面横向溢出。
- 完成快照 CSS 动画冻结，避免额度数字与进度条在副本插入时重新闪入。
- 最终完整质量门通过；隔离浏览器九项布局/动画检查通过，原生检查存在环境限制（见下）。

# Implementation Result

## Changes

- 展开、切换和收起使用同一段 600ms FLIP；先测量当前呈现位置，再取消旧动画，
  防止连续点击时从旧终点重新开始，旧完成回调也不能清掉新动画。
- 通过 `aria-hidden`、`inert`、无指针事件的只读副本承载过渡，脱离 Dock 裁剪，
  新旧内容交叉呈现且反向抵消缩放，不再整块淡到 0.12。
- 副本移除重复 DOM ID、自动聚焦标记及动画标记，并冻结原有 CSS 动画；
  动画结束、卸载、窗口变化及用户滚动时清理，不修改业务状态、路由或真实凭证。
- Dock 固定为单行横向滚动，提供完整品牌名称、账号数、Tooltip、左右滚动按钮、
  方向键及 Home/End；根据子项尺寸更新溢出状态，不再多行挤压。
- 去掉详情总外框、概览内层边框、OpenAI 额度内层框和 Token 数据小方框，
  只保留账号自身的卡片层级。内容区小于 44rem 时自动单栏。
- 焦点转移使用 `preventScroll`，展开面板按提供商隔离节点；独立详情路由兼容性保留。

## Files Changed

- `frontend/src/features/accounts/{use-flip-reflow,flip-snapshot}.ts` 与对应测试。
- `frontend/src/features/accounts/{accounts-view,provider-switcher-dock,provider-dock-card}.tsx` 与对应测试。
- `frontend/src/features/accounts/bento-accounts.css`、账号卡片及额度/Token 样式。
- `frontend/src/i18n/locales/{zh,en}.ts`。
- `docs/product/APPEARANCE.md`、`docs/plans/CURRENT.md` 与本记录。

## Technical Notes

没有新增依赖、后端接口、迁移或凭证格式变更。保留既有缓动
`cubic-bezier(0.22, 1, 0.36, 1)` 与减少动画回退。过渡副本不挂载 React，
不会额外触发查询、订阅或账号动作。

# Verification

## Tests

- [x] 四个变更相关测试文件：24 项通过，覆盖动画打断/清理、快照安全、
  减少动画、零尺寸/无 WAAPI 回退、Dock 键盘/滚动、焦点与路由。
- [x] 既有独立详情、账号管理、Token 与额度测试包含在全量测试中。
- [x] 隔离 Chromium 使用真实组件和合成账号数据，完成九项检查：
  11 个提供商的动画起点、600ms 时长、120ms 中间帧、文字比例、
  快速切换续接、反向收起与焦点恢复、单行品牌全名、
  窄窗无横向溢出、减少动画及 Home/End 滚动。无页面异常。
- [ ] 原生 WebView2 自动视觉验收未完成：隔离 production/test 实例仅创建了环境，
  未加载出可操作的账号导航，CDP 端口也不可用；不以浏览器通过替代原生验收。
  已停止该隔离实例及子进程，未重启或改动用户安装版进程及真实数据目录。

## Quality Checks

```text
task verify
  format / lint / typecheck / Go tests / 137 文件 839 项前端测试通过；
  增量构建依赖阶段长时间无后续输出，主动中断。

task --force verify
  PASS，exit 0
  Prettier / go vet / staticcheck / ESLint / TypeScript：PASS
  Go tests：PASS
  Frontend：138 files / 841 tests PASS
  Vite production + Windows amd64 production build：PASS

git diff --check
  PASS
```

完整门禁最终重跑覆盖新增快照测试及最终代码，不依赖中途测试结果。
构建保留既有大 chunk 提示，jsdom 保留既有 `window.scrollTo` 未实现警告，
均未导致门禁失败，不在本任务中做无关修正。
本地检查材料保存在忽略目录 `tmp/bento-motion-polish/`，未提交演示账号或运行数据。
未重新制作安装包、未发布版本，原生实机最终观感仍需用户确认。

# Documentation

- 已更新 `APPEARANCE.md`，说明单行 Dock、容器响应式与新版可打断动画。
- 已同步 CURRENT；无产品范围、架构、技术栈或 ADR 变更。

# Deviations

- 初始计划仅修正现有 FLIP；逐帧检查发现横向滚动容器裁剪问题，
  因此增加轻量只读过渡层，没有引入动画框架。
- 原生验收已尝试但受上述环境限制，保留明确未验收项。

# Git

```text
8e3c36527652bc4de9956f289c80d031a396b0ce
fix(accounts): 修复 Bento 展开闪动并优化单行切换 Dock
```

已本地提交实现与测试，不推送远端；随后归档本记录并同步 CURRENT。

# Result

已完成账号 Bento 动画与视觉层级修复，完整质量门及隔离浏览器验证通过。
Windows production 二进制已构建，安装包未重打；原生实机观感仍待确认。

# Completed

2026-10-28 20:31（Asia/Singapore）
