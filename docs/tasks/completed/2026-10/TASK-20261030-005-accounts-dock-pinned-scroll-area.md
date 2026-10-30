# TASK-20261030-005: 账号页切换提供商栏固定、仅详情区滚动

## Status

Completed

## Created

2026-10-30 21:20（Asia/Singapore）

## Request

账号页展开某个提供商后，底部「切换提供商」栏会随页面滚动一起向上移动；用户要求它固定不动，
只让中间的详情区域（图上红框）滚动。

## Goal

展开态下页面本身不再滚动：详情区独占可滚动范围，底部停靠区始终贴在内容区底部。

## Background

TASK-20261030-002（`ab47f044`）为这个问题引入了「展开后绝对定位覆盖主视口 + 详情区独立滚动」的方案，
本地 `v2.2.1-beta.1` 已包含。用户在该构建（安装版 `--webview-exe-version=2.2.1-beta.1`）上截图，
页面仍在滚动：内容整体上移 98px，而侧栏与标题栏未动。

## Scope

- 账号页展开态的滚动归属：外层页面滚动区锁定、详情区独占滚动、Dock 固定。
- 对应单元测试与隔离 Chromium 几何核对。

## Out of Scope

账号数据语义、后端接口、Dock 的供应商切换逻辑与 FLIP 动画时长、其他页面布局。

## Existing Behavior

页面外壳：`html, body, #root` 均 `overflow: hidden`，唯一滚动容器是根布局 `ScrollRegion` 的
`main.scroll-region__viewport`。展开态由 `.accounts-page[data-expanded="true"]` 绝对定位覆盖该视口的
内容区，内部 `.bento-accounts` 用 `minmax(0,1fr) auto` 两行分给详情区和 Dock。

---

# Analysis

## 现象定量

两张截图（1200×800，等于 `main.go` 里的默认窗口尺寸）做整幅平移比对：最佳匹配位移 98px，
且侧栏导航项、标题栏、窗口按钮在两图里完全重合。据此可以确定：

- 窗口和 `html/body` 没有滚动（否则侧栏会一起动）；
- 移动的是 `main.scroll-region__viewport` —— 它当时有多出约 98px 的可滚动范围并已滚动；
- 展开态覆盖层是绝对定位元素，位于滚动内容坐标系里，所以外层一旦滚动，覆盖层（连同 Dock）整体上移，
  同时覆盖层下方露出滚动容器里没有任何内容的空白 —— 与截图中 Dock 下方的空白一致。

## 隔离复现结果

用真实组件 + 真实 CSS（Vite 开发服务 + 隔离 Chromium，1200×800/1200×640/1200×560/960×600/1600×1000，
账号数量 1/2/4/7 及订阅登录账号）测量：`main` 的 `scrollHeight - clientHeight` 恒为 0，
把 `scrollTop` 设成 400 仍是 0，Dock 坐标不变，详情区自身可滚动。也就是说尺寸和内容量本身不会产生溢出，
真实会话里的溢出只能来自残留在滚动内容里的其他元素（过渡层副本、浮动层等），CSS 无法穷举。

## 结论

不依赖「一定没有溢出」，而把不变量显式化：进入展开态就锁死外层页面滚动，把「可滚动」这件事
只留给详情区；同时去掉对 `:has()` 结构推断的依赖。

---

# Implementation Plan

1. `AccountsView` 展开时给最近的 `.scroll-region` 打 `data-locked`，并把外层 `scrollTop` 归零；
   收起时移除标记。
2. `scroll-region.css`：`[data-locked="true"]` 的视口 `overflow-y: clip`（`clip` 连程序化滚动也不允许），
   并隐藏该区域的滚动条轨道。
3. `bento-accounts.css`：覆盖层加 `overflow: clip`；两行网格由 `data-has-dock` 显式声明，替换 `:has(> .bento-dock)`。
4. 补单测（锁定/解除、`data-has-dock`）与隔离 Chromium 溢出注入核对，然后跑 `task verify`。

---

# Implementation Result

## 代码

| 位置 | 改动 |
| --- | --- |
| `frontend/src/features/accounts/accounts-view.tsx` | 新增 `useLayoutEffect`：展开时 `region.dataset.locked = "true"`、外层 `scrollTop = 0`，收起时移除；`.bento-accounts` 增加 `data-has-dock` |
| `frontend/src/components/common/scroll-region.css` | `.scroll-region[data-locked="true"] > .scroll-region__viewport { overflow-y: clip }`；同区域轨道 `display: none` |
| `frontend/src/features/accounts/bento-accounts.css` | `.accounts-page[data-expanded="true"]` 增加 `overflow: clip`；`:has(> .bento-dock)` 改为 `[data-has-dock="true"]` |
| `frontend/src/features/accounts/accounts-view.test.tsx` | `renderPage({ shell: true })` 复刻根布局滚动区；新增「展开时锁定页面滚动」测试 |

行为：展开后外层视口既不能被滚轮滚动，也不能被程序化 `scrollTop` 抖动（`clip` 语义），
`scrollTop` 归零保证折叠网格里滚动过的位置不会带到展开态；详情区仍是内部 `ScrollRegion`，
Dock 与详情同处不滚动的两行网格。

