# OmniProxy Documentation

`docs/` is the long-term shared knowledge base of the project. It describes the
current real state of the project — not chat logs, not wishful thinking.

## Directory map

```text
product/        Product requirements and product boundaries
architecture/   System architecture and technology stack
development/    Coding, Git, testing and build rules
plans/          Current development status dashboard
tasks/          Task records — the source of truth for task history
decisions/      Architecture Decision Records (ADRs)
```

## Which document should I read?

| Task                                   | Read                                             |
| -------------------------------------- | ------------------------------------------------ |
| Any non-trivial change                 | `../AGENTS.md`                                   |
| "What are we building?"                | `product/PRD.md`                                 |
| "How is it put together?"              | `architecture/OVERVIEW.md`                       |
| "Which libraries/versions?"            | `architecture/TECH_STACK.md`                     |
| "How should I write this code?"        | `development/CODING_RULES.md`                    |
| "How do I commit?"                     | `development/GIT_WORKFLOW.md`                    |
| "Where do tests go / how to run them?" | `development/TESTING.md`                         |
| "How do I build / package / run?"      | `development/BUILD.md`                           |
| "What is happening right now?"         | `plans/CURRENT.md`                               |
| "Why was X decided?"                   | `decisions/`                                     |
| "Why was this change made, how, and which commit?" | `tasks/` (find by TASK ID)         |

## Rules

- One fact has one home. Do not duplicate content across documents.
- Documentation follows implementation: update docs in the same change as the
  code they describe.
- Keep `plans/CURRENT.md` short; it is shared short-term context between
  agents, not a changelog.
- Record irreversible or cross-cutting technical choices as ADRs. Do not
  fabricate ADRs for decisions that were never made.
- Every development task starts with a Task Record in `tasks/active/` before
  any code is touched — see `tasks/README.md`.
