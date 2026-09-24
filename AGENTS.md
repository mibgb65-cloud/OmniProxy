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
7. `docs/plans/CURRENT.md` (short-term status)

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
Understand the requirement
  → read the relevant docs
  → inspect the existing code
  → confirm the impact scope
  → plan the implementation
  → implement
  → test
  → run `task verify`
  → update the affected docs
  → update docs/plans/CURRENT.md
  → review `git diff`
```

Do not add ceremonial steps that serve no purpose.

## 5. Verification

`task verify` is the single quality gate:

```text
format check → lint → typecheck → test → build
```

Run it after any significant change. CI runs the same gate.

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