## 隔离 Chromium 核对

1200×800，先在折叠网格里滚动 420px 再展开，然后往滚动内容里注入 660px 的绝对定位残留元素：

| 项目 | 结果 |
| --- | --- |
| `region.dataset.locked` | `true`（收起后移除） |
| 外层 `overflow-y` | `clip` |
| 轨道显示 | `none` |
| 注入溢出后的 `scrollHeight - clientHeight` | 660 |
| 注入溢出后 `main.scrollTop`（尝试设为 600） | 0 |
| 注入溢出后 Dock `top` | 657（与未注入时一致） |
| 详情区可滚动量 / 滚动 300 后 Dock | 145 / 657（不动） |

截图：`tmp/accounts-dock-fixed/lock-expanded-top.png`、`lock-expanded-with-stray.png`。

## 交付构建（v2.2.1-beta.2）

安装版是打包产物，源码修复必须重新出包才能在本机生效。版本源递增至 `2.2.1-beta.2`，
新增双语发布说明 `docs/releases/v2.2.1-beta.2.md`，执行
`scripts/build-release.ps1 -Version v2.2.1-beta.2 -OutputDirectory release-assets/v2.2.1-beta.2`
（日志 `tmp/release-v2.2.1-beta.2.log`，exit 0）：

| 项目 | 结果 |
| --- | --- |
| 脚本内 `task --force verify` | exit 0（145 文件／885 项前端测试） |
| `build/windows/info.json` | 数字版本 `2.2.1.0`、`ProductVersion` `2.2.1-beta.2` |
| NSIS 打包 | `Total size: 20354025 / 81188656 bytes (25.0%)` |
| 安装器 | `release-assets/v2.2.1-beta.2/OmniProxy-Setup-v2.2.1-beta.2-windows-amd64.exe`，20,354,025 bytes |
| SHA256 | `0d2ad9a56c2edfc44ec6ee1314ef315bc72732e41d8c8b71d1935c031b3d866f` |
| 内嵌前端的 `bin/omniproxy.exe` | 23,297,024 bytes，SHA256 `399bd1aa82fa3f2ded133ed322fbd90a9fee2874d5b1bc1103fa09f436d1d460` |
| `test-windows-metadata.ps1` / `verify-release-assets.ps1` | 通过／通过 |

`build/config.yml` 与 `build/windows/nsis/wails_tools.nsh` 的 `2.2.1` 已在上一个 beta 写入，本次无需改；
`info.json` 被脚本改写为 CRLF，提交前已还原为 LF。

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
| `pnpm vitest run src/features/accounts src/components/common` | 46 文件／257 项通过 |
| `accounts-view.test.tsx` | 14 项通过（含新增锁定测试） |
| `task verify` | exit 0（145 文件／885 项前端测试、Go 全量包与 Windows production 构建） |
| 隔离 Chromium 几何核对 | 见上表，「页面不可滚动、Dock 不动、详情区滚动」全部成立 |

未覆盖（需用户实机确认）：原生 WebView2 下滚轮手感与 Dock 观感。

---

# Documentation

- 更新 `docs/plans/CURRENT.md`（顶部摘要、`Current Version`、`Recently Completed`）。
- 新增 `docs/releases/v2.2.1-beta.2.md`（中英双语）。
- 本记录：`docs/tasks/completed/2026-10/TASK-20261030-005-accounts-dock-pinned-scroll-area.md`。

---

# Deviations

1. 未能先在隔离环境复现「外层滚动区出现 98px 溢出」的触发条件：真实组件 + 真实 CSS 在多尺寸、
   多账号数量与订阅登录账号下都测不到溢出。因此修复不是「堵住某个具体溢出源」，而是把
   「展开态页面不可滚动」变成显式不变量，并用注入 660px 残留溢出的场景证明它成立。
2. 视口用 `overflow-y: clip` 而不是 `hidden`：`hidden` 仍允许程序化 `scrollTop`，
   而滚动位置恢复逻辑会写 `scrollTop`。
3. 修复与出包合成一条记录（`v2.2.1-beta.2`）：本次出包只承载这一处修复，单独记录会重复同样的验证信息。

---

# Git

## Commit

```text
902cc49c fix(accounts): 锁定展开态页面滚动并固定切换提供商栏
```

归档与 CURRENT.md 回填为后续文档提交。未推送、未打标签、未签名。

---

# Result

账号页展开态的滚动归属已修正：外层页面滚动区在展开时被锁定（`data-locked` + `overflow-y: clip`）
并把偏移归零，覆盖层自身 `overflow: clip`，Dock 所在两行网格改用显式 `data-has-dock`；
即使滚动内容里残留 660px 溢出，页面也不可滚动、Dock 坐标不变，只有详情区滚动。
`task verify` exit 0（145 文件／885 项前端测试、Windows production 构建），
`v2.2.1-beta.2` 安装包 20,354,025 bytes，SHA256
`0d2ad9a56c2edfc44ec6ee1314ef315bc72732e41d8c8b71d1935c031b3d866f`，未签名、未自动安装、未推送、未公开发布。

---

# Completed

2026-10-30 21:45
