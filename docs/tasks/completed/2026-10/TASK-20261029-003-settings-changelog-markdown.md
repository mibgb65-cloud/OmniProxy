# TASK-20261029-003: 设置页当前版本更新日志改用 Markdown 渲染

## Status

Completed

## Created

2026-10-29 10:35（Asia/Singapore）

## Request

「这个设置里面的这个位置显示信息的也需要适配吧」——设置与关于页「当前版本更新日志」卡片
仍把 `docs/releases/v*.md` 原文按纯文本结构化显示：`##`、`**粗体**`、反引号、`~~删除线~~`
和链接全部裸露。

## Goal

设置页的当前版本更新日志与升级浮窗共用同一套 Markdown 渲染，标题、段落、行内代码、粗体／
斜体／删除线、链接、有序／无序及嵌套列表、引用、代码块与分隔线正常排版，并保持既有安全边界。

## Background

TASK-20261029-001 为升级浮窗新增了零依赖 Markdown 解析与渲染
（`frontend/src/lib/markdown/`、`frontend/src/components/common/markdown.tsx`）。
`frontend/src/features/about/current-release-card.tsx` 仍使用
`frontend/src/features/about/current-release.ts` 的 `parseReleaseNotes`，该函数只识别标题、
段落与无序列表，其余原样输出，同一份发行说明在设置页和浮窗呈现不一致。

## Scope

- `current-release-card.tsx` 改用共享 Markdown 渲染。
- 删除只服务于该卡片的 `parseReleaseNotes`／`ReleaseNoteBlock`，并把其安全意图的断言迁到渲染测试。
- 保留「卡片不再重复顶级版本标题」的既有行为。
- 更新 `docs/product/UPDATE.md` 的行为描述。
- 回归测试与完整质量门。

## Out of Scope

- 不改动升级浮窗、更新下载／校验／安装链路、后端快照结构。
- 不新增依赖。

## Existing Behavior

卡片正文只显示标题（`##` 起的层级）、段落与 `- ` 列表，`**粗体**`、反引号、链接均按原文输出；
`# ` 顶级标题被丢弃，正文不含任何可点击链接。

## Expected Behavior

卡片正文与升级浮窗一致：Markdown 语义排版；`http(s)` 与站内相对链接可点击并带
`target="_blank" rel="noreferrer"`；HTML 标签与 `javascript:` 等协议仍是纯文本；顶级版本标题
仍由卡片页眉版本号承担，不在正文重复。

---

# Analysis

- 渲染组件 `Markdown` 已具备所需能力与安全边界（无 `dangerouslySetInnerHTML`、链接白名单、
  不加载远端图片），迁移只涉及展示层。
- 现有测试 `current-release.test.ts` 的两条用例只断言 `parseReleaseNotes` 的数据形状；
  函数删除后其真实意图（不解释 HTML／不安全链接、丢弃顶级标题）由新的渲染断言承接。
- `update-page.test.tsx` 通过 `getByRole("heading", { name: "登录修复" })` 断言卡片内容，
  Markdown 渲染后 `##` 变为 `h4`，角色不变，用例无需修改。

## Risks

- 卡片位于设置页窄列：长标题与代码块需可换行、可横向滚动 → 复用已有 `overflow-wrap` 与
  `pre` 的 `overflow-x-auto`，并用隔离浏览器截图核对。

---

# Implementation Plan

1. 卡片改用 `Markdown`，新增 `withoutTopHeading` 工具替代原先丢弃 `# ` 的行为。
2. 删除 `parseReleaseNotes`／`ReleaseNoteBlock`，迁移其安全断言到卡片渲染测试。
3. 更新 `docs/product/UPDATE.md`；执行格式、Lint、类型、测试与完整质量门。
4. 隔离 Chromium 截图核对窄列排版；提交、回填并归档。

---

# Implementation Result

- `current-release-card.tsx` 改用共享组件 `Markdown`，与升级浮窗同源同排版；
  卡片页眉继续显示运行版本，正文通过新增的 `withoutTopHeading` 去掉顶级版本标题，避免重复。
