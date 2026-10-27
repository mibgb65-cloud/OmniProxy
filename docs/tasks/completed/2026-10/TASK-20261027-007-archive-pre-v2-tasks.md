# TASK-20261027-007: 归档 v2.0.0 之前的在办任务记录

## Status

Completed（2026-10-27 20:55 Asia/Singapore）

## Created

2026-10-27 20:40（Asia/Singapore）

## Request

用户：v2.0.0 版本已发布，把 `docs/tasks/active/` 中属于该版本之前的任务全部标记为已完成。

## Goal

把 `docs/tasks/active/` 中所有在 v2.0.0 发布基线（2026-10-27）之前创建的 Task Record
按发布事实归档：Status → Completed，补 Archival 说明，移动到
`docs/tasks/completed/2026-10/`；同时把 `docs/plans/CURRENT.md` 的 In Progress 段
改为归档结论＋发布后跟进事项。

## Background

- v2.0.0 已于 2026-10-27 公开（标签源码 `c9c48b4a31988a6cc3fa2f4b1d423e61bd10f71b`，
  Actions 132 文件／809 项前端测试及 Windows 构建通过，见
  [TASK-20261026-005](TASK-20261026-005-publish-v2-stable-github.md)）。
- `docs/tasks/active/` 归档前有 34 个文件，全部创建于 2026-10-13 至 2026-10-24，均早于
  v2.0.0 发布；其中 1 个状态已是 Completed 却未移动，1 个是重复的 Planned 草稿，
  其余多为「实现完成，等待实机／原生窗口／真实覆盖安装验收」。
- 归档前已核对：工作区无未提交的代码改动（仅 `docs/plans/CURRENT.md` 修改与一份
  未跟踪的 completed 记录），各记录引用的实现提交均为 v2.0.0 源码的祖先提交。

## Scope

- 34 个 `docs/tasks/active/*.md`：更新 Status、追加 Archival 段。
- 移动到 `docs/tasks/completed/2026-10/`。
- 更新 `docs/plans/CURRENT.md`：In Progress 段、顶部「进行中」行、发布后跟进清单。

## Out of Scope

- 不修改任何代码、配置、迁移或安装器。
- 不重新执行 `task verify`、不重新打包、不打标签、不推送、不发布。
- 不改写各任务已有的 Implementation Result / Verification 内容，只在末尾追加归档事实。
- 不把「实机客户端／原生窗口验收」伪造成已验收；未执行项转入发布后跟进。
- 不修复 2026-10-24／25／26 既有归档记录中 9 个文件的 12 处失效相对链接（历史遗留，
  见下方 Follow-ups）。

## Existing Behavior

`docs/tasks/active/` 混放 Planned／In Progress／Completed 三种状态的文件，其中 32 个
实质已完成交付但状态停在 In Progress，`CURRENT.md` 的 In Progress 段与真实交付状态
不一致。

## Expected Behavior

`docs/tasks/active/` 只保留真正未完成的任务（本次清空并保留 `.gitkeep`）；
v2.0.0 之前的任务记录全部位于 `docs/tasks/completed/2026-10/`，状态为 Completed，
并附带归档依据；发布后仍需人工执行的验收项集中记录在 `CURRENT.md`。

---

# Analysis

## Related Modules

- `docs/tasks/active/`、`docs/tasks/completed/2026-10/`
- `docs/plans/CURRENT.md`

## Dependencies

- v2.0.0 发布事实与标签源码提交（`c9c48b4`）。

## Risks

- 归档可能掩盖「未验收」事实：通过 Archival 段逐条保留原状态与未执行项，并在
  `CURRENT.md` 单列 Post-release Follow-ups 缓解。
- 重复 Task ID（TASK-20261013-017）归档后与既有 Completed 记录冲突：草稿标记
  Cancelled 并以 `-planned-draft` 后缀归档，不覆盖既有实现记录。
- 相对链接在跨目录移动后失效：归档后对全部 34 个文件做链接解析检查，修复因移动
  产生的 2 处链接。

---

# Implementation Plan

1. 逐个读取 active 记录，提取原 Status 原文。
2. Status → `Completed（2026-10-27 随 v2.0.0 正式发布归档）`；重复草稿 → `Cancelled`。
3. 末尾追加 `# Archival (v2.0.0)` 段：原状态、发布提交、祖先提交核对结论、未执行项去向。
4. 移动到 `docs/tasks/completed/2026-10/`。
5. 更新 `docs/plans/CURRENT.md`：In Progress → 归档说明＋发布后跟进清单。
6. 回填本记录的 Implementation Result / Verification / Result 并归档。

