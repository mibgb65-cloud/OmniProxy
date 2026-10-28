# TASK-20261028-010: 修复账号页提供商切换区越界与遮挡

## Status

Completed

## Created

2026-10-28 22:47

## Request

用户反馈两个账号页布局 Bug：底部提供商图标不应出现在左侧操作栏区域；提供商切换区不应与账号卡片重叠、遮挡内容。

## Goal

将提供商切换区及过渡动画约束在主内容区，保证账号卡片可完整查看和操作，保留现有切换与键盘交互。

## Scope

- 检查账号页布局、提供商 Dock 和 FLIP 过渡层。
- 针对根因修复并增加回归验证，不改变账号业务逻辑。

## Out of Scope

调整产品需求、重做页面设计、修改后端或升级依赖。

## Analysis

- 已用含 223px 侧栏、44px 标题栏及真实 ScrollRegion 的隔离浏览器复现两张截图。
- Dock 横向滚动后，视口外子项仍有矩形；FLIP 副本直接挂在 body，固定定位且 z-index 为 40，绕过主内容区裁剪。
- 同一层级还高于 Dock，长账号面板的副本压住停靠区，产生文字、图标与卡片叠加。
- 保留现有 sticky Dock，避免改成普通文档流而失去长列表中的快捷切换；把动画层放回账号网格，受主视口裁剪，详情副本低于 Dock、飞行中的迷你卡片高于 Dock，同时使用不透明底色。

## Implementation Plan

1. 确认主内容区、Dock 和动画层的定位、裁剪及滚动关系。
2. 实施最小修复，保留既有交互。
3. 增加回归测试并进行真实浏览器布局验证。
4. 运行 `task verify`，记录结果并复查差异。

## Progress

- 已定位并复现两个问题；使用假账号数据，不访问或复制真实凭证。
- 完成相对坐标、主视口裁剪、三层绘制次序与共享过渡容器清理。
- 隔离 Chromium 九组检查通过；完整质量门通过（139 文件、851 项前端测试）。
- 已尝试隔离原生测试实例，但没有可操作的原生窗口或 CDP 端点；仅停止本任务创建的进程，不干预用户运行中的应用。

## Implementation Result

### Changes

- FLIP 从 body 上的固定层改为账号网格内的共享绝对定位容器，坐标改为相对账号网格，保留侧栏/主视口裁剪与容器响应式上下文。
- 详情副本、Dock、迷你卡片副本分别采用 1／2／3 层级；Dock 改用不透明主题底色，不再透出账号文字。
- 保留 600ms 过渡、横向滚动、键盘导航、快速切换和减少动画行为。
- 完成、打断与卸载时清理共享容器，增加相对坐标、层级标记、多动画共享容器生命周期回归测试。

### Files Changed

- `frontend/src/features/accounts/bento-accounts.css`
- `frontend/src/features/accounts/flip-snapshot.ts`
- `frontend/src/features/accounts/use-flip-reflow.ts`
- `frontend/src/features/accounts/use-flip-reflow.test.tsx`
- `docs/product/APPEARANCE.md`、`docs/plans/CURRENT.md` 及本记录

### Technical Notes

- 过渡容器刻意不创建独立层叠上下文，否则迷你卡片会被 Dock 不透明底色遮住，无法连续飞入。
- 测试文件超过 200 行，保留同一 WAAPI 模拟夹具，集中验证坐标、连续打断及共享层清理，不为行数做无关拆分。
- 无新增依赖、后端、凭证、接口、版本号或安装器变更。

## Root Cause

动画绕过页面裁剪和局部层叠关系，不是提供商图标资源或侧栏导航本身有问题。

## Regression Prevention

单测限制副本挂载位置与相对坐标，验证详情/迷你卡片标记和共享容器清理；真实浏览器检查多主题、尺寸、侧栏状态及关键动画帧。

## Verification

```text
task --force verify
PASS（exit 0）
Format / Go vet / staticcheck / ESLint / TypeScript / Go tests：PASS
Frontend：139 files / 851 tests PASS
Vite production / Windows amd64 production build：PASS

git diff --check
PASS
```

隔离 Chromium 九组验证：

1. 1200×800 深色主题。
2. 1200×800 浅色主题。
3. 900×680 窄窗，账号自动降为单栏。
4. 侧栏收起至 64px，账号恢复双栏。
5. River → Oxion AI 反向切换，120／300／590ms 侧栏像素保持不变。
6. Dock Home／End 键盘导航。
7. 快速连续切换后没有残留副本或隐藏标记。
8. 减少动画模式不创建过渡层。
9. 窗口尺寸变化立即取消过期动画。

前四组逐帧检查 0／120／300／590ms，侧栏和标题栏像素完全不变；固定迷你卡片后，
详情飞行不污染 Dock（窄窗仅有最大 2／255 的文字抗锯齿差异，其余为 0）。
长列表滚到底时最后一张卡片完整位于 Dock 上方，主视口无横向溢出。

验证材料在忽略目录 `tmp/provider-dock-layout/`。保留既有 jsdom `window.scrollTo`
未实现及 Vite 大 chunk 警告，均不影响门禁通过。

原生限制：使用本次新构建、独立 test 数据/配置/浏览器目录及临时端口启动原生实例，
没有生成可操作窗口，CDP 端口持续拒绝连接。因此未宣称原生视觉验收通过。
已核对进程路径并停止隔离实例，未操作用户运行中的安装版；未自动安装、打包或发布。

## Documentation

已同步 `APPEARANCE.md` 的裁剪/分层规则及 CURRENT；无产品范围、架构、技术栈或 ADR 变更。

## Deviations

保留停靠区和既有页面滚动，不额外引入账号区嵌套滚动或普通文档流 Dock。
原生视觉验收受上述环境限制，需在实际运行新版应用后确认。

## Git

```text
970990e0ea460cbfd0fb68e2f6f987ed9e9dce91
fix(accounts): 约束提供商切换动画并修复 Dock 遮挡
```

已本地提交实现、测试及验证记录；随后归档本记录并同步 CURRENT，不推送远端。

## Result

两处截图问题已在源码修复，并通过完整质量门及隔离浏览器验证。
新 Windows 可执行文件已生成；未替换用户正在运行的安装版。

后续本地交付：[TASK-20261028-011](TASK-20261028-011-build-local-v2.0.1-beta.7-installer.md)
已将本修复纳入 `v2.0.1-beta.7` Windows production 安装包并完成版本资源与 SHA256 校验；
安装及原生视觉确认仍由用户执行。

## Completed

2026-10-28 23:10（Asia/Singapore）
