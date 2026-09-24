# Task Records

`docs/tasks/` 是 OmniProxy 任务历史的 Source of Truth。

核心原则：

> No task without a record.
> 没有任务记录，不开始实现。

任何开发工作——包括但不限于：新功能、Bug 修复、UI 调整、文案修改、重构、性能优化、依赖升级、配置修改、文档修改、CI 修改、构建修改——在修改项目代码或配置之前，都必须先创建对应的 Task Record。

规范性要求见项目根目录《新项目初始化规范.md》§44。本文件是日常操作指南。

---

## 1. 目录结构

```text
docs/tasks/
├─ README.md
├─ active/          # Planned / In Progress / Blocked
└─ completed/       # Completed / Cancelled，可按年月归档
    └─ 2026-09/
```

任务完成后，Task 文件从 `active/` 移动到 `completed/`（建议按 `YYYY-MM` 归档）。

## 2. Task ID 与文件名

ID 格式：

```text
TASK-YYYYMMDD-NNN
```

同一天按任务创建顺序递增（001、002、003…）。

文件名：

```text
TASK-YYYYMMDD-NNN-short-description.md
```

描述使用简短、明确的英文，例如：

```text
TASK-20260924-001-node-latency-monitoring.md
TASK-20260924-002-fix-settings-button-label.md
```

## 3. 任务状态

只使用以下状态：

| 状态         | 含义                                               |
| ------------ | -------------------------------------------------- |
| `Planned`    | 已记录，尚未开始修改                               |
| `In Progress`| 正在分析或实现                                     |
| `Blocked`    | 因依赖、缺少信息、技术问题等暂时无法继续           |
| `Completed`  | 实现、验证和文档更新已完成                         |
| `Cancelled`  | 明确不再执行（不删除文件，记录取消原因）           |

## 4. 强制流程

```text
用户提出需求
↓
创建 Task Record（Status: Planned）
↓
记录原始需求（Request）
↓
分析现有代码（Analysis）
↓
补充实现计划（Implementation Plan）
↓
Status → In Progress
↓
实现
↓
测试
↓
Verify（task verify）
↓
记录实际实现情况（Implementation Result）
↓
记录验证结果（Verification）
↓
更新相关 docs
↓
检查 Git diff
↓
提交 Commit
↓
把 Commit 写入 Task Record
↓
Status → Completed，移动到 completed/
```

禁止先实现、后补任务记录。

## 5. Task Record 模板

````md
# TASK-YYYYMMDD-NNN: Task Title

## Status

Planned

## Created

YYYY-MM-DD HH:mm

## Request

记录用户提出的原始需求。尽量保留原始意图，不要过度改写。

## Goal

这个任务最终需要达到什么结果。

## Background

必要背景。如果没有：N/A

## Scope

本任务预计包含：

- ...

## Out of Scope

本任务明确不处理：

- ...

## Existing Behavior

修改前系统当前行为。

## Expected Behavior

完成后系统预期行为。

---

# Analysis

## Related Modules

预计涉及：

- ...

## Dependencies

可能涉及：

- ...

没有：N/A

## Risks

可能风险。没有明显风险：Low

---

# Implementation Plan

1. ...
2. ...
3. ...

---

# Progress

只记录重要节点（定位到根因、完成核心实现、增加测试、修改方案、遇到阻塞）。
不要记录每一次代码编辑。

---

# Implementation Result

任务完成后填写。

## Changes

实际完成：

- ...

## Files Changed

主要修改文件：

- `...`

无需机械列出自动生成文件。

## Technical Notes

值得保留的实现说明。如果没有：N/A

---

# Verification

## Tests

- [ ] Unit Tests
- [ ] Integration Tests
- [ ] E2E Tests

根据项目实际情况填写。不适用：N/A

## Quality Checks

- [ ] Format
- [ ] Lint
- [ ] Type Check
- [ ] Test
- [ ] Build
- [ ] Verify

## Verification Result

例如：

```text
task verify
PASS
```

存在失败必须说明原因。

---

# Documentation

检查是否更新：

- [ ] PRD
- [ ] Architecture
- [ ] Tech Stack
- [ ] Coding Rules
- [ ] Testing
- [ ] Build
- [ ] ADR
- [ ] CURRENT

没有需要更新的文档时：

```text
No documentation changes required.
```

---

# Deviations

实际实现与最初 Implementation Plan 的区别。如果没有：None.

发生改变时说明：原计划 / 实际方案 / 改变原因。

---

# Git

## Commit

```text
<commit hash> <commit message>
```

如果当前任务不允许 Agent 自动提交：

```text
Not committed by Agent.
```

---

# Result

简要总结最终结果。

---

# Completed

YYYY-MM-DD HH:mm
````

## 6. 小任务

即使只是把按钮文字从 Save 改成 Apply，也创建 Task Record，但允许使用简化内容（省略 Analysis、Progress 等章节，保留 Request / Goal / Implementation Result / Verification / Git / Result）。任务文档复杂度与任务复杂度匹配，不要为文字修改生成数百行说明。

## 7. Bug Fix 任务

除模板外，建议增加：

```md
## Root Cause

问题根因。

## Fix

实际修复方式。

## Regression Prevention

如何避免再次发生（尽可能增加回归测试）。
```

原则：Bug Fix 不只记录修了什么，还要记录为什么出现。

## 8. Refactor 任务

必须明确 Behavior Before / Behavior After。Refactor 默认不改变外部行为；如果改变，必须明确记录。

## 9. Blocked / Cancelled

- `Blocked`：增加 `## Blocker`（阻塞原因）和 `## Required To Continue`（恢复需要什么）。不得假装已完成。
- `Cancelled`：增加 `## Cancellation Reason`，移动到 `completed/`。不要删除文件——避免未来 Agent 重复提出已被否决的方案。

## 10. 完成标准

只有同时满足以下条件才能标记 `Completed`：

- [ ] 实现完成
- [ ] 必要测试完成
- [ ] 必要质量检查通过
- [ ] Verify 已运行，或明确说明为什么无法运行
- [ ] 相关文档已同步
- [ ] Implementation Result 已写入
- [ ] Deviations 已记录
- [ ] Git 状态已检查
- [ ] Commit 已完成并写入 Task Record（若项目要求 Agent 提交）

## 11. 与其他体系的关系

- **CURRENT.md**：只是当前状态看板（In Progress / Recently Completed 列 TASK ID），完整信息在 Task 文件中。`docs/tasks/` 是 Source of Truth。
- **Commit**：一个 Task 可对应 1 个或多个 Commit。Task 描述完整需求生命周期；Commit 描述原子变更。Task Record 必须记录相关 Commit。
- **ADR**：普通任务用 Task Record；任务过程中产生重大架构决策时同时创建 ADR，并在 Task 中引用（`Related ADR: ADR-XXX-....md`）。Task 记录"这次做了什么"，ADR 记录"为什么长期这样选"。

## 12. 追溯记录

在本工作流建立之前已完成的工作，可以补建 Task Record 并在文件开头注明：

```text
> Retroactive record: 本记录于 Task Record 工作流建立时补建（见 TASK-YYYYMMDD-NNN），
> 工作本身在此之前已完成。
```

禁止为从未发生过的工作伪造任务记录。