- 删除只服务该卡片的 `parseReleaseNotes`／`ReleaseNoteBlock`：同一份发行说明此前存在两套
  解析器，其中一套无法处理行内格式与链接。
- 安全边界随共享渲染器保持一致：无 `dangerouslySetInnerHTML`；仅 `http(s)` 与以 `/` 开头的
  站内路径渲染为链接（`target="_blank" rel="noreferrer"`）；HTML 标签、`javascript:` 等协议仍是
  纯文本；远端图片不加载。原先「不解释 HTML／不安全链接」的断言迁移到卡片渲染测试。
- 卡片内边距与 `aria-busy`、`role="region"`、页眉版本号与失败／加载／无日志三种提示均保持原样。

## Files Changed

- `frontend/src/features/about/current-release-card.tsx`
- `frontend/src/features/about/current-release.ts`（删除 `parseReleaseNotes`，新增 `withoutTopHeading`）
- `frontend/src/features/about/current-release.test.ts`
- `frontend/src/features/about/current-release-card.test.tsx`（新增）
- `docs/product/UPDATE.md`

## Technical Notes

`withoutTopHeading` 只去掉首行 `# ` 标题及其后的空行，保留原文其余字节，因此 CRLF 文档
仍按原文渲染；渲染测试直接读取真实 `docs/releases/v2.1.1-beta.1.md`，保证卡片与随包日志同步。

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

```text
pnpm vitest run src/features/about
PASS：7 文件 / 81 项

task verify
PASS（exit 0；日志 tmp/verify-settings-changelog.log）
Format / ESLint / tsc / go vet / staticcheck / Go tests：PASS
Frontend：143 files / 873 tests PASS
Windows amd64 production build：PASS
```

- 新增 3 项卡片渲染测试（Markdown 排版且无原始记号、链接白名单与 HTML／`javascript:` 纯文本、
  未知版本提示）与 3 项 `withoutTopHeading` 用例；删除的 2 项 `parseReleaseNotes` 数据形状用例
  其安全意图已由上述渲染断言承接。
- 视觉核对：隔离 Chromium 渲染真实 `v2.1.1-beta.1` 随包日志，深色／浅色各一张，
  686 宽窄列无横向溢出（`tmp/settings-changelog-20261029/*.png`）。
- 原生 WebView2 窗口与 140% 字号观感属人工验收，未据此宣称通过。

---

# Documentation

- [x] `docs/product/UPDATE.md`：「当前版本更新日志」改为与浮窗共用 Markdown 排版，
  并写明链接白名单、HTML／`javascript:` 纯文本、不加载远端图片。
- [x] `docs/plans/CURRENT.md`：顶部摘要、`## Current Version` 与 `## Recently Completed` 在归档提交中更新。
- [ ] PRD / Architecture / Tech Stack / Coding Rules / Testing / Build / ADR：无变更
  （只改前端渲染路径，接口、依赖与构建流程不变）。

---

# Deviations

原计划只改卡片渲染；实现时把「丢弃顶级标题」从解析器行为改为显式的 `withoutTopHeading` 工具，
以便在删除 `parseReleaseNotes` 后仍保留既有呈现，并能单测覆盖。

---

# Git

## Commit

```text
ae97d7b8 fix(about): 设置页当前版本更新日志改用 Markdown 渲染
```

任务归档与 CURRENT.md 回填为后续独立文档提交。

---

# Result

设置页「当前版本更新日志」已改用与升级浮窗相同的 Markdown 渲染，原始记号消失，安全边界不变；
卡片原有的页眉版本号、加载／失败／无日志提示与「不重复顶级标题」呈现保持不变。完整质量门与
窄列视觉核对通过，随后随 `v2.1.1-beta.2` 本地安装包交付（TASK-20261029-004）。

---

# Completed

2026-10-29 11:05（Asia/Singapore）
