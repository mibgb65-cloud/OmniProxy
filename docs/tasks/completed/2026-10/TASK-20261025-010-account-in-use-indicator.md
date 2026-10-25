# TASK-20261025-010: 账号卡片的「正在使用」状态指示替代闪烁提示

## Status

Completed

## Created

2026-10-25 12:35（Asia/Singapore）

## Request

用户指出：River 账号卡片在请求进行中时，底部那行提示文字
「包含进行中或用量未完整返回的请求，当前数值仅为已记录用量。」会一直闪烁（反复出现／消失，卡片高度跟着跳动）。
要求去掉闪烁，改为在 API Key 与「已启用」徽标之间增加状态显示：

> 如果有正在使用，则显示一个那种加载中的圆环转圈的动画，鼠标悬浮时，显示提示框：该apikey正在使用中

## Goal

请求进行中时，卡片头部的固定位置出现旋转圆环，悬浮提示「该 API Key 正在使用中」；提示文字不再出现，卡片高度稳定，不产生闪烁或跳动。

## Background

- `frontend/src/features/accounts/account-usage-panel.tsx` 在本机用量下方按 `usage.incompleteRequests > 0` 渲染一行说明；请求开始（记录 `state='pending'`）时出现、结束（写入 `usage_complete=1`）时消失，配合 `activity:updated` 事件反复刷新，形成闪烁并改变卡片高度。
- `incompleteRequests` 的语义是「历史上用量未完整或被中断的记录数」，并不等于「此刻正在使用」。账号 `rv_1fjfj` 已有历史不完整记录时该行会长期存在，因此不能直接用它驱动「正在使用」指示。
- 现有用量接口 `AccountUsage.AccountUsage` 已按账号身份聚合 `request_records`，可顺带给出 `state='pending'` 的条数作为实时信号。

## Scope

- Go：`AccountTokenUsage` 增加 `pendingRequests`（`state='pending'` 记录数），复用现有聚合查询；更新后端测试与 Wails 绑定。
- 前端：抽出共享用量查询 Hook；移除闪烁说明行；新增「正在使用」圆环指示组件（固定占位、悬浮提示、短暂保持避免抖动）。
- 卡片视图在 API Key 与「已启用」之间显示指示；表格视图状态列同步显示同一指示。
- 更新中英文案、测试与产品文档。

## Out of Scope

- 不改 `incompleteRequests` 的语义，也不删除该字段（仍由接口返回）。
- 不改网关记账、事件节流（`activity:updated` 500ms 合并）与用量统计口径。
- 配额页等其它使用同一用量面板的页面只跟随面板变化，不新增指示。

## Existing Behavior

- 请求进行中：Token 用量下方出现「包含进行中或用量未完整返回的请求…」一行，请求结束即消失；每次事件刷新都可能重新出现。
- 卡片高度随该行出现／消失变化，网格中的卡片位置跟着跳动，用户看到「一直闪烁」。

## Expected Behavior

- 卡片头部的 API Key 行右侧固定保留一个 16px 槽位，「正在使用」时显示旋转圆环，其余时间留空，徽标位置不变。
- 圆环悬浮／键盘聚焦时显示产品统一 Tooltip：「该 API Key 正在使用中」。
- 判断依据为进行中的请求（`pendingRequests > 0`）；观测到进行中后至少保持 1.2 秒，避免连续请求之间闪断。
- 说明文字不再渲染，卡片高度稳定；Token 数值与 `≥` 前缀规则不变。

---

# Analysis

## Related Modules

- `internal/application/activity/account_token_usage.go`（聚合查询与模型）
- `frontend/bindings/omniproxy/internal/application/activity/models.ts`（生成绑定）
- `frontend/src/features/accounts/account-usage-panel.tsx`、`account-details-card.tsx`、`account-details-list.tsx`
- `frontend/src/i18n/locales/{zh,en}.ts`、`activity-{zh,en}.ts`

## Dependencies

- 复用现有 TanStack Query、`Tooltip`、lucide／CSS 动画；无新依赖。

## Risks

- Low–Medium：新增一个后端字段需要重新生成 Wails 绑定；字段为纯增量，旧前端行为不受影响。
- 若 `pendingRequests` 因进程异常遗留 `pending` 记录，圆环可能长时间显示；`recovery.go` 已有把陈旧 pending 记为 cancelled 的机制，沿用即可。

---

# Implementation Plan

