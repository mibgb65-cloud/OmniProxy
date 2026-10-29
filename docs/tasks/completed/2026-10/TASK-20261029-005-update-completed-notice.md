# TASK-20261029-005: 更新完成后提示新版本与新功能

## Status

Completed

## Created

2026-10-29 11:40（Asia/Singapore）

## Request

用户反馈：「我感觉现在每次更新完了没有任何提示，我需要你在用户更新完成后，弹窗提示用户更新已完成，提示一下更新了哪些新功能。」

## Goal

应用完成自我更新并重新打开后，自动弹出一次「更新完成」浮窗，显示升级前版本 → 当前运行版本，
并按 Markdown 排版列出本版新增内容；用户确认后不再重复弹出。

## Background

现状：`InstallApplicationUpdate` 启动独立安装辅助程序后应用退出，安装器完成后自动重新打开新版本，
但新版本启动后没有任何「已升级」的反馈；侧栏徽标与设置页只显示「有无可用新版本」，
`updater` 的内存快照虽然会在 `restore()` 中把状态置为 `installed`，前端从不展示该状态。

## Scope

- 后端 `updater`：在持久化记录中区分「安装前版本」与「已提示版本」，快照新增 `installed` 通知。
- 新增绑定方法 `AcknowledgeApplicationUpdateInstalled`，只清除本次提示标记。
- 前端：新增「更新完成」浮窗与应用级监视器，正文复用运行版本内置的发布说明（`docs/releases/`），
  沿用既有 Markdown 渲染与三段式布局。
- 文案 `zh.ts` / `en.ts`；`docs/product/UPDATE.md` 补充该提示的行为与边界。

## Out of Scope

自动安装、跳过用户确认、签名、打标签、推送、公开发布、改动更新渠道或下载校验逻辑、
联网获取已安装版本的日志（正文只来自本地内置说明）。

## Existing Behavior

- 安装成功后应用自动重启，界面上没有任何「更新完成」提示（用户本次反馈的问题）。
- `restore()` 在 `version.Compare(当前, 记录的包版本) >= 0` 时把 `download.state` 置为 `installed`，
  但该字段没有任何界面使用，也没有「已提示」标记，无法区分「刚升级」与「一直如此」。
- 设置页「当前版本更新日志」已能按 Markdown 渲染本地内置的 `docs/releases/<当前版本>.md`。

---

# Implementation Plan

1. `updater` 记录结构新增 `previous`（安装前运行版本）与 `noticeSeen`（已提示的运行版本）。
2. `Install` 成功启动辅助程序时记录 `previous`；`restore()` 在判定升级已生效且 `noticeSeen` 不一致时
   填充 `Snapshot.installed{version, from}`。
3. 新增 `Service.AcknowledgeInstalled()` 与 Wails 绑定，写入 `noticeSeen` 并清空快照通知。
4. 前端新增 `update-completed-dialog` 与 `update-completed-watcher`，挂载在根布局；
   正文取运行版本的本地发布说明（去掉重复顶级标题），缺失时给出通用说明。
5. 关闭浮窗（确认、Esc、遮罩、关闭按钮）即回执；同一会话内不再重复弹出。
6. 补充 Go 与前端测试、更新文档、运行 `task verify`、提交并归档。

---

# Implementation Result

## 后端（`internal/application/updater`）

| 位置 | 改动 |
| --- | --- |
| `service.go` | 新增 `Installed{version, from}`；`Snapshot` 新增 `installed,omitempty`；`Snapshot()` 抽出 `snapshotLocked()`，`Configure` 重置候选时保留该通知 |
| `persistence.go` | 记录新增 `previous` / `noticeSeen`；`restore()` 在 `version.Compare(运行版本, 包版本) >= 0` 且 `noticeSeen != 运行版本` 时填充 `installed`；新增 `AcknowledgeInstalled()` |
| `transfer.go` | `Install` 写入 `previous = 本次运行版本` |
| `app_updates.go` | 新增 `AcknowledgeApplicationUpdateInstalled()` 绑定 |

判定细节：

