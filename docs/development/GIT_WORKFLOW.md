# Git Workflow

## Branches

本机日常开发与启动统一使用 `E:/go/new/wails3-v/OmniProxy`，当前开发分支为 `main`。需要切换分支时在同一目录使用 `git switch`；不要把多个长期工作树当作项目备份。

Git worktree 是同一仓库的额外工作目录，共享提交历史和分支。临时工作树使用结束后，通过 `git worktree remove` 清理；清理前保留未提交的独有内容，移除目录不删除分支或提交。

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

应用和前端包遵循 Semantic Versioning，完整预发布版本保存在 `internal/version` 与 `frontend/package.json`。Windows `build/config.yml`／NSIS 使用数字基础版本，PE 数字资源采用四段数字，`build/windows/info.json` 的 ProductVersion 保留完整预发布字符串。发布时统一这些版本映射，不能将 Beta 后缀写入数字版本字段；详见 `docs/product/UPGRADE.md`。
