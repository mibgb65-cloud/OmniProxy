# TASK-20261029-001: 升级浮窗布局优化与发布说明 Markdown 渲染

## Status

Completed

## Created

2026-10-29 09:12（Asia/Singapore）

## Request

「这个升级页面的UI布局需要进行优化，还有更新内容的md的正常的格式显示适配」

附图为侧栏升级标记打开的升级浮窗：版本摘要为四行纯文本，更新内容为 `whitespace-pre-wrap`
原文，`# 标题`、`**加粗**`、反引号与嵌套列表都原样显示，滚动条直接出现在正文上，操作按钮与
发布页链接混在正文流里。

## Goal

升级浮窗改为「固定标题栏 + 可滚动正文 + 固定操作区」三段式布局，版本摘要改为标签／值结构，
发布说明按 Markdown 语义排版显示（标题、段落、加粗、行内代码、链接、有序／无序及嵌套列表、
引用、代码块、分隔线），且不执行发布说明中的 HTML。

## Background

`frontend/src/features/app-update/update-dialog.tsx` 目前把 `release.notes` 当纯文本用
`whitespace-pre-wrap` 展示；GitHub Release 正文是 Markdown（`docs/releases/v*.json` 同源），
因此用户看到的是带 `#`、`**`、反引号的原始源码。

设置页的「当前版本更新日志」已有一个只处理标题／段落／列表的纯文本结构化器
（`frontend/src/features/about/current-release.ts` 的 `parseReleaseNotes`），已在
`docs/product/UPDATE.md` 明确记录为「只展示安全文本标题／段落／列表，不执行 HTML 或正文中的链接」。
本任务不改变该处的既定行为。

## Scope

- 升级浮窗布局：标题栏、版本摘要、发布说明区、操作区、发布页入口重新排布。
- 新增发布说明 Markdown 渲染能力（块级 + 行内），用于升级浮窗正文。
- 新增中英文案键，替换原「当前版本：…／可升级版本：…」整句键。
- 单元测试：Markdown 解析与安全边界、浮窗渲染回归。
- 文档：`docs/product/UPDATE.md` 浮窗行为描述。

## Out of Scope

- 不引入 `react-markdown`／`remark` 等新依赖，不改动更新下载、校验、安装链路。
- 不改变设置页「当前版本更新日志」的纯文本结构化行为与相关测试。
- 不改动后端 `internal/application/updater` 的说明截断（12000 字符）与快照结构。
- 不重新打包安装程序、不发布。

## Existing Behavior

浮窗为单个 `max-w-lg` 滚动容器：标题与关闭按钮、`bg-muted/50` 四行摘要、`更新内容`
标题 + `max-h-52` 原文段落（`whitespace-pre-wrap`）、下载／重启按钮组、发布页链接。

## Expected Behavior

- 标题栏与操作区固定，只有正文（版本摘要 + 发布说明）滚动；浮窗更宽（`min(880px, …)`）。
- 版本摘要以「当前版本 / 可升级版本」标签 + 等宽版本值呈现，发布时间与包大小作为元信息，
  发布页入口位于摘要卡片内。
- 发布说明按 Markdown 渲染：`#` 标题不再显示井号，`**x**` 为加粗，反引号为行内代码，
  `- `／`  - ` 为两级列表，`---` 为分隔线；HTML 标签与 `javascript:` 链接一律按纯文本显示。
- 没有发布说明时保留「发布者尚未提供更新说明。」。

---

# Analysis

## Related Modules

- `frontend/src/features/app-update/`（浮窗、操作区、测试）
- `frontend/src/components/common/`（新增 Markdown 渲染组件）
- `frontend/src/lib/`（新增 Markdown 解析）
- `frontend/src/i18n/locales/{zh,en}.ts`

## Dependencies

无新依赖。项目规则（`docs/development/CODING_RULES.md`）要求新依赖必须有当前具体需求；
发布说明只需 GitHub 常用子集，仓库已有同类轻量解析先例，手写解析更可控也更小。

## Risks

