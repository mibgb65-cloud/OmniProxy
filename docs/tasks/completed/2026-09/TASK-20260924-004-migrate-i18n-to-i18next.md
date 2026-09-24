# TASK-20260924-004: Migrate i18n to i18next + react-i18next

## Status

Completed

## Created

2026-09-24 15:13

## Request

“直接引入i18n多语言依赖吧。”

## Goal

将 TASK-20260924-003 的手写 i18n 实现迁移到标准 i18next 生态（i18next + react-i18next），保留类型安全与既有交互行为。

## Background

用户决定引入正式 i18n 依赖。需在迁移后保持：key 类型安全、Zustand 持有语言偏好、切换即时生效、持久化不失效。

## Scope

- 引入 i18next + react-i18next
- 重构 `frontend/src/i18n/`（config/i18next.d.ts/provider）
- 组件改用 react-i18n 的 `useTranslation`
- 插值语法迁移 `{var}` → `{{var}}`
- 测试初始化共享实例；文档同步

## Out of Scope

- 语言自动检测（navigator 探测）：初始语言仍取自 Zustand store，默认中文
- 懒加载语言包（两种语言体积极小，无必要）

## Existing Behavior

手写词典 + 自定义 `useTranslation` hook；`translate()` 自行实现 `{var}` 替换。

## Expected Behavior

i18next 实例在应用启动时初始化；`t()` 仅接受 zh 词典中存在的 key（编译期约束）；语言切换经 store → I18nProvider → `changeLanguage`；行为与迁移前一致。

---

# Analysis

## Related Modules

- `frontend/src/i18n/**`、`frontend/src/components/common/**`、`frontend/src/app/routes/index.tsx`、`frontend/src/test/setup.ts`

## Dependencies

- `i18next@26.4.2`、`react-i18next@17.0.15`

## Risks

- i18next 为主版本 26，需验证 resources 结构与类型增强行为（已实测）
- 测试环境需在渲染前完成实例初始化（setup 中 `await initI18n()`）

---

# Implementation Plan

1. 安装依赖
2. 词典插值改为 `{{var}}`；新增 i18next.d.ts 模块增强（t() key 类型约束）
3. config.ts：语言在前的 resources、`keySeparator: false`、初始语言取自 store
4. provider.tsx：模块加载即初始化 + effect 同步 store → i18next，维护 document lang/title
5. 组件迁移到 react-i18n；删除自定义 hook
6. 测试：setup 初始化、组件测试包裹 I18nProvider、i18next 实例级测试
7. 文档同步；task verify；提交

---

# Progress

- 迁移后所有翻译返回 key 本身：调试发现 `resources` 误用命名空间在前结构（`{ translation: { zh, en } }`），i18next 要求语言在前（`{ zh: { translation: {...} } }`），导致 store 中 bundle 为空；修正后全部恢复
- 用临时探针组件验证模块增强生效：错误 key 产生编译错误

---

# Implementation Result

## Changes

- i18next + react-i18next 接入；resources 语言在前、扁平点分 key + `keySeparator: false`
- i18next 模块增强：`t()` 只接受 `locales/zh.ts` 中存在的 key（拼写错误 = 编译错误）
- 语言偏好仍由 Zustand UI store 持有并持久化；I18nProvider 是唯一同步点（`changeLanguage`），并维护 `<html lang>`/窗口标题
- 插值统一为 i18next `{{var}}` 语法
- 测试在 setup 中 `await initI18n()`，组件测试经 I18nProvider 渲染

## Files Changed

- `frontend/package.json`、`frontend/pnpm-lock.yaml`
- `frontend/src/i18n/{config.ts,i18next.d.ts,provider.tsx,translation.ts,i18n.test.ts,locales/*.ts}`
- `frontend/src/components/common/{app-sidebar,language-switcher,page-placeholder}.*`、`frontend/src/app/routes/index.tsx`、`frontend/src/test/setup.ts`
- 删除 `frontend/src/hooks/use-translation.ts`

## Technical Notes

- i18next 的 `resources` 必须是 `{ [lng]: { [ns]: keys } }`（语言在前），命名空间在前的写法会静默得到空 bundle
- `keySeparator: false` 与模块增强兼容：扁平点分 key 既能在运行时按字面查找，也保留完整 key 联合类型

---

# Verification

## Tests

- [x] Unit Tests（词典对齐、i18next 实例双语/插值/回退、切换器、跨组件切换）
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
PASS（18 个前端测试通过；另以临时探针验证错误 key 编译报错）
```

---

# Documentation

- [x] Tech Stack（i18next 方案与资源结构说明）
- [x] Coding Rules（文案规则指向 react-i18n）
- [x] CURRENT

---

# Deviations

- 原计划与最终实现一致。执行中修复了 resources 结构错误（语言在前），属于实现细节修正，不改变方案。

---

# Git

## Commit

```text
a2c955d refactor(frontend): migrate i18n to i18next + react-i18next
```

---

# Result

i18n 迁移至 i18next 生态完成，行为与类型安全保持不变，`task verify` 全绿。

---

# Completed

2026-09-24 15:04

---

> Retroactive record: 本记录于 Task Record 工作流建立时补建（见 TASK-20260924-005），工作本身于 2026-09-24 下午完成。