1. Go：`AccountTokenUsage` 增加 `PendingRequests`，SQL 增加 `SUM(CASE WHEN state='pending' THEN 1 ELSE 0 END)`，更新注释与测试。
2. 重新生成 Wails 绑定（`task build` 的 `generate:bindings` 依赖）。
3. 前端新增 `useAccountTokenUsage`，面板与指示组件共用同一查询键（含 `activity:updated`／可见性刷新）。
4. 新增 `AccountInUseIndicator`：固定 `size-4` 槽位 + 旋转圆环 + Tooltip + 1.2 秒保持。
5. 卡片与表格视图接入指示；删除面板中的闪烁说明行与对应文案。
6. 测试：面板不再渲染说明、指示在 `pendingRequests>0` 时出现并保持、卡片布局不产生说明行；更新 Go 测试断言。
7. 文档：更新 `docs/product/APPEARANCE.md`（如涉及）、`docs/plans/CURRENT.md`。

---

# Progress

- Go 字段与 SQL、注释完成，绑定重新生成。
- 共享 Hook 与圆环组件完成，卡片／列表接入完成。
- 说明行与对应文案删除，产品文档同步。
- 测试补齐（含悬浮 Tooltip 用例），全量质量门运行中。

---

# Implementation Result

已实现：后端在 `AccountTokenUsage` 中新增 `pendingRequests`（`state='pending'` 的记录数），前端抽出共享用量查询并在卡片／列表头部渲染固定槽位的「正在使用」圆环，移除会改变卡片高度的说明行。

## Changes

- Go `internal/application/activity/account_token_usage.go`：`AccountTokenUsage` 增加 `PendingRequests int64`（`json:"pendingRequests"`），聚合 SQL 增加 `COALESCE(SUM(CASE WHEN state='pending' THEN 1 ELSE 0 END),0)`；注释明确「历史未完整记录不算进行中」。`incompleteRequests` 与既有统计口径未改动。
- 重新生成 Wails 绑定（`frontend/bindings/.../activity/models.ts` 出现 `pendingRequests`）。
- 新增 `useAccountTokenUsage`：把原来面板内的 TanStack Query（键 `["account-token-usage", account.id, account.updatedAt]`、`staleTime` 30s、`retry:false`）与 `activity:updated`／`visibilitychange` 刷新抽出，面板与圆环共用同一个查询，避免重复请求；支持 `enabled` 以跳过不需要的账号。
- 新增 `AccountInUseIndicator`：固定 `size-4` 槽位（`shrink-0`）、`size-3.5` 旋转圆环（`border-primary/25` + `border-t-primary`、`animate-spin`、`motion-reduce:animate-none`）、`role="status"` + `aria-label` 与产品统一 Tooltip；仅在 `pending` 时渲染圆环，观测到进行中立即显示、消失延后 1200ms（连续请求之间不闪断）。只有 `authMethod !== "subscription_login"` 且 `providerId === "builtin_river"` 的账号参与（其余账号没有本机用量面板）。
- `account-details-card.tsx`：API Key 预览与「已启用」徽标之间的右侧簇改为 `gap-2` 并插入圆环；`account-details-list.tsx`：状态列同样前置圆环。
- `account-usage-panel.tsx`：删除 `incompleteRequests` 说明行，改用共享 Hook；加载／失败／重试行为不变。
- i18n：新增 `accounts.inUse`（`该 API Key 正在使用中` / `This API key is in use`），删除不再使用的 `activity.accountUsageIncomplete`。

## Files Changed

- `internal/application/activity/account_token_usage.go`
- `internal/application/activity/account_token_usage_test.go`
- `internal/application/activity/account_usage_test.go`
- `frontend/bindings/omniproxy/internal/application/activity/models.ts`
- `frontend/src/features/accounts/use-account-token-usage.ts`（新增）
- `frontend/src/features/accounts/account-in-use-indicator.tsx`（新增）
- `frontend/src/features/accounts/account-usage-panel.tsx`
- `frontend/src/features/accounts/account-details-card.tsx`
- `frontend/src/features/accounts/account-details-list.tsx`
- `frontend/src/features/accounts/account-usage-panel.test.tsx`
- `frontend/src/features/accounts/account-token-summary.test.tsx`
- `frontend/src/features/accounts/provider-accounts-view.test.tsx`
- `frontend/src/features/accounts/subscription-quota-usd.test.tsx`
- `frontend/src/features/quota/quota-test-helpers.tsx`
- `frontend/src/i18n/locales/zh.ts`、`en.ts`、`activity-zh.ts`、`activity-en.ts`
- `docs/product/RIVER.md`、`docs/product/ACTIVITY.md`、`docs/plans/CURRENT.md`

## Technical Notes