- 「刚刚升级」只在既有安装记录上出现——只下载未安装、安装回执失败、无记录的手动覆盖安装都不提示。
- 发布标签带 `v` 前缀而运行版本不带，来源版本只用 `version.Compare(...) == 0` 判定同一版本，避免把 `v2.0.0-beta.2`
  与 `2.0.0-beta.2` 当成两个版本或凭空编造 `from`。
- 回执写入 `status.json` 的 `noticeSeen`；写入失败时回滚内存状态并如实返回错误，下次启动会再提示，
  不假装已读。安装包与 `install-result.json` 不被删除。

## 前端

- `update-completed-dialog.tsx`：三段式浮窗（固定标题栏／可滚动正文／固定操作区），
  `w-[min(680px,calc(100vw-2rem))]`、`max-h-[calc(100dvh-2rem)]`；标题栏含图标、`Dialog.Title`
  与「已从 X 升级到 Y」说明；正文依次为版本过渡条（`role="region"`「版本信息」）与
  「本版新功能」Markdown 面板；页脚左侧为一次性提示说明，右侧「查看完整更新日志」「知道了」。
- `update-completed-watcher.tsx`：应用级监视器，读取快照里的 `installed`，同一版本每会话只提示一次；
  确认、关闭按钮、Esc、遮罩点击都调用回执，「查看完整更新日志」先回执再用
  `useAppPreferences().show("about")` 打开设置页关于分区（不在浮窗之上叠加第二个模态框）。
- `__root.tsx`：与 `CodexQuotaActivationWatcher` 并列挂载，不依赖设置面板是否打开。
- 正文来源：`getCurrentReleaseNotes(运行版本)` 读取构建时收录的 `docs/releases/v<版本>.md`，
  经 `withoutTopHeading` 去掉重复顶级标题后交给共用 `Markdown` 组件——与升级浮窗、设置页日志同一条渲染链，
  链接白名单与「HTML 只作文本、远端图片只留替代文字」的边界自动生效，且不额外联网。
- i18n：`appUpdate.completed*` 9 条（中英各一套）。
- `use-application-update.ts`：新增 `acknowledge` mutation，成功即用返回快照替换缓存。

## 测试

- `internal/application/updater/notice_test.go`（4 项）：升级并重启后产生一次通知（含 `from`）、
  回执后再次启动不再出现、只下载未安装不提示、安装回执失败不提示。
- `frontend/src/features/app-update/update-completed-dialog.test.tsx`（3 项）：渲染真实内置说明、
  无 `from` 时的说明与无版本过渡条、无匹配说明时的兜底文案；并断言 `# OmniProxy v2.1.1-beta.2`、
  `## 新特性`、`**升级浮窗三段式布局**` 等原始记号不再出现。
- `frontend/src/features/app-update/update-completed-watcher.test.tsx`（4 项）：自动弹出、
  确认后回执并关闭、「查看完整更新日志」回执后打开关于分区、无待提示时不打扰。
- `frontend/src/app/page-scroll-restoration.test.tsx`：为新挂载的应用级监视器补上同 `CodexQuotaActivationWatcher`
  的桩，保持该测试只验证滚动隔离（无 QueryClient 依赖）。

## 视觉核对

隔离 Chromium（生产 CSS）截取四态，存 `tmp/update-completed-ui-20261029/`：
`dark.png`、`light.png`（1440×900）、`dark-narrow.png`（900×560）、`light-narrow.png`（760×560），
另有 `dark-beta3.png`（本版真实说明）与 `dark-beta3-unknown.png`（无说明兜底）。
深色／浅色、宽窗／窄窗下标题栏与页脚固定、正文独立滚动，窄窗页脚两个按钮仍在一行；
兜底态浮窗压缩为内容高度，不出现空白滚动区。原生 WebView2 与 140% 字号观感仍需用户实机确认。

## 本地构建

