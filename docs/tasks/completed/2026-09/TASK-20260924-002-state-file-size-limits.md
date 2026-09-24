# TASK-20260924-002: State File and Function Size Limits in Coding Rules

## Status

Completed

## Created

2026-09-24 15:11

## Request

用户询问：“有做手写代码文件中的代码行数最大不超过 300 行的要求吗？”

## Goal

核对项目现行的大小约束规定；如发现文档缺口则修复，确保编码规范入口明确记录该限制。

## Background

技术基线《技术架构选择.md》§32 规定“单文件 ≤ 200 行，超过约 200 行主动评估拆分，禁止 2000+ 行组件 / God 文件”，但初始化时编写的 `docs/development/CODING_RULES.md` 未写入该数字，违反“同一事实只有一个主要来源”的文档原则。

## Scope

- 核对规范文档与技术基线中的长度约束
- 在 CODING_RULES.md 中补充文件/函数长度规则

## Out of Scope

- 修改长度数值本身（维持技术基线的 200 行标准）
- 改造任何代码

## Existing Behavior

CODING_RULES.md 只有“保持函数小巧”等定性描述，无具体数值；《新项目初始化规范.md》将具体限制下放给技术栈文档规定。

## Expected Behavior

CODING_RULES.md 明确写出：手写文件 ≤ 200 行、超长主动评估拆分、禁止 God 文件、函数约 50 行软性指引、生成文件豁免。

---

# Analysis

## Related Modules

- `docs/development/CODING_RULES.md`
- `技术架构选择.md` §32（事实来源）

## Dependencies

N/A

## Risks

Low

---

# Implementation Plan

1. 核对三处文档（初始化规范、技术基线、CODING_RULES）的现行规定
2. 实测现有手写文件行数确认合规
3. 在 CODING_RULES.md 的 File Organization / Functions 章节补充明确限制

---

# Progress

- 确认项目无 300 行要求；现行有效规定为技术基线 §32 的 200 行标准
- 实测最大手写文件为 `internal/platform/profile/profile.go`（148 行），全部合规

---

# Implementation Result

## Changes

- CODING_RULES.md 新增“File size limits”小节：≤ 200 行、超长评估拆分、禁止 2000+ 行组件与 God Service/Store/Provider/Handler、超长需在 PR 说明理由
- Functions 小节新增约 50 行 / 一屏内的软性拆分指引

## Files Changed

- `docs/development/CODING_RULES.md`

## Technical Notes

N/A

---

# Verification

## Tests

N/A（纯文档修改）

## Quality Checks

- [x] Format — N/A（根目录文档不经过 Prettier 门禁）
- [x] Lint — N/A
- [x] Type Check — N/A
- [x] Test — N/A
- [x] Build — N/A
- [x] Verify — 文档修改未单独运行；本次为文档变更，无代码影响

## Verification Result

```text
文档变更，无代码影响。未运行 task verify（无代码/配置改动）。
```

---

# Documentation

- [x] Coding Rules（本任务的修改对象）
- 其他文档无需更新

---

# Deviations

None.

---

# Git

## Commit

```text
1cf1fd2 docs: state file and function size limits in coding rules
```

---

# Result

编码规范入口现已明确记录 200 行文件上限与函数长度指引，与技术基线保持一致。

---

# Completed

2026-09-24 12:15

---

> Retroactive record: 本记录于 Task Record 工作流建立时补建（见 TASK-20260924-005），工作本身于 2026-09-24 中午完成。