- 闪烁根因不是渲染频率，而是信号选错：`incompleteRequests` 统计历史上未完整／被中断的记录，可能长期为真，而说明行随 `activity:updated` 刷新反复出现／消失并改变卡片高度。新信号只认 `state='pending'`，与「此刻正在使用」语义一致。
- 保持 1.2 秒用单个 `setTimeout` 实现（显示用 0ms、隐藏用 `HOLD_MS`），避免 `react-hooks/set-state-in-effect` 直接同步 `setState`。
- 流式响应在结束前一直保持 `state='pending'`，因此圆环覆盖整个流式过程；这符合「正在使用」的含义。
- 进程异常遗留的陈旧 pending 仍由既有 `Recover()`（10 分钟）收口，圆环不会永久悬挂。
- `pendingRequests` 为纯增量字段，旧前端忽略它即可；`incompleteRequests` 仍由接口返回，供其它视图继续使用。

---

# Verification

## Tests

- [x] Unit Tests（Go：`pendingRequests` 聚合；前端：圆环出现／保持／隐藏、非 River 账号不参与、说明行不再渲染、悬浮 Tooltip 文案）
- [x] Integration Tests（既有 `internal/transport/wails` 事件与绑定测试随全量包通过）
- [ ] E2E Tests（仓库无 E2E 套件）

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

- `task --force verify` exit 0，日志 `tmp/ui-verify-20261025/verify-010.log`：Prettier／gofmt 通过、ESLint／go vet／staticcheck 通过、TypeScript 类型检查通过、Go 全包测试通过、前端 129 文件／785 项测试通过、Wails 绑定重新生成（107 models）、Windows production 构建成功。
- 定向验证：`npx tsc --noEmit`、`npx eslint .`、`pnpm exec prettier --check .`、`npx vitest run src/features/accounts src/features/quota`（38 文件／214 项）、`go test ./internal/application/activity/... ./internal/transport/wails/...` 全部通过。
- 原生窗口视觉核对（隔离数据目录 `tmp/ui-verify-20261025/data`，1200×800 原生窗口，`OMNIPROXY_DATA_DIR` 指向隔离目录，未触碰用户数据）：截图 `r6-3.png` 显示圆环出现在 API Key 与「已启用」之间、说明行消失、Token 数值（1.26B／2.81M／≥ 1.21B）不受影响；`r6-hover.png` 显示悬浮提示「该 API Key 正在使用中」。
- 未覆盖：浅色主题与 80%／140% 字号下的圆环与 Tooltip 位置待用户视觉验收；未在真实 River 账号上跑真实请求，使用隔离数据库中 `state='pending'` 记录驱动。

---

# Documentation

- [x] APPEARANCE（无需改动：文档未描述该说明行）
- [ ] PRD
- [ ] Architecture
- [ ] ADR
- [x] CURRENT
- [x] RIVER（账号卡片 Token 用量段落改为描述「正在使用」圆环）
- [x] ACTIVITY（账号本机用量段落区分 `incompleteRequests` 与 `pendingRequests`）

---

# Deviations

- 计划里的文档目标是 `APPEARANCE.md`／`CURRENT.md`；实际说明行写在 `RIVER.md` 与 `ACTIVITY.md`，因此改这两处，`APPEARANCE.md` 无对应描述未改。
- 额外增加一条悬浮用例（`userEvent.hover` → `role="tooltip"` 内容），让 Tooltip 文案在测试中直接可证，而不是只靠截图。
- `account-details-list.tsx`（表格视图）也接入了圆环，保持与卡片视图一致；计划中只写了卡片，但两者共用同一行级状态语义，缺失会造成不一致。

---

# Git

## Commit

- `a3b288fc8c93b5c681554f50e42486000038895a` fix(frontend): 账号卡片改用「正在使用」圆环替代闪烁提示（22 文件，+369/−44）

---

# Result

- 请求进行中时，River 账号卡片与列表的状态列出现旋转圆环，悬浮（或键盘聚焦）提示「该 API Key 正在使用中」；说明行与 `activity.accountUsageIncomplete` 文案删除，卡片高度不再随请求变化，用户报告的闪烁消失。
- 「正在使用」由 `pendingRequests`（`state='pending'`）驱动，历史未完整记录不再误报；消失延后 1.2 秒避免连续请求闪断。
- 影响面限于账号卡片／列表与本机用量面板；Token 数值口径、`incompleteRequests` 语义、配额页展示均未改变。
- 质量门 exit 0（129 文件／785 项前端测试），原生窗口圆环与悬浮提示截图核对通过。

---

# Completed

2026-10-25 12:52（Asia/Singapore）
