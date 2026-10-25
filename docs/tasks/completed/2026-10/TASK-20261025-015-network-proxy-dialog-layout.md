# TASK-20261025-015: 网络代理弹窗布局重做（两栏分区 + 固定页脚）

## Status

Completed

## Created

2026-10-25 21:55

## Request

用户附「提供商」页「网络代理」弹窗截图（`ProxySettingsDialog` 展开态）并提出：

> 这个是感觉UI不太行啊内容太拥挤了

截图反映的实际问题：560px 宽的单栏弹窗把所有内容堆在一条纵向滚动里——标题、说明、启用开关、
协议、地址＋端口、用户名＋密码、清除密码、绕过列表、探测／测试、检测结果、13 行提供商分流
（每行 3 个模式按钮）、保存／取消。提供商列表行高只有 ~26px，三态按钮挤成一片灰；保存按钮
被 13 行列表推到滚动区底部，用户改完设置要滚回最下面才能保存。

## Goal

- 同一个弹窗内信息分区清晰：左「代理服务器」、右「使用代理的提供商」，两栏并列，不再一条长列表。
- 提供商分流列表拥有自己的滚动区，行高与三态控件对齐成一列，视觉噪声下降但仍是「一眼可见、
  一次点击」的三态。
- 页脚（保存／取消）固定在弹窗底部，任何滚动位置都能保存；保存失败提示也随页脚常驻可见。
- 提供商数量增长时仍可用：支持按名称筛选，并显示三态计数摘要。
- 不改变任何后端接口、字段语义、三态取值与「全部走代理／全部直连」的批量语义。

## Background

`frontend/src/features/providers/proxy-settings-dialog.tsx` 由 TASK-20261026-001 引入，是单栏
`overflow-y-auto` 的 `Dialog.Content`（`w-[min(560px,...)]`、`p-6`、`space-y-6`）。固化的问题：

- 内容总量 ≈ 900px+，而弹窗可视高度 ≈ 640px，必须滚动；标题与页脚都会滚出视野。
- 提供商列表每行 `px-3 py-2` + `size="xs"` 三态按钮，13 行占 ~350px，且三态按钮在每行内
  随提供商名长度左右浮动，右边界不对齐。
- 字段标签统一 `text-sm font-medium`，每 60px 一条强对比标签，纵向密度高。

## Scope

- `ProxySettingsDialog` 的弹窗骨架：`flex flex-col` + 固定头部 + 可滚动主体 + 固定页脚。
- 主体改为两栏卡片（≥768px 两列，窄屏堆叠）：左「代理服务器」，右「使用代理的提供商」。
- 提供商列表：独立滚动区、行高加大、三态控件右对齐成列、按名称筛选、三态计数摘要。
- 「清除密码」从独立一行移入密码标签行，减少一行空白。
- 中英文新增文案（筛选占位、无匹配提示、计数摘要、未启用提示）。
- 更新 `proxy-settings-dialog.test.tsx` 覆盖筛选与摘要；补充文档。

## Out of Scope

- 不改 `proxy_settings` / `providers.proxy_mode` 的数据模型、迁移与 Go 侧逻辑。
- 不改「探测本机代理」「测试」「保存」「全部走代理／全部直连」的接口与语义。
- 不改提供商页其他组件（卡片、目录、列表）的样式。
- 不引入新的 UI 依赖或设计 token；沿用现有 `styles.css` 语义色与 `components/ui`。

## Existing Behavior

- 单栏 560px 弹窗，整块内容一起滚动；页脚按钮在内容末尾。
- 提供商三态用 `size="xs"` 的 ghost 按钮组，无筛选、无计数。
- 「清除密码」独占一行 checkbox。

## Expected Behavior

- 弹窗 880px（窄屏自适应），头部标题／说明固定，主体分区并可滚动，页脚常驻。
- 左栏所有代理服务器字段与操作在一屏内可见；检测结果与探测芯片在操作按钮下方就近显示。
- 右栏列出全部提供商，每行「名称 + 右对齐三态控件」；列表自身滚动，列表下方显示
  「共 N 个 · 走代理 a · 直连 b · 跟随全局 c」；输入筛选词后只显示匹配项并提示显示数量。
- 关闭按钮、Esc、取消、保存、失败提示的行为保持原样。

---

# Analysis

## Related Modules

- `frontend/src/features/providers/proxy-settings-dialog.tsx`（本任务唯一实现文件）
- `frontend/src/features/providers/proxy-settings-dialog.test.tsx`
- `frontend/src/features/providers/use-proxy-settings.ts`（复用，不改）
- `frontend/src/components/ui/{button,input,select-field,switch}.tsx`（复用，不改）
- `frontend/src/i18n/locales/{zh,en}.ts`
- `docs/product/NETWORK_PROXY.md`、`docs/plans/CURRENT.md`

