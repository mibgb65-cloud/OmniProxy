# TASK-20260924-003: Frontend i18n (Chinese/English)

## Status

Completed

## Created

2026-09-24 15:12

## Request

“我们先做前端，不着急该后端，首先就是多语言，加入可以选择中英语的多语言。”

## Goal

前端接入中文/英文切换：所有用户可见文案通过翻译层获取，语言选择持久化，切换即时生效。

## Background

技术栈文档未定义 i18n 方案。按项目依赖规则（最小依赖、Typed First、小模块）初始判断：两种语言的静态文案不值得引入 i18n 框架，采用手写类型安全词典。

## Scope

- `frontend/src/i18n/`：zh/en 词典、translate、Provider
- 语言状态接入 Zustand UI store（localStorage 持久化）
- 侧边栏语言切换器
- 全部现有界面文案迁移为翻译 key（导航、Dashboard、占位页）
- 测试与文档同步

## Out of Scope

- 引入 i18next 等第三方依赖（当时判断不必要；后由 TASK-20260924-004 应用户要求迁移）
- 复数规则、日期/数字格式化

## Existing Behavior

界面全部为硬编码英文/中文混合文案，无语言切换能力。

## Expected Behavior

侧边栏底部可切换中/英；导航、Dashboard、占位页、窗口标题即时切换；选择重启后保持。

---

# Analysis

## Related Modules

- `frontend/src/i18n/**`、`frontend/src/stores/ui-store.ts`、`frontend/src/components/common/**`、`frontend/src/app/routes/**`

## Dependencies

- Zustand（已在技术栈中）

## Risks

- 手写方案在文案量增长后可维护性下降（已知，接受）
- 词典 key 与使用方可能不同步（用类型系统约束：en 必须满足 zh 的 key 集合）

---

# Implementation Plan

1. 创建 zh.ts（唯一事实来源）/en.ts（类型受约束）/translation.ts/provider.tsx
2. ui-store 增加 language + persist
3. 侧边栏增加 LanguageSwitcher
4. 全部文案迁移为 key
5. 测试（词典对齐、插值、切换器、跨组件切换）
6. 文档同步（TECH_STACK/OVERVIEW/CODING_RULES/CURRENT）

---

# Progress

- 初始实现为手写词典 + 自定义 useTranslation hook，18 → 19 个测试通过
- 用户随后要求引入正式 i18n 依赖，形成 TASK-20260924-004

---

# Implementation Result

## Changes

- 类型安全词典（zh 为 key 唯一事实来源，en 以 `Record<keyof typeof zh, string>` 约束）
- `useTranslation` hook + I18nProvider（同步 `<html lang>` 与窗口标题）
- 语言状态存入 Zustand UI store 并持久化（含损坏值防御性回退）
- 侧边栏底部分段切换器（折叠态为图标一键切换）
- 全部现有 UI 文案迁移为翻译 key

## Files Changed

- `frontend/src/i18n/{locales/zh.ts,locales/en.ts,translation.ts,provider.tsx,i18n.test.ts}`
- `frontend/src/hooks/use-translation.ts`（后被 TASK-20260924-004 删除）
- `frontend/src/stores/ui-store.ts`、`frontend/src/components/common/{app-sidebar,language-switcher,page-placeholder}.*`、`frontend/src/app/routes/**`、`frontend/src/app/providers.tsx`

## Technical Notes

- 插值采用 `{var}` 自定义语法（迁移到 i18next 后改为 `{{var}}`）

---

# Verification

## Tests

- [x] Unit Tests（词典对齐、插值、切换器、跨组件语言切换）
- [ ] Integration Tests — N/A
- [ ] E2E Tests — N/A

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
PASS（19 个前端测试通过）
```

---

# Documentation

- [x] Tech Stack（记录手写方案及“为何不用 i18next”的说明）
- [x] Architecture（前端职责注明 i18n）
- [x] Coding Rules（新增“用户可见文案必须走 i18n”）
- [x] CURRENT

---

# Deviations

- 原计划：最小依赖手写方案。实际：先落动手写方案，随后按用户要求迁移至 i18next（TASK-20260924-004）。原因：用户明确要求引入 i18n 依赖。

---

# Git

## Commit

```text
0584167 feat(frontend): add Chinese/English internationalization
```

---

# Result

前端具备中/英切换能力，全部文案经翻译层，选择持久化。实现方案后在 TASK-20260924-004 中迁移到 i18next。

---

# Completed

2026-09-24 14:46

---

> Retroactive record: 本记录于 Task Record 工作流建立时补建（见 TASK-20260924-005），工作本身于 2026-09-24 下午完成。
