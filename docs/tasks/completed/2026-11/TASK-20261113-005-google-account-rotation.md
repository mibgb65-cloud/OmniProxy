# TASK-20261113-005: Google 订阅账号的同池轮换方式

## Status

Completed

## Created

2026-11-13 09:30（Asia/Singapore）

## Request

> 谷歌这边也需要类似 OpenAI 的这些功能，你看看怎么加入

（截图中的 OpenAI 账号设置包含「Codex WebSocket」「自动激活 Codex 额度窗口」「账号轮换方式」三项。）

## Goal

- Google 账号页提供与 OpenAI／River 一致的「Google 账号设置」面板，先落地**账号轮换方式**
  （队列轮换／平衡切换）。
- 只对 Google AI 订阅账号生效；API Key 账号不参与重排，固定账号不切换。
- 设置持久化并可热切换，不需要重启网关。

## Background

- 现有模式：`CodexSettingsService`（`codex_settings` 表）给 Codex 提供
  WebSocket／自动激活／轮换三项；`RiverSettingsService`（`river_settings` 表）只提供轮换。
- `rotateAccounts` 原先只处理 `openai` 与 `river`，Google 订阅请求直接沿用候选顺序。
- 截图中的另外两项在本轮的判断：
  - **WebSocket**：Google 订阅上游只有 HTTP／SSE（`language_server` 网关与
    `streamGenerateContent` 都是 HTTP），没有可对接的 WebSocket 传输，增加同名开关只会
    变成无效设置，故不实现。
  - **自动激活额度窗口**：技术上可行（Google 也有 5h／周窗口），但会给账号发送真实消息、
    消耗额度，属于需要用户明确确认的产品决策，单独排期（见 Result）。

## Scope

- 迁移 `047_google_account_rotation.sql`：`google_settings`（单行，默认 `queue`）。
- `internal/application/gateway/google_settings.go`：读写与校验。
- `internal/application/gateway/google_rotation.go` + `account_rotation.go`：Google 订阅候选
  的队列／均衡重排。
- Wails 绑定 `GetGoogleSettings`／`SetGoogleAccountRotationMode`。
- 前端「Google 账号设置」面板、i18n（中／英）、账号页入口。
- 单元测试（Go 轮换与设置、前端面板）。

## Out of Scope

- WebSocket（上游无此传输）。
- 自动激活额度窗口（另开任务，需用户确认会发送真实消息）。
- 不改 Codex／River 的既有设置与轮换实现。

## Existing Behavior

Google 订阅账号总是按路由给出的候选顺序使用，界面上没有轮换设置。

## Expected Behavior

`queue`（默认）保持原顺序；`balanced` 在同池可用订阅账号间轮流分配新请求，
设置读取或候选复核失败时返回明确错误码，不静默退回队列。

---

# Analysis

## Related Modules

- `internal/application/gateway/account_rotation.go`、`river_rotation.go`（参照实现）
- `internal/transport/wails/gateway.go`（启动装配）
- `frontend/src/features/accounts/provider-accounts-view.tsx`

## Dependencies

无新增依赖；沿用 `rotationState` 的平滑加权轮询。

## Risks

- 均衡模式会改变账号选择顺序：只重排候选，不新增重试，也不改变固定账号与故障切换开关。
- 多账号共享同一官方额度，请求数均衡不等于 Token 均衡，界面文案已说明。

---

# Implementation Plan

1. 创建本记录。
2. 迁移 + 设置服务 + 轮换分支 + 装配 + 绑定。
3. 前端面板、i18n 与账号页入口。
4. 补测试，跑 `task verify`。
5. 更新文档（`docs/product/GOOGLE_SUBSCRIPTION.md`、CURRENT），提交。

---

# Progress

- 2026-11-13 09:30：创建记录，完成迁移／服务／轮换／绑定与前端面板。
- Go 与前端针对性测试通过；完整质量门进行中。

---

# Implementation Result

## Changes

- 迁移 `047_google_account_rotation.sql`：新建 `google_settings`（`id=1`，
  `account_rotation_mode`，默认 `queue`，CHECK 限制两个取值）。
- `gateway.GoogleSettingsService`：`Read`／`SetAccountRotationMode`／`AccountRotationMode`，
  未知取值 400、缺行 503，与 River 设置同构。
- `rotateGoogleAccounts`：只接受同提供商、已启用、`subscription_login` 且已授权的候选；
  健康冷却中的账号跳过；全部不可用时返回 `account_pool_cooling` 429；选择用既有
  `rotationState.choose`（平滑加权轮询，权重 1）。
- `account_rotation.go`：`kind == "google"` 且 `authMethod == subscription_login` 时走 Google 分支。
- Wails 绑定 `GetGoogleSettings`／`SetGoogleAccountRotationMode`（已重新生成 bindings）。
- 前端：`GoogleAccountSettings` 弹出面板（与 River 同版式），账号页在
  `provider.kind === "google"` 时显示；新增中／英文案，明确“只对订阅账号生效”。

## Files Changed

- `migrations/047_google_account_rotation.sql`
- `internal/application/gateway/google_settings.go`、`google_rotation.go`、`account_rotation.go`、`service.go`
- `internal/application/gateway/google_rotation_test.go`
- `internal/transport/wails/google_settings.go`、`gateway.go`
- `frontend/src/services/google-settings.ts`、
  `frontend/src/features/accounts/google-account-settings.tsx`（+ 测试）、`provider-accounts-view.tsx`
- `frontend/src/i18n/locales/google-settings-{zh,en}.ts` 与 `zh.ts`／`en.ts`
- `frontend/bindings/**`（重新生成）

## Technical Notes

- 设置每个请求读取一次（`func(context.Context) (string, error)`），保存后立即生效。
- 均衡与 Codex 的平衡模式共用同一份轮询状态机，但各自独立的模式来源与候选池。

---

# Verification

## Tests

- [x] Unit Tests：`go test ./internal/application/gateway/ -run Google` 通过。
- [x] 前端：`vitest run src/features/accounts/google-account-settings.test.tsx` 通过。
- [ ] Integration Tests：多 Google 订阅账号的实机均衡待用户环境（本机仅一个订阅账号）。

## Quality Checks

- [x] Format
- [x] Lint
- [x] Type Check
- [x] Test
- [x] Build
- [x] Verify

## Verification Result

```text
task --force verify
VERIFY_EXIT=0（148 文件 / 902 项前端测试，全部 Go 包与 Windows production 构建）
```

- go test ./internal/application/gateway/ -run Google：轮换顺序、API Key 排除、设置校验通过。
- vitest google-account-settings：4 项通过（保存、失败保留、保存中锁定、读取失败）。

---

# Documentation

- [x] `docs/product/GOOGLE_SUBSCRIPTION.md`：新增轮换方式与生效范围
- [x] CURRENT

---

# Git

## Commit

```text
40f1f92a feat(google): Google 订阅账号支持队列／均衡轮换
```

未打标签、未推送。

---

# Result

Google 账号页现在有「Google 账号设置」入口，可选择队列轮换（默认）或平衡切换；
只影响 Google AI 订阅账号，API Key 账号与固定账号不受影响。

截图中的另两项：WebSocket 对 Google 无对应上游传输，不实现；自动激活额度窗口可行，
但会给账号发送真实消息并消耗额度，需要用户确认后再排期。

---

# Completed

2026-11-13 09:46
