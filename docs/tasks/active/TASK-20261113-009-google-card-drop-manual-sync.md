# TASK-20261113-009: Google 卡片移除多余的「重新同步到 AI Agent」

## Status

Completed

## Created

2026-11-13 11:10（Asia/Singapore）

## Request

> 这个《重新同步到 AI Agent》不是做了自动的吗？这里就不需要了吧？感觉放到这里很影响体验

## Goal

- Google 订阅账号卡片不再显示「重新同步到 AI Agent」按钮与那段自动同步说明，模型区回到
  “标题 + 刷新 + 模型摘要”的紧凑形态。
- 目录同步仍然自动发生，且保留一条可用的手动重试路径。
- 不减少实际能力：同步失败时用户仍能让它重来。

## Background

- 导入旧版数据后自动同步（TASK-20261113-002）之后，目录同步已覆盖：启动、登录、刷新模型列表、
  导入旧版数据、新建账号。
- `AppService.GoogleSubscriptionModels`（模型列表刷新按钮调用的绑定）在成功读取官方选择器后
  就会 `requestGoogleModelSync()`，因此卡片上的刷新图标本身就是一次同步 + 重试入口。
- 也就是说卡片里的独立按钮是重复入口，还带一段长说明，占用空间且干扰阅读。

## Scope

- `frontend/src/features/google/google-subscription-models.tsx`：移除同步区块。
- 删除 `frontend/src/features/google/google-model-sync.tsx`。
- i18n：删除 5 个同步相关键，把自动同步说明并入模型列表的提示。
- 更新相关测试。

## Out of Scope

- 不改后端同步实现与触发点。
- 不改路由／模型目录页的「同步到客户端」入口（那是另一个页面，仍需要）。

## Existing Behavior

卡片内显示说明 + 按钮，按钮调用 `ModelSync(accountID)`。

## Expected Behavior

卡片内只有模型摘要与刷新；自动同步说明放在展开的模型列表底部，刷新图标同时承担重试。

---

# Analysis

## Related Modules

- `frontend/src/features/google/google-subscription-models.tsx`
- `frontend/src/services/models.ts`（`syncAccountModels` 仍被路由页使用，保留）
- `internal/transport/wails/google_subscription.go`（刷新即触发同步）

## Dependencies

无。

## Risks

- 若用户习惯用该按钮确认“已同步”的反馈，移除后少一处成功提示；改为在模型列表提示里说明
  自动同步范围，刷新图标仍可重试。

---

# Implementation Plan

1. 创建本记录。
2. 删除同步区块与组件，调整提示文案。
3. 更新测试（断言按钮不再出现、刷新仍工作）。
4. `task verify`，提交并构建本地 Beta。

---

# Progress

- 2026-11-13 11:10：创建记录。

---

# Implementation Result

## Changes

- `google-subscription-models.tsx`：移除底部的同步区块，卡片只保留标题、刷新图标与模型摘要；
  展开面板底部的提示改为说明自动同步范围（启动／登录／导入／刷新模型列表）。
- 删除 `google-model-sync.tsx`（唯一使用方就是这张卡片）。
- i18n 删除 5 个仅服务该按钮的键（`autoSync`／`sync`／`syncing`／`synced`／`syncFailed`），
  `models.hint` 合并自动同步说明。
- 测试：改为断言按钮不存在、自动同步说明出现在展开面板内；删除随组件一起失效的同步重试用例。

## Files Changed

- `frontend/src/features/google/google-subscription-models.tsx`
- `frontend/src/features/google/google-subscription-models.test.tsx`
- `frontend/src/features/google/google-model-sync.tsx`（删除）
- `frontend/src/i18n/locales/google-{zh,en}.ts`
- `docs/product/GOOGLE_SUBSCRIPTION.md`

## Technical Notes

- 手动重试没有丢失：`AppService.GoogleSubscriptionModels` 在成功读取官方选择器后会
  `requestGoogleModelSync()`，因此卡片上的刷新图标就是一次同步；路由页的同步入口保留。
- `services/models.ts` 的 `syncAccountModels` 仍被模型目录页使用，未删除。

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
task --force verify
VERIFY_EXIT=0（148 文件 / 901 项前端测试，全部 Go 包与 Windows production 构建）

vitest src/features/google
Test Files 5 passed (5) / Tests 16 passed (16)
```

---

# Documentation

- [x] CURRENT
- [x] `docs/product/GOOGLE_SUBSCRIPTION.md`
- [x] `docs/releases/`：随下一版本发行说明

---

# Git

## Commit

（未开始）

---

# Result

Google 账号卡片回到紧凑形态：不再有重复的手动同步按钮和长说明，同步说明移入模型列表底部。
自动同步与刷新即重试的能力都保留。

---

# Completed

2026-11-13 11:24
