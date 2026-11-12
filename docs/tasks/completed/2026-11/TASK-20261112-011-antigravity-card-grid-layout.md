# TASK-20261112-011: Antigravity 卡片改为与其它客户端一致的网格卡片

## Status

In Progress

## Created

2026-11-12 20:24（Asia/Singapore）

## Request

用户覆盖安装 `v2.2.3-beta.11` 前查看网关页「客户端接入」，指出 Antigravity 卡片占满整行、
版式与下方客户端卡片不一致：

> 这个UI不至于这样吧，就正常的就行，和下面的卡片一样

即：Antigravity 应作为 `agent-client-grid` 里的一张普通卡片，与 Codex、Claude Code、
OpenCode、DeepSeek Harness、Kilo Code 等分组卡片同尺寸、同结构。

## Goal

- Antigravity 卡片进入 `agent-client-grid` 网格，宽度与相邻分组卡片一致，不再独占整行。
- 版式对齐 `AgentClientGroupTile`：品牌图标 + 分组标题 + 描述 + 单条客户端块（标题、
  状态徽标、安装状态）+ 分隔线 + 整宽操作按钮。
- 保留既有行为：连接预览、默认模型选择、应用、断开并恢复、设置文件路径、错误与成功提示。

## Background

- TASK-20261112-009 新增 `AntigravityClientCard`，由 `LocalAgents` 直接渲染在分组网格
  之外，因此整行拉伸、内部信息按宽卡片排布，与下方 5 张分组卡片视觉不一致。
- Antigravity 与其它客户端不同：没有安装目标（`agent-installation-targets`）、没有详情页
  （`/gateway?tab=clients&client=…` 只覆盖已注册客户端），所以不能直接复用
  `AgentClientGroupTile` 与 `AgentClientTile`（它们是链接，且依赖安装快照）。

## Scope

- `frontend/src/features/gateway/antigravity-client-card.tsx`：改为分组卡片结构。
- `frontend/src/features/gateway/local-agents.tsx`：把卡片移入 `agent-client-grid`。
- `frontend/src/i18n/locales/agents-{zh,en}.ts`：新增分组标题、精简提示文案。
- `frontend/src/features/gateway/local-agents.test.tsx`：补充网格内呈现与交互断言。

## Out of Scope

- 不改后端 `agentconfig`、模型目录、网关路由或数据。
- 不为 Antigravity 新增安装检测目标、安装器或详情页。
- 不修改其它客户端卡片。
- 不重新打包、不安装、不推送。

## Existing Behavior

`LocalAgents` 在标题下先渲染 `AntigravityClientCard`（`Card` 组件、占满容器宽度），
再渲染 `agent-client-grid` 网格中的 5 张分组卡片。

## Expected Behavior

Antigravity 作为网格中的一张分组卡片：图标与标题为「Antigravity」，描述沿用现有文案，
卡片内一条「Antigravity 桌面端」条目带状态徽标与「已安装／未安装」，底部整宽按钮按
未配置／选择模型／已连接三种状态显示「连接 Antigravity」「应用连接」「断开并恢复」。

---

# Analysis

## Related Modules

- `frontend/src/features/gateway/local-agents.tsx`
- `frontend/src/features/gateway/antigravity-client-card.tsx`
- `frontend/src/features/gateway/agent-client-group-tile.tsx`（外观参照）
- `frontend/src/features/gateway/agent-client-grid.css`
- `frontend/src/i18n/locales/agents-{zh,en}.ts`

## Dependencies

无新增依赖。沿用 `services/agents` 的 `previewAgentPlan`／`applyAgentConfig`／
`restoreAgentConfig` 与既有图标资源 `provider-icons/google-color.svg`。

## Risks

- 网格用 `minmax(min(100%, 22rem), 1fr)`，窄窗口下卡片会变单列；卡片内部需保持
  `min-w-0` 与可换行，避免长路径与长文案撑破。
- 现有测试按按钮名与 `combobox` 定位，重构时保持同样的可访问名称，避免无意义改测试。
- 状态徽标与安装状态两处都可能显示「未配置／未安装」，措辞需区分配置与安装。

---

# Implementation Plan

1. 创建本记录。
2. 重写 `AntigravityClientCard` 为分组卡片结构（`section` + 标题 + 单条客户端块 +
   底部整宽按钮），保留全部交互与提示。
3. `LocalAgents` 中把卡片放入 `agent-client-grid`，移除网格外的整行渲染。
4. 补充 i18n：分组标题与精简提示；同步英文。
5. 更新测试：断言卡片位于网格内、状态与按钮按状态切换。
6. 运行前端 typecheck／lint／format 与相关测试，必要时全量。
7. 填写结果，归档并提交。

---

# Progress

- 2026-11-12 20:24：创建记录；确认问题来自 `LocalAgents` 在网格外单独渲染整行卡片。
- 重写卡片为分组卡片结构并移入 `agent-client-grid`；补分组标题与精简提示文案。
- 新增 2 项测试；前端 15 项网关卡片测试、`src/features/gateway` 30 文件／200 项通过。

---

# Implementation Result

## Changes