## Dependencies

无新增依赖。图标沿用 `lucide-react`（`Globe`、`Search`、`X`）。

## Risks

- 弹窗内部出现三层可滚动容器会导致滚动冲突（skill §5 `scroll-behavior`）：本方案只用一层
  主体滚动 + 列表自身的独立滚动区（有明确边框边界），头部与页脚不滚动。
- 两栏在窄窗口下被压扁：以 `md:` 断点堆叠，弹窗宽度用 `min(880px, calc(100vw-2rem))`。
- 新增筛选输入框可能让既有测试的 `getByRole` 查询出现多个匹配：筛选框用 `searchbox` 语义
  （`type="search"`）而不是与现有按钮／标签重名。
- 三态按钮仍用 `aria-pressed`，既有 `getByRole("group", { name: "OpenAI的出站方式" })` 断言不变。

---

# Implementation Plan

1. 弹窗骨架：`Dialog.Content` 改 `flex flex-col overflow-hidden`，`rounded-2xl`、`w-[min(880px,…)]`；
   加载态／错误态保留原样但补内边距。
2. 表单内分三段：头部（标题＋说明＋关闭）、`app-scroll` 主体（两栏 grid）、页脚（失败提示＋取消／保存）。
3. 左栏卡片「代理服务器」：启用开关行（下方未启用提示）、协议、地址＋端口、用户名、密码
   （标签行右侧「清除密码」）、绕过列表、探测／测试按钮与结果。
4. 右栏卡片「使用代理的提供商」：标题行右侧「全部走代理／全部直连」，筛选输入，独立滚动列表，
   行内名称截断 + 右对齐三态控件，底部计数摘要。
5. 抽两个内部小部件：`SectionCard`（分区卡片）与 `ModeToggle`（三态控件），保持单文件。
6. i18n 新增 5 个键（中英同键）。
7. 测试：保留全部既有断言，新增筛选与摘要断言。
8. 文档：`docs/product/NETWORK_PROXY.md` 用户可见行为补一句布局与筛选说明；`CURRENT.md` 记一条。

---

# Progress

- 已确认拥挤的成因（单栏 900px 内容、列表行高 26px、页脚随内容滚动），并核对既有测试的
  全部可访问性查询点，避免改布局时打断断言。

---

# Implementation Result

## 实现摘要

`ProxySettingsDialog` 从「560px 单栏整块滚动」改为「880px 两栏分区 + 固定头部 + 固定页脚」。

弹窗骨架：

```text
Dialog.Content  flex max-h-[calc(100dvh-2rem)] w-[min(880px,calc(100vw-2rem))] flex-col
                overflow-hidden rounded-2xl border shadow-2xl
├── header      Globe 图标块 + Title + Description + ghost 关闭按钮（不滚动）
├── body        app-scroll min-h-0 flex-1 overflow-y-auto
│   └── grid gap-4 md:grid-cols-2
│       ├── SectionCard「代理服务器」
│       └── SectionCard「使用代理的提供商」
└── footer      bg-muted/20 border-t，左侧失败提示 + 右侧 取消 / 保存（不滚动）
```

左栏「代理服务器」字段顺序与语义完全保持：启用开关（关闭时下方给出「所有请求按直连发出」提示）
→ 分隔线 → 协议 → 服务器地址 + 端口（`grid-cols-[minmax(0,1fr)_6.5rem]`，端口错误就近内联）
→ 用户名 → 密码（值仍为占位提示，永不回显；「清除密码」并入标签行右侧）→ 不使用代理的地址
→ 分隔线 → 探测本机代理 / 测试 + 探测结果芯片 + 测试结论（`role="status"` / `role="alert"`）。

右栏「使用代理的提供商」：标题行右侧常驻「全部走代理 / 全部直连」；`type="search"` 名称筛选；
列表 `max-h-[min(22.5rem,50vh)]` 独立滚动（外层边框包裹，`aria-label` 分组，行 `px-2 py-1.5`，
名称 `truncate` + 右对齐三态控件成列）；列表下方「共 N 个提供商 · 走代理 a · 直连 b · 跟随全局 c」，
筛选时改为「显示 x / N 个提供商」。

内部抽两个同文件小部件：`SectionCard`（`aria-labelledby` 分区卡片）、`ModeToggle`
（`role="group"` + `Button size="xs" aria-pressed` 的三态分段控件）。字段标签降为
`text-muted-foreground text-xs font-medium` 的弱化 kicker，主控件仍 36px 高。

## 文件变更