- 手写 Markdown 解析对边界输入（未闭合 `**`、嵌套列表缩进、CJK 与行内代码混排）处理不当
  → 用解析器单元测试覆盖，未知语法回退为纯文本而不是抛错。
- 渲染 `release.notes` 里的链接会新增外链入口 → 只允许 `http`/`https`/站内相对路径，
  统一 `target="_blank" rel="noreferrer"`，其余保持纯文本。
- 浮窗布局从单列流改为三段式，需确认窄窗口与 140% 字号下不横向溢出。

---

# Implementation Plan

1. 新增 `frontend/src/lib/markdown.ts`：块级（标题、段落、列表、引用、代码块、分隔线）与
   行内（粗体、斜体、删除线、行内代码、链接、自动链接、硬换行）解析，输出纯数据 AST。
2. 新增 `frontend/src/components/common/markdown.tsx`：按产品排版渲染 AST，未知内容回退文本。
3. 重写 `update-dialog.tsx` 为三段式布局，新增 `update-version-summary.tsx` 承载版本摘要。
4. 调整 `update-controls.tsx` 以适配固定操作区（按钮右对齐、说明文字在左）。
5. 新增／同步 i18n 键，补 `markdown.test.ts` 与浮窗测试，执行前端质量门。
6. 更新 `docs/product/UPDATE.md`、`docs/plans/CURRENT.md`，提交并回填记录。

---

# Progress

- 2026-10-29：定位问题——浮窗把 GitHub Release Markdown 正文当纯文本渲染；确认仓库无
  Markdown 依赖，设置页仅有一个不复用链接／行内格式的纯文本结构化器。
- 2026-10-29：完成解析器（16 项解析／渲染单测）与浮窗三段式布局，用隔离 Chromium 截图核对
  深色／浅色、空闲／可重启，以及 900×560 窄窗。

---

# Implementation Result

任务完成后填写。

## Changes

- 升级浮窗改为三段式布局：`header`（标题／描述／关闭）与 `footer`（操作区）固定，
  中间 `.app-scroll` 区域滚动版本摘要与发布说明；浮窗宽度改为 `min(720px, calc(100vw-2rem))`，
  不再让正文承担整窗滚动。
- 新增 `update-version-summary.tsx`：摘要卡按「当前版本 →（箭头）可升级版本」两栏排布，
  下方元信息行显示发布时间、包大小，并在右侧提供「打开发布页面」入口。
- 新增 Markdown 渲染链路（零新依赖）：
  `lib/markdown/{types,inline,blocks,index}.ts` 解析为纯数据 AST，
  `components/common/markdown.tsx` 按产品排版渲染标题、段落、粗体／斜体／删除线、行内代码、
  链接、自动链接、有序／无序及嵌套列表、引用、代码块与分隔线。
- 发布说明解析的安全边界：不解释 HTML（无 `dangerouslySetInnerHTML`），只把 `http(s)` 与站内
  相对路径渲染为链接（`target="_blank" rel="noreferrer"`），远端图片不加载、只保留替代文字，
  未闭合定界符与 `javascript:` 等协议按纯文本显示。
- 操作区改为「状态说明在左、按钮在右」；新增下载前提示文案，避免空闲态按钮孤立。
- i18n：`appUpdate.current`／`appUpdate.target` 整句键替换为
  `appUpdate.currentLabel`／`appUpdate.targetLabel`／`appUpdate.releaseInfo`，
  并新增 `appUpdate.downloadHint`；中英文同步。

## Files Changed

- `frontend/src/features/app-update/update-dialog.tsx`
- `frontend/src/features/app-update/update-version-summary.tsx`（新增）
- `frontend/src/features/app-update/update-controls.tsx`
- `frontend/src/components/common/markdown.tsx`（新增）
- `frontend/src/lib/markdown/{types,inline,blocks,index}.ts`（新增）
- `frontend/src/i18n/locales/{zh,en}.ts`
- `docs/product/UPDATE.md`

## Technical Notes