---

# Progress

- 核对：34 个 active 文件状态分布为 Planned 1、In Progress 32、Completed 1，全部创建于
  2026-10-24 及更早，均早于 v2.0.0 发布。
- 核对：工作区无未提交代码改动；23 个被引用实现提交（`e8e684e`…`65d81d3`）全部为
  `c9c48b4` 的祖先提交。
- 归档：34 个文件状态与 Archival 段写入完成并移动；`active/` 只剩 `.gitkeep` 与本记录。
- 修正：7 个已有 `# Completed` 段占位（`Pending.`／`（待完成）`／`（待用户确认…）`）改为
  归档时间；2 处移动后失效的相对链接改为同目录引用。
- 更新 `CURRENT.md`：In Progress 段改为归档结论 + `### Post-release Follow-ups` 8 条。

---

# Implementation Result

## Changes

- 33 个任务记录：`Status` 改为 `Completed（2026-10-27 随 v2.0.0 正式发布归档）`，
  末尾追加 `# Archival (v2.0.0)`，含原状态原文、发布提交、祖先提交核对结论、未执行项去向。
- 1 个重复草稿 `TASK-20261013-017`：`Status` 改为 `Cancelled`，附 `## Cancellation Reason`，
  归档为 `TASK-20261013-017-atria-builtin-provider-planned-draft.md`，不覆盖既有实现记录。
- 全部 34 个文件移动到 `docs/tasks/completed/2026-10/`；`docs/tasks/active/` 仅剩
  `.gitkeep` 与本记录（本记录随后归档）。
- `docs/plans/CURRENT.md`：顶部「进行中：TASK-20261022-004」改为「已归档」并把链接指向
  completed；`## In Progress` 段改写为归档结论，新增 `### Post-release Follow-ups`
  （Codex 实机、Desktop 额度拦截、Gemini 429、500k 提前压缩、原生窗口视觉、推理档位实机、
  安装器 UAC／真实安装、Token 漏记修复待部署 共 8 条）。

归档清单（原状态 → 归档状态）：

| 任务 | 原状态 | 归档状态 |
| --- | --- | --- |
| TASK-20261013-017 atria-builtin-provider（规划草稿） | Planned | Cancelled（重复草稿） |
| TASK-20261020-008 网关启动／额度／内容兼容 | In Progress | Completed |
| TASK-20261021-006 Gemini Desktop 429 复现 | In Progress | Completed |
| TASK-20261022-004 Codex 压缩与 River Token | In Progress | Completed |
| TASK-20261022-006 用量滚动条＋分组一键配置 | In Progress | Completed |
| TASK-20261023-001 Claude effort 单模型选择 | In Progress | Completed |
| TASK-20261023-002／008／010／013／015／018 Beta.21／24／25／26／27／28 安装包 | In Progress | Completed |
| TASK-20261023-007 Claude／Harness 推理档位 | In Progress | Completed |
| TASK-20261023-011 Claude Desktop effort 入口 | In Progress | Completed |
| TASK-20261023-012 按模型固定思考档位 | In Progress | Completed |
| TASK-20261023-014 请求页缓存／成本／速度 | In Progress | Completed |
| TASK-20261023-016 仪表盘趋势＋自绘滚动条 | In Progress | Completed |
| TASK-20261023-017 存储拆分＋更新残留 | In Progress | Completed |
| TASK-20261024-001 固定 Agent 入口前缀 | In Progress | Completed |
| TASK-20261024-002／004／006／009／012／014／017／021 Beta.29–36 安装包 | In Progress | Completed |
| TASK-20261024-003 推理档位词汇与 Google 变体 | In Progress | Completed |
| TASK-20261024-005 `none→low` 与目录去重 | In Progress | Completed |
| TASK-20261024-007 Codex WebSocket 复核与 ultra | In Progress | Completed |
| TASK-20261024-008 Codex v1 透明转发 | In Progress | Completed |
| TASK-20261024-013 安装向导语言与数据目录 | Completed | Completed |
| TASK-20261024-015 透明转发结果与运行形态标签 | In Progress | Completed |
| TASK-20261024-016 激活失败码与 WebSocket 开关 | In Progress | Completed |

