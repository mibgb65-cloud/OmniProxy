# AGENTS.md

This file is the highest-level development guide for AI agents working on
OmniProxy. It exists so that any agent (Codex, Claude Code, Cursor, Gemini CLI,
…) follows the same rules without relying on chat history.

## 1. Authoritative sources

When documents disagree, trust them in this order:

1. Code and tests (the implemented truth)
2. `docs/product/PRD.md` (what the product is and is not)
3. `docs/architecture/OVERVIEW.md` (components and boundaries)
4. `docs/architecture/TECH_STACK.md` (technology baseline)
5. `docs/development/*` (coding, Git, testing, build rules)
6. `docs/decisions/` (accepted architecture decisions)
7. `docs/plans/CURRENT.md` (short-term status dashboard)
8. `docs/tasks/` (permanent task history: why, how, verification, commits)

Never invent product requirements. If information is missing, mark it `TBD`
or `Needs confirmation` — do not fabricate.

## 2. Before starting a larger task, read

```text
docs/product/PRD.md
docs/architecture/OVERVIEW.md
docs/architecture/TECH_STACK.md
docs/development/CODING_RULES.md
docs/plans/CURRENT.md
```

Additionally, when the task involves:

- Git → `docs/development/GIT_WORKFLOW.md`
- Tests → `docs/development/TESTING.md`
- Building/packaging → `docs/development/BUILD.md`
- Architecture changes → check `docs/decisions/` and consider a new ADR

If you are continuing an existing task, read its Task Record in
`docs/tasks/active/` first.

Simple changes do not require reading the whole docs tree.

## 3. Non-negotiable rules

- Do not change the core technology stack (see `docs/architecture/TECH_STACK.md`).
- Do not introduce large dependencies without a clear, current need.
- Understand and extend the existing implementation before rewriting anything.
- Do not rewrite working modules without reason.
- Avoid duplicated code; avoid over-engineering for imaginary requirements.
- Respect module boundaries: Core must not depend on Wails or React.
- Fix root causes, not symptoms.
- Never silently swallow errors; never delete reasonable tests to make CI
  green; never bypass the type system to avoid proper modeling.
- Do not change public API behavior without evaluating compatibility.
- Do not modify code unrelated to the current task.

## 4. Standard development flow

```text
User raises a requirement
  → create the Task Record (docs/tasks/active/, Status: Planned)
  → understand the requirement
  → read the relevant docs
  → inspect the existing code
  → confirm the impact scope
  → fill in Analysis and Implementation Plan
  → Status → In Progress
  → implement
  → test
  → run `task verify`
  → fill in Implementation Result and Verification
  → update the affected docs
  → review `git diff`
  → commit and write the commit hash into the Task Record
  → Status → Completed, move to docs/tasks/completed/
```

The Task Record must exist before any code, config or project file is
modified. Never implement first and backfill the record afterwards.

Do not add ceremonial steps that serve no purpose.

## 5. Verification

`task verify` is the single quality gate:

```text
format check → lint → typecheck → test → build
```

Run it after any significant change. CI runs the same gate. The verification
result must be written back into the Task Record.

## 6. Security rules

- Never commit real API keys, tokens, passwords, private keys or `.env` files.
- Never hardcode secrets in source code.
- Never log credentials: all logs pass through the central Redactor
  (`internal/infrastructure/logger`).
- Credentials live in the OS credential store (DPAPI on Windows); the database
  only stores `credential_ref` values.

## 7. Git safety rules

Without explicit user authorization, an agent must not:

- `git push --force`, `git reset --hard`, delete commits, or rewrite shared
  history
- rebase already-shared history or delete uncommitted user changes
- commit files unrelated to the current task, or commit `.env`

If the working tree contains changes that do not belong to the current task:
leave them untouched — do not overwrite, delete or commit them.

After committing, record the commit hash in the Task Record.

## 8. Task Records

任何开发任务，无论大小，在修改代码、配置或项目文件之前，都必须创建 Task Record。

路径：

`docs/tasks/`

完整规范：

`docs/tasks/README.md`

基本流程：

Request
→ Task Record
→ Analysis
→ Implementation Plan
→ Implementation
→ Verification
→ Documentation
→ Git
→ Result

禁止先实现再补任务记录。

任务完成后必须在 Task Record 中填写实际 Implementation Result 和 Verification Result。

`docs/tasks/` 是任务历史的 Source of Truth。

`docs/plans/CURRENT.md` 只维护当前任务概览。