`scripts/build-release.ps1 -Version v2.1.1-beta.3 -OutputDirectory release-assets/v2.1.1-beta.3`（exit 0，
日志 `tmp/release-v2.1.1-beta.3.log`）：安装器 20,340,868 bytes，
SHA256 `6f9068194a06b2fddab3292b3a95a7bd45fd2a2fd9ef1e7ee0ca4ec7509e35fe`；
NSIS `Total size: 20340868 / 81065145 bytes (25.0%)`；应用 `2.1.1-beta.3`／安装器 `2.1.1`／数字版本 `2.1.1.0`。

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
| `task verify` | exit 0（首次失败见 Deviations 1、2，修正后重跑通过） |
| gofmt / prettier `format:check` | 通过 |
| `go vet` / `staticcheck` / `eslint` | 通过 |
| `tsc` typecheck | 通过 |
| Vitest | 145 文件／880 项全部通过（本次新增 7 项） |
| Go 全量包测试 | 通过，含新增 4 项 `updater` 通知测试 |
| Windows production 构建 | 通过 |
| 发布脚本内 `task --force verify` | exit 0（145 文件／880 项） |
| NSIS 打包 | 通过（25.0%） |
| `test-windows-metadata.ps1` | 「Windows 应用和安装器版本资源校验通过」 |
| `verify-release-assets.ps1` | 「发布资产校验通过：v2.1.1-beta.3」 |
| SHA256 独立复算 | 与 `.sha256` 一致（`6f906819…35fe`） |

已知无害噪声：Vite 大 chunk 警告、jsdom `window.scrollTo` 未实现、`tail`／`uname` 在 Windows 缺失；
均不影响结果，未跳过或放宽任何检查。

---

# Documentation

- `docs/product/UPDATE.md` 新增「更新完成提示」小节：出现时机、三段式结构、正文来源与链接白名单、
  三种确认方式、`noticeSeen` 回执与失败重提示、以及「哪些情况不提示」。
- `docs/releases/v2.1.1-beta.3.md`：本版双语发布说明（该说明同时是本浮窗正文的来源）。
- `docs/plans/CURRENT.md`：顶部摘要、`## Current Version`、`## Recently Completed`。

---

# Deviations

1. 首次 `task verify` 失败于 `tsc`：新测试的渲染辅助函数参数默认值被推断为 `{}`，显式标注为
   `{ confirm?: () => void; notes?: () => void }` 后通过。
2. 第二次 `task verify` 失败于 `page-scroll-restoration.test.tsx`：根布局新增的监视器使用了
   react-query hook，而该测试刻意不提供 `QueryClient`。按同文件既有做法
   （`CodexQuotaActivationWatcher` 已被桩掉）为 `UpdateCompletedWatcher` 加桩，而非给测试塞 Provider。
3. 实现过程中发现自锁：`AcknowledgeInstalled()` 持锁后又调用会再次加锁的 `Snapshot()`，
   导致 Go 测试挂起。抽出 `snapshotLocked()` 供持锁路径使用，修正后测试通过。
4. `AcknowledgeInstalled()` 的回执标记写入失败时会回滚内存通知并返回错误，
   前端因此仍会在下次启动再次提示——这是有意的「不假装已读」，不是遗漏。
5. 版本递增到 `2.1.1-beta.3`：Windows 数字版本仍为 `2.1.1.0`（与 beta.1／beta.2 相同，发布脚本按 patch 段映射）。
6. `build/windows/info.json` 被发布脚本改写为 CRLF，提交前还原为 LF，最终 diff 只含版本值。

---

# Git

## Commit

```text
84e3b785 feat(update): 更新完成并重启后提示本版新功能
```

版本源与发布说明为独立的 `chore(release)` 提交；归档与 CURRENT.md 回填为后续文档提交。
未推送、未打标签、未签名。

---

# Result

升级并重启后会弹出一次「更新完成」浮窗，说明「升级前版本 → 当前版本」并按 Markdown 列出本版新增内容；
确认、关闭按钮、Esc 与遮罩点击都写入 `updates/status.json` 的 `noticeSeen`，同一版本只提示一次。
`task verify` exit 0（145 文件／880 项前端测试、Windows production 构建），
本地 `v2.1.1-beta.3` 安装包 20,340,868 bytes，SHA256 `6f9068194a06b2fddab3292b3a95a7bd45fd2a2fd9ef1e7ee0ca4ec7509e35fe`。
未签名、未自动安装、未推送、未公开发布。

---

# Completed

2026-10-29 12:10
