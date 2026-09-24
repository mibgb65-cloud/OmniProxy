# Git Workflow

## Branches

- `main` is the primary branch and stays releasable.
- Short-lived `feature/*`, `fix/*`, `refactor/*` branches for larger work.
- No `develop`, `release/*`, `hotfix/*` or `staging` branches unless a real
  need appears.

## Commit messages

This project uses [Conventional Commits](https://www.conventionalcommits.org/):

```text
<type>(<scope>): <description>
```

Examples:

```text
feat(gateway): add model routing table
fix(config): handle missing config file
refactor(parser): extract shared parser
docs: update architecture documentation
test(config): add malformed config tests
ci: add Windows build workflow
```

### Types

| Type       | Meaning                                  |
| ---------- | ---------------------------------------- |
| `feat`     | New feature                              |
| `fix`      | Bug fix                                  |
| `refactor` | Behavior-preserving refactor             |
| `docs`     | Documentation only                       |
| `test`     | Tests only                               |
| `style`    | Formatting only, no behavior change      |
| `perf`     | Performance improvement                  |
| `build`    | Build system or dependencies             |
| `ci`       | CI/CD                                    |
| `chore`    | Maintenance                              |

Suggested scopes: `gateway`, `provider`, `scheduler`, `routing`, `account`,
`quota`, `usage`, `db`, `logger`, `wails`, `frontend`, `control-api`, `ci`,
`docs`.

## Commit principles

- One commit focuses on one problem.
- Keep commits atomic and independently understandable.
- Never mix unrelated changes in one commit.
- Keep commit size reasonable.

Bad examples:

```text
update
fix stuff
feat: add feature and fix ui and update dependencies
```

## AI agent Git safety rules

Without explicit user authorization, an agent must not:

- `git push --force`
- `git reset --hard`
- Delete existing commits or rewrite public branch history
- Rebase already-shared history
- Delete uncommitted user changes
- Commit files unrelated to the current task
- Commit `.env`, tokens, API keys, passwords or private keys

If the working tree contains changes that do not belong to the current task:
**keep them — do not overwrite, delete, or commit them.**

## Versioning

Semantic Versioning (`MAJOR.MINOR.PATCH`). The current version lives in
`internal/version` and `build/config.yml`; keep them in sync.