- `antigravity-client-card.tsx`：`AntigravityClientCard` 改为 `AntigravityGroupTile`，
  外观与 `AgentClientGroupTile` 同构——`bg-card` 圆角边框、`size-14` 品牌图标、
  分组标题（「Antigravity」）与描述、单条客户端块（「Antigravity 桌面端」+ 状态徽标 +
  已安装／未安装）、底部分隔线与整宽 `size="sm"` 按钮。
- 状态徽标改用 Antigravity 自己的语义：已写入网关配置时显示「已连接本地网关」，
  配置异常显示「配置需要检查」，否则「尚未配置」；不再复用「接入 Key 已启用」。
- 三条操作路径全部收在卡片底部：未配置「连接 Antigravity」、预览态「应用连接／关闭」、
  已连接「断开并恢复」；模型选择在客户端块内，保留原有 `previewAgentPlan` →
  `applyAgentConfig` → `restoreAgentConfig` 调用与 revision／catalogRevision 参数。
- 设置文件路径、错误提示、保存成功提示与「不使用 Google 登录」说明保留在卡片内，
  长文案用 `text-xs leading-5` 与 `wrap-anywhere` 收在 22rem 列宽内。
- `local-agents.tsx`：卡片移入 `agent-client-grid` 作为第一个子项，不再独占整行。
- i18n：新增 `agents.antigravityGroup`；`antigravityHint` 精简为一行
  （Gemini、Claude、GPT-OSS 全部订阅模型 + 不使用 Google 登录）；
  `antigravityGoogleOff` 精简，不再逐条列出斜杠命令。

## Not Changed

- 后端 `agentconfig`、模型目录、网关路由、备份与恢复逻辑。
- 其它客户端分组卡片、安装检测目标与详情页。
- 请求链路：桌面端仍复用现有 Google 订阅多账号路由。

---

# Verification

## Tests

- [x] Unit Tests：前端全量 147 文件／898 项通过（新增 2 项）。
- [ ] Integration Tests：N/A
- [ ] E2E Tests：N/A（原生窗口观感由用户装后确认）

## Quality Checks

- [x] Format：`pnpm exec prettier --check` 通过（改动涉及 3 个文件已 `--write`）。
- [x] Lint：`pnpm exec eslint` 三个改动文件无输出。
- [x] Type Check：`pnpm exec tsc --noEmit -p tsconfig.json` 通过。
- [x] Test：`pnpm test -- --maxWorkers=4` 147 文件／898 项通过。
- [x] Build：`task verify` 内的 `task build` exit 0（Windows production 编译）。
- [x] Verify：`task --force verify` exit 0（format check → lint → typecheck → test → build）。

## Verification Result

```text
pnpm exec tsc --noEmit -p tsconfig.json                  → exit 0
pnpm exec eslint <3 个改动文件>                          → exit 0（无输出）
pnpm exec prettier --check <5 个改动文件>                → All matched files use Prettier code style
pnpm exec vitest run src/features/gateway --maxWorkers=4 → 30 文件／200 项通过
task format:check                                        → exit 0
task lint                                                → exit 0（仅 2 条既有 react-refresh 警告）
task typecheck                                           → exit 0
task test                                                → exit 0（147 文件／898 项）
task --force verify                                      → exit 0（VERIFY_EXIT=0）
```

- 新增测试断言卡片位于 `agent-client-grid` 内（`parentElement` 含该类）、标题层级为
  `h4`、已安装／未安装与三种按钮状态，以及缺少设置文件时「连接 Antigravity」禁用。
- 布局由类名同构保证（与 `AgentClientGroupTile` 相同的 `section`／`header`／条目／
  `footer` 结构与 `p-6`／`gap-5`／`rounded-xl border`），网格为
  `repeat(auto-fit, minmax(min(100%, 22rem), 1fr))`，窗口变窄时自动减少列数。
- 原生 WebView2 观感待用户覆盖安装后确认。

---

# Documentation

- [x] CURRENT：已更新为卡片改版
- [x] PRD／Architecture／Tech Stack／Coding Rules／Testing／Build／ADR：无需修改

---

# Git

## Commit

```text
b2e501bb fix(agents): Antigravity 卡片改为与其它客户端一致的网格卡片
```

本记录随实现提交一起提交；归档提交 `bd1e81ea`。未打标签、未推送。

---

# Result

Antigravity 卡片改为与 Codex、Claude Code 等一致的网格分组卡片：图标 + 标题 + 描述 +
一条客户端条目 + 底部分隔线与整宽按钮，宽度随网格列变化，不再独占整行。连接、选择默认
模型、应用、断开并恢复的行为与文案边界不变；前端全量与 `task verify` 通过。

## Deviation

- 首次以 `task --force verify 2>&1 | Select-Object -Last 40` 运行得到 exit 1，单独重跑
  `task format:check`／`lint`／`typecheck`／`test`／`build`，以及
  `task --force verify *> tmp/verify.log` 均为 exit 0；差异来自 PowerShell 管道与 stderr
  的退出码传播，不是质量门失败。

---

# Completed

2026-11-12 20:35
