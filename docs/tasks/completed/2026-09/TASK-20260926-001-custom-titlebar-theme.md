# TASK-20260926-001: 自绘标题栏与亮暗模式

## Status

Completed

## Created

2026-09-26 08:53

## Request

“修改这个操作栏，不要用原生的，要自己实现，并且加入亮暗模式的开关按钮，增加亮色暗色模式。”

## Goal

将 Windows 原生标题栏替换为与应用风格一致的自绘标题栏，保留拖动、最小化、最大化/还原、关闭，并允许切换和记住亮暗模式。

## Existing Behavior

- Wails 使用系统标题栏；前端根布局从侧边栏开始。
- 样式已定义亮暗变量，但 `index.html` 固定 `dark` 类，没有主题切换与持久化。

## Expected Behavior

- 自绘标题栏包含应用标识、主题切换和窗口控制；窗口仍可拖动、调整大小及最大化/还原。
- 所有页面随主题切换；重启后恢复用户选择；按钮有中文/英文可访问名称和键盘焦点。

---

# Analysis

## Related Modules

- `internal/transport/wails/app.go`：无边框窗口配置。
- `frontend/src/app/routes/__root.tsx`、`components/common`、`styles.css`：标题栏和布局。
- `frontend/src/stores/ui-store.ts`、`main.tsx`、`i18n/locales`：主题持久化与文案。

## Dependencies

- 现有 Wails 3 运行时 API、Zustand、Lucide React；不新增依赖。

## Risks

- 无边框窗口需验证拖动、缩放与窗口按钮的实际 Windows 行为。
- 首屏必须在 React 渲染前应用已保存主题，避免亮暗闪烁。

---

# Implementation Plan

1. Wails 主窗口启用 `Frameless`，保留原尺寸约束与 Windows 装饰。
2. 添加自绘标题栏和拖动区域，接入 Wails 最小化、最大化/还原及关闭 API。
3. 为 Zustand UI 状态添加经校验的持久化主题；在首屏与后续切换时同步根元素的主题类。
4. 补齐标题栏视觉状态与中英文案，添加关键交互测试。
5. 运行 `task verify`，实际启动 Windows 程序检查界面和窗口操作，再更新记录与状态看板。

---

# Progress

- 已确认 Wails 3.0.0-beta.25 支持 `Frameless`、`--wails-draggable` 与前端 `Window` 控制 API；现有样式已具备亮暗颜色变量。
- 完成自绘标题栏和主题持久化；针对性测试、完整 `task verify` 通过。
- 在真实 Windows 窗口验证两套主题与最小化、最大化/还原、拖动、缩放。

---

# Implementation Result

## Changes

- Wails 主窗口启用 `Frameless`；React 根布局在侧边栏和主内容上方增加自绘标题栏。
- 标题栏含应用标识、亮暗切换、最小化、最大化/还原及关闭按钮；窗口控制使用 Wails Runtime，最大化图标跟随窗口事件更新，失败时有可见提示。
- 主题保存在现有 Zustand UI 状态中，重启时先于 React 首帧应用；从存储恢复时校验主题、语言及侧边栏状态，避免异常值覆盖操作函数。
- 补齐中英文按钮文案和可访问名称，按钮支持键盘焦点。

## Files Changed

- `internal/transport/wails/app.go`
- `frontend/src/components/common/app-titlebar.tsx`、`app-titlebar.test.tsx`
- `frontend/src/app/routes/__root.tsx`、`theme-provider.tsx`、`providers.tsx`
- `frontend/src/stores/ui-store.ts`、`ui-store.test.ts`、`frontend/src/lib/theme.ts`
- `frontend/src/main.tsx`、`styles.css`、`i18n/locales/{zh,en}.ts`
- `docs/plans/CURRENT.md`、本任务记录

## Technical Notes

- 使用 Wails CSS 拖动区域与框架自带的无边框窗口缩放能力；未启用实验性的 WebView2 Composition Hosting。
- 默认保持原有暗色外观，用户可切换并持久保存亮色选择。

---

# Verification

## Tests

- [x] 前端标题栏交互测试：主题切换与持久化、窗口控制调用、最大化事件、失败提示。
- [x] UI 状态测试：已保存主题恢复、异常存储值回退。
- [x] 既有 Go 与前端测试。

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

- `task verify` PASS；Go 测试全通过，前端 11 个测试文件 / 41 个测试全通过，Windows 生产构建成功。
- 开发窗口实测：`/healthz` 返回 `ok`；无系统标题栏；亮色/暗色均正常渲染。
- Windows UI Automation 调用标题栏按钮：最大化、还原、最小化均成功；手动模拟标题栏拖动与右下角缩放成功，测试后恢复位置与尺寸。
- `git diff --check` PASS。

---

# Documentation

- [x] `docs/plans/CURRENT.md`：同步任务状态。
- PRD、架构、技术栈及 ADR 无需更新：本任务沿用现有 UI 栈与窗口边界，没有改变产品能力边界或核心架构。

---

# Deviations

None.

---

# Git

## Commit

```text
15587fadf30db0b9f4e5860fe54641ff93e2d1f0 feat(frontend): add custom titlebar and light-dark theme
```

（本任务的 commit hash 由紧随其后的记录提交回填。）

---

# Result

自绘标题栏和可持久化的亮暗模式已实现，自动化质量门禁与真实 Windows 窗口操作验证通过。

---

# Completed

2026-09-26 09:11
