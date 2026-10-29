# TASK-20261029-006: 关闭设置窗口时补上退出动画

## Status

Completed

## Created

2026-10-29 12:25（Asia/Singapore）

## Request

用户反馈：「这个页面在退出关闭时，没有动画，闪一下就消失了，很影响体验。」

## Goal

关闭设置与关于窗口时播放与打开对称的退出动画（遮罩淡出 + 窗口淡出并轻微缩小），
不再瞬间消失；升级浮窗与「更新完成」浮窗的退出行为同步修正。

## Background

`AppPreferencesDialog` 用 `{open ? <PreferencesWindow /> : null}` 条件挂载内容，
`open` 变为 `false` 时 React 立即卸载 `Dialog.Portal`，Radix 的退出状态（`data-state="closed"`）
没有机会渲染，因此没有退出动画，观感是闪一下就消失。同时该窗口的 Overlay 与 Content 只有定位与配色类，
打开时也没有任何过渡。`UpdateBadge` 与 `UpdateCompletedWatcher` 采用同样的条件挂载写法，
同样是「打开有动画、关闭没有」。

## Scope

- 设置与关于窗口：内容常驻 `Dialog.Root`，由 Radix Presence 控制挂载生命周期，补上进入／退出动画类。
- 升级浮窗：同样改为常驻，补齐退出动画。
- 「更新完成」浮窗：把 `open` 与通知数据分离，关闭时先播动画再卸载。
- 复用既有测试覆盖「关闭后内容确实释放」这一约束。

## Out of Scope

其他业务弹窗（新增账号、请求详情、删除确认等）的动画；改变动画时长／缓动体系；
原生窗口层面的动画；后台状态与更新逻辑。

## Existing Behavior

- 关闭设置窗口：DOM 立即移除，无过渡。
- 打开设置窗口：同样无过渡（直接出现）。
- 升级浮窗：有进入动画（`animate-in / fade-in-0 / zoom-in-95`），关闭时瞬间消失。
- 「更新完成」浮窗：有进入动画，关闭时瞬间消失。

---

# Implementation Plan

1. `app-preferences-dialog.tsx`：`PreferencesWindow` 常驻渲染；Overlay 与 Content 增加
   `motion-safe:data-[state=open|closed]` 的淡入淡出／缩放类。
2. `update-badge.tsx` 改为常驻 `<UpdateDialog />`；`update-dialog.tsx` 补齐退出动画类。
3. `update-completed-watcher.tsx` 拆分 `open` 与 `notice` 状态；`update-completed-dialog.tsx` 补齐退出动画类。
4. 复查既有测试（关闭后内容释放、焦点归还、不提前加载数据）仍成立。
5. 运行 `task verify`、核对生产 CSS 与截图、更新文档与任务记录。

---

# Implementation Result

## 代码

| 文件 | 改动 |
| --- | --- |
| `frontend/src/features/settings/app-preferences-dialog.tsx` | `{open ? <PreferencesWindow /> : null}` → 常驻 `PreferencesWindow`；Overlay 由无动画改为 `fade-in-0` / `fade-out-0`；Content 增加 `fade-in-0` + `zoom-in-95` / `fade-out-0` + `zoom-out-95`，全部带 `motion-safe:` |
| `frontend/src/features/app-update/update-badge.tsx` | 浮窗改为常驻，关闭动画由 Radix Presence 决定卸载时机 |
| `frontend/src/features/app-update/update-dialog.tsx` | 补 `data-[state=closed]:animate-out / fade-out-0 / zoom-out-95` |
| `frontend/src/features/app-update/update-completed-dialog.tsx` | 同上 |
| `frontend/src/features/app-update/update-completed-watcher.tsx` | 拆出 `open` 状态：通知数据保留到卸载，关闭时先播退出动画再让 Radix 卸载 |

要点：

- Radix 的 `Dialog.Portal` 内部用 Presence 监听 `getComputedStyle(node).animationName`；只要关闭态有动画，
  节点就会保留到 `animationend` 才卸载。因此「常驻 Root + 不条件渲染」是拿到退出动画的前提，
  而「关闭后释放表单与异步结果」的既有约束仍由 Presence 保证。
- 居中用 Tailwind v4 的独立 `translate` 属性（`.-translate-x-1\/2{translate:...}`），
  `@keyframes exit` 只改 `transform` 与 `opacity`，两者叠加不会在缩放时跳位。
