# TASK-20261022-006: 用量列表滚动条样式与本地 Agent 分组一键配置

## Status

In Progress

## Created

2026-10-22 20:45

## Request

用户提供两张截图并指出两点需要优化：

1. 图一（用量页「按账号成员」／「按模型」两张卡片）：滚动条很丑，与当前软件设计不搭配，需要优化 UI。
2. 图二（网关 → 客户端接入 → 本地 Agent 接入）：需要在这个外部每个卡片上增加一个「一键配置」按钮，点击后直接把所有模型用默认配置写好，不需要手动选择模型。

## Goal

- 用量页两张分布卡片的纵向滚动条改为与产品视觉一致的细圆角滑块，去掉 Windows 原生轨道与箭头。
- 本地 Agent 目录页每个分组卡片提供「一键配置」按钮，一次为该分组内全部客户端写入完整默认模型目录，无需进入详情手动选择。

## Background

- 主内容区与侧栏使用 `components/common/scroll-region.tsx`（隐藏原生滚动条 + 自绘轨道），而用量页分布列表仍用 `overflow-auto`，因此显示 Windows 原生滚动条。
- 已有的详情页「一键配置」（`agents.configure`）只打开配置编辑器并预选全部模型，仍需用户点击「应用配置」；后端 `agentconfig.Service.Apply` 在未提供 `ModelIDs` 时写入完整目录，因此分组级一键配置可以复用同一契约。

## Scope

- `frontend/src/styles.css`：新增细圆角滚动条工具类。
- `frontend/src/features/activity/usage-breakdown.tsx`：用量分布列表改用该工具类。
- `frontend/src/features/gateway/agent-client-group-tile.tsx`：分组卡片底部增加「一键配置」按钮与结果反馈。
- 新增 `frontend/src/features/gateway/use-agent-group-config.ts`：按分组内去重客户端依次预览并应用完整模型目录。
- `frontend/src/features/gateway/local-agents.tsx`：把可用模型列表传入分组卡片。
- 中英文案 `frontend/src/i18n/locales/agents-{zh,en}.ts`。
- 新增回归测试。

## Out of Scope

- 不改后端 agentconfig 应用/回滚逻辑与协议。
- 不改详情页既有「一键配置 → 预览 → 应用」流程。
- 不改其他页面（请求详情、旧版导入、设置弹窗）的原生滚动条。

## Existing Behavior

- 用量分布列表使用浏览器原生滚动条。
- 目录页只能进入单个客户端详情后手动配置模型。

## Expected Behavior

- 用量分布列表显示 4px 圆角细滑块、无轨道、无箭头，深浅主题一致，滚动行为不变。
- 每个分组卡片底部有「一键配置」按钮：为分组内每个客户端写入完整默认模型目录（无需选择模型），逐项反馈进度，失败时列出具体客户端与原因，成功后刷新客户端与模型查询。

---

# Analysis

## Related Modules

- 前端 UI：`features/activity`、`features/gateway`、`components/ui`、`styles.css`
- 绑定：`services/agents`（`AgentPreviewPlan` / `AgentApply`）
- 后端契约（只读确认）：`internal/application/agentconfig/apply.go`

## Dependencies

无新增依赖。

## Risks

- 分组内客户端共享配置文件（Codex CLI/Desktop、OpenCode CLI/Desktop 等）需按 client 去重，避免重复写入与重复备份。
- 分组一键配置会直接写客户端配置，需依赖既有加密备份与「恢复原配置」；失败必须逐项可见，不能静默。
- 目录页既有测试断言「目录页没有名为『一键配置』的按钮」，新按钮需用带分组名的可访问名称区分。

---

# Implementation Plan

1. `styles.css` 增加 `.app-scroll` 细滑块工具类（Chromium 伪元素，`scrollbar-gutter: stable`）。
2. `usage-breakdown.tsx` 的 `max-h-96 overflow-auto` 容器改用 `.app-scroll`。
3. 新增 `use-agent-group-config.ts`：去重客户端 → `previewAgentPlan` → `applyAgentConfig`（完整目录）→ 逐项结果与查询刷新。
4. `agent-client-group-tile.tsx` 底部增加按钮与状态区域（可访问名称含分组名）。
5. `local-agents.tsx` 传入 `models`。
6. 补中英文案与回归测试。
7. `task verify`。

---

# Progress

- 2026-10-22 20:45 建立任务记录，确认滚动条现状与后端「未指定 ModelIDs 即完整目录」契约。
- 2026-10-22 21:00 完成 `.app-scroll` 与分组一键配置实现，新增 4 项回归测试；前端 122 文件 / 728 项通过。
- 2026-10-22 21:10 `task verify` exit 0（含 Go 测试与 Windows production 构建），前端绑定重新生成无差异。

---

# Implementation Result

## Changes

已实现：

- `styles.css` 新增 `.app-scroll`：10px 滚动槽内只画 4px 圆角滑块（`border: 3px solid transparent` +
  `background-clip: content-box`），轨道与两端箭头透明，`background-color` 由
  `--muted-foreground` 与 `transparent` 的 `color-mix` 得到，悬停加深；`scrollbar-gutter: stable`
  预留滚动槽，避免列表出现滚动条时内容横向跳动。深浅主题共用语义色。