- 行内解析区分 `**`/`__` 与 `*`/`_`，`_` 在词内（如 `snake_case`）不视为斜体；粗体优先于斜体，
  避免 `**x**` 被拆成两层斜体。
- 段落内软换行合并为空格，行尾两个空格或反斜杠按硬换行输出，兼容 GitHub Release 正文习惯。
- 设置页「当前版本更新日志」仍沿用原有纯文本结构化器（`parseReleaseNotes`），
  其行为在 UPDATE.md 有明确记录，本任务未改动以避免扩大范围。

---

# Verification

## Tests

- [x] Unit Tests
- [ ] Integration Tests
- [ ] E2E Tests

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task verify
PASS (exit 0)
```

- 新增 18 项前端测试：解析器 12 项（标题／段落／分隔线、两级列表与行内格式、有序列表起始、
  围栏代码块、引用与硬换行、斜体／删除线／自动链接、词内下划线、未闭合定界符、链接白名单、
  HTML 纯文本、Windows 换行、空输入），渲染 3 项（语义元素、外链属性、无脚本／不安全链接），
  浮窗 3 项（摘要分区、Markdown 排版无原文记号、无说明提示）。
- `task verify` 覆盖：Prettier、ESLint、`tsc --noEmit`、`go vet`、`staticcheck`、全部 Go 测试、
  142 文件／869 项前端测试、Wails 绑定生成与 Windows production 构建，退出 0。
- 首轮 `task verify` 在最后的 `.syso` 清理步骤瞬时报 exit 1；同一条命令重跑（无代码改动）
  exit 0，日志分别在 `tmp/update-dialog-ui-20261029/verify.log` 与 `verify2.log`，未跳过或放宽任何检查。
- 视觉核对：隔离 Chromium 截图检查深色／浅色 × 空闲／可重启四态与 900×560 窄窗
  （`tmp/update-dialog-ui-20261029/*.png`），标题栏与操作区固定、正文独立滚动、无横向溢出、
  Markdown 标题／加粗／行内代码／嵌套列表／引用／代码块均按预期排版。原生 WebView2 窗口与
  140% 字号下的观感属人工验收，本任务未据此宣称通过。

---

# Documentation

检查结果（勾选＝本次有改动）：

- [ ] PRD — 不改：`PRD.md` 只描述「侧栏升级标记打开版本浮窗」，细粒度行为在 `UPDATE.md`
- [ ] Architecture / Tech Stack / Coding Rules / Testing / Build / ADR — 不改：无新依赖、无接口或构建流程变更
- [x] CURRENT
- [x] `docs/product/UPDATE.md`

已更新 `docs/product/UPDATE.md` 的浮窗描述：三段式布局、摘要分区、Markdown 排版范围与
安全边界（HTML／`javascript:` 纯文本、远端图片只保留替代文字）。

---

# Deviations

- 原计划未打算改动操作区；实现时发现三段式布局下按钮孤立在右下角，因此把
  `update-controls.tsx` 的状态说明与按钮改为左右两栏，并新增 `appUpdate.downloadHint`
  作为空闲态提示，不影响任何下载／安装行为。
- 计划中的「复用现有 `parseReleaseNotes`」未采用：该结构化器只处理标题／段落／列表，
  且其「不渲染链接」行为已写入 `docs/product/UPDATE.md`；新解析器与它并存属于不同边界
  （浮窗正文 vs 离线当前版本日志），本任务不改变设置页行为。

---

# Git

## Commit

```text
cd67d25b feat(updates): 重排升级浮窗并渲染发布说明 Markdown
```

任务归档与 CURRENT.md 更新为后续独立文档提交。

---

# Result

升级浮窗改为固定标题栏／滚动正文／固定操作区，版本摘要按标签值分区并带发布时间、包大小与
发布页入口；发布说明由新增的零依赖 Markdown 解析与渲染链路按语义排版，HTML 与不安全链接
只作纯文本。完整质量门与隔离浏览器视觉核对通过，未打包、未发布。

---

# Completed

2026-10-29 10:05（Asia/Singapore）