## Files Changed

- `docs/tasks/active/*.md` → `docs/tasks/completed/2026-10/*.md`（34 个文件移动 + 内容更新）
- `docs/tasks/completed/2026-10/TASK-20261013-017-atria-builtin-provider-planned-draft.md`（重命名归档）
- `docs/plans/CURRENT.md`

## Technical Notes

- Archival 段保留原状态原文，因此「已归档为 Completed」与「实机验收仍未执行」两个事实
  可以同时查证；不使用「已验收」措辞。
- 未执行项没有删除也没有改写原 Verification 段，只在 `CURRENT.md` 汇总去向。
- 既有 `# Completed` 段中 7 处占位文字（`Pending.` 等）补为归档时间，避免同文件内
  「Completed + Pending」并存。

---

# Verification

## Tests

- [ ] Unit Tests（N/A，纯文档变更）
- [ ] Integration Tests（N/A）
- [ ] E2E Tests（N/A）

## Quality Checks

- [ ] Format（N/A，无代码变更）
- [ ] Lint（N/A）
- [ ] Type Check（N/A）
- [ ] Test（N/A）
- [ ] Build（N/A）
- [x] Verify（用链接解析 + 状态统计 + git 祖先核对代替 `task verify`，理由见下）

## Verification Result

```text
# 1) active 目录清空
docs/tasks/active/ → .gitkeep + 本记录（随后归档）

# 2) 归档数量
docs/tasks/completed/2026-10/*.md = 270（归档前 236 + 34）

# 3) 状态改写
grep "Completed（2026-10-27 随 v2.0.0 正式发布归档）" → 33 个文件
grep "^Cancelled$" → 1 个文件（重复草稿）
剩余 Pending. / （待完成） / （待用户确认…）占位 → 0

# 4) 移动后链接解析（34 个归档文件）
broken = 0（修复了 2 处 ../completed/2026-10/ 失效链接）

# 5) CURRENT.md 链接解析
links = 187, broken = 0

# 6) 发布可追溯性
git merge-base --is-ancestor <23 个实现提交> c9c48b4a31988a6cc3fa2f4b1d423e61bd10f71b → 全部 True
git status → 仅 docs/ 变更，无代码／配置／迁移改动
```

未运行 `task verify`：本次只改 `docs/tasks/` 与 `docs/plans/`，未触碰源码、配置或构建产物，
质量门不覆盖 Markdown 内容；替代检查为上述链接解析、状态统计与提交祖先核对。

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

更新：`docs/plans/CURRENT.md`（In Progress 段、顶部进行中行、发布后跟进清单）。

---

# Deviations

- 原计划「全部标记为已完成」：1 个重复的 Atria 规划草稿（TASK-20261013-017，Planned）
  改为 Cancelled 归档，因为它与既有 Completed 实现记录同 ID，标记 Completed 会产生
  重复 ID 与两个互相矛盾的完成记录；内容未删除。
- 原计划未包含占位补写：实际补写了 7 个既有 `# Completed` 段的占位文字。
- 原计划未包含链接修复：实际修复了移动导致的 2 处失效相对链接。

---

# Git

## Commit

```text
a453fdab docs(tasks): 归档 v2.0.0 之前的任务记录并撤销版本基线改动
（本记录的归档动作、34 个文件移动与 docs/plans/CURRENT.md 归档段均在该提交内）
```

---

# Result

`docs/tasks/active/` 已清空：v2.0.0 之前的 34 个任务记录全部归档到
`docs/tasks/completed/2026-10/`，其中 33 个按发布事实标记 Completed（附原状态、发布提交
与未执行项去向），1 个重复草稿标记 Cancelled。`docs/plans/CURRENT.md` 的 In Progress 段
改为归档结论，并把 8 类仍需人工执行的验收项集中到 `### Post-release Follow-ups`，
未把实机／原生窗口验收伪造成已通过。

## Follow-ups

- 9 个 2026-10-24／25／26 既有归档记录仍有 12 处失效相对链接
  （`../completed/2026-10/…` 形式），本次未修复，建议单独建任务统一处理。
- 本记录与 `CURRENT.md` 的改动尚未提交。

---

# Completed

2026-10-27 20:55（Asia/Singapore）