- 用量页「按账号成员」「按模型」两张卡片的滚动容器改用 `.app-scroll`，滚动行为仍为原生。
- 新增 `use-agent-group-config.ts`：按 `group.entries` 去重客户端，逐个
  `previewAgentPlan({client, revision})` → `applyAgentConfig({client, revision, catalogRevision})`，
  不传 `modelIds`，由后端写入完整默认目录；单项失败不中断，记录 `{client, error}`，结束后刷新
  `agent-clients` 与 `agent-configured-models` 查询。
- `agent-client-group-tile.tsx` 底部新增卡片级「一键配置」按钮：读取中／配置中／无可用目录时禁用，
  配置中显示进度，完成后显示成功数量与失败清单（客户端名 + `agents.error.*` 原因）；
  无可用模型路由时显示「暂无可用模型路由」。可访问名称带分组名（「一键配置 Codex 全部模型」），
  与详情页同名按钮区分。
- `local-agents.tsx` 把网关可用模型列表传入分组卡片。
- 中英文案各新增 5 个 `agents.groupConfigure*` / `agents.groupConfigured` /
  `agents.groupConfigureFailedItem` / `agents.groupConfigureUnavailable` 键。

## Files Changed

- `frontend/src/styles.css`
- `frontend/src/features/activity/usage-breakdown.tsx`
- `frontend/src/features/gateway/agent-client-group-tile.tsx`
- `frontend/src/features/gateway/use-agent-group-config.ts`（新增）
- `frontend/src/features/gateway/local-agents.tsx`
- `frontend/src/features/gateway/agent-group-configure.test.tsx`（新增）
- `frontend/src/i18n/locales/agents-zh.ts`、`frontend/src/i18n/locales/agents-en.ts`
- `docs/product/ACTIVITY.md`、`docs/product/LOCAL_AGENTS.md`、`docs/plans/CURRENT.md`

## Technical Notes

- Chromium（WebView2）下 `scrollbar-color` 会覆盖 `::-webkit-scrollbar` 自定义样式，因此这里只用
  webkit 伪元素实现圆角滑块，未设置标准属性。
- 分组内 Codex／OpenCode／Kilo／Qwen／Goose／MiMo 的 CLI 与 Desktop 共享同一 client，去重后只写一次，
  避免重复备份与重复写入。
- 乐观并发仍由后端 `revision` / `catalogRevision` 校验；目录页数据过期时单项会以
  `config_changed` / `catalog_changed` 失败并显示原因，不静默重试。

---

# Verification

## Tests

- [x] Unit Tests（`agent-group-configure.test.tsx` 4 项：完整目录写入、共享配置只写一次、单项失败保留其余结果并列出原因、无可用模型时禁用）
- [x] Integration Tests（前端 122 个文件 / 728 项测试全部通过）
- [ ] E2E Tests（N/A，无 E2E 框架）

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
PASS（format:check → lint → typecheck → test → build，exit 0；前端 122 文件 / 728 项测试通过）
```

视觉核对（非原生窗口）：

- 用量卡片：以真实构建产物 CSS + jsdom 渲染出的真实 DOM 复现两张卡片，滚动条为 4px 浅色圆角滑块，
  无轨道与箭头（对照图：原生滚动条带灰色轨道与上下箭头）。
- 本地 Agent 目录：同样方式渲染全部 11 张分组卡片，底部「一键配置」按钮为整宽 outline 按钮 +
  图标，等宽对齐、与卡片内边距一致。
- 原生 Wails 窗口视觉验收与真实客户端写入未在本任务内执行，待用户确认。

---

# Documentation

- [ ] PRD
- [ ] Architecture
- [ ] Tech Stack
- [ ] Coding Rules
- [ ] Testing
- [ ] Build
- [ ] ADR
- [x] CURRENT

更新：`docs/product/ACTIVITY.md`（用量卡片滚动条）、`docs/product/LOCAL_AGENTS.md`（分组卡片一键配置）、
`docs/plans/CURRENT.md`（本任务条目）。

---

# Deviations

- 原计划在分组卡片标题右侧放置按钮；实际改为卡片底部整宽按钮。原因：卡片最小宽度 22rem，
  标题区在 140% 字号下会让描述过度换行，底部操作区更稳定且与卡片底边对齐。
- 原计划失败清单使用 `agentTitle()` 名称；实际改用 `agentClients[client].title`，与详情页标题同源。
- 原计划无可用模型时复用 `agents.noModels` 长提示；实际新增短文案「暂无可用模型路由」，
  避免 11 张卡片重复两行长句。

---

# Git

## Commit

`a5bf9bc` — feat(gateway): 用量卡片细圆角滚动条与本地 Agent 分组一键配置（10 个文件；含 docs/product/LOCAL_AGENTS.md 与 docs/tasks/active/ 本记录）。同任务对 docs/product/ACTIVITY.md 的滚动条说明此前随 `211a98f` 一并提交，docs/plans/CURRENT.md 条目随 `65d81d3` 提交。验收状态保持 In Progress：原生窗口视觉与真实客户端写入待用户确认。

---

# Result

用量页两张分布卡片的原生滚动条已替换为细圆角滑块；本地 Agent 目录中每张分组卡片新增「一键配置」，
一次性为该分组全部客户端写入完整默认模型目录，逐项反馈结果并刷新接入状态。完整质量门通过，
原生窗口与真实客户端写入待用户验收。

---

# Completed

（待完成）