| 文件 | 变更 |
| --- | --- |
| `frontend/src/features/providers/proxy-settings-dialog.tsx` | 重写弹窗布局（707 行级改动） |
| `frontend/src/features/providers/proxy-settings-dialog.test.tsx` | 新增 3 个用例（摘要、筛选、未启用提示） |
| `frontend/src/i18n/locales/zh.ts` | 新增 5 个键 |
| `frontend/src/i18n/locales/en.ts` | 同步 5 个同键英文文案 |
| `docs/product/NETWORK_PROXY.md` | 「用户可见行为」改为两栏描述 |
| `docs/plans/CURRENT.md` | In Progress 记一条 |
| `docs/tasks/active/TASK-20261025-015-*.md` | 本记录 |

## 技术要点

- **行为不变**：`getProxySettings` / `saveProxySettings` / `testProxySettings` /
  `discoverProxyPorts` / `setProviderProxyMode` 的调用点与参数未改；三态取值仍是
  `inherit|direct|proxy`；端口 1–65535 校验、`safeErrorDetail` 脱敏、保存成功即关闭 /
  失败留在弹窗内提示的行为保持。
- **布局约束**：全弹窗只有两层滚动（主体 + 列表自身），头部与页脚 `shrink-0`；
  `max-h` 用 `100dvh`，窗口高度不足时主体先让位。
- **三态控件对齐**：每行 `flex items-center justify-between`，控件固定贴右，13 行形成单一
  纵向列，不再随名称长度左右浮动。
- **规模**：筛选 + 计数摘要让提供商继续增长时仍可定位；筛选空结果与加载态各有一行文案。
- **响应式**：`md:` 以下两栏堆叠为单栏，弹窗宽度随视口收敛。
- **可访问性**：分区 `aria-labelledby`、开关独立 `aria-label`、筛选框 `searchbox` 语义、
  三态组 `aria-label="{{name}}的出站方式"`、错误/状态 `role="alert"|"status"` 全部保留。
- 无新增依赖，无新增设计 token。

## 测试

`proxy-settings-dialog.test.tsx` 由 9 个用例增至 12 个，新增：

1. 计数摘要文案 `共 3 个提供商 · 走代理 1 · 直连 1 · 跟随全局 1`。
2. 筛选：输入 `deep` 后摘要变为 `显示 1 / 3 个提供商`，OpenAI 分组隐藏、DeepSeek 分组可见；
   输入 `zzz` 显示 `没有匹配的提供商。`。
3. 关闭代理开关后出现提示 `代理未启用：所有请求按直连发出。`。

既有的 9 个用例（保存、取消、失败提示、密码不新增、端口校验、探测、测试、批量设置、
三态切换）未改动且全部通过。

---

# Verification

`task verify`（format:check → lint → typecheck → test → build）在仓库根目录执行：

```text
task: [format:check]           gofmt 无未格式化文件
task: [frontend:format:check]  All matched files use Prettier code style!
task: [lint]                   go vet / staticcheck / eslint 均无输出
task: [frontend:typecheck]     tsc --noEmit 通过
task: [test]                   go test ./... 通过
task: [frontend:test]          Test Files 132 passed (132) · Tests 809 passed (809)
task: [windows:build:native]   bin/omniproxy.exe 产出成功
EXIT=0
```

另做人工视觉核对：用 `tmp/ui-preview/` 的独立预览页（真实组件 + mock 服务）截图确认深色 /
浅色 / 窄屏（760px）三种形态，两栏在窄屏正确堆叠、页脚始终可见、保存不被列表顶出视口。

---

# Documentation

- `docs/product/NETWORK_PROXY.md`：用户可见行为改写为两栏布局 + 筛选 + 固定页脚描述。
- `docs/plans/CURRENT.md`：`## In Progress` 增加本任务链接。
- 接口与数据模型无变更，`AGENTS.md` 与其他架构文档无需改动。

---

# Git

## Commit

```text
3d831c8 feat(frontend): 网络代理弹窗改为两栏分区与固定页脚
```

工作区改动仅限本任务涉及的 6 个文件 + 本记录，无夹带无关文件；未打标签、未推送。

---

# Result

达成 Goal 全部条目：

- 拥挤问题解决：内容由「一条 900px 长列表」拆成两张有边框的分区卡片，同屏可见。
- 保存／取消固定在页脚，滚动到任何位置都能保存，失败提示常驻。
- 提供商列表行高加大，三态控件右对齐成列，配合筛选与计数摘要。
- 后端接口、数据模型、三态语义、脱敏行为零改动；本文件测试 12/12 通过，全量 809 通过。

已随提交 `3d831c8` 落地，并在 Beta.44 本地生产安装包中交付。

---

# Completed

2026-10-25 22:28（Asia/Shanghai）