- 全部过渡带 `motion-safe:` 前缀，系统要求减少动画时不播放，直接切换。
- 「更新完成」浮窗关闭后 `notice` 保留但 `open=false`，Portal 不渲染；同一版本已有 `seen` 集合去重，
  不会因此重复弹出。

## 未改动

其余业务弹窗（新增账号、请求详情、删除确认、Google 项目等）仍使用 `{open ? <Portal/> : null}`，
本次未改：它们的出现／消失频率低，且逐个改动会扩大回归面。需要时可按同一手法逐个处理。

---

# Verification

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

| 项目 | 结果 |
| --- | --- |
| `task verify` | exit 0 |
| gofmt / prettier `format:check` | 通过 |
| `go vet` / `staticcheck` / `eslint` | 通过 |
| `tsc` typecheck | 通过 |
| Vitest | 145 文件／880 项全部通过（本次未新增测试，见 Deviations 1） |
| Windows production 构建 | 通过 |

既有测试覆盖到的行为（`app-preferences-dialog.test.tsx`、`update-badge.test.tsx`、
`update-completed-watcher.test.tsx`）：

- 关闭后 `dialog` 从 DOM 移除（jsdom 不执行 CSS 动画，Presence 立即卸载，说明卸载路径正常）。
- 关闭后焦点回到入口按钮、路由与草稿不变。
- 未打开时不加载数据存储与关于信息（常驻 Portal 未破坏按需加载）。
- 「更新完成」确认后回执一次并关闭。

## 生产 CSS 核对

`frontend/dist/assets/index-*.css`（生产构建产物）中确认存在所需工具类：

```text
.data-\[state\=closed\]\:animate-out[data-state=closed]{animation:exit …}
.data-\[state\=closed\]\:fade-out-0[data-state=closed]{--tw-exit-opacity:0}
.data-\[state\=closed\]\:zoom-out-95[data-state=closed]{--tw-exit-scale:.95}
@keyframes exit{to{opacity:var(--tw-exit-opacity,1);transform:…scale3d(var(--tw-exit-scale,1)…)}}
```

同一 CSS 中 `.-translate-x-1/2{translate:…}` 使用独立 `translate` 属性，不会被 `transform` 关键帧覆盖。

## 静态采样

用生产 CSS 渲染 `tmp/update-completed-ui-20261029/exit-proof.html`（同一组 `data-state="closed"` 类），
在 40ms／90ms／160ms／400ms 四个虚拟时间点截图：40ms 与 90ms 的窗口透明度明显不同，
证明该组类产生的是随时间推进的淡出而不是瞬间切换。

---

# Documentation

- `docs/product/APPEARANCE.md` 新增一段：弹窗出现／消失的过渡（150ms 淡入淡出 + `.95` 缩放）、
  关闭时先播动画再卸载、居中用 `translate` 不跳位、减少动画偏好下直接切换。
- `docs/plans/CURRENT.md`：顶部摘要与 `## Recently Completed`。

---

# Deviations

1. 未新增前端测试：退出动画由 CSS 关键帧驱动，jsdom 不执行动画，无法在单测中断言；
   改为核对生产 CSS 中的工具类与关键帧，并用静态采样截图确认随时间推进的淡出。
   「关闭后内容释放」等既有约束由现有 5 个测试继续覆盖，未放宽任何断言。
2. 「更新完成」浮窗的 `notice` 在关闭后仍保留在 state 中（`open=false`，Portal 不渲染），
   以便退出动画期间仍有数据可渲染；这是有意的，不是泄漏——数据本身来自一次性快照。
3. 其余弹窗沿用原写法，未在本次一并修改，原因见 Implementation Result「未改动」。

---

# Git

## Commit

```text
7d4ff154 fix(ui): 弹窗关闭时补上退出动画
```

归档与 CURRENT.md 回填为后续文档提交。未推送、未打标签。

---

# Result

设置与关于窗口现在打开有淡入 + 轻微放大、关闭有淡出 + 轻微缩小（150ms，`motion-safe`），
遮罩同步淡入淡出，不再闪一下就消失；升级浮窗与「更新完成」浮窗同步补齐退出动画。
关闭后内容仍由 Radix Presence 释放，焦点归还、按需加载与草稿保留等既有行为不变。
`task verify` exit 0（145 文件／880 项前端测试、Windows production 构建）。
原生 WebView2 的实机观感仍需用户在本地安装版上确认。

---

# Completed

2026-10-29 12:55
