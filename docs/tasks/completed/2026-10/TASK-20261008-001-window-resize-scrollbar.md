# TASK-20261008-001: 优化滚动条与窗口右侧缩放交互

## Status

Completed

## Created

2026-10-08 09:58（Asia/Singapore）

## Request

窗口右侧的滚动条影响了右侧的侧边的大小拖动，需要进行优化。

## Goal

消除主内容滚动条与无边框窗口右侧缩放区域的交互冲突，保留滚动条点击、拖动与鼠标滚轮操作。

## Analysis

- `ScrollRegion` 的 12px 轨道位于主内容最右侧。其 `pointerdown` 调用 `preventDefault()` 并捕获指针，导致边缘点击的兼容鼠标事件被取消，Wails 无法开始窗口缩放。
- 已通过浏览器复现：窗口右侧距边缘 2px 的命中目标是滚动条轨道，横向拖动未产生 Wails 缩放调用。
- 本地 Wails v3.0.0-beta.25 的 `drag.ts` 依据系统边框宽度识别窗口边缘，角落额外扩展 10px；当前系统边框指标为 4px。
- 比较方案：缩窄滚动条不能消除区域重叠；改变 Wails 配置或添加全局鼠标拦截会扩大影响。采用仅主内容适用的布局内缩，保留共享滚动条的原有交互逻辑与侧栏布局。

## Implementation Plan

1. 主内容滚动区域右侧预留 8px，滚动轨道底部预留 20px，避开窗口右侧及右下角缩放区域。
2. 在多种窗口尺寸、100%–200% 缩放及亮暗主题下验证真实浏览器命中区域、轨道点击、滑块拖动、键盘／滚轮及 Wails 缩放调用。
3. 运行现有滚动组件测试和 `task verify`；质量门禁期间暂停本次启动的开发进程，完成后恢复开发环境并检查健康状态。
4. 核对工作区变更，填写实际结果并同步任务看板。

## Implementation Result

- 根布局仅为主内容启用 `scroll-region--window-edge` 样式：右侧预留 8px，轨道底部预留 20px。
- 原有 12px 轨道、悬停渐显、点击跳转、指针捕获拖动、滚轮和键盘滚动逻辑保持可用；侧栏滚动区域布局不受影响。
- 实际修改为 `frontend/src/app/routes/__root.tsx` 和 `frontend/src/components/common/scroll-region.css`；未增加依赖，也未改变窗口配置或业务接口。
- 完整检查后恢复 `task dev`，后台任务 PID 为 38384，桌面 PID 为 34268，Vite PID 为 31384。

## Verification

- 修复前回归复现：距右边缘 2px 的命中目标为滚动轨道，拖动不触发窗口缩放。
- 修复后浏览器检查通过：960×600、1200×800、1440×900、1920×1080；设备缩放分别为 100%、125%、150%、200%，并模拟对应系统边框指标，覆盖亮／暗主题。
- 四组检查均确认右侧预留 8px、轨道底部预留 20px、轨道宽度保持 12px，无文档横向溢出；轨道点击、滑块拖动、拖动越过右侧预留区域、悬停显隐、PageDown 和滚轮均通过，侧栏位置保持独立。
- 使用项目当前 Wails 运行时代码验证：右侧和右下角产生 `wails:resize:e-resize` 与 `wails:resize:se-resize`，拖动滚动条时不触发窗口缩放。
- 桌面真实鼠标输入检查通过：右侧拖动使窗口从 1200×800 变为 1235×800，右下角拖动变为 1235×835；检查后恢复窗口原尺寸、置顶状态和鼠标位置。
- `pnpm exec vitest run src/components/common/scroll-region.test.tsx` PASS，2/2 测试通过。
- `task verify` PASS：格式、lint、类型检查、全部 Go 测试、25 个前端测试文件共 139 项测试及 Windows 生产构建通过。
- 开发环境恢复检查通过：Vite 首页 HTTP 200；Control API 健康接口 HTTP 200，`status=ok`、`profile=development`；桌面窗口可见。
- `git diff --check` PASS。启动前后 SHA-256 核对确认其他任务既有文件内容保持一致，包括自动生成绑定。
- 临时浏览器与桌面检查脚本、结果位于 `C:/Users/mimanchi/AppData/Local/Temp/omniproxy-resize-qa-7a6b5748/`，未添加项目测试依赖。

## Documentation

已将本任务加入 `docs/plans/CURRENT.md`。产品、架构和构建说明无需变更。

## Deviations

桌面自动化首次未命中被其他窗口遮挡的开发窗口；加入窗口命中校验、临时提高窗口层级及状态恢复后验证通过，未因此修改产品代码。

## Remaining Risks

已验证当前 Windows 桌面环境及 100%–200% 浏览器缩放组合；其他系统显示缩放未做实机验证。构建中既有的包体积及外部工具诊断未影响质量门禁通过。

## Git

`eead77532f8c169a0c4ddcd867ee5cc87cfc50c2 fix(frontend): 避免滚动条占用窗口边缘缩放区域`

仅提交本任务代码及任务文档；看板只暂存新增条目，保留其他任务的既有未提交修改。

## Result

主内容滚动条与窗口右侧、右下角缩放区域分离，滚动交互、实际桌面缩放与完整质量门禁通过，开发环境已恢复。

## Completed

2026-10-08 10:14（Asia/Singapore）
